package delivery

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Store is the narrow SQL boundary used by delivery persistence.
//
// seam: Store
type Store interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// Service owns storage configuration, artifact metadata, and provider-backed
// upload and download operations for the delivery domain.
type Service struct {
	db        Store
	providers map[string]StorageProvider
}

func NewService(db Store, providers ...StorageProvider) *Service {
	registered := map[string]StorageProvider{}
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
		registered["s3"] = S3StorageProvider{}
	}

	return &Service{
		db:        db,
		providers: registered,
	}
}

var ErrStorageNotConfigured = errors.New("download storage not configured")

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

func (s *Service) HasProvider(providerKey string) bool {
	_, ok := s.providers[strings.TrimSpace(providerKey)]
	return ok
}

func (s *Service) GetSettings(ctx context.Context, bundleKey string) (*StorageSettings, error) {
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
	var settings StorageSettings
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

func (s *Service) SettingsSnapshot(ctx context.Context, bundleKey string) (*StorageSettingsSnapshot, error) {
	settings, err := s.GetSettings(ctx, bundleKey)
	if err != nil {
		return nil, err
	}

	if settings == nil {
		return &StorageSettingsSnapshot{
			Provider:             "s3",
			SignedURLTTLSeconds:  900,
			CredentialsFromEnv:   true,
			SettingsRowAvailable: false,
		}, nil
	}

	credentialsFromEnv := strings.TrimSpace(settings.AccessKeyID) == "" && strings.TrimSpace(settings.SecretAccessKey) == ""

	return &StorageSettingsSnapshot{
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

func (s *Service) ValidateStorageSettings(settings StorageSettings) error {
	providerKey := strings.TrimSpace(settings.Provider)
	if providerKey == "" {
		providerKey = "s3"
	}
	if _, ok := s.providers[providerKey]; !ok {
		return fmt.Errorf("unsupported provider %q", providerKey)
	}

	return ValidateSettings(settings)
}

func (s *Service) SaveSettings(ctx context.Context, bundleKey string, update StorageSettingsUpdate) (*StorageSettingsSnapshot, error) {
	bundleKey = strings.TrimSpace(bundleKey)
	if bundleKey == "" {
		return nil, fmt.Errorf("bundle_key is required")
	}

	existing, err := s.GetSettings(ctx, bundleKey)
	if err != nil {
		return nil, err
	}

	settings := StorageSettings{
		BundleKey:           bundleKey,
		Provider:            "s3",
		SignedURLTTLSeconds: 900,
	}
	if existing != nil {
		settings = *existing
	}

	settings = ApplySettingsUpdate(settings, update)

	if err := s.ValidateStorageSettings(settings); err != nil {
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

func (s *Service) resolveStorage(ctx context.Context, settings StorageSettings) (Storage, error) {
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

func (s *Service) requireConfiguredSettings(ctx context.Context, bundleKey string) (*StorageSettings, error) {
	settings, err := s.GetSettings(ctx, bundleKey)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return nil, ErrStorageNotConfigured
	}
	if strings.TrimSpace(settings.Bucket) == "" {
		return nil, ErrStorageNotConfigured
	}
	if settings.SignedURLTTLSeconds <= 0 {
		settings.SignedURLTTLSeconds = 900
	}
	return settings, nil
}

func (s *Service) TestConnection(ctx context.Context, bundleKey string) error {
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

func (s *Service) PresignUpload(ctx context.Context, bundleKey string, req PresignUploadRequest) (*PresignUploadResponse, error) {
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

	objectKey, err := BuildObjectKey(*settings, bundleKey, req)
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
		StableObjectURI: StableS3URI(settings.Bucket, objectKey),
	}, nil
}

func (s *Service) CommitArtifact(ctx context.Context, bundleKey string, req CommitArtifactRequest) (*Artifact, error) {
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

	appKey := strings.TrimSpace(req.AppKey)

	query := `
		INSERT INTO download_artifacts (
			bundle_key, app_key, provider, bucket, object_key, etag, size_bytes, sha256, sha512,
			release_id, git_commit_hash, content_type, original_filename, platform, release_version, metadata, updated_at
		) VALUES ($1,$2,'s3',$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15, NOW())
		ON CONFLICT (bundle_key, bucket, object_key) DO UPDATE SET
			app_key = COALESCE(EXCLUDED.app_key, download_artifacts.app_key),
			etag = EXCLUDED.etag,
			size_bytes = EXCLUDED.size_bytes,
			sha256 = EXCLUDED.sha256,
			sha512 = EXCLUDED.sha512,
			release_id = COALESCE(EXCLUDED.release_id, download_artifacts.release_id),
			git_commit_hash = EXCLUDED.git_commit_hash,
			content_type = EXCLUDED.content_type,
			original_filename = EXCLUDED.original_filename,
			platform = EXCLUDED.platform,
			release_version = EXCLUDED.release_version,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, bundle_key, app_key, provider, bucket, object_key, etag, size_bytes, sha256, sha512,
		          release_id, git_commit_hash, content_type, original_filename, platform, release_version, metadata, created_at, updated_at
	`

	row := s.db.QueryRowContext(ctx, query,
		bundleKey,
		normalizeOptionalString(&appKey),
		bucket,
		strings.TrimSpace(req.ObjectKey),
		normalizeOptionalString(&etag),
		size,
		normalizeOptionalString(&req.SHA256),
		normalizeOptionalString(&req.SHA512),
		normalizeOptionalString(&req.ReleaseID),
		normalizeOptionalString(&req.GitCommitHash),
		normalizeOptionalString(&contentType),
		normalizeOptionalString(&req.OriginalFilename),
		normalizeOptionalString(&req.Platform),
		normalizeOptionalString(&req.ReleaseVersion),
		metadataBytes,
	)

	var t ArtifactScanTargets
	if err := row.Scan(t.ScanDest()...); err != nil {
		return nil, err
	}
	artifact := t.Hydrate()

	return &artifact, nil
}

func (s *Service) GetArtifact(ctx context.Context, bundleKey string, id int64) (*Artifact, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, bundle_key, app_key, provider, bucket, object_key, etag, size_bytes, sha256, sha512,
		       release_id, git_commit_hash, content_type, original_filename, platform, release_version, metadata, created_at, updated_at
		FROM download_artifacts
		WHERE bundle_key = $1 AND id = $2
		LIMIT 1
	`, bundleKey, id)

	var t ArtifactScanTargets
	if err := row.Scan(t.ScanDest()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	artifact := t.Hydrate()

	return &artifact, nil
}

func (s *Service) ListArtifacts(ctx context.Context, bundleKey string, query, platform, appKey string, page, pageSize int) (*ListArtifactsResult, error) {
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
	appKey = strings.TrimSpace(appKey)

	where := []string{"bundle_key = $1"}
	args := []interface{}{bundleKey}

	if platform != "" {
		args = append(args, platform)
		where = append(where, fmt.Sprintf("platform = $%d", len(args)))
	}

	if appKey != "" {
		args = append(args, appKey)
		where = append(where, fmt.Sprintf("app_key = $%d", len(args)))
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
		SELECT id, bundle_key, app_key, provider, bucket, object_key, etag, size_bytes, sha256, sha512,
		       release_id, git_commit_hash, content_type, original_filename, platform, release_version, metadata, created_at, updated_at
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
		Artifacts: []Artifact{},
		Page:      page,
		PageSize:  pageSize,
		Total:     total,
	}

	for rows.Next() {
		var t ArtifactScanTargets
		if err := rows.Scan(t.ScanDest()...); err != nil {
			return nil, err
		}
		result.Artifacts = append(result.Artifacts, t.Hydrate())
	}

	return result, nil
}

// ListArtifactsByApp returns artifacts for a specific app/platform with is_current flags
// indicating which artifact is currently active (linked in download_assets).
func (s *Service) ListArtifactsByApp(ctx context.Context, bundleKey, appKey, platform string, page, pageSize int) (*ListArtifactsResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	appKey = strings.TrimSpace(appKey)
	platform = strings.TrimSpace(platform)

	if appKey == "" {
		return nil, fmt.Errorf("app_key is required")
	}

	where := []string{"a.bundle_key = $1", "a.app_key = $2"}
	args := []interface{}{bundleKey, appKey}

	if platform != "" {
		args = append(args, platform)
		where = append(where, fmt.Sprintf("a.platform = $%d", len(args)))
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM download_artifacts a WHERE %s`, whereClause), args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)

	// Join with download_assets to determine which artifact is current for each platform
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT a.id, a.bundle_key, a.app_key, a.provider, a.bucket, a.object_key, a.etag, a.size_bytes, a.sha256, a.sha512,
		       a.release_id, a.git_commit_hash, a.content_type, a.original_filename, a.platform, a.release_version, a.metadata, a.created_at, a.updated_at,
		       CASE WHEN da.artifact_id = a.id THEN true ELSE false END AS is_current
		FROM download_artifacts a
		LEFT JOIN download_assets da ON da.bundle_key = a.bundle_key AND da.app_key = a.app_key AND da.platform = a.platform
		WHERE %s
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &ListArtifactsResult{
		Artifacts: []Artifact{},
		Page:      page,
		PageSize:  pageSize,
		Total:     total,
	}

	for rows.Next() {
		var t ArtifactScanTargets
		var isCurrent sql.NullBool
		dest := append(t.ScanDest(), &isCurrent)
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		a := t.Hydrate()
		a.IsCurrent = isCurrent.Valid && isCurrent.Bool
		result.Artifacts = append(result.Artifacts, a)
	}

	return result, nil
}

// GetCurrentArtifactByFilename returns the current artifact for an app/variant matching a filename.
func (s *Service) GetCurrentArtifactByFilename(ctx context.Context, bundleKey, appKey, variantKey, filename string) (*Artifact, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT da.id, da.bundle_key, da.app_key, da.provider, da.bucket, da.object_key,
		       da.etag, da.size_bytes, da.sha256, da.sha512, da.release_id, da.git_commit_hash, da.content_type,
		       da.original_filename, da.platform, da.release_version, da.metadata,
		       da.created_at, da.updated_at
		FROM download_artifacts da
		JOIN download_assets das ON das.artifact_id = da.id
		WHERE das.bundle_key = $1 AND das.app_key = $2 AND das.variant_key = $3
		  AND da.original_filename = $4
		LIMIT 1
	`, bundleKey, appKey, variantKey, filename)

	var t ArtifactScanTargets
	if err := row.Scan(t.ScanDest()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	artifact := t.Hydrate()

	return &artifact, nil
}

func (s *Service) PresignGetArtifact(ctx context.Context, bundleKey string, artifact Artifact) (string, error) {
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

// HeadArtifact checks if an artifact's S3 object is accessible.
func (s *Service) HeadArtifact(ctx context.Context, bundleKey string, artifact Artifact) error {
	settings, err := s.requireConfiguredSettings(ctx, bundleKey)
	if err != nil {
		return err
	}
	storage, err := s.resolveStorage(ctx, *settings)
	if err != nil {
		return err
	}
	_, _, _, err = storage.HeadObject(ctx, artifact.Bucket, artifact.ObjectKey)
	return err
}
