package upload

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"wegugin/config"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioUploader struct {
	client *minio.Client
}

func NewMinioUploader() (*MinioUploader, error) {
	cfg := config.Load()
	client, err := minio.New(cfg.Minio.MINIO_ENDPOINT, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Minio.MINIO_ACCESS_KEY_ID, cfg.Minio.MINIO_SECRET_ACCESS_KEY, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %v", err)
	}

	return &MinioUploader{client: client}, nil
}

func (m *MinioUploader) UploadFile(bucketName string, file multipart.File, header *multipart.FileHeader) (string, error) {
	ctx := context.Background()

	// Generate unique filename
	fileExt := filepath.Ext(header.Filename)
	newFileName := uuid.NewString() + fileExt

	// Upload the file
	_, err := m.client.PutObject(ctx, bucketName, newFileName, file, header.Size, minio.PutObjectOptions{
		ContentType: getContentType(fileExt),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %v", err)
	}

	// Set bucket policy for public access
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"AWS": ["*"]
				},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, bucketName)

	err = m.client.SetBucketPolicy(ctx, bucketName, policy)
	if err != nil {
		return "", fmt.Errorf("failed to set bucket policy: %v", err)
	}

	// Generate URL
	// UploadFile funktsiyasida:
	url := fmt.Sprintf("http://%s/%s/%s", config.Load().Minio.MINIO_ENDPOINT, bucketName, newFileName)
	return url, nil
}

func getContentType(fileExt string) string {
	switch fileExt {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	default:
		return "application/octet-stream"
	}
}

func (m *MinioUploader) DeleteFile(bucketName, fileName string) error {
	ctx := context.Background()
	err := m.client.RemoveObject(ctx, bucketName, fileName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}
	return nil
}
