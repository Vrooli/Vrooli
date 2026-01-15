package providers

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// EnvironmentReader abstracts environment variable access.
type EnvironmentReader interface {
	GetEnv(key string) string
}

// S3TargetConfig contains the configuration needed for S3 uploads.
type S3TargetConfig struct {
	Name               string
	Provider           string
	Bucket             string
	Endpoint           string
	Region             string
	PathPrefix         string
	AccessKeyIDEnv     string
	SecretAccessKeyEnv string
	ACL                string
	CDNUrl             string
	Retry              *RetryConfig
}

// UploadRequest contains parameters for an upload operation.
type UploadRequest struct {
	LocalPath        string
	Key              string
	ContentType      string
	Metadata         map[string]string
	ProgressCallback func(bytesUploaded, totalBytes int64)
}

// UploadResult contains the result of an upload operation.
type UploadResult struct {
	Key      string
	URL      string
	ETag     string
	Size     int64
	Duration time.Duration
}

// ObjectInfo contains metadata about a stored object.
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
}

// ProgressReader wraps an io.Reader to track read progress.
type ProgressReader struct {
	reader      io.Reader
	total       int64
	read        int64
	callback    func(bytesRead, total int64)
	lastReport  int64
	reportEvery int64
}

// NewProgressReader creates a progress-tracking reader.
func NewProgressReader(r io.Reader, total int64, callback func(bytesRead, total int64)) *ProgressReader {
	return &ProgressReader{
		reader:      r,
		total:       total,
		callback:    callback,
		reportEvery: 1024 * 1024, // Report every 1MB
	}
}

// Read implements io.Reader.
func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.read += int64(n)

	if pr.callback != nil && (pr.read-pr.lastReport) >= pr.reportEvery {
		pr.callback(pr.read, pr.total)
		pr.lastReport = pr.read
	}

	return n, err
}

// S3Uploader implements Uploader for S3-compatible storage.
type S3Uploader struct {
	client      *s3.Client
	presigner   *s3.PresignClient
	target      *S3TargetConfig
	retryConfig *RetryConfig
}

// S3UploaderOption configures an S3Uploader.
type S3UploaderOption func(*S3Uploader)

// WithS3RetryConfig sets custom retry configuration.
func WithS3RetryConfig(cfg *RetryConfig) S3UploaderOption {
	return func(u *S3Uploader) {
		u.retryConfig = cfg
	}
}

// NewS3Uploader creates a new S3 uploader for the given target.
func NewS3Uploader(ctx context.Context, target *S3TargetConfig, envReader EnvironmentReader, opts ...S3UploaderOption) (*S3Uploader, error) {
	// Get credentials from environment
	accessKeyID := envReader.GetEnv(target.AccessKeyIDEnv)
	secretAccessKey := envReader.GetEnv(target.SecretAccessKeyEnv)

	if accessKeyID == "" {
		return nil, fmt.Errorf("environment variable %s not set", target.AccessKeyIDEnv)
	}
	if secretAccessKey == "" {
		return nil, fmt.Errorf("environment variable %s not set", target.SecretAccessKeyEnv)
	}

	// Build AWS config options
	awsOpts := []func(*config.LoadOptions) error{
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"", // session token (not used)
		)),
	}

	// Set region (required for S3, optional for others)
	region := target.Region
	if region == "" {
		region = "auto" // R2 and some S3-compatible services use "auto"
	}
	awsOpts = append(awsOpts, config.WithRegion(region))

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx, awsOpts...)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	// Build S3 client options
	s3Opts := []func(*s3.Options){}

	// Set custom endpoint for R2 and S3-compatible services
	if target.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(target.Endpoint)
			o.UsePathStyle = true // Required for most S3-compatible services
		})
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg, s3Opts...)
	presigner := s3.NewPresignClient(client)

	uploader := &S3Uploader{
		client:      client,
		presigner:   presigner,
		target:      target,
		retryConfig: target.Retry,
	}

	// Apply options
	for _, opt := range opts {
		opt(uploader)
	}

	// Use default retry config if not set
	if uploader.retryConfig == nil {
		uploader.retryConfig = DefaultRetryConfig
	}

	return uploader, nil
}

