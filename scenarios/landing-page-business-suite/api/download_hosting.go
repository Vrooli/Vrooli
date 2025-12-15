package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type DownloadStorage interface {
	TestConnection(ctx context.Context, bucket string) error
	PresignGet(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
	PresignPut(ctx context.Context, bucket, key string, ttl time.Duration, contentType string) (string, map[string]string, error)
	HeadObject(ctx context.Context, bucket, key string) (etag string, size int64, contentType string, err error)
}

type DownloadStorageProvider interface {
	ProviderKey() string
	New(ctx context.Context, settings DownloadStorageSettings) (DownloadStorage, error)
}

type DownloadStorageSettings struct {
	ID                  int64
	BundleKey           string
	Provider            string
	Bucket              string
	Region              string
	Endpoint            string
	ForcePathStyle      bool
	DefaultPrefix       string
	SignedURLTTLSeconds int
	PublicBaseURL       string
	AccessKeyID         string
	SecretAccessKey     string
	SessionToken        string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type DownloadStorageSettingsSnapshot struct {
	Provider             string `json:"provider"`
	Bucket               string `json:"bucket,omitempty"`
	Region               string `json:"region,omitempty"`
	Endpoint             string `json:"endpoint,omitempty"`
	ForcePathStyle       bool   `json:"force_path_style"`
	DefaultPrefix        string `json:"default_prefix,omitempty"`
	SignedURLTTLSeconds  int    `json:"signed_url_ttl_seconds"`
	PublicBaseURL        string `json:"public_base_url,omitempty"`
	AccessKeyIDSet       bool   `json:"access_key_id_set"`
	SecretAccessKeySet   bool   `json:"secret_access_key_set"`
	SessionTokenSet      bool   `json:"session_token_set"`
	CredentialsFromEnv   bool   `json:"credentials_from_env"`
	SettingsRowAvailable bool   `json:"settings_row_available"`
}

type DownloadStorageSettingsUpdate struct {
	Provider            *string `json:"provider"`
	Bucket              *string `json:"bucket"`
	Region              *string `json:"region"`
	Endpoint            *string `json:"endpoint"`
	ForcePathStyle      *bool   `json:"force_path_style"`
	DefaultPrefix       *string `json:"default_prefix"`
	SignedURLTTLSeconds *int    `json:"signed_url_ttl_seconds"`
	PublicBaseURL       *string `json:"public_base_url"`
	AccessKeyID         *string `json:"access_key_id"`
	SecretAccessKey     *string `json:"secret_access_key"`
	SessionToken        *string `json:"session_token"`
}

type DownloadArtifact struct {
	ID                int64                  `json:"id"`
	BundleKey         string                 `json:"bundle_key"`
	Provider          string                 `json:"provider"`
	Bucket            string                 `json:"bucket"`
	ObjectKey         string                 `json:"object_key"`
	ETag              string                 `json:"etag,omitempty"`
	SizeBytes         int64                  `json:"size_bytes,omitempty"`
	SHA256            string                 `json:"sha256,omitempty"`
	ContentType       string                 `json:"content_type,omitempty"`
	OriginalFilename  string                 `json:"original_filename,omitempty"`
	Platform          string                 `json:"platform,omitempty"`
	ReleaseVersion    string                 `json:"release_version,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	StableObjectURI   string                 `json:"stable_object_uri,omitempty"`
	SignedDownloadURL string                 `json:"signed_download_url,omitempty"`
}

type DownloadHostingService struct {
	db        *sql.DB
	providers map[string]DownloadStorageProvider
}

func NewDownloadHostingService(db *sql.DB, providers ...DownloadStorageProvider) *DownloadHostingService {
	registered := map[string]DownloadStorageProvider{}
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		key := strings.TrimSpace(provider.ProviderKey())
		if key == "" {
			continue
		}
		registered[key] = provider
	}

	if _, ok := registered["s3"]; !ok {
		registered["s3"] = S3DownloadStorageProvider{}
	}

	return &DownloadHostingService{
		db:        db,
		providers: registered,
	}
}

var ErrDownloadStorageNotConfigured = errors.New("download storage not configured")

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (s *DownloadHostingService) GetSettings(ctx context.Context, bundleKey string) (*DownloadStorageSettings, error) {
	bundleKey = strings.TrimSpace(bundleKey)
	if bundleKey == "" {
		return nil, fmt.Errorf("bundle_key is required")
	}

	query := `
		SELECT id, bundle_key, provider, bucket, region, endpoint, force_path_style, default_prefix,
		       signed_url_ttl_seconds, public_base_url, access_key_id, secret_access_key, session_token,
		       created_at, updated_at
		FROM download_storage_settings
		WHERE bundle_key = $1
		LIMIT 1
	`

	row := s.db.QueryRowContext(ctx, query, bundleKey)
	var settings DownloadStorageSettings
	var provider, bucket, region, endpoint, defaultPrefix, publicBaseURL sql.NullString
	var accessKeyID, secretAccessKey, sessionToken sql.NullString
	var ttl sql.NullInt64
	if err := row.Scan(
		&settings.ID,
		&settings.BundleKey,
		&provider,
		&bucket,
		&region,
		&endpoint,
		&settings.ForcePathStyle,
		&defaultPrefix,
		&ttl,
		&publicBaseURL,
		&accessKeyID,
		&secretAccessKey,
		&sessionToken,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	settings.Provider = strings.TrimSpace(provider.String)
	if settings.Provider == "" {
		settings.Provider = "s3"
	}
	settings.Bucket = strings.TrimSpace(bucket.String)
	settings.Region = strings.TrimSpace(region.String)
	settings.Endpoint = strings.TrimSpace(endpoint.String)
	settings.DefaultPrefix = strings.TrimSpace(defaultPrefix.String)
	settings.PublicBaseURL = strings.TrimSpace(publicBaseURL.String)
	settings.AccessKeyID = strings.TrimSpace(accessKeyID.String)
	settings.SecretAccessKey = strings.TrimSpace(secretAccessKey.String)
	settings.SessionToken = strings.TrimSpace(sessionToken.String)
	if ttl.Valid && ttl.Int64 > 0 {
		settings.SignedURLTTLSeconds = int(ttl.Int64)
	} else {
		settings.SignedURLTTLSeconds = 900
	}

	return &settings, nil
}

func (s *DownloadHostingService) SettingsSnapshot(ctx context.Context, bundleKey string) (*DownloadStorageSettingsSnapshot, error) {
	settings, err := s.GetSettings(ctx, bundleKey)
	if err != nil {
		return nil, err
	}

	if settings == nil {
		return &DownloadStorageSettingsSnapshot{
			Provider:             "s3",
			SignedURLTTLSeconds:  900,
			CredentialsFromEnv:   true,
			SettingsRowAvailable: false,
		}, nil
	}

	credentialsFromEnv := strings.TrimSpace(settings.AccessKeyID) == "" && strings.TrimSpace(settings.SecretAccessKey) == ""

	return &DownloadStorageSettingsSnapshot{
		Provider:             settings.Provider,
		Bucket:               settings.Bucket,
		Region:               settings.Region,
		Endpoint:             settings.Endpoint,
		ForcePathStyle:       settings.ForcePathStyle,
		DefaultPrefix:        settings.DefaultPrefix,
		SignedURLTTLSeconds:  settings.SignedURLTTLSeconds,
		PublicBaseURL:        settings.PublicBaseURL,
		AccessKeyIDSet:       strings.TrimSpace(settings.AccessKeyID) != "",
		SecretAccessKeySet:   strings.TrimSpace(settings.SecretAccessKey) != "",
		SessionTokenSet:      strings.TrimSpace(settings.SessionToken) != "",
		CredentialsFromEnv:   credentialsFromEnv,
		SettingsRowAvailable: true,
	}, nil
}

func (s *DownloadHostingService) validateStorageSettings(settings DownloadStorageSettings) error {
	providerKey := strings.TrimSpace(settings.Provider)
	if providerKey == "" {
		providerKey = "s3"
	}
	if _, ok := s.providers[providerKey]; !ok {
		return fmt.Errorf("unsupported provider %q", providerKey)
	}

	if settings.Endpoint != "" {
		parsed, err := url.Parse(settings.Endpoint)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("endpoint must be a valid URL including scheme (e.g. https://...)")
		}
	}

	if settings.SignedURLTTLSeconds <= 0 {
		return fmt.Errorf("signed_url_ttl_seconds must be > 0")
	}
	if settings.SignedURLTTLSeconds > 24*60*60 {
		return fmt.Errorf("signed_url_ttl_seconds must be <= 86400")
	}

	if settings.PublicBaseURL != "" {
		parsed, err := url.Parse(settings.PublicBaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("public_base_url must be a valid URL including scheme (e.g. https://...)")
		}
	}

	hasAccessKey := strings.TrimSpace(settings.AccessKeyID) != ""
	hasSecret := strings.TrimSpace(settings.SecretAccessKey) != ""
	if hasAccessKey != hasSecret {
		return fmt.Errorf("access_key_id and secret_access_key must both be provided (or both left blank)")
	}

	return nil
}

func (s *DownloadHostingService) SaveSettings(ctx context.Context, bundleKey string, update DownloadStorageSettingsUpdate) (*DownloadStorageSettingsSnapshot, error) {
	bundleKey = strings.TrimSpace(bundleKey)
	if bundleKey == "" {
		return nil, fmt.Errorf("bundle_key is required")
	}

	existing, err := s.GetSettings(ctx, bundleKey)
	if err != nil {
		return nil, err
	}

	settings := DownloadStorageSettings{
		BundleKey:           bundleKey,
		Provider:            "s3",
		SignedURLTTLSeconds: 900,
	}
	if existing != nil {
		settings = *existing
	}

	if update.Provider != nil {
		settings.Provider = strings.TrimSpace(*update.Provider)
	}
	if update.Bucket != nil {
		settings.Bucket = strings.TrimSpace(*update.Bucket)
	}
	if update.Region != nil {
		settings.Region = strings.TrimSpace(*update.Region)
	}
	if update.Endpoint != nil {
		settings.Endpoint = strings.TrimSpace(*update.Endpoint)
	}
	if update.ForcePathStyle != nil {
		settings.ForcePathStyle = *update.ForcePathStyle
	}
	if update.DefaultPrefix != nil {
		settings.DefaultPrefix = strings.TrimSpace(*update.DefaultPrefix)
	}
	if update.SignedURLTTLSeconds != nil {
		settings.SignedURLTTLSeconds = *update.SignedURLTTLSeconds
	}
	if update.PublicBaseURL != nil {
		settings.PublicBaseURL = strings.TrimSpace(*update.PublicBaseURL)
	}
	if update.AccessKeyID != nil {
		settings.AccessKeyID = strings.TrimSpace(*update.AccessKeyID)
	}
	if update.SecretAccessKey != nil {
		settings.SecretAccessKey = strings.TrimSpace(*update.SecretAccessKey)
	}
	if update.SessionToken != nil {
		settings.SessionToken = strings.TrimSpace(*update.SessionToken)
	}

	if err := s.validateStorageSettings(settings); err != nil {
		return nil, err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO download_storage_settings (
			bundle_key, provider, bucket, region, endpoint, force_path_style, default_prefix,
			signed_url_ttl_seconds, public_base_url, access_key_id, secret_access_key, session_token, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12, NOW())
		ON CONFLICT (bundle_key) DO UPDATE SET
			provider = EXCLUDED.provider,
			bucket = EXCLUDED.bucket,
			region = EXCLUDED.region,
			endpoint = EXCLUDED.endpoint,
			force_path_style = EXCLUDED.force_path_style,
			default_prefix = EXCLUDED.default_prefix,
			signed_url_ttl_seconds = EXCLUDED.signed_url_ttl_seconds,
			public_base_url = EXCLUDED.public_base_url,
			access_key_id = EXCLUDED.access_key_id,
			secret_access_key = EXCLUDED.secret_access_key,
			session_token = EXCLUDED.session_token,
			updated_at = NOW()
	`, bundleKey, settings.Provider,
		normalizeOptionalString(&settings.Bucket),
		normalizeOptionalString(&settings.Region),
		normalizeOptionalString(&settings.Endpoint),
		settings.ForcePathStyle,
		normalizeOptionalString(&settings.DefaultPrefix),
		settings.SignedURLTTLSeconds,
		normalizeOptionalString(&settings.PublicBaseURL),
		normalizeOptionalString(&settings.AccessKeyID),
		normalizeOptionalString(&settings.SecretAccessKey),
		normalizeOptionalString(&settings.SessionToken),
	)
	if err != nil {
		return nil, fmt.Errorf("save download storage settings: %w", err)
	}

	return s.SettingsSnapshot(ctx, bundleKey)
}

type s3DownloadStorage struct {
	client    *s3.Client
	presigner *s3.PresignClient
}

type S3DownloadStorageProvider struct{}

func (S3DownloadStorageProvider) ProviderKey() string {
	return "s3"
}

func (S3DownloadStorageProvider) New(ctx context.Context, settings DownloadStorageSettings) (DownloadStorage, error) {
	return newS3DownloadStorage(ctx, settings)
}

func endpointResolverForS3(endpointURL string) aws.EndpointResolverWithOptionsFunc {
	return func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID && strings.TrimSpace(endpointURL) != "" {
			return aws.Endpoint{
				URL:               endpointURL,
				HostnameImmutable: true,
				SigningRegion:     region,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	}
}

func newS3DownloadStorage(ctx context.Context, settings DownloadStorageSettings) (*s3DownloadStorage, error) {
	region := strings.TrimSpace(settings.Region)
	if region == "" {
		region = "us-east-1"
	}

	loadOptions := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}
	if settings.Endpoint != "" {
		loadOptions = append(loadOptions, config.WithEndpointResolverWithOptions(endpointResolverForS3(settings.Endpoint)))
	}
	if strings.TrimSpace(settings.AccessKeyID) != "" || strings.TrimSpace(settings.SecretAccessKey) != "" {
		loadOptions = append(loadOptions, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			settings.AccessKeyID,
			settings.SecretAccessKey,
			settings.SessionToken,
		)))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = settings.ForcePathStyle
	})

	return &s3DownloadStorage{
		client:    client,
		presigner: s3.NewPresignClient(client),
	}, nil
}

