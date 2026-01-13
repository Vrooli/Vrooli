package distribution

import (
	"context"
	"time"

	"scenario-to-desktop-api/distribution/providers"
)

// DefaultUploaderFactory creates uploaders for S3-compatible storage.
type DefaultUploaderFactory struct{}

// Create creates an Uploader for the given target config.
// inlineCredentials is an optional map of env var names to values that override env lookups.
func (f *DefaultUploaderFactory) Create(target *DistributionTarget, envReader EnvironmentReader, inlineCredentials map[string]string) (Uploader, error) {
	// Convert to providers types
	s3Config := targetToS3Config(target)

	// Wrap env reader to check inline credentials first
	providerEnvReader := &envReaderWithInline{
		envReader:         envReader,
		inlineCredentials: inlineCredentials,
	}

	// Create the S3 uploader
	s3Uploader, err := providers.NewS3Uploader(context.Background(), s3Config, providerEnvReader)
	if err != nil {
		return nil, err
	}

	return &uploaderAdapter{uploader: s3Uploader, target: target}, nil
}

// envReaderWithInline wraps an EnvironmentReader to check inline credentials first.
type envReaderWithInline struct {
	envReader         EnvironmentReader
	inlineCredentials map[string]string
}

func (e *envReaderWithInline) GetEnv(key string) string {
	// Check inline credentials first
	if e.inlineCredentials != nil {
		if val, ok := e.inlineCredentials[key]; ok {
			return val
		}
	}
	// Fall back to environment
	return e.envReader.GetEnv(key)
}

// targetToS3Config converts DistributionTarget to providers.S3TargetConfig.
func targetToS3Config(target *DistributionTarget) *providers.S3TargetConfig {
	var retryConfig *providers.RetryConfig
	if target.Retry != nil {
		retryConfig = &providers.RetryConfig{
			MaxAttempts:       target.Retry.MaxAttempts,
			InitialBackoffMs:  target.Retry.InitialBackoffMs,
			MaxBackoffMs:      target.Retry.MaxBackoffMs,
			BackoffMultiplier: target.Retry.BackoffMultiplier,
		}
	}

	return &providers.S3TargetConfig{
		Name:               target.Name,
		Provider:           target.Provider,
		Bucket:             target.Bucket,
		Endpoint:           target.Endpoint,
		Region:             target.Region,
		PathPrefix:         target.PathPrefix,
		AccessKeyIDEnv:     target.AccessKeyIDEnv,
		SecretAccessKeyEnv: target.SecretAccessKeyEnv,
		ACL:                target.ACL,
		CDNUrl:             target.CDNUrl,
		Retry:              retryConfig,
	}
}

// targetToProviderTarget converts DistributionTarget to providers.DistributionTarget.
func targetToProviderTarget(target *DistributionTarget) *providers.DistributionTarget {
	return &providers.DistributionTarget{
		Name:               target.Name,
		Provider:           target.Provider,
		Bucket:             target.Bucket,
		Endpoint:           target.Endpoint,
		Region:             target.Region,
		AccessKeyIDEnv:     target.AccessKeyIDEnv,
		SecretAccessKeyEnv: target.SecretAccessKeyEnv,
	}
}

// uploaderAdapter adapts providers.S3Uploader to the Uploader interface.
type uploaderAdapter struct {
	uploader *providers.S3Uploader
	target   *DistributionTarget
}

func (a *uploaderAdapter) Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error) {
	// Convert request
	providerReq := &providers.UploadRequest{
		LocalPath:   req.LocalPath,
		Key:         req.Key,
		ContentType: req.ContentType,
		Metadata:    req.Metadata,
	}
	if req.ProgressCallback != nil {
		providerReq.ProgressCallback = func(bytesUploaded, totalBytes int64) {
			req.ProgressCallback(bytesUploaded, totalBytes)
		}
	}

	// Call provider
	providerResult, err := a.uploader.Upload(ctx, providerReq)
	if err != nil {
		return nil, err
	}

	// Convert result
	return &UploadResult{
		Key:      providerResult.Key,
		URL:      providerResult.URL,
		ETag:     providerResult.ETag,
		Size:     providerResult.Size,
		Duration: providerResult.Duration,
	}, nil
}

func (a *uploaderAdapter) ValidateCredentials(ctx context.Context) error {
	return a.uploader.ValidateCredentials(ctx)
}

func (a *uploaderAdapter) ListObjects(ctx context.Context, prefix string, maxKeys int) ([]ObjectInfo, error) {
	providerObjects, err := a.uploader.ListObjects(ctx, prefix, maxKeys)
	if err != nil {
		return nil, err
	}

	objects := make([]ObjectInfo, len(providerObjects))
	for i, obj := range providerObjects {
		objects[i] = ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
			ETag:         obj.ETag,
		}
	}
	return objects, nil
}

func (a *uploaderAdapter) DeleteObject(ctx context.Context, key string) error {
	return a.uploader.DeleteObject(ctx, key)
}

func (a *uploaderAdapter) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return a.uploader.GetPresignedURL(ctx, key, expiry)
}

func (a *uploaderAdapter) GetPublicURL(key string) string {
	return a.uploader.GetPublicURL(key)
}