// Upload uploads a file to S3.
func (u *S3Uploader) Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error) {
	startTime := time.Now()

	// Open the file
	file, err := os.Open(req.LocalPath)
	if err != nil {
		return nil, &UploadError{
			Operation: "upload",
			Target:    u.target.Name,
			Key:       req.Key,
			Retryable: false,
			Cause:     fmt.Errorf("open file: %w", err),
		}
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return nil, &UploadError{
			Operation: "upload",
			Target:    u.target.Name,
			Key:       req.Key,
			Retryable: false,
			Cause:     fmt.Errorf("stat file: %w", err),
		}
	}
	fileSize := stat.Size()

	// Determine content type
	contentType := req.ContentType
	if contentType == "" {
		contentType = ContentTypeFromFilename(filepath.Base(req.LocalPath))
	}

	// Build the object key with prefix
	objectKey := req.Key
	if u.target.PathPrefix != "" {
		objectKey = u.target.PathPrefix + "/" + req.Key
	}

	// Wrap reader with progress tracking
	var reader io.Reader = file
	if req.ProgressCallback != nil {
		reader = NewProgressReader(file, fileSize, req.ProgressCallback)
	}

	// Build put object input
	input := &s3.PutObjectInput{
		Bucket:      aws.String(u.target.Bucket),
		Key:         aws.String(objectKey),
		Body:        reader,
		ContentType: aws.String(contentType),
	}

	// Set ACL if configured
	if u.target.ACL != "" {
		input.ACL = s3types.ObjectCannedACL(u.target.ACL)
	}

	// Set metadata if provided
	if len(req.Metadata) > 0 {
		input.Metadata = req.Metadata
	}

	// Upload with retry
	var result *s3.PutObjectOutput
	err = RetryWithBackoff(ctx, u.retryConfig, func(attempt int) error {
		var uploadErr error
		result, uploadErr = u.client.PutObject(ctx, input)
		if uploadErr != nil {
			// Reset file position for retry
			file.Seek(0, io.SeekStart)
			return &UploadError{
				Operation: "upload",
				Target:    u.target.Name,
				Key:       objectKey,
				Attempt:   attempt,
				Retryable: true,
				Cause:     uploadErr,
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Build result
	uploadResult := &UploadResult{
		Key:      objectKey,
		Size:     fileSize,
		Duration: time.Since(startTime),
	}

	if result.ETag != nil {
		uploadResult.ETag = *result.ETag
	}

	// Set public URL if CDN configured
	uploadResult.URL = u.GetPublicURL(objectKey)

	return uploadResult, nil
}

// ValidateCredentials verifies the credentials work by listing the bucket.
func (u *S3Uploader) ValidateCredentials(ctx context.Context) error {
	_, err := u.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(u.target.Bucket),
	})
	if err != nil {
		return &UploadError{
			Operation: "validate",
			Target:    u.target.Name,
			Retryable: false,
			Cause:     fmt.Errorf("bucket access check failed: %w", err),
		}
	}
	return nil
}

// ListObjects lists objects at a given prefix.
func (u *S3Uploader) ListObjects(ctx context.Context, prefix string, maxKeys int) ([]ObjectInfo, error) {
	// Add target path prefix
	fullPrefix := prefix
	if u.target.PathPrefix != "" {
		fullPrefix = u.target.PathPrefix + "/" + prefix
	}

	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(u.target.Bucket),
		Prefix:  aws.String(fullPrefix),
		MaxKeys: aws.Int32(int32(maxKeys)),
	}

	result, err := u.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, &UploadError{
			Operation: "list",
			Target:    u.target.Name,
			Key:       prefix,
			Retryable: true,
			Cause:     err,
		}
	}

	objects := make([]ObjectInfo, 0, len(result.Contents))
	for _, obj := range result.Contents {
		objects = append(objects, ObjectInfo{
			Key:          *obj.Key,
			Size:         *obj.Size,
			LastModified: *obj.LastModified,
			ETag:         *obj.ETag,
		})
	}

	return objects, nil
}

// DeleteObject removes an object.
func (u *S3Uploader) DeleteObject(ctx context.Context, key string) error {
	// Add target path prefix
	objectKey := key
	if u.target.PathPrefix != "" {
		objectKey = u.target.PathPrefix + "/" + key
	}

	_, err := u.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(u.target.Bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return &UploadError{
			Operation: "delete",
			Target:    u.target.Name,
			Key:       objectKey,
			Retryable: true,
			Cause:     err,
		}
	}
	return nil
}

// GetPresignedURL generates a temporary download URL.
func (u *S3Uploader) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	// Add target path prefix
	objectKey := key
	if u.target.PathPrefix != "" {
		objectKey = u.target.PathPrefix + "/" + key
	}

	presignResult, err := u.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(u.target.Bucket),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return "", &UploadError{
			Operation: "presign",
			Target:    u.target.Name,
			Key:       objectKey,
			Retryable: false,
			Cause:     err,
		}
	}

	return presignResult.URL, nil
}

// GetPublicURL returns the public URL for an object.
func (u *S3Uploader) GetPublicURL(key string) string {
	if u.target.CDNUrl != "" {
		// Use CDN URL
		return u.target.CDNUrl + "/" + key
	}

	// Build S3 URL
	if u.target.Endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", u.target.Endpoint, u.target.Bucket, key)
	}

	// AWS S3 URL
	region := u.target.Region
	if region == "" {
		region = "us-east-1"
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", u.target.Bucket, region, key)
}
