package main

import (
	"context"
	"log"
	"log/slog"
	"time"

	"wegugin/api"
	"wegugin/api/handler"
	"wegugin/config"
	"wegugin/genproto/cruds"
	"wegugin/genproto/user"
	"wegugin/logs"
	"wegugin/storage"
	"wegugin/storage/mongosh"
	"wegugin/storage/redis"
	"wegugin/upload"

	"github.com/casbin/casbin/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conf := config.Load()
	mdb, err := mongosh.Connect(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	rdbs := redis.ConnectRDB()
	logger := logs.NewLogger()
	dbs := storage.NewStorage(mdb, rdbs)
	defer func() {
		if err := dbs.CloseRDB(); err != nil {
			logger.Error("Failed to close Redis connection", "error", err)
		}
	}()

	go startTopCarsCleanup(dbs, logger)
	hand := NewHandler(conf, logger, dbs)
	router := api.Router(hand)
	log.Printf("server is running...")
	log.Fatal(router.Run(conf.Server.HTTP_PORT))
}

func NewHandler(conf *config.Config, logs *slog.Logger, st storage.IStorage) *handler.Handler {

	connUser, err := grpc.NewClient(conf.Server.USER_PORT, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	connCrud, err := grpc.NewClient(conf.Server.GRPC_PORT, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	User := user.NewUserClient(connUser)
	Crud := cruds.NewCrudsServiceClient(connCrud)
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
		Cruds:    st,
		User:     User,
		Crud:     Crud,
		Log:      logs,
		Enforcer: enforcer,
		MINIO:    uploader,
	}
}

func startTopCarsCleanup(storage storage.IStorage, logger *slog.Logger) {
	ticker := time.NewTicker(30 * time.Minute) // har 30 daqiqada ishga tushadi
	defer ticker.Stop()

	logger.Info("TopCars cleanup goroutine started", "interval", "30 minutes")

	for range ticker.C {
		cleanupExpiredTopCars(storage, logger)
	}
}

func cleanupExpiredTopCars(storage storage.IStorage, logger *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()

	deletedCount, err := storage.TopCars().DeleteFinishedTopCars(ctx)
	if err != nil {
		logger.Error("Failed to cleanup expired top cars",
			"error", err,
			"duration", time.Since(start))
		return
	}

	if deletedCount > 0 {
		logger.Info("Successfully cleaned up expired top cars",
			"deleted_count", deletedCount,
			"duration", time.Since(start))
	} else {
		logger.Debug("No expired top cars found for cleanup",
			"duration", time.Since(start))
	}
}
