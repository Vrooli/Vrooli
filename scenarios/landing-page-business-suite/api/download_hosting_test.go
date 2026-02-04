package main

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
)

// mockDownloadStorage implements DownloadStorage for testing
type mockDownloadStorage struct {
	testConnectionErr error
	presignGetURL     string
	presignGetErr     error
	presignPutURL     string
	presignPutHeaders map[string]string
	presignPutErr     error
	headEtag          string
	headSize          int64
	headContentType   string
	headErr           error
}

func (m *mockDownloadStorage) TestConnection(ctx context.Context, bucket string) error {
	return m.testConnectionErr
}

func (m *mockDownloadStorage) PresignGet(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	return m.presignGetURL, m.presignGetErr
}

func (m *mockDownloadStorage) PresignPut(ctx context.Context, bucket, key string, ttl time.Duration, contentType string) (string, map[string]string, error) {
	return m.presignPutURL, m.presignPutHeaders, m.presignPutErr
}

func (m *mockDownloadStorage) HeadObject(ctx context.Context, bucket, key string) (etag string, size int64, contentType string, err error) {
	return m.headEtag, m.headSize, m.headContentType, m.headErr
}

// mockStorageProvider implements DownloadStorageProvider for testing
type mockStorageProvider struct {
	storage *mockDownloadStorage
	newErr  error
}

func (m *mockStorageProvider) ProviderKey() string {
	return "s3"
}

func (m *mockStorageProvider) New(ctx context.Context, settings DownloadStorageSettings) (DownloadStorage, error) {
	if m.newErr != nil {
		return nil, m.newErr
	}
	return m.storage, nil
}

func TestNewDownloadHostingService(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewDownloadHostingService(db)
	if service == nil {
		t.Fatal("NewDownloadHostingService returned nil")
	}
	if service.db != db {
		t.Error("Expected service to hold reference to provided db")
	}
	// Should have default s3 provider
	if _, ok := service.providers["s3"]; !ok {
		t.Error("Expected default s3 provider to be registered")
	}
}

func TestNewDownloadHostingService_WithCustomProvider(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	customProvider := &mockStorageProvider{storage: &mockDownloadStorage{}}
	service := NewDownloadHostingService(db, customProvider)

	if _, ok := service.providers["s3"]; !ok {
		t.Error("Expected s3 provider to be registered")
	}
}

func TestDownloadHostingService_GetSettings_NotConfigured(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	service := NewDownloadHostingService(db)
	ctx := context.Background()

	settings, err := service.GetSettings(ctx, "nonexistent_bundle")
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if settings != nil {
		t.Error("Expected nil settings for non-configured bundle")
	}
}

func TestDownloadHostingService_GetSettings_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	// Insert test settings
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, endpoint, force_path_style, default_prefix, signed_url_ttl_seconds, public_base_url)
		VALUES ('test_bundle', 's3', 'test-bucket', 'us-east-1', 'https://s3.test.com', true, 'downloads/', 3600, 'https://cdn.test.com')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test settings: %v", err)
	}

	service := NewDownloadHostingService(db)
	ctx := context.Background()

	settings, err := service.GetSettings(ctx, "test_bundle")
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if settings == nil {
		t.Fatal("Expected settings, got nil")
	}
	if settings.Provider != "s3" {
		t.Errorf("Expected provider 's3', got '%s'", settings.Provider)
	}
	if settings.Bucket != "test-bucket" {
		t.Errorf("Expected bucket 'test-bucket', got '%s'", settings.Bucket)
	}
	if settings.Region != "us-east-1" {
		t.Errorf("Expected region 'us-east-1', got '%s'", settings.Region)
	}
	if !settings.ForcePathStyle {
		t.Error("Expected ForcePathStyle to be true")
	}
	if settings.SignedURLTTLSeconds != 3600 {
		t.Errorf("Expected TTL 3600, got %d", settings.SignedURLTTLSeconds)
	}
}

func TestDownloadHostingService_GetSettings_EmptyBundleKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewDownloadHostingService(db)
	ctx := context.Background()

	_, err := service.GetSettings(ctx, "")
	if err == nil {
		t.Error("Expected error for empty bundle_key")
	}
}

