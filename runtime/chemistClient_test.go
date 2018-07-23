package runtime

import (
	"fmt"
	"golang.org/x/net/context"
	"testing"
	"golang.org/x/oauth2/google"
	"log"

	sc "google.golang.org/api/servicecontrol/v1"
	"time"
)

func TestAllocateQuota(t *testing.T) {
	t.SkipNow()
	metricName := "projects/edge-mint-counterservice/prod_1_metrics"
	operationId := "prod_1_metrics"
	serviceName := "rajanishgj-1-test-service-created-using-api.sandbox.googleapis.com"
	quotaMode := "BEST_EFFORT"

	ctx := context.Background()
	oauthClient, err := google.DefaultClient(ctx, sc.CloudPlatformScope)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	sCtlService, err := sc.New(oauthClient)

	if err != nil {
		log.Fatal("failed to create new service control service", err)
	}

	apiKeyString := "api_key:AIzaSyCQajptBEJSBgvlwWbqWoX5pBLc2qyrTK0"
	for i := 1; i <= 10; i++ {
		fmt.Println("apiKeyString : ", apiKeyString, " call number : ", i)
		e := AllocateQuota(sCtlService, serviceName, metricName, operationId, quotaMode, apiKeyString, 1)
		fmt.Println("response is ", e)
	}

	apiKeyString = "api_key:AIzaSyBm4d5Px9JBBZ2nIVM9-BeKgpnh0fM_DbA"
	for i := 1; i <= 10; i++ {
		fmt.Println("apiKeyString : ", apiKeyString, " call number : ", i)
		e := AllocateQuota(sCtlService, serviceName, metricName, operationId, quotaMode, apiKeyString, 1)
		fmt.Println("response is ", e)
	}
}

func TestAllocateQuotaGrpc(*testing.T) {

	metricName := "projects/edge-mint-counterservice/prod_1_metrics"
	operationId := "prod_1_metrics"
	serviceName := "rajanishgj-1-test-service-created-using-api.sandbox.googleapis.com"
	quotaMode := "BEST_EFFORT"

	ctx := context.Background()


	apiKeyString := "api_key:AIzaSyCQajptBEJSBgvlwWbqWoX5pBLc2qyrTK0"
	pastState := true
	for i := 1; i <= 200; i++ {
		res, e := AllocateQuotaGrpc(ctx, serviceName, metricName, operationId, quotaMode, apiKeyString, 1)
		blocked := len(res.AllocateErrors) > 0
		if (pastState && !blocked) || (!pastState && blocked) {
			pastState = blocked
			fmt.Println(time.Now(), " ", e," apiKeyString : ", apiKeyString, " call number : ", i , " and response service config id : " , res.ServiceConfigId)
		}
		time.Sleep(1000000000)
	}

	//apiKeyString = "api_key:AIzaSyBm4d5Px9JBBZ2nIVM9-BeKgpnh0fM_DbA"
	//for i := 1; i <= 10; i++ {
	//	res, e := AllocateQuotaGrpc(ctx, serviceName, metricName, operationId, quotaMode, apiKeyString, 1)
	//	fmt.Println(time.Now(), " ", e," apiKeyString : ", apiKeyString, " call number : ", i , " and response service config id : " , res.ServiceConfigId)
	//}
}


