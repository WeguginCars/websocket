package main

import (
	"log"

	"wegugin/api"
	"wegugin/api/handler"
	"wegugin/config"
	"wegugin/genproto/cruds"
	"wegugin/genproto/user"
	"wegugin/logs"
	"wegugin/upload"

	"github.com/casbin/casbin/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conf := config.Load()
	hand := NewHandler()
	router := api.Router(hand)
	log.Printf("server is running...")
	log.Fatal(router.Run(conf.Server.HTTP_PORT))
}

func NewHandler() *handler.Handler {
	conf := config.Load()
	connUser, err := grpc.NewClient(conf.Server.GRPC_PORT, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	connCrud, err := grpc.NewClient(conf.Server.GRPC_PORT, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	User := user.NewUserClient(connUser)
	Crud := cruds.NewCrudsServiceClient(connCrud)

	logs := logs.NewLogger()
	enforcer, err := casbin.NewEnforcer("casbin/model.conf", "casbin/policy.csv")
	if err != nil {
		log.Fatal(err)
	}
	//minio
	uploader, err := upload.NewMinioUploader()
	if err != nil {
		log.Fatal(err)
	}
	return &handler.Handler{
		User:     User,
		Crud:     Crud,
		Log:      logs,
		Enforcer: enforcer,
		MINIO:    uploader,
	}
}