func (s *s3DownloadStorage) TestConnection(ctx context.Context, bucket string) error {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	if err != nil {
		return err
	}
	return nil
}

func (s *s3DownloadStorage) PresignGet(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	req, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = ttl
	})
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (s *s3DownloadStorage) PresignPut(ctx context.Context, bucket, key string, ttl time.Duration, contentType string) (string, map[string]string, error) {
	headers := map[string]string{}
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if strings.TrimSpace(contentType) != "" {
		input.ContentType = aws.String(contentType)
	}

	req, err := s.presigner.PresignPutObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = ttl
	})
	if err != nil {
		return "", nil, err
	}

	for key, values := range req.SignedHeader {
		if len(values) == 0 {
			continue
		}
		headers[key] = values[0]
	}

	return req.URL, headers, nil
}

func (s *s3DownloadStorage) HeadObject(ctx context.Context, bucket, key string) (etag string, size int64, contentType string, err error) {
	resp, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", 0, "", err
	}

	if resp.ETag != nil {
		etag = strings.Trim(*resp.ETag, "\"")
	}
	if resp.ContentLength != nil {
		size = *resp.ContentLength
	}
	if resp.ContentType != nil {
		contentType = *resp.ContentType
	}

	return etag, size, contentType, nil
}

func (s *DownloadHostingService) resolveStorage(ctx context.Context, settings DownloadStorageSettings) (DownloadStorage, error) {
	providerKey := strings.TrimSpace(settings.Provider)
	if providerKey == "" {
		providerKey = "s3"
	}
	provider, ok := s.providers[providerKey]
	if !ok {
		return nil, fmt.Errorf("unsupported provider %q", providerKey)
	}
	return provider.New(ctx, settings)
}

func (s *DownloadHostingService) requireConfiguredSettings(ctx context.Context, bundleKey string) (*DownloadStorageSettings, error) {
	settings, err := s.GetSettings(ctx, bundleKey)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return nil, ErrDownloadStorageNotConfigured
	}
	if strings.TrimSpace(settings.Bucket) == "" {
		return nil, ErrDownloadStorageNotConfigured
	}
	if settings.SignedURLTTLSeconds <= 0 {
		settings.SignedURLTTLSeconds = 900
	}
	return settings, nil
}

func (s *DownloadHostingService) TestConnection(ctx context.Context, bundleKey string) error {
	settings, err := s.requireConfiguredSettings(ctx, bundleKey)
	if err != nil {
		return err
	}
	storage, err := s.resolveStorage(ctx, *settings)
	if err != nil {
		return err
	}
	return storage.TestConnection(ctx, settings.Bucket)
}

type PresignUploadRequest struct {
	Filename       string                 `json:"filename"`
	ContentType    string                 `json:"content_type"`
	AppKey         string                 `json:"app_key"`
	Platform       string                 `json:"platform"`
	ReleaseVersion string                 `json:"release_version"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type PresignUploadResponse struct {
	UploadURL       string            `json:"upload_url"`
	RequiredHeaders map[string]string `json:"required_headers"`
	Bucket          string            `json:"bucket"`
	ObjectKey       string            `json:"object_key"`
	ExpiresAt       time.Time         `json:"expires_at"`
	StableObjectURI string            `json:"stable_object_uri"`
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func sanitizeObjectFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	filename = path.Base(filename)
	if filename == "." || filename == "/" {
		return "artifact.bin"
	}
	filename = strings.ReplaceAll(filename, " ", "-")
	filename = strings.ReplaceAll(filename, "..", ".")
	filename = strings.Trim(filename, "/")
	if filename == "" {
		return "artifact.bin"
	}
	return filename
}

func buildObjectKey(settings DownloadStorageSettings, bundleKey string, req PresignUploadRequest) (string, error) {
	prefix := strings.Trim(strings.TrimSpace(settings.DefaultPrefix), "/")
	filename := sanitizeObjectFilename(req.Filename)
	nonce, err := randomHex(6)
	if err != nil {
		return "", err
	}
	segments := []string{}
	if prefix != "" {
		segments = append(segments, prefix)
	}
	segments = append(segments, bundleKey)
	if app := strings.TrimSpace(req.AppKey); app != "" {
		segments = append(segments, app)
	}
	if platform := strings.TrimSpace(req.Platform); platform != "" {
		segments = append(segments, platform)
	}
	if version := strings.TrimSpace(req.ReleaseVersion); version != "" {
		segments = append(segments, version)
	}
	segments = append(segments, fmt.Sprintf("%d-%s-%s", time.Now().UTC().Unix(), nonce, filename))
	return strings.Join(segments, "/"), nil
}

func stableS3URI(bucket, key string) string {
	return fmt.Sprintf("s3://%s/%s", bucket, key)
}

func (s *DownloadHostingService) PresignUpload(ctx context.Context, bundleKey string, req PresignUploadRequest) (*PresignUploadResponse, error) {
	settings, err := s.requireConfiguredSettings(ctx, bundleKey)
	if err != nil {
		return nil, err
	}
	storage, err := s.resolveStorage(ctx, *settings)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(req.Filename) == "" {
		return nil, fmt.Errorf("filename is required")
	}

	objectKey, err := buildObjectKey(*settings, bundleKey, req)
	if err != nil {
		return nil, err
	}

	ttl := 10 * time.Minute
	uploadURL, requiredHeaders, err := storage.PresignPut(ctx, settings.Bucket, objectKey, ttl, req.ContentType)
	if err != nil {
		return nil, err
	}

	return &PresignUploadResponse{
		UploadURL:       uploadURL,
		RequiredHeaders: requiredHeaders,
		Bucket:          settings.Bucket,
		ObjectKey:       objectKey,
		ExpiresAt:       time.Now().UTC().Add(ttl),
		StableObjectURI: stableS3URI(settings.Bucket, objectKey),
	}, nil
}

type CommitArtifactRequest struct {
	Bucket           string                 `json:"bucket"`
	ObjectKey        string                 `json:"object_key"`
	OriginalFilename string                 `json:"original_filename"`
	ContentType      string                 `json:"content_type"`
	Platform         string                 `json:"platform"`
	ReleaseVersion   string                 `json:"release_version"`
	SHA256           string                 `json:"sha256"`
	Metadata         map[string]interface{} `json:"metadata"`
}

func (s *DownloadHostingService) CommitArtifact(ctx context.Context, bundleKey string, req CommitArtifactRequest) (*DownloadArtifact, error) {
	settings, err := s.requireConfiguredSettings(ctx, bundleKey)
	if err != nil {
		return nil, err
	}
	storage, err := s.resolveStorage(ctx, *settings)
	if err != nil {
		return nil, err
	}

	bucket := strings.TrimSpace(req.Bucket)
	if bucket == "" {
		bucket = settings.Bucket
	}
	if strings.TrimSpace(bucket) == "" || strings.TrimSpace(req.ObjectKey) == "" {
		return nil, fmt.Errorf("bucket and object_key are required")
	}

	etag, size, headContentType, err := storage.HeadObject(ctx, bucket, req.ObjectKey)
	if err != nil {
		return nil, err
	}

	metadataBytes, err := json.Marshal(req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	contentType := strings.TrimSpace(req.ContentType)
	if contentType == "" {
		contentType = headContentType
	}

	var artifact DownloadArtifact
	var insertedMetadata []byte
	query := `
		INSERT INTO download_artifacts (
			bundle_key, provider, bucket, object_key, etag, size_bytes, sha256,
			content_type, original_filename, platform, release_version, metadata, updated_at
		) VALUES ($1,'s3',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11, NOW())
		ON CONFLICT (bundle_key, bucket, object_key) DO UPDATE SET
			etag = EXCLUDED.etag,
			size_bytes = EXCLUDED.size_bytes,
			sha256 = EXCLUDED.sha256,
			content_type = EXCLUDED.content_type,
			original_filename = EXCLUDED.original_filename,
			platform = EXCLUDED.platform,
			release_version = EXCLUDED.release_version,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, bundle_key, provider, bucket, object_key, etag, size_bytes, sha256,
		          content_type, original_filename, platform, release_version, metadata, created_at, updated_at
	`

	row := s.db.QueryRowContext(ctx, query,
		bundleKey,
		bucket,
		strings.TrimSpace(req.ObjectKey),
		normalizeOptionalString(&etag),
		size,
		normalizeOptionalString(&req.SHA256),
		normalizeOptionalString(&contentType),
		normalizeOptionalString(&req.OriginalFilename),
		normalizeOptionalString(&req.Platform),
		normalizeOptionalString(&req.ReleaseVersion),
		metadataBytes,
	)

	var provider, bucketOut, objectKey, etagOut, shaOut, ctypeOut, fnameOut, platformOut, versionOut sql.NullString
	var sizeOut sql.NullInt64
	if err := row.Scan(
		&artifact.ID,
		&artifact.BundleKey,
		&provider,
		&bucketOut,
		&objectKey,
		&etagOut,
		&sizeOut,
		&shaOut,
		&ctypeOut,
		&fnameOut,
		&platformOut,
		&versionOut,
		&insertedMetadata,
		&artifact.CreatedAt,
		&artifact.UpdatedAt,
	); err != nil {
		return nil, err
	}

	artifact.Provider = provider.String
	artifact.Bucket = bucketOut.String
	artifact.ObjectKey = objectKey.String
	artifact.ETag = etagOut.String
	if sizeOut.Valid {
		artifact.SizeBytes = sizeOut.Int64
	}
	artifact.SHA256 = shaOut.String
	artifact.ContentType = ctypeOut.String
	artifact.OriginalFilename = fnameOut.String
	artifact.Platform = platformOut.String
	artifact.ReleaseVersion = versionOut.String
	if len(insertedMetadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(insertedMetadata, &meta); err == nil {
			artifact.Metadata = meta
		}
	}
	artifact.StableObjectURI = stableS3URI(artifact.Bucket, artifact.ObjectKey)

	return &artifact, nil
}

func (s *DownloadHostingService) GetArtifact(ctx context.Context, bundleKey string, id int64) (*DownloadArtifact, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, bundle_key, provider, bucket, object_key, etag, size_bytes, sha256,
		       content_type, original_filename, platform, release_version, metadata, created_at, updated_at
		FROM download_artifacts
		WHERE bundle_key = $1 AND id = $2
		LIMIT 1
	`, bundleKey, id)

	var artifact DownloadArtifact
	var provider, bucket, objectKey, etag, shaOut, ctypeOut, fnameOut, platformOut, versionOut sql.NullString
	var sizeOut sql.NullInt64
	var metadataBytes []byte
	if err := row.Scan(
		&artifact.ID,
		&artifact.BundleKey,
		&provider,
		&bucket,
		&objectKey,
		&etag,
		&sizeOut,
		&shaOut,
		&ctypeOut,
		&fnameOut,
		&platformOut,
		&versionOut,
		&metadataBytes,
		&artifact.CreatedAt,
		&artifact.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	artifact.Provider = provider.String
	artifact.Bucket = bucket.String
	artifact.ObjectKey = objectKey.String
	artifact.ETag = etag.String
	if sizeOut.Valid {
		artifact.SizeBytes = sizeOut.Int64
	}
	artifact.SHA256 = shaOut.String
	artifact.ContentType = ctypeOut.String
	artifact.OriginalFilename = fnameOut.String
	artifact.Platform = platformOut.String
	artifact.ReleaseVersion = versionOut.String
	if len(metadataBytes) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(metadataBytes, &meta); err == nil {
			artifact.Metadata = meta
		}
	}
	artifact.StableObjectURI = stableS3URI(artifact.Bucket, artifact.ObjectKey)

	return &artifact, nil
}

type ListArtifactsResult struct {
	Artifacts []DownloadArtifact `json:"artifacts"`
	Page      int                `json:"page"`
	PageSize  int                `json:"page_size"`
	Total     int                `json:"total"`
}

func (s *DownloadHostingService) ListArtifacts(ctx context.Context, bundleKey string, query, platform string, page, pageSize int) (*ListArtifactsResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	query = strings.TrimSpace(query)
	platform = strings.TrimSpace(platform)

	where := []string{"bundle_key = $1"}
	args := []interface{}{bundleKey}

	if platform != "" {
		args = append(args, platform)
		where = append(where, fmt.Sprintf("platform = $%d", len(args)))
	}

	if query != "" {
		like := "%" + strings.ToLower(query) + "%"
		args = append(args, like)
		param := len(args)
		where = append(where, fmt.Sprintf("(LOWER(original_filename) LIKE $%d OR LOWER(object_key) LIKE $%d OR LOWER(release_version) LIKE $%d)", param, param, param))
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM download_artifacts WHERE %s`, whereClause), args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, bundle_key, provider, bucket, object_key, etag, size_bytes, sha256,
		       content_type, original_filename, platform, release_version, metadata, created_at, updated_at
		FROM download_artifacts
		WHERE %s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &ListArtifactsResult{
		Artifacts: []DownloadArtifact{},
		Page:      page,
		PageSize:  pageSize,
		Total:     total,
	}

	for rows.Next() {
		var artifact DownloadArtifact
		var provider, bucket, objectKey, etag, shaOut, ctypeOut, fnameOut, platformOut, versionOut sql.NullString
		var sizeOut sql.NullInt64
		var metadataBytes []byte
		if err := rows.Scan(
			&artifact.ID,
			&artifact.BundleKey,
			&provider,
			&bucket,
			&objectKey,
			&etag,
			&sizeOut,
			&shaOut,
			&ctypeOut,
			&fnameOut,
			&platformOut,
			&versionOut,
			&metadataBytes,
			&artifact.CreatedAt,
			&artifact.UpdatedAt,
		); err != nil {
			return nil, err
		}
		artifact.Provider = provider.String
		artifact.Bucket = bucket.String
		artifact.ObjectKey = objectKey.String
		artifact.ETag = etag.String
		if sizeOut.Valid {
			artifact.SizeBytes = sizeOut.Int64
		}
		artifact.SHA256 = shaOut.String
		artifact.ContentType = ctypeOut.String
		artifact.OriginalFilename = fnameOut.String
		artifact.Platform = platformOut.String
		artifact.ReleaseVersion = versionOut.String
		if len(metadataBytes) > 0 {
			var meta map[string]interface{}
			if err := json.Unmarshal(metadataBytes, &meta); err == nil {
				artifact.Metadata = meta
			}
		}
		artifact.StableObjectURI = stableS3URI(artifact.Bucket, artifact.ObjectKey)

		result.Artifacts = append(result.Artifacts, artifact)
	}

	return result, nil
}

func (s *DownloadHostingService) PresignGetArtifact(ctx context.Context, bundleKey string, artifact DownloadArtifact) (string, error) {
	settings, err := s.requireConfiguredSettings(ctx, bundleKey)
	if err != nil {
		return "", err
	}
	storage, err := s.resolveStorage(ctx, *settings)
	if err != nil {
		return "", err
	}

	ttl := time.Duration(settings.SignedURLTTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	return storage.PresignGet(ctx, artifact.Bucket, artifact.ObjectKey, ttl)
}
