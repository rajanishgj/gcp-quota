# gcp-quota
quota service integration for gcp

Notes : 
install deb using : curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh 
go get google.golang.org/api/servicemanagement/v1/
dep ensure -add google.golang.org/api/servicemanagement/v1
dep ensure -add google.golang.org/api/servicecontrol/v1

dep ensure -add google.golang.org/grpc
dep ensure -add google.golang.org/genproto/googleapis/api/servicemanagement/v1
dep ensure -add google.golang.org/api/servicecontrol/v1
dep ensure -add github.com/golang/protobuf/ptypes/timestamp
dep ensure -add google.golang.org/genproto/googleapis/longrunning
dep ensure -add google.golang.org/genproto/googleapis/api/metric

Proto-Gen :
cd $GOPATH
protoc -I src/github.com/googleapis/googleapis src/github.com/googleapis/googleapis/google/api/servicemanagement/v1/servicemanager.proto  -I src/github.com/google/protobuf/src --go_out=out/

Run :
go run main.go
open url in browser : http://localhost:8081/gcp-quota