func TestDownloadHostingService_SaveSettings_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	service := NewDownloadHostingService(db)
	ctx := context.Background()

	bucket := "new-bucket"
	region := "eu-west-1"
	ttl := 1800

	update := DownloadStorageSettingsUpdate{
		Bucket:              &bucket,
		Region:              &region,
		SignedURLTTLSeconds: &ttl,
	}

	snapshot, err := service.SaveSettings(ctx, "new_bundle", update)
	if err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}
	if snapshot == nil {
		t.Fatal("Expected snapshot, got nil")
	}
	if snapshot.Bucket != "new-bucket" {
		t.Errorf("Expected bucket 'new-bucket', got '%s'", snapshot.Bucket)
	}
	if snapshot.Region != "eu-west-1" {
		t.Errorf("Expected region 'eu-west-1', got '%s'", snapshot.Region)
	}
	if snapshot.SignedURLTTLSeconds != 1800 {
		t.Errorf("Expected TTL 1800, got %d", snapshot.SignedURLTTLSeconds)
	}
	if !snapshot.SettingsRowAvailable {
		t.Error("Expected SettingsRowAvailable to be true")
	}
}

func TestDownloadHostingService_SaveSettings_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	// Insert initial settings
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('update_bundle', 's3', 'old-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert initial settings: %v", err)
	}

	service := NewDownloadHostingService(db)
	ctx := context.Background()

	newBucket := "updated-bucket"
	update := DownloadStorageSettingsUpdate{
		Bucket: &newBucket,
	}

	snapshot, err := service.SaveSettings(ctx, "update_bundle", update)
	if err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}
	if snapshot.Bucket != "updated-bucket" {
		t.Errorf("Expected bucket 'updated-bucket', got '%s'", snapshot.Bucket)
	}
	// Region should be preserved
	if snapshot.Region != "us-east-1" {
		t.Errorf("Expected region 'us-east-1' to be preserved, got '%s'", snapshot.Region)
	}
}

func TestDownloadHostingService_SaveSettings_ValidationErrors(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	service := NewDownloadHostingService(db)
	ctx := context.Background()

	tests := []struct {
		name   string
		update DownloadStorageSettingsUpdate
	}{
		{
			name: "invalid endpoint",
			update: func() DownloadStorageSettingsUpdate {
				endpoint := "not-a-url"
				bucket := "test-bucket"
				return DownloadStorageSettingsUpdate{Bucket: &bucket, Endpoint: &endpoint}
			}(),
		},
		{
			name: "invalid TTL zero",
			update: func() DownloadStorageSettingsUpdate {
				bucket := "test-bucket"
				ttl := 0
				return DownloadStorageSettingsUpdate{Bucket: &bucket, SignedURLTTLSeconds: &ttl}
			}(),
		},
		{
			name: "TTL too large",
			update: func() DownloadStorageSettingsUpdate {
				bucket := "test-bucket"
				ttl := 100000 // > 86400
				return DownloadStorageSettingsUpdate{Bucket: &bucket, SignedURLTTLSeconds: &ttl}
			}(),
		},
		{
			name: "mismatched credentials",
			update: func() DownloadStorageSettingsUpdate {
				bucket := "test-bucket"
				accessKey := "key"
				return DownloadStorageSettingsUpdate{Bucket: &bucket, AccessKeyID: &accessKey}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.SaveSettings(ctx, "validation_test", tt.update)
			if err == nil {
				t.Error("Expected validation error")
			}
		})
	}
}

func TestDownloadHostingService_SettingsSnapshot_NotConfigured(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	service := NewDownloadHostingService(db)
	ctx := context.Background()

	snapshot, err := service.SettingsSnapshot(ctx, "unconfigured_bundle")
	if err != nil {
		t.Fatalf("SettingsSnapshot failed: %v", err)
	}
	if snapshot == nil {
		t.Fatal("Expected snapshot, got nil")
	}
	if snapshot.SettingsRowAvailable {
		t.Error("Expected SettingsRowAvailable to be false")
	}
	if snapshot.Provider != "s3" {
		t.Errorf("Expected default provider 's3', got '%s'", snapshot.Provider)
	}
	if snapshot.SignedURLTTLSeconds != 900 {
		t.Errorf("Expected default TTL 900, got %d", snapshot.SignedURLTTLSeconds)
	}
	if !snapshot.CredentialsFromEnv {
		t.Error("Expected CredentialsFromEnv to be true for unconfigured settings")
	}
}

