package mgmt

import "fmt"

import (
	sm "google.golang.org/api/servicemanagement/v1"

	smp "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	grpcOauth "google.golang.org/grpc/credentials/oauth"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"crypto/x509"
	"google.golang.org/genproto/googleapis/api/serviceconfig"
	"sync"
	"time"
	"errors"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/genproto/googleapis/api/metric"
	"github.com/Sirupsen/logrus"
	"encoding/json"
)

const (
	scope              = sm.CloudPlatformScope
	serverAddr         = "servicemanagement.googleapis.com"
	serverAddrWithPort = "servicemanagement.googleapis.com:443"
)

func SetupQuota(projectId, serviceName string, units int64, token string) (*serviceconfig.Service, error) {

	var err error
	var conn *grpc.ClientConn
	ctx := context.Background()
	// 0. create or reuse connection
	//if token != "" {
	conn = createGrpcConnectionWithToken(serverAddr, serverAddrWithPort, token)
	//}else {
	//	conn = createGrpcConnection(ctx,serverAddr,serverAddrWithPort,scope)
	//}
	// 1. get service config
	logrus.Info("Get service config for pre-existence validation ", serviceName)
	service, err := GrpcGetServiceConfig(ctx, conn, serviceName)
	if err == nil {
		// return error if service with this name already exists
		res, _ := json.Marshal(service)
		return nil, errors.New(serviceName + " already exists. " + string(res))
	}
	// 2a. create service
	logrus.Info("grpcCreateService")
	err = grpcCreateService(ctx, conn, projectId, serviceName)
	if err != nil {
		// return error if service create fails
		return nil, err
	}
	// 2b. create service config
	logrus.Info("CreateServiceConfig")
	sc, err := CreateServiceConfig(ctx, conn, projectId, serviceName, units)
	if err != nil {
		// return error if service-config create fails
		return nil, err
	}
	fmt.Println(sc.Id)
	// 3. rollout service config to 100%
	logrus.Info("RolloutServiceConfig")
	err = RolloutServiceConfig(ctx, conn, serviceName, serviceName)
	if err != nil {
		// return error if service-config rollout fails
		return nil, err
	}
	return GrpcGetServiceConfig(ctx, conn, serviceName)
}

func RolloutServiceConfig(ctx context.Context, conn *grpc.ClientConn, serviceName, serviceConfigId string) (error) {

	trafficSplit := make(map[string]float64)
	trafficSplit[serviceConfigId] = 100
	rStrategy := smp.Rollout_TrafficPercentStrategy_{
		TrafficPercentStrategy: &smp.Rollout_TrafficPercentStrategy{
			Percentages: trafficSplit,
		},
	}
	rollout := smp.Rollout{
		ServiceName: serviceName,
		Strategy:    &rStrategy,
	}
	createServiceRolloutRequest := smp.CreateServiceRolloutRequest{
		ServiceName: serviceName,
		Rollout:     &rollout,
	}
	future, err := smp.NewServiceManagerClient(conn).CreateServiceRollout(ctx, &createServiceRolloutRequest)

	if err != nil {
		return err
	}

	// TODO: handle future in better way.
	retryCounter := 1
	for i := 1; i <= 10; i++ {
		if !future.Done && retryCounter <= 10 {
			time.Sleep(1000000000)
			retryCounter++
			err = errors.New("no response from future or longrunning after waiting for 10 secs in step : RolloutServiceConfig")
		} else {
			if future.GetError() != nil {
				return errors.New(future.GetError().Message)
			}
		}
	}

	return err
}

func GrpcGetServiceConfig(ctx context.Context, conn *grpc.ClientConn, serviceName string) (*serviceconfig.Service, error) {

	getServiceConfigProtoReq := smp.GetServiceConfigRequest{
		ServiceName: serviceName,
	}
	gscp, err := smp.NewServiceManagerClient(conn).GetServiceConfig(ctx, &getServiceConfigProtoReq)
	if err != nil {
		fmt.Println("Error while making grpc call: ", err)
	}
	fmt.Println("grpc call get name : ", gscp.GetName())
	return gscp, err
}

