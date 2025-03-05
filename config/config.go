package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cast"
)

type Config struct {
	Postgres PostgresConfig
	Server   ServerConfig
	Token    TokensConfig
	Minio    MinioConfig
}

type PostgresConfig struct {
	DB_NAME     string
	DB_PORT     string
	DB_PASSWORD string
	DB_USER     string
	DB_HOST     string
}

type RedisConfig struct {
	RDB_ADDRESS  string
	RDB_PASSWORD string
}

type ServerConfig struct {
	HTTP_PORT string
	GRPC_PORT string
}

type TokensConfig struct {
	ACCES_KEY string
}

type MinioConfig struct {
	MINIO_ENDPOINT          string
	MINIO_ACCESS_KEY_ID     string
	MINIO_SECRET_ACCESS_KEY string
	MINIO_BUCKET_NAME       string
	MINIO_PUBLIC_URL        string
}

func Load() *Config {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("error while loading .env file: %v", err)
	}

	return &Config{
		Postgres: PostgresConfig{
			DB_HOST:     cast.ToString(coalesce("DB_HOST", "localhost")),
			DB_PORT:     cast.ToString(coalesce("DB_PORT", "5432")),
			DB_USER:     cast.ToString(coalesce("DB_USER", "postgres")),
			DB_NAME:     cast.ToString(coalesce("DB_NAME", "postgres")),
			DB_PASSWORD: cast.ToString(coalesce("DB_PASSWORD", "3333")),
		},
		Server: ServerConfig{
			HTTP_PORT: cast.ToString(coalesce("HTTP_PORT", ":1234")),
			GRPC_PORT: cast.ToString(coalesce("GRPC_PORT", ":5678")),
		},
		Token: TokensConfig{
			ACCES_KEY: cast.ToString(coalesce("ACCES_KEY", "access_key")),
		},
		Minio: MinioConfig{
			MINIO_ENDPOINT:          cast.ToString(coalesce("MINIO_ENDPOINT", "access_key")),
			MINIO_ACCESS_KEY_ID:     cast.ToString(coalesce("MINIO_ACCESS_KEY_ID", "access_key")),
			MINIO_SECRET_ACCESS_KEY: cast.ToString(coalesce("MINIO_SECRET_ACCESS_KEY", "access_key")),
			MINIO_BUCKET_NAME:       cast.ToString(coalesce("MINIO_BUCKET_NAME", "twit_images")),
			MINIO_PUBLIC_URL:        cast.ToString(coalesce("MINIO_PUBLIC_URL", "http://localhost:9000/minio/")),
		},
	}
}

func coalesce(key string, value interface{}) interface{} {
	val, exist := os.LookupEnv(key)
	if exist {
		return val
	}
	return value
}
