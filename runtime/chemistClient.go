package runtime

import "fmt"

import (
	sc "google.golang.org/api/servicecontrol/v1"
	sm "google.golang.org/api/servicemanagement/v1"

	scp "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
	grpcOauth "google.golang.org/grpc/credentials/oauth"
	"github.com/golang/protobuf/ptypes/timestamp"
	"golang.org/x/net/context"
	"time"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"crypto/x509"
	"sync"
	"errors"

)

const (
	scope = sm.CloudPlatformScope
	serverAddr = "servicecontrol.googleapis.com"
	serverAddrWithPort = "servicecontrol.googleapis.com:443"
)

func AllocateQuotaGrpc(ctx context.Context,serviceName, metricName, operationId, quotaMode, apiKeyString string, weight int64)  (*scp.AllocateQuotaResponse, error) {

	conn := createGrpcConnection(ctx, serverAddr, serverAddrWithPort, scope)

	quotaRequest := constructQuotaRequestGrpc(serviceName, metricName, operationId, quotaMode, apiKeyString, weight)
	quotaRes, err := scp.NewQuotaControllerClient(conn).AllocateQuota(ctx, &quotaRequest)

	if err != nil {
		fmt.Println("Error while making grpc call: ", err)
	} else if len(quotaRes.AllocateErrors) > 0 {
		quotaViolationErrorsBA := quotaRes.AllocateErrors[0].Code.String()
		err = errors.New(quotaViolationErrorsBA)
	}

	return quotaRes, err
}

func createGrpcConnection(ctx context.Context, serverAddr, serverAddrWithPort, scope string) *grpc.ClientConn {
	pool, _ := x509.SystemCertPool()
	// error handling omitted
	creds := credentials.NewClientTLSFromCert(pool, serverAddrWithPort)
	creds.OverrideServerName(serverAddr)
	perRPC, _ := grpcOauth.NewApplicationDefault(ctx, scope)
	conn, _ := grpc.Dial(
		serverAddrWithPort,
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(perRPC),
	)
	return conn
}

func AllocateQuota(sCtlService *sc.Service, serviceName, metricName, operationId, quotaMode, apiKeyString string, weight int64) error {

	quotaRequest := constructQuotaRequest(metricName, operationId, quotaMode, apiKeyString, weight)
	allocateQuota := sCtlService.Services.AllocateQuota(serviceName, &quotaRequest)
	allocateQuotaRes, e := allocateQuota.Do()

	if e == nil {
		var quotaViolationErrors string
		if len(allocateQuotaRes.AllocateErrors) > 0 {
			quotaViolationErrorsBA, _ := allocateQuotaRes.AllocateErrors[0].MarshalJSON()
			quotaViolationErrors = string(quotaViolationErrorsBA[:])
			e = errors.New(quotaViolationErrors)
		}
	}
	return e
}

func constructQuotaRequest(metricName, operationId, quotaMode, apiKeyString string, weight int64) sc.AllocateQuotaRequest {

	qm := make([]*sc.MetricValueSet, 1)
	mv := make([]*sc.MetricValue, 1)
	timeNow := time.Now().Format(time.RFC3339Nano)
	mv[0] = &sc.MetricValue{
		StartTime:  timeNow,
		EndTime:    timeNow,
		Int64Value: &weight,
	}

	qm[0] = &sc.MetricValueSet{
		MetricName:   metricName,
		MetricValues: mv,
	}

	allocateOp := sc.QuotaOperation{
		ConsumerId:   apiKeyString,
		OperationId:  operationId,
		QuotaMetrics: qm,
		QuotaMode:    quotaMode,
	}
	quotaRequest := sc.AllocateQuotaRequest{
		AllocateOperation: &allocateOp,
	}
	return quotaRequest
}

func constructQuotaRequestGrpc(serviceName, metricName, operationId, quotaMode, apiKeyString string, weight int64) scp.AllocateQuotaRequest {

	qm := make([]*scp.MetricValueSet, 1)
	mv := make([]*scp.MetricValue, 1)

	timeNow := &timestamp.Timestamp{Seconds: time.Now().Unix()}
	value := scp.MetricValue_Int64Value{
		Int64Value: weight,
	}
	//timeNow := timestamp.Timestamp{Seconds: time.Now().Unix()}

	mv[0] = &scp.MetricValue{
		StartTime: timeNow,
		EndTime: timeNow,
		Value: &value,
	}

	qm[0] = &scp.MetricValueSet{
		MetricName:   metricName,
		MetricValues: mv,
	}
	// TODO : read it from argument
	mode := scp.QuotaOperation_BEST_EFFORT
	allocateOp := scp.QuotaOperation{
		ConsumerId:   apiKeyString,
		OperationId:  operationId,
		QuotaMetrics: qm,
		QuotaMode:    mode,
	}
	quotaRequest := scp.AllocateQuotaRequest{
		ServiceName:serviceName,
		AllocateOperation: &allocateOp,
	}
	return quotaRequest
}

// customJwt represents PerRPCCredentials via provided JWT signing key.
type customJwt struct {
	mu    sync.Mutex
	token string
}

func (s *customJwt) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]string{
		"authorization": s.token,
	}, nil
}

func (s *customJwt) RequireTransportSecurity() bool {
	return true
}
