//go:build integration
// +build integration

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// These tests require MinIO running. Use: go test -tags=integration ./...

func integrationS3Provider(accessKey, secretKey string) *S3DownloadStorageProvider {
	return &S3DownloadStorageProvider{ResolveCredential: func(_ context.Context, field string) (string, error) {
		switch field {
		case "delivery-s3-access-key-id":
			return accessKey, nil
		case "delivery-s3-secret-access-key":
			return secretKey, nil
		default:
			return "", nil
		}
	}}
}

func TestS3Integration_NewStorage_Success(t *testing.T) {
	endpoint, accessKey, secretKey := setupMinIOContainer(t)

	ctx := context.Background()
	provider := integrationS3Provider(accessKey, secretKey)

	settings := DownloadStorageSettings{
		Provider:       "s3",
		Bucket:         "test-bucket",
		Region:         "us-east-1",
		Endpoint:       endpoint,
		ForcePathStyle: true,
	}

	storage, err := provider.New(ctx, settings)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	if storage == nil {
		t.Fatal("expected non-nil storage")
	}
}

func TestS3Integration_NewStorage_InvalidCredentials(t *testing.T) {
	endpoint, _, _ := setupMinIOContainer(t)

	ctx := context.Background()
	provider := integrationS3Provider("invalid", "invalid")

	settings := DownloadStorageSettings{
		Provider:       "s3",
		Bucket:         "test-bucket",
		Region:         "us-east-1",
		Endpoint:       endpoint,
		ForcePathStyle: true,
	}

	storage, err := provider.New(ctx, settings)
	// Creation should succeed; auth error happens on operation
	if err != nil {
		t.Fatalf("creation failed: %v", err)
	}
	if storage == nil {
		t.Fatal("expected storage instance")
	}
}

func TestS3Integration_TestConnection_ValidBucket(t *testing.T) {
	endpoint, accessKey, secretKey := setupMinIOContainer(t)

	// Create bucket first using AWS SDK
	ctx := context.Background()
	createTestBucket(t, ctx, endpoint, accessKey, secretKey, "connection-test-bucket")

	provider := integrationS3Provider(accessKey, secretKey)
	settings := DownloadStorageSettings{
		Provider:       "s3",
		Bucket:         "connection-test-bucket",
		Region:         "us-east-1",
		Endpoint:       endpoint,
		ForcePathStyle: true,
	}

	storage, err := provider.New(ctx, settings)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	err = storage.TestConnection(ctx, "connection-test-bucket")
	if err != nil {
		t.Errorf("TestConnection failed: %v", err)
	}
}

func TestS3Integration_TestConnection_InvalidBucket(t *testing.T) {
	endpoint, accessKey, secretKey := setupMinIOContainer(t)

	ctx := context.Background()
	provider := integrationS3Provider(accessKey, secretKey)

	settings := DownloadStorageSettings{
		Provider:       "s3",
		Bucket:         "nonexistent-bucket-12345",
		Region:         "us-east-1",
		Endpoint:       endpoint,
		ForcePathStyle: true,
	}

	storage, err := provider.New(ctx, settings)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	err = storage.TestConnection(ctx, "nonexistent-bucket-12345")
	if err == nil {
		t.Error("expected error for nonexistent bucket")
	}
}

func TestS3Integration_PresignGet_Success(t *testing.T) {
	endpoint, accessKey, secretKey := setupMinIOContainer(t)

	ctx := context.Background()
	bucket := "presign-get-bucket"
	createTestBucket(t, ctx, endpoint, accessKey, secretKey, bucket)
	uploadTestObject(t, ctx, endpoint, accessKey, secretKey, bucket, "test-file.txt", []byte("test content"))

	provider := integrationS3Provider(accessKey, secretKey)
	settings := DownloadStorageSettings{
		Provider:       "s3",
		Bucket:         bucket,
		Region:         "us-east-1",
		Endpoint:       endpoint,
		ForcePathStyle: true,
	}

	storage, err := provider.New(ctx, settings)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	url, err := storage.PresignGet(ctx, bucket, "test-file.txt", 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignGet failed: %v", err)
	}
	if url == "" {
		t.Error("expected non-empty URL")
	}

	// Try to actually download the file using the presigned URL
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("failed to GET presigned URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "test content" {
		t.Errorf("expected 'test content', got '%s'", string(body))
	}
}

