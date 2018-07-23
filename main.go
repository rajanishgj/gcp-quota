package main

import "fmt"

import (
	sc "google.golang.org/api/servicecontrol/v1"
	sm "google.golang.org/api/servicemanagement/v1"
	//oauth "google.golang.org/api/oauth2/v2"
	smp "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	grpcOauth "google.golang.org/grpc/credentials/oauth"
	"flag"
	"os"
	"io/ioutil"
	"log"
	"strings"
	//"google.golang.org/api/option"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"time"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"crypto/x509"
	"google.golang.org/genproto/googleapis/api/serviceconfig"
	"sync"
	"net/http"
	"github.com/rajanishgj/gcp-quota/runtime"
	"strconv"
	"github.com/rajanishgj/gcp-quota/mgmt"
	"encoding/json"
)

func consumeFromQuota(w http.ResponseWriter, r *http.Request) {

	var message string
	var weight int64
	// 1. extract params
	quotaId := r.URL.Query().Get("quota_id")
	consumerId := r.URL.Query().Get("consumer_id")
	timestamp := r.URL.Query().Get("timestamp")
	tokens := r.URL.Query().Get("tokens")
	// 2a. validate
	if quotaId == "" {
		message = "query param quota_id is required"
	}
	if consumerId == "" {
		message += "\n query param consumer_id is required"
	}

	if message != "" {
		w.Write([]byte(message))
		return
	}

	// 2b. set defaults
	if timestamp == "" {
		// TODO : pass this to quota service.
		timestamp = time.Now().String()
	}
	if tokens != "" {
		weight, _ = strconv.ParseInt(tokens, 10, 64)
	} else {
		weight = 1
	}
	// 3. call chemist
	ctx := context.Background()
	_, violation := runtime.AllocateQuotaGrpc(ctx, quotaId, quotaId, quotaId, "NORMAL", consumerId, weight)
	// 4. parse response and write to Response
	if violation != nil {
		rs := "[ \"" + violation.Error() + " \"]"
		w.WriteHeader(403)
		w.Write([]byte(rs))
	} else {
		w.Write([]byte("[ \"success\" ] "))
	}
}

func configureQuota(w http.ResponseWriter, r *http.Request) {

	var message, projectId, serviceName, token, units string

	switch r.Method {
	case "GET":
		projectId = r.URL.Query().Get("project_id")
		serviceName = r.URL.Query().Get("service_name")
		units = r.URL.Query().Get("units")
		token = r.Header.Get("token")
	case "POST":
		// Call ParseForm() to parse the raw query and update r.PostForm and r.Form.
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}

		projectId = r.FormValue("project_id")
		units = r.FormValue("units")
		serviceName = r.FormValue("service_name")
		token = r.FormValue("token")

	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}

	if projectId == "" {
		message = "request parameter project_id is required"
	}
	if serviceName == "" {
		message += "\nrequest parameter service_name is required"
	}
	if token == "" {
		message += "\nrequest parameter token is required"
	} else if strings.LastIndex(token, "Bearer") != 0 {
		token = "Bearer " + token
	}

	iUnits, err := strconv.ParseInt(units, 0,64)

	if iUnits == 0 || err != nil {
		message += "\nrequest parameter units (per min) is required and should be integer"
	}

	if message != "" {
		w.Write([]byte(message))
		return
	}
	reqParams := "\n projectId : "+ projectId +"\n serviceName : "+ serviceName +" \n units per min : "+ units +"\n token : ****** "
	fmt.Println(reqParams)
	service, err := mgmt.SetupQuota(projectId, serviceName, iUnits, token)
	if err != nil {
		errMsg := " ERROR : " + err.Error() + reqParams
		w.WriteHeader(400)
		w.Write([]byte(errMsg))
	} else {
		sjon, _ := json.Marshal(service)
		w.Write(sjon)
	}
}

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func main() {
	http.HandleFunc("/consume", consumeFromQuota)
	http.HandleFunc("/configure", configureQuota)
	//http.Handle("/rajanishgj/", http.StripPrefix("/rajanishgj", http.FileServer(http.Dir("/Users/rajanishgj"))))
	http.Handle("/gcp-quota", http.StripPrefix("/gcp-quota", http.FileServer(http.Dir("/Users/rajanishgj/git/go_workspace/src/github.com/rajanishgj/gcp-quota"))))

	http.HandleFunc("/ping", ping)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func main_old() {
	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	sa := f.String("sa", "", "path to the service account json")
	gcpProject := f.String("project", "", "gcp project id/name")

	f.Parse(os.Args[1:])

	fmt.Println("CloudPlatformScope is : ", sm.CloudPlatformScope)
	fmt.Println("service account file location : ", *sa)
	fmt.Println("gcp project : ", *gcpProject)
	ctx := context.Background()

	_, err := grpcGetServiceConfig(sa, ctx)
	grpcGetServiceConfigWithOauthToken(sa, ctx)
	// GRPC call ends here

	//oauthHttpClient, _ := oauth.New(http.DefaultClient)
	//option.WithCredentialsFile(*sa)
	//fmt.Println(oauthHttpClient)

	oauthClient, err := google.DefaultClient(ctx, sm.CloudPlatformScope, sm.ServiceManagementScope)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	sCtlService, err := sc.New(oauthClient)

	//var tt oauth2.Transport = oauth2.Transport(oauthClient.Transport)

	//fmt.Println(tt)

	if err != nil {
		log.Fatal("failed to create new service control service", err)
	}
	sMgmtService, err := sm.New(oauthClient)
	getConfig(sMgmtService)

	// servicesList(sMgmtService)

	for i := 1; i <= 10; i++ {
		fmt.Println("call number : ", i)
		apiKeyString := "api_key:AIzaSyCQajptBEJSBgvlwWbqWoX5pBLc2qyrTK0"
		allocateQuota(sCtlService, apiKeyString)
	}

	for i := 1; i <= 10; i++ {
		fmt.Println("call number : ", i)
		apiKeyString := "api_key:AIzaSyBm4d5Px9JBBZ2nIVM9-BeKgpnh0fM_DbA"
		allocateQuota(sCtlService, apiKeyString)
	}

}

const serviceName = "rajanishgj-1-test-service-created-using-api.sandbox.googleapis.com"

func grpcGetServiceConfig(sa *string, ctx context.Context) (*serviceconfig.Service, error) {
	scope := sm.CloudPlatformScope
	serverAddr := "servicemanagement.googleapis.com"
	serverAddrWithPort := "servicemanagement.googleapis.com:443"
	getServiceConfigProtoReq := smp.GetServiceConfigRequest{
		ServiceName: serviceName,
	}
	pool, _ := x509.SystemCertPool()
	// error handling omitted
	creds := credentials.NewClientTLSFromCert(pool, serverAddrWithPort)
	creds.OverrideServerName(serverAddr)
	perRPC, _ := grpcOauth.NewServiceAccountFromFile(*sa, scope)
	conn, _ := grpc.Dial(
		serverAddrWithPort,
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(perRPC),
	)
	gscp, err := smp.NewServiceManagerClient(conn).GetServiceConfig(ctx, &getServiceConfigProtoReq)
	if err != nil {
		fmt.Println("Error while making grpc call: ", err)
	}
	fmt.Println("grpc call get name : ", gscp.GetName())
	return gscp, err
}

// TODO : pass externally generated (jwt) auth header
func grpcGetServiceConfigWithOauthToken(sa *string, ctx context.Context) (*serviceconfig.Service, error) {
	//scope := sm.CloudPlatformScope
	serverAddr := "servicemanagement.googleapis.com"
	serverAddrWithPort := "servicemanagement.googleapis.com:443"
	getServiceConfigProtoReq := smp.GetServiceConfigRequest{
		ServiceName: serviceName,
	}
	pool, _ := x509.SystemCertPool()
	// error handling omitted
	creds := credentials.NewClientTLSFromCert(pool, serverAddrWithPort)
	creds.OverrideServerName(serverAddr)
	//perRPC, _ := grpcOauth.NewServiceAccountFromFile(*sa, scope)
	bearerToken := "Bearer ya29.c.ElrSBZKqpjJDEyFjqpfWF1s62FplR8at1Lvt2NDxFKShwNzJr6x2T0YK6ycldNv_ZlA4aNxBjL1jmZdBmjvf6733o8G9sCsxDWHWNgy9Wewz7Fz_Jo7bSaz0psc"

	//md := metadata.Pairs("Authorization", bearerToken)
	//cos := grpc.HeaderCallOption{
	//	HeaderAddr: &md,
	//}

	perRPC := customJwt{
		token: bearerToken,
	}

	conn, _ := grpc.Dial(
		serverAddrWithPort,
		grpc.WithPerRPCCredentials(&perRPC),
		grpc.WithTransportCredentials(creds),
	)
	//grpc.Header()
	gscp, err := smp.NewServiceManagerClient(conn).GetServiceConfig(ctx, &getServiceConfigProtoReq)
	if err != nil {
		fmt.Println("Error while making grpc call: ", err)
	}
	fmt.Println("grpc call get name : ", gscp.GetName())
	return gscp, err
}

func getConfig(apiService *sm.APIService) {
	fmt.Println("apiService.BasePath is : ", apiService.BasePath)
	servicesGetConfigCall := apiService.Services.GetConfig(serviceName)
	res, error := servicesGetConfigCall.Do()
	printResponse(error, res)
}

func allocateQuota(sCtlService *sc.Service, apiKeyString string) {
	//metricName := "airport_requests"
	//apiKeyString := "api_key:AIzaSyAWoARcZrvk0qJ0URFpDDW8isWsMtoUNjM"
	//methodName := "AirportName"
	//operationId := "1.edge_mint_counterservice_appspot_com"

	metricName := "projects/edge-mint-counterservice/prod_1_metrics"
	//apiKeyString := "api_key:AIzaSyAWoARcZrvk0qJ0URFpDDW8isWsMtoUNjM"
	methodName := ""
	operationId := "prod_1_metrics"

	quotaRequest := constructQuotaRequest(metricName, apiKeyString, methodName, operationId)
	//qrString, _ :=  json.Marshal(quotaRequest)
	//fmt.Println("quota request is : " , string(qrString))
	allocateQuota := sCtlService.Services.AllocateQuota(serviceName, &quotaRequest)
	allocateQuotaRes, error := allocateQuota.Do()
	if error != nil {
		fmt.Println(error)
	} else {
		var quotaViolationErrors string
		if len(allocateQuotaRes.AllocateErrors) > 0 {
			quotaViolationErrorsBA, _ := allocateQuotaRes.AllocateErrors[0].MarshalJSON()
			quotaViolationErrors = string(quotaViolationErrorsBA[:])
		}

		for i, v := range allocateQuotaRes.QuotaMetrics {
			fmt.Println(i, " : ", v.MetricName)
		}
		fmt.Println(allocateQuotaRes.QuotaMetrics[0].MetricName, " ", quotaViolationErrors)
		//resString, _ :=json.Marshal(allocateQuotaRes)
		//fmt.Println("response is ", string(resString))
	}

}

func constructQuotaRequest(metricName, apiKeyString, methodName, operationId string) sc.AllocateQuotaRequest {

	qm := make([]*sc.MetricValueSet, 2)
	mv := make([]*sc.MetricValue, 1)
	mvWithLabel := make([]*sc.MetricValue, 1)
	lables := make(map[string]string)
	weight := int64(-1)
	timeNow := time.Now().Format(time.RFC3339Nano)
	mv[0] = &sc.MetricValue{
		StartTime:  timeNow,
		EndTime:    timeNow,
		Int64Value: &weight,
	}
	lables["/protocol"] = "https"
	lables["/response_code"] = "200"
	lables["/response_code_class"] = "OK"
	lables["/status_code"] = "0"

	mvWithLabel[0] = &sc.MetricValue{
		StartTime:  timeNow,
		EndTime:    timeNow,
		Int64Value: &weight,
		Labels:     lables,
	}

	qm[0] = &sc.MetricValueSet{
		MetricName:   metricName,
		MetricValues: mv,
	}

	qm[1] = &sc.MetricValueSet{
		MetricName:   "serviceruntime.googleapis.com/api/consumer/request_count",
		MetricValues: mvWithLabel,
	}

	allocateOp := sc.QuotaOperation{
		ConsumerId:   apiKeyString,
		MethodName:   methodName,
		OperationId:  operationId,
		QuotaMetrics: qm,
		QuotaMode:    "BEST_EFFORT",
	}
	quotaRequest := sc.AllocateQuotaRequest{
		AllocateOperation: &allocateOp,
	}
	return quotaRequest
}

//func constructReportRequest(metricName, apiKeyString, methodName, operationId string) sc.ReportRequest {
//
//	req := []*
//	reportRequest := sc.ReportRequest{
//		ServiceConfigId: operationId,
//
//	}
//
//	qm := make([]*sc.MetricValueSet, 1)
//	mv := make([]*sc.MetricValue, 1)
//	weight := int64(2)
//	timeNow := time.Now().Format(time.RFC3339Nano)
//	mv[0] = &sc.MetricValue{
//		StartTime:  timeNow,
//		EndTime:    timeNow,
//		Int64Value: &weight,
//	}
//
//	qm[0] = &sc.MetricValueSet{
//		MetricName:   metricName,
//		MetricValues: mv,
//	}
//
//	allocateOp := sc.QuotaOperation{
//		ConsumerId:   apiKeyString,
//		MethodName:   methodName,
//		OperationId:  operationId,
//		QuotaMetrics: qm,
//		QuotaMode:    "BEST_EFFORT",
//	}
//	quotaRequest := sc.AllocateQuotaRequest{
//		AllocateOperation: &allocateOp,
//	}
//	return quotaRequest
//}

func printResponse(error error, res *sm.Service) {
	if error != nil {
		fmt.Println(error)
	} else {
		fmt.Println("no error")
		fmt.Println(res.ServerResponse.HTTPStatusCode)
		fmt.Println(res.Title)
		fmt.Println(res.ConfigVersion)
		resString, err := res.MarshalJSON()
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(string(resString))
		}
	}
}
func servicesList(service *sm.APIService) {
	res, error := service.Services.List().Do();
	if error == nil {
		for i, v := range res.Services {
			fmt.Println(i, " : ", string(v.ServiceName))
		}
	} else {
		fmt.Println(error)
	}

}

func valueOrFileContents(value string, filename string) string {
	if value != "" {
		return value
	}
	slurp, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error reading %q: %v", filename, err)
	}
	return strings.TrimSpace(string(slurp))
}

//// explicit reads credentials from the specified path.
//func explicit(jsonPath, projectID string) {
//	ctx := context.Background()
//	client, err := sm.(ctx, option.WithCredentialsFile(jsonPath))
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println("Buckets:")
//	it := client.Buckets(ctx, projectID)
//	for {
//		battrs, err := it.Next()
//		if err == iterator.Done {
//			break
//		}
//		if err != nil {
//			log.Fatal(err)
//		}
//		fmt.Println(battrs.Name)
//	}
//}

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
