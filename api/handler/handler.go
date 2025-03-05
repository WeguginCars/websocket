package handler

import (
	"log/slog"
	"wegugin/genproto/cruds"
	"wegugin/genproto/user"
	"wegugin/upload"

	"github.com/casbin/casbin/v2"
)

type Handler struct {
	User     user.UserClient
	Crud     cruds.CrudsServiceClient
	Log      *slog.Logger
	Enforcer *casbin.Enforcer
	MINIO    *upload.MinioUploader
}