func CreateServiceConfig(ctx context.Context, conn *grpc.ClientConn, projectId, serviceName string, units int64) (*serviceconfig.Service, error) {

	configV := wrappers.UInt32Value{
		Value: 3,
	}
	control := serviceconfig.Control{
		Environment: "servicecontrol.googleapis.com",
	}
	metricType := "custom.googleapis.com/" + projectId + "/" + serviceName
	metrics := make([]*metric.MetricDescriptor, 1)
	metrics[0] = &metric.MetricDescriptor{
		Name:        serviceName,
		MetricKind:  metric.MetricDescriptor_DELTA,
		ValueType:   metric.MetricDescriptor_INT64,
		Unit:        "1",
		Description: serviceName,
		DisplayName: serviceName,
		Type:        metricType,
	}

	limits := make([]*serviceconfig.QuotaLimit, 1)
	limitValues := make(map[string]int64)
	limitValues["STANDARD"] = units
	limits[0] = &serviceconfig.QuotaLimit{
		Name:        "test-limit",
		DisplayName: serviceName,
		Description: serviceName,
		Unit:        "1/min/{project}",
		Metric:      metrics[0].Name,
		Values:      limitValues,
	}
	quotaConfig := serviceconfig.Quota{
		Limits: limits,
	}

	serviceConfig := serviceconfig.Service{
		Id:                serviceName,
		Name:              serviceName,
		Title:             serviceName,
		ConfigVersion:     &configV,
		Control:           &control,
		ProducerProjectId: projectId,
		Quota:             &quotaConfig,
		Metrics:           metrics,
	}

	createServiceConfigProtoReq := smp.CreateServiceConfigRequest{
		ServiceName:   serviceName,
		ServiceConfig: &serviceConfig,
	}
	gscp, err := smp.NewServiceManagerClient(conn).CreateServiceConfig(ctx, &createServiceConfigProtoReq)
	if err != nil {
		fmt.Println("Error while making grpc call: ", err)
	}
	fmt.Println("grpc call get name : ", gscp.GetName())
	return gscp, err
}

func grpcCreateService(ctx context.Context, conn *grpc.ClientConn, producerProjectId, serviceName string) (error) {

	service := smp.ManagedService{
		ServiceName:       serviceName,
		ProducerProjectId: producerProjectId,
	}
	createServiceProtoReq := smp.CreateServiceRequest{
		Service: &service,
	}
	future, err := smp.NewServiceManagerClient(conn).CreateService(ctx, &createServiceProtoReq)
	if err != nil {
		fmt.Println("Error while making grpc call: ", err)
		return err
	}

	// TODO: handle future in better way.
	retryCounter := 1
	for i := 1; i <= 10; i++ {
		if !future.Done && retryCounter <= 10 {
			time.Sleep(1000000000)
			retryCounter++
			err = errors.New("no response from future or longrunning after waiting for 10 secs in step : CreateService")
		} else {
			if future.GetError() != nil {
				return errors.New(future.GetError().Message)
			}
		}
	}

	return err
}

func GetServiceConfigUsingOauthToken(ctx context.Context, serviceName, bearerToken string) (*serviceconfig.Service, error) {
	//scope := sm.CloudPlatformScope
	serverAddr := "servicemanagement.googleapis.com"
	serverAddrWithPort := "servicemanagement.googleapis.com:443"
	getServiceConfigProtoReq := smp.GetServiceConfigRequest{
		ServiceName: serviceName,
	}

	conn := createGrpcConnectionWithToken(serverAddr, serverAddrWithPort, bearerToken)
	//grpc.Header()
	gscp, err := smp.NewServiceManagerClient(conn).GetServiceConfig(ctx, &getServiceConfigProtoReq)
	if err != nil {
		fmt.Println("Error while making grpc call: ", err)
	}
	fmt.Println("grpc call get name : ", gscp.GetName())
	return gscp, err
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

func createGrpcConnection(ctx context.Context, serverAddr, serverAddrWithPort, scope string) *grpc.ClientConn {
	pool, _ := x509.SystemCertPool()
	// error handling omitted
	creds := credentials.NewClientTLSFromCert(pool, serverAddrWithPort)
	creds.OverrideServerName(serverAddr)
	perRPC, _ := grpcOauth.NewApplicationDefault(ctx, sm.ServiceManagementScope, sm.CloudPlatformScope)
	conn, _ := grpc.Dial(
		serverAddrWithPort,
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(perRPC),
	)
	return conn
}

func createGrpcConnectionWithToken(serverAddr, serverAddrWithPort, bearerToken string) *grpc.ClientConn {
	pool, _ := x509.SystemCertPool()
	// error handling omitted
	creds := credentials.NewClientTLSFromCert(pool, serverAddrWithPort)
	creds.OverrideServerName(serverAddr)
	perRPC := customJwt{
		token: bearerToken,
	}
	conn, _ := grpc.Dial(
		serverAddrWithPort,
		grpc.WithPerRPCCredentials(&perRPC),
		grpc.WithTransportCredentials(creds),
	)
	return conn
}
