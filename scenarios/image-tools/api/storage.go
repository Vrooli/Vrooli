package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorage struct {
	client      *minio.Client
	bucketName  string
	publicURL   string
}

func NewMinIOStorage() (StorageService, error) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		return nil, fmt.Errorf("MINIO_ENDPOINT environment variable is required")
	}

	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		return nil, fmt.Errorf("MINIO_ACCESS_KEY environment variable is required")
	}

	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		return nil, fmt.Errorf("MINIO_SECRET_KEY environment variable is required")
	}
	
	useSSL := os.Getenv("MINIO_USE_SSL") == "true"
	
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}
	
	bucketName := "image-tools-processed"
	
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		log.Printf("Warning: Could not check if bucket exists: %v", err)
	}
	
	if !exists {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Printf("Warning: Could not create bucket: %v", err)
		}
	}
	
	publicURL := os.Getenv("MINIO_PUBLIC_URL")
	if publicURL == "" {
		publicURL = fmt.Sprintf("http://%s", endpoint)
	}
	
	return &MinIOStorage{
		client:     client,
		bucketName: bucketName,
		publicURL:  publicURL,
	}, nil
}

func (m *MinIOStorage) Save(key string, data []byte) (string, error) {
	ctx := context.Background()
	
	key = strings.TrimPrefix(key, "/")
	
	_, err := m.client.PutObject(ctx, m.bucketName, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{
			ContentType: getContentType(key),
		})
	if err != nil {
		return "", fmt.Errorf("failed to save to MinIO: %w", err)
	}
	
	return fmt.Sprintf("%s/%s/%s", m.publicURL, m.bucketName, key), nil
}

func (m *MinIOStorage) Get(key string) ([]byte, error) {
	ctx := context.Background()
	
	key = strings.TrimPrefix(key, "/")
	
	object, err := m.client.GetObject(ctx, m.bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get from MinIO: %w", err)
	}
	defer object.Close()
	
	return io.ReadAll(object)
}

func (m *MinIOStorage) Delete(key string) error {
	ctx := context.Background()
	
	key = strings.TrimPrefix(key, "/")
	
	err := m.client.RemoveObject(ctx, m.bucketName, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete from MinIO: %w", err)
	}
	
	return nil
}

func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}

type LocalStorage struct {
	basePath string
}

func NewLocalStorage() StorageService {
	basePath := "/tmp/image-tools"
	os.MkdirAll(basePath, 0755)
	return &LocalStorage{basePath: basePath}
}

func (l *LocalStorage) Save(key string, data []byte) (string, error) {
	path := filepath.Join(l.basePath, key)
	dir := filepath.Dir(path)
	
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	
	return fmt.Sprintf("file://%s", path), nil
}

func (l *LocalStorage) Get(key string) ([]byte, error) {
	path := filepath.Join(l.basePath, key)
	return os.ReadFile(path)
}

func (l *LocalStorage) Delete(key string) error {
	path := filepath.Join(l.basePath, key)
	return os.Remove(path)
}