package upload

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/url"
	"path/filepath"
	"strings"
	"wegugin/config"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioUploader struct {
	client *minio.Client
	cfg    *config.Config
}

func NewMinioUploader() (*MinioUploader, error) {
	fmt.Println("Minio client yaratilmoqda...")
	cfg := config.Load()

	client, err := minio.New(cfg.Minio.MINIO_ENDPOINT, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Minio.MINIO_ACCESS_KEY_ID, cfg.Minio.MINIO_SECRET_ACCESS_KEY, ""),
		Secure: false, // localhost uchun false
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %v", err)
	}

	fmt.Println("Minio client muvaffaqiyatli yaratildi")
	return &MinioUploader{client: client, cfg: cfg}, nil
}

func (m *MinioUploader) UploadFile(bucketName string, file multipart.File, header *multipart.FileHeader) (string, error) {
	fmt.Println("Fayl yuklash boshlandi...")
	ctx := context.Background()

	// Bucket mavjudligini tekshirish va yaratish
	exists, err := m.client.BucketExists(ctx, bucketName)
	if err != nil {
		return "", fmt.Errorf("failed to check bucket existence: %v", err)
	}

	if !exists {
		err = m.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to create bucket: %v", err)
		}
		fmt.Printf("Bucket '%s' yaratildi\n", bucketName)
	}

	// Fayl nomi generatsiya qilish
	fileExt := filepath.Ext(header.Filename)
	newFileName := uuid.NewString() + fileExt

	// Faylni yuklash
	_, err = m.client.PutObject(ctx, bucketName, newFileName, file, header.Size, minio.PutObjectOptions{
		ContentType: getContentType(fileExt),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %v", err)
	}

	// Bucket policy o'rnatish (faqat bir marta kerak)
	err = m.setBucketPolicyIfNeeded(ctx, bucketName)
	if err != nil {
		// Policy o'rnatish xatosi bo'lsa ham, fayl yuklandi, shuning uchun logga yozamiz
		fmt.Printf("Warning: failed to set bucket policy: %v\n", err)
	}

	// Public URL yaratish
	url := fmt.Sprintf("%s/%s/%s", m.cfg.Minio.MINIO_PUBLIC_URL, bucketName, newFileName)
	fmt.Printf("Fayl muvaffaqiyatli yuklandi: %s\n", url)

	return url, nil
}

func (m *MinioUploader) setBucketPolicyIfNeeded(ctx context.Context, bucketName string) error {
	// Avval mavjud policy tekshiramiz
	_, err := m.client.GetBucketPolicy(ctx, bucketName)
	if err == nil {
		// Policy allaqachon o'rnatilgan
		return nil
	}

	// Policy o'rnatish
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

	return m.client.SetBucketPolicy(ctx, bucketName, policy)
}

// URL dan fayl nomini ajratib olish
func (m *MinioUploader) extractFileNameFromURL(fileURL string) (string, error) {
	parsedURL, err := url.Parse(fileURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %v", err)
	}

	// Path dan oxirgi qismni olish (/bucket/filename.ext -> filename.ext)
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return "", fmt.Errorf("invalid URL format")
	}

	return pathParts[len(pathParts)-1], nil
}

// URL orqali faylni o'chirish
func (m *MinioUploader) DeleteFileByURL(bucketName, fileURL string) error {
	fileName, err := m.extractFileNameFromURL(fileURL)
	if err != nil {
		return err
	}

	return m.DeleteFile(bucketName, fileName)
}

// Fayl nomini bilgan holda o'chirish
func (m *MinioUploader) DeleteFile(bucketName, fileName string) error {
	fmt.Printf("Fayl o'chirilmoqda: %s/%s\n", bucketName, fileName)
	ctx := context.Background()

	// Avval fayl mavjudligini tekshirish
	_, err := m.client.StatObject(ctx, bucketName, fileName, minio.StatObjectOptions{})
	if err != nil {
		// Agar fayl mavjud bo'lmasa
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return fmt.Errorf("file not found: %s", fileName)
		}
		return fmt.Errorf("failed to check file existence: %v", err)
	}

	// Faylni o'chirish
	err = m.client.RemoveObject(ctx, bucketName, fileName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}

	fmt.Printf("Fayl muvaffaqiyatli o'chirildi: %s\n", fileName)
	return nil
}

// Ko'p fayllarni bir vaqtda o'chirish
func (m *MinioUploader) DeleteFiles(bucketName string, fileNames []string) error {
	ctx := context.Background()

	// RemoveObjects uchun kanal yaratish
	objectsCh := make(chan minio.ObjectInfo)

	go func() {
		defer close(objectsCh)
		for _, fileName := range fileNames {
			objectsCh <- minio.ObjectInfo{Key: fileName}
		}
	}()

	// Fayllarni o'chirish
	for rErr := range m.client.RemoveObjects(ctx, bucketName, objectsCh, minio.RemoveObjectsOptions{}) {
		if rErr.Err != nil {
			return fmt.Errorf("failed to delete file %s: %v", rErr.ObjectName, rErr.Err)
		}
		fmt.Printf("Fayl o'chirildi: %s\n", rErr.ObjectName)
	}

	return nil
}

func getContentType(fileExt string) string {
	switch strings.ToLower(fileExt) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}