func TestS3Integration_PresignPut_Success(t *testing.T) {
	endpoint, accessKey, secretKey := setupMinIOContainer(t)

	ctx := context.Background()
	bucket := "presign-put-bucket"
	createTestBucket(t, ctx, endpoint, accessKey, secretKey, bucket)

	provider := integrationS3Provider(accessKey, secretKey)
	settings := DownloadStorageSettings{
		Provider:       "s3",
		Bucket:         bucket,
		Region:         "us-east-1",
		Endpoint:       endpoint,
		ForcePathStyle: true,
	}

	storage, err := provider.New(ctx, settings)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	url, headers, err := storage.PresignPut(ctx, bucket, "upload-test.txt", 15*time.Minute, "text/plain")
	if err != nil {
		t.Fatalf("PresignPut failed: %v", err)
	}
	if url == "" {
		t.Error("expected non-empty URL")
	}

	// Try to actually upload a file using the presigned URL
	content := []byte("uploaded content")
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "text/plain")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to PUT: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected 200 OK, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestS3Integration_HeadObject_Exists(t *testing.T) {
	endpoint, accessKey, secretKey := setupMinIOContainer(t)

	ctx := context.Background()
	bucket := "head-object-bucket"
	createTestBucket(t, ctx, endpoint, accessKey, secretKey, bucket)
	uploadTestObject(t, ctx, endpoint, accessKey, secretKey, bucket, "existing.txt", []byte("existing content"))

	provider := integrationS3Provider(accessKey, secretKey)
	settings := DownloadStorageSettings{
		Provider:       "s3",
		Bucket:         bucket,
		Region:         "us-east-1",
		Endpoint:       endpoint,
		ForcePathStyle: true,
	}

	storage, err := provider.New(ctx, settings)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	etag, size, contentType, err := storage.HeadObject(ctx, bucket, "existing.txt")
	if err != nil {
		t.Fatalf("HeadObject failed: %v", err)
	}

	if etag == "" {
		t.Error("expected non-empty etag")
	}
	if size != 16 { // len("existing content")
		t.Errorf("expected size 16, got %d", size)
	}
	if contentType == "" {
		t.Error("expected non-empty content-type")
	}
}

func TestS3Integration_HeadObject_NotFound(t *testing.T) {
	endpoint, accessKey, secretKey := setupMinIOContainer(t)

	ctx := context.Background()
	bucket := "head-notfound-bucket"
	createTestBucket(t, ctx, endpoint, accessKey, secretKey, bucket)

	provider := integrationS3Provider(accessKey, secretKey)
	settings := DownloadStorageSettings{
		Provider:       "s3",
		Bucket:         bucket,
		Region:         "us-east-1",
		Endpoint:       endpoint,
		ForcePathStyle: true,
	}

	storage, err := provider.New(ctx, settings)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	_, _, _, err = storage.HeadObject(ctx, bucket, "nonexistent.txt")
	if err == nil {
		t.Error("expected error for nonexistent object")
	}
}

func TestS3Integration_EndToEnd_UploadDownload(t *testing.T) {
	endpoint, accessKey, secretKey := setupMinIOContainer(t)

	ctx := context.Background()
	bucket := "end-to-end-bucket"
	createTestBucket(t, ctx, endpoint, accessKey, secretKey, bucket)

	provider := integrationS3Provider(accessKey, secretKey)
	settings := DownloadStorageSettings{
		Provider:       "s3",
		Bucket:         bucket,
		Region:         "us-east-1",
		Endpoint:       endpoint,
		ForcePathStyle: true,
	}

	storage, err := provider.New(ctx, settings)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	objectKey := "test/upload-download.bin"
	content := []byte("binary content for end-to-end test")

	// Step 1: Get presigned PUT URL
	putURL, headers, err := storage.PresignPut(ctx, bucket, objectKey, 15*time.Minute, "application/octet-stream")
	if err != nil {
		t.Fatalf("PresignPut failed: %v", err)
	}

	// Step 2: Upload using presigned URL
	req, _ := http.NewRequest(http.MethodPut, putURL, bytes.NewReader(content))
	req.Header.Set("Content-Type", "application/octet-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload returned %d", resp.StatusCode)
	}

	// Step 3: Verify with HeadObject
	etag, size, contentType, err := storage.HeadObject(ctx, bucket, objectKey)
	if err != nil {
		t.Fatalf("HeadObject failed: %v", err)
	}

	if etag == "" {
		t.Error("expected etag")
	}
	if size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), size)
	}
	if contentType != "application/octet-stream" {
		t.Errorf("expected content-type 'application/octet-stream', got '%s'", contentType)
	}

	// Step 4: Get presigned GET URL
	getURL, err := storage.PresignGet(ctx, bucket, objectKey, 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignGet failed: %v", err)
	}

	// Step 5: Download and verify
	getResp, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	defer getResp.Body.Close()

	downloaded, _ := io.ReadAll(getResp.Body)
	if !bytes.Equal(downloaded, content) {
		t.Error("downloaded content doesn't match uploaded content")
	}
}

// Helper functions for integration tests

func createTestBucket(t *testing.T, ctx context.Context, endpoint, accessKey, secretKey, bucket string) {
	t.Helper()

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: endpoint, HostnameImmutable: true}, nil
			})),
	)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	// Ignore "bucket already exists" errors
	if err != nil && !isBucketAlreadyExistsError(err) {
		t.Fatalf("failed to create bucket: %v", err)
	}
}

func uploadTestObject(t *testing.T, ctx context.Context, endpoint, accessKey, secretKey, bucket, key string, content []byte) {
	t.Helper()

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: endpoint, HostnameImmutable: true}, nil
			})),
	)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(content),
	})
	if err != nil {
		t.Fatalf("failed to upload object: %v", err)
	}
}

func isBucketAlreadyExistsError(err error) bool {
	// Simple string check for bucket exists error
	return err != nil && (
	// AWS S3
	fmt.Sprintf("%v", err) == "BucketAlreadyOwnedByYou" ||
		// MinIO returns different errors
		fmt.Sprintf("%v", err) == "BucketAlreadyExists" ||
		// Check error message
		contains(fmt.Sprintf("%v", err), "already own") ||
		contains(fmt.Sprintf("%v", err), "already exists"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}