func TestDownloadHostingService_TestConnection_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	mockStorage := &mockDownloadStorage{testConnectionErr: nil}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	service := NewDownloadHostingService(db, mockProvider)
	ctx := context.Background()

	// Insert settings
	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('connection_test', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	err = service.TestConnection(ctx, "connection_test")
	if err != nil {
		t.Errorf("TestConnection failed: %v", err)
	}
}

func TestDownloadHostingService_TestConnection_NotConfigured(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	service := NewDownloadHostingService(db)
	ctx := context.Background()

	err := service.TestConnection(ctx, "not_configured")
	if err == nil {
		t.Error("Expected error for not configured bundle")
	}
	if !errors.Is(err, ErrDownloadStorageNotConfigured) {
		t.Errorf("Expected ErrDownloadStorageNotConfigured, got %v", err)
	}
}

func TestDownloadHostingService_TestConnection_Error(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	mockStorage := &mockDownloadStorage{testConnectionErr: errors.New("connection failed")}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	service := NewDownloadHostingService(db, mockProvider)
	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('connection_error', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	err = service.TestConnection(ctx, "connection_error")
	if err == nil {
		t.Error("Expected error from connection test")
	}
}

func TestDownloadHostingService_PresignUpload_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	mockStorage := &mockDownloadStorage{
		presignPutURL:     "https://s3.test.com/upload?signed=yes",
		presignPutHeaders: map[string]string{"x-amz-meta-test": "value"},
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	service := NewDownloadHostingService(db, mockProvider)
	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, default_prefix, signed_url_ttl_seconds)
		VALUES ('upload_test', 's3', 'test-bucket', 'us-east-1', 'artifacts/', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	req := PresignUploadRequest{
		Filename:       "test-app.zip",
		ContentType:    "application/zip",
		AppKey:         "my-app",
		Platform:       "windows",
		ReleaseVersion: "1.0.0",
	}

	resp, err := service.PresignUpload(ctx, "upload_test", req)
	if err != nil {
		t.Fatalf("PresignUpload failed: %v", err)
	}
	if resp == nil {
		t.Fatal("Expected response, got nil")
	}
	if resp.UploadURL != "https://s3.test.com/upload?signed=yes" {
		t.Errorf("Expected upload URL, got '%s'", resp.UploadURL)
	}
	if resp.Bucket != "test-bucket" {
		t.Errorf("Expected bucket 'test-bucket', got '%s'", resp.Bucket)
	}
	if resp.ObjectKey == "" {
		t.Error("Expected ObjectKey to be set")
	}
}

func TestDownloadHostingService_PresignUpload_MissingFilename(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	mockStorage := &mockDownloadStorage{}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	service := NewDownloadHostingService(db, mockProvider)
	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('missing_filename', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	req := PresignUploadRequest{
		Filename: "", // Missing
	}

	_, err = service.PresignUpload(ctx, "missing_filename", req)
	if err == nil {
		t.Error("Expected error for missing filename")
	}
}

func TestDownloadHostingService_PresignUpload_NotConfigured(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	service := NewDownloadHostingService(db)
	ctx := context.Background()

	req := PresignUploadRequest{Filename: "test.zip"}

	_, err := service.PresignUpload(ctx, "not_configured", req)
	if err == nil {
		t.Error("Expected error for not configured bundle")
	}
}

func TestDownloadHostingService_CommitArtifact_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadArtifacts(t, db)

	mockStorage := &mockDownloadStorage{
		headEtag:        "abc123",
		headSize:        1024000,
		headContentType: "application/zip",
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	service := NewDownloadHostingService(db, mockProvider)
	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('commit_test', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	req := CommitArtifactRequest{
		Bucket:           "test-bucket",
		ObjectKey:        "artifacts/my-app/1.0.0/app.zip",
		OriginalFilename: "app.zip",
		ContentType:      "application/zip",
		Platform:         "windows",
		ReleaseVersion:   "1.0.0",
	}

	artifact, err := service.CommitArtifact(ctx, "commit_test", req)
	if err != nil {
		t.Fatalf("CommitArtifact failed: %v", err)
	}
	if artifact == nil {
		t.Fatal("Expected artifact, got nil")
	}
	if artifact.ID == 0 {
		t.Error("Expected artifact ID to be set")
	}
	if artifact.ETag != "abc123" {
		t.Errorf("Expected ETag 'abc123', got '%s'", artifact.ETag)
	}
	if artifact.SizeBytes != 1024000 {
		t.Errorf("Expected size 1024000, got %d", artifact.SizeBytes)
	}
}

func TestDownloadHostingService_CommitArtifact_Upsert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadArtifacts(t, db)

	mockStorage := &mockDownloadStorage{
		headEtag:        "abc123",
		headSize:        1024000,
		headContentType: "application/zip",
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	service := NewDownloadHostingService(db, mockProvider)
	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('upsert_test', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	req := CommitArtifactRequest{
		Bucket:         "test-bucket",
		ObjectKey:      "artifacts/upsert/app.zip",
		ReleaseVersion: "1.0.0",
	}

	artifact1, err := service.CommitArtifact(ctx, "upsert_test", req)
	if err != nil {
		t.Fatalf("First CommitArtifact failed: %v", err)
	}

	// Update version
	mockStorage.headEtag = "def456"
	req.ReleaseVersion = "2.0.0"

	artifact2, err := service.CommitArtifact(ctx, "upsert_test", req)
	if err != nil {
		t.Fatalf("Second CommitArtifact failed: %v", err)
	}

	if artifact1.ID != artifact2.ID {
		t.Errorf("Expected same ID on upsert, got %d and %d", artifact1.ID, artifact2.ID)
	}
	if artifact2.ReleaseVersion != "2.0.0" {
		t.Errorf("Expected updated version '2.0.0', got '%s'", artifact2.ReleaseVersion)
	}
}

func TestDownloadHostingService_CommitArtifact_HeadObjectError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	mockStorage := &mockDownloadStorage{
		headErr: errors.New("object not found"),
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	service := NewDownloadHostingService(db, mockProvider)
	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('head_error', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	req := CommitArtifactRequest{
		Bucket:    "test-bucket",
		ObjectKey: "nonexistent/file.zip",
	}

	_, err = service.CommitArtifact(ctx, "head_error", req)
	if err == nil {
		t.Error("Expected error from HeadObject failure")
	}
}

func TestDownloadHostingService_GetArtifact_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadArtifacts(t, db)

	mockStorage := &mockDownloadStorage{
		headEtag:        "abc123",
		headSize:        512000,
		headContentType: "application/zip",
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	service := NewDownloadHostingService(db, mockProvider)
	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('get_artifact', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	// Create artifact first
	req := CommitArtifactRequest{
		Bucket:    "test-bucket",
		ObjectKey: "get/test.zip",
	}
	created, err := service.CommitArtifact(ctx, "get_artifact", req)
	if err != nil {
		t.Fatalf("CommitArtifact failed: %v", err)
	}

	// Now get it
	artifact, err := service.GetArtifact(ctx, "get_artifact", created.ID)
	if err != nil {
		t.Fatalf("GetArtifact failed: %v", err)
	}
	if artifact == nil {
		t.Fatal("Expected artifact, got nil")
	}
	if artifact.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, artifact.ID)
	}
}

func TestDownloadHostingService_GetArtifact_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewDownloadHostingService(db)
	ctx := context.Background()

	artifact, err := service.GetArtifact(ctx, "any_bundle", 99999)
	if err != nil {
		t.Fatalf("GetArtifact should not return error for not found: %v", err)
	}
	if artifact != nil {
		t.Error("Expected nil artifact for not found")
	}
}

func TestDownloadHostingService_ListArtifacts_Pagination(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadArtifacts(t, db)

	mockStorage := &mockDownloadStorage{
		headEtag:        "abc123",
		headSize:        1000,
		headContentType: "application/octet-stream",
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	service := NewDownloadHostingService(db, mockProvider)
	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('list_test', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	// Create multiple artifacts
	for i := 0; i < 15; i++ {
		req := CommitArtifactRequest{
			Bucket:    "test-bucket",
			ObjectKey: "list/" + string(rune('a'+i)) + ".zip",
		}
		_, err := service.CommitArtifact(ctx, "list_test", req)
		if err != nil {
			t.Fatalf("CommitArtifact failed: %v", err)
		}
	}

	// Test first page
	result, err := service.ListArtifacts(ctx, "list_test", "", "", "", 1, 5)
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}
	if result.Total != 15 {
		t.Errorf("Expected total 15, got %d", result.Total)
	}
	if len(result.Artifacts) != 5 {
		t.Errorf("Expected 5 artifacts on first page, got %d", len(result.Artifacts))
	}
	if result.Page != 1 {
		t.Errorf("Expected page 1, got %d", result.Page)
	}

	// Test second page
	result, err = service.ListArtifacts(ctx, "list_test", "", "", "", 2, 5)
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}
	if len(result.Artifacts) != 5 {
		t.Errorf("Expected 5 artifacts on second page, got %d", len(result.Artifacts))
	}
	if result.Page != 2 {
		t.Errorf("Expected page 2, got %d", result.Page)
	}
}

func TestDownloadHostingService_PresignGetArtifact_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)

	mockStorage := &mockDownloadStorage{
		presignGetURL: "https://s3.test.com/download?signed=yes",
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	service := NewDownloadHostingService(db, mockProvider)
	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('presign_get', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	artifact := DownloadArtifact{
		Bucket:    "test-bucket",
		ObjectKey: "download/test.zip",
	}

	url, err := service.PresignGetArtifact(ctx, "presign_get", artifact)
	if err != nil {
		t.Fatalf("PresignGetArtifact failed: %v", err)
	}
	if url != "https://s3.test.com/download?signed=yes" {
		t.Errorf("Expected signed URL, got '%s'", url)
	}
}

func TestSanitizeObjectFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple.txt", "simple.txt"},
		{"  spaced.txt  ", "spaced.txt"},
		{"path/to/file.txt", "file.txt"},
		{"with spaces.txt", "with-spaces.txt"},
		{"../../../etc/passwd", "passwd"},
		{"", "artifact.bin"},
		{".", "artifact.bin"},
		{"/", "artifact.bin"},
		{"file..name.txt", "file.name.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeObjectFilename(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestBuildObjectKey(t *testing.T) {
	settings := DownloadStorageSettings{
		DefaultPrefix: "downloads/",
	}

	req := PresignUploadRequest{
		Filename:       "app.zip",
		AppKey:         "my-app",
		Platform:       "windows",
		ReleaseVersion: "1.0.0",
	}

	key, err := buildObjectKey(settings, "test_bundle", req)
	if err != nil {
		t.Fatalf("buildObjectKey failed: %v", err)
	}

	// Key should contain prefix, bundle key, app key, platform, version, and filename
	if key == "" {
		t.Error("Expected non-empty key")
	}
	// Check it contains expected segments
	if len(key) < 10 {
		t.Errorf("Key seems too short: '%s'", key)
	}
}

// cleanupDownloadStorageSettings removes all entries from download_storage_settings table
func cleanupDownloadStorageSettings(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec("DELETE FROM download_storage_settings"); err != nil {
		t.Fatalf("Failed to cleanup download_storage_settings table: %v", err)
	}
}

// cleanupDownloadArtifacts removes all entries from download_artifacts table
func cleanupDownloadArtifacts(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec("DELETE FROM download_artifacts"); err != nil {
		t.Fatalf("Failed to cleanup download_artifacts table: %v", err)
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestNormalizeOptionalString_Nil(t *testing.T) {
	result := normalizeOptionalString(nil)
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}

func TestNormalizeOptionalString_Empty(t *testing.T) {
	empty := ""
	result := normalizeOptionalString(&empty)
	if result != nil {
		t.Errorf("expected nil for empty string, got %v", result)
	}
}

func TestNormalizeOptionalString_Whitespace(t *testing.T) {
	whitespace := "   "
	result := normalizeOptionalString(&whitespace)
	if result != nil {
		t.Errorf("expected nil for whitespace-only string, got %v", result)
	}
}

func TestNormalizeOptionalString_ValidValue(t *testing.T) {
	value := "  test-value  "
	result := normalizeOptionalString(&value)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if *result != "test-value" {
		t.Errorf("expected 'test-value', got '%s'", *result)
	}
}

func TestStableS3URI_Basic(t *testing.T) {
	result := stableS3URI("my-bucket", "path/to/object.zip")
	expected := "s3://my-bucket/path/to/object.zip"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestStableS3URI_EmptyKey(t *testing.T) {
	result := stableS3URI("my-bucket", "")
	expected := "s3://my-bucket/"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestS3DownloadStorageProvider_ProviderKey(t *testing.T) {
	provider := S3DownloadStorageProvider{}
	if key := provider.ProviderKey(); key != "s3" {
		t.Errorf("expected 's3', got '%s'", key)
	}
}

func TestRandomHex_Generates12Chars(t *testing.T) {
	result, err := randomHex(6)
	if err != nil {
		t.Fatalf("randomHex failed: %v", err)
	}
	if len(result) != 12 { // 6 bytes = 12 hex chars
		t.Errorf("expected 12 characters, got %d", len(result))
	}
}

func TestRandomHex_GeneratesUnique(t *testing.T) {
	results := make(map[string]bool)
	for i := 0; i < 100; i++ {
		hex, err := randomHex(6)
		if err != nil {
			t.Fatalf("randomHex failed: %v", err)
		}
		if results[hex] {
			t.Errorf("duplicate hex generated: %s", hex)
		}
		results[hex] = true
	}
}

func TestBuildObjectKey_WithAllSegments(t *testing.T) {
	settings := DownloadStorageSettings{
		DefaultPrefix: "artifacts/",
	}

	req := PresignUploadRequest{
		Filename:       "app.zip",
		AppKey:         "my-app",
		Platform:       "windows",
		ReleaseVersion: "1.0.0",
	}

	key, err := buildObjectKey(settings, "test_bundle", req)
	if err != nil {
		t.Fatalf("buildObjectKey failed: %v", err)
	}

	// Should contain: prefix/bundle/app/platform/version/timestamp-nonce-filename
	if key == "" {
		t.Error("expected non-empty key")
	}
	if !strings.Contains(key, "artifacts/") {
		t.Error("expected key to contain prefix")
	}
	if !strings.Contains(key, "test_bundle") {
		t.Error("expected key to contain bundle key")
	}
	if !strings.Contains(key, "my-app") {
		t.Error("expected key to contain app key")
	}
	if !strings.Contains(key, "windows") {
		t.Error("expected key to contain platform")
	}
	if !strings.Contains(key, "1.0.0") {
		t.Error("expected key to contain version")
	}
	if !strings.Contains(key, "app.zip") {
		t.Error("expected key to contain filename")
	}
}

func TestBuildObjectKey_WithoutOptionalSegments(t *testing.T) {
	settings := DownloadStorageSettings{}

	req := PresignUploadRequest{
		Filename: "simple.bin",
	}

	key, err := buildObjectKey(settings, "bundle", req)
	if err != nil {
		t.Fatalf("buildObjectKey failed: %v", err)
	}

	if !strings.Contains(key, "bundle") {
		t.Error("expected key to contain bundle key")
	}
	if !strings.Contains(key, "simple.bin") {
		t.Error("expected key to contain filename")
	}
}

func TestDownloadHostingService_ValidateStorageSettings_UnsupportedProvider(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewDownloadHostingService(db)

	settings := DownloadStorageSettings{
		Provider:            "unsupported",
		Bucket:              "test-bucket",
		SignedURLTTLSeconds: 900,
	}

	err := service.validateStorageSettings(settings)
	if err == nil {
		t.Error("expected error for unsupported provider")
	}
	if !strings.Contains(err.Error(), "unsupported provider") {
		t.Errorf("expected error about unsupported provider, got: %v", err)
	}
}

func TestDownloadHostingService_ValidateStorageSettings_InvalidEndpoint(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewDownloadHostingService(db)

	settings := DownloadStorageSettings{
		Provider:            "s3",
		Bucket:              "test-bucket",
		Endpoint:            "not-a-url",
		SignedURLTTLSeconds: 900,
	}

	err := service.validateStorageSettings(settings)
	if err == nil {
		t.Error("expected error for invalid endpoint")
	}
}

func TestDownloadHostingService_ValidateStorageSettings_InvalidTTL(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewDownloadHostingService(db)

	tests := []struct {
		name string
		ttl  int
	}{
		{"zero", 0},
		{"negative", -100},
		{"too_large", 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := DownloadStorageSettings{
				Provider:            "s3",
				Bucket:              "test-bucket",
				SignedURLTTLSeconds: tt.ttl,
			}

			err := service.validateStorageSettings(settings)
			if err == nil {
				t.Errorf("expected error for TTL %d", tt.ttl)
			}
		})
	}
}

func TestDownloadHostingService_ValidateStorageSettings_MismatchedCredentials(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewDownloadHostingService(db)

	// Only access key, no secret
	settings := DownloadStorageSettings{
		Provider:            "s3",
		Bucket:              "test-bucket",
		SignedURLTTLSeconds: 900,
		AccessKeyID:         "access-key",
	}

	err := service.validateStorageSettings(settings)
	if err == nil {
		t.Error("expected error for mismatched credentials")
	}

	// Only secret, no access key
	settings = DownloadStorageSettings{
		Provider:            "s3",
		Bucket:              "test-bucket",
		SignedURLTTLSeconds: 900,
		SecretAccessKey:     "secret-key",
	}

	err = service.validateStorageSettings(settings)
	if err == nil {
		t.Error("expected error for mismatched credentials")
	}
}

func TestDownloadHostingService_ValidateStorageSettings_ValidSettings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewDownloadHostingService(db)

	settings := DownloadStorageSettings{
		Provider:            "s3",
		Bucket:              "test-bucket",
		Region:              "us-east-1",
		Endpoint:            "https://s3.amazonaws.com",
		SignedURLTTLSeconds: 900,
		AccessKeyID:         "access-key",
		SecretAccessKey:     "secret-key",
	}

	err := service.validateStorageSettings(settings)
	if err != nil {
		t.Errorf("expected no error for valid settings, got: %v", err)
	}
}

func TestDownloadHostingService_ListArtifacts_WithSearchQuery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadArtifacts(t, db)

	mockStorage := &mockDownloadStorage{
		headEtag:        "abc123",
		headSize:        1000,
		headContentType: "application/octet-stream",
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	service := NewDownloadHostingService(db, mockProvider)
	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('search_test', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	// Create artifacts with different names
	artifacts := []CommitArtifactRequest{
		{Bucket: "test-bucket", ObjectKey: "search/alpha.zip", OriginalFilename: "alpha.zip"},
		{Bucket: "test-bucket", ObjectKey: "search/beta.zip", OriginalFilename: "beta.zip"},
		{Bucket: "test-bucket", ObjectKey: "search/gamma.zip", OriginalFilename: "gamma.zip"},
	}

	for _, req := range artifacts {
		_, err := service.CommitArtifact(ctx, "search_test", req)
		if err != nil {
			t.Fatalf("CommitArtifact failed: %v", err)
		}
	}

	// Search for "alpha"
	result, err := service.ListArtifacts(ctx, "search_test", "alpha", "", "", 1, 50)
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}

	if result.Total != 1 {
		t.Errorf("expected 1 result for 'alpha' search, got %d", result.Total)
	}
}

func TestDownloadHostingService_ListArtifacts_WithPlatformFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadStorageSettings(t, db)
	cleanupDownloadArtifacts(t, db)

	mockStorage := &mockDownloadStorage{
		headEtag:        "abc123",
		headSize:        1000,
		headContentType: "application/octet-stream",
	}
	mockProvider := &mockStorageProvider{storage: mockStorage}

	service := NewDownloadHostingService(db, mockProvider)
	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO download_storage_settings (bundle_key, provider, bucket, region, signed_url_ttl_seconds)
		VALUES ('platform_test', 's3', 'test-bucket', 'us-east-1', 900)
	`)
	if err != nil {
		t.Fatalf("Failed to insert settings: %v", err)
	}

	// Create artifacts with different platforms
	artifacts := []CommitArtifactRequest{
		{Bucket: "test-bucket", ObjectKey: "platform/win.exe", Platform: "windows"},
		{Bucket: "test-bucket", ObjectKey: "platform/mac.dmg", Platform: "macos"},
		{Bucket: "test-bucket", ObjectKey: "platform/linux.tar", Platform: "linux"},
	}

	for _, req := range artifacts {
		_, err := service.CommitArtifact(ctx, "platform_test", req)
		if err != nil {
			t.Fatalf("CommitArtifact failed: %v", err)
		}
	}

	// Filter by platform
	result, err := service.ListArtifacts(ctx, "platform_test", "", "windows", "", 1, 50)
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}

	if result.Total != 1 {
		t.Errorf("expected 1 result for 'windows' platform, got %d", result.Total)
	}
}
