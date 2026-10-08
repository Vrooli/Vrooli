package delivery

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type s3Storage struct {
	client    *s3.Client
	presigner *s3.PresignClient
	region    string
}

// S3StorageProvider is the AWS S3-compatible storage implementation for the
// delivery domain, including custom endpoints such as MinIO and R2.
type CredentialResolver func(context.Context, string) (string, error)

type S3StorageProvider struct {
	ResolveCredential CredentialResolver
}

func (S3StorageProvider) ProviderKey() string { return "s3" }

func (p S3StorageProvider) New(ctx context.Context, settings StorageSettings) (Storage, error) {
	return newS3Storage(ctx, settings, p.ResolveCredential)
}

//nolint:staticcheck // legacy resolver remains required by custom S3-compatible endpoints.
func endpointResolverForS3(endpointURL string) aws.EndpointResolverWithOptionsFunc {
	return func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID && strings.TrimSpace(endpointURL) != "" {
			return aws.Endpoint{URL: endpointURL, HostnameImmutable: true, SigningRegion: region}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	}
}

func newS3Storage(ctx context.Context, settings StorageSettings, resolve CredentialResolver) (*s3Storage, error) {
	region := strings.TrimSpace(settings.Region)
	if region == "" {
		region = "us-east-1"
	}
	loadOptions := []func(*config.LoadOptions) error{config.WithRegion(region)}
	if settings.Endpoint != "" {
		//nolint:staticcheck // legacy resolver remains required by custom S3-compatible endpoints.
		loadOptions = append(loadOptions, config.WithEndpointResolverWithOptions(endpointResolverForS3(settings.Endpoint)))
	}
	if resolve != nil {
		accessKey, accessErr := resolveOptionalS3Credential(ctx, resolve, "delivery-s3-access-key-id")
		secretKey, secretErr := resolveOptionalS3Credential(ctx, resolve, "delivery-s3-secret-access-key")
		sessionToken, sessionErr := resolveOptionalS3Credential(ctx, resolve, "delivery-s3-session-token")
		if accessErr != nil || secretErr != nil || sessionErr != nil {
			return nil, fmt.Errorf("resolve download bucket credentials: %w", firstCredentialError(accessErr, secretErr, sessionErr))
		}
		if strings.TrimSpace(accessKey) != "" || strings.TrimSpace(secretKey) != "" {
			loadOptions = append(loadOptions, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken)))
		}
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) { options.UsePathStyle = settings.ForcePathStyle })
	return &s3Storage{client: client, presigner: s3.NewPresignClient(client), region: region}, nil
}

func resolveOptionalS3Credential(ctx context.Context, resolve CredentialResolver, field string) (string, error) {
	value, err := resolve(ctx, field)
	if errors.Is(err, credentialauthority.ErrUnconfigured) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func firstCredentialError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *s3Storage) TestConnection(ctx context.Context, bucket string) error {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return newDiagnosticError(CodeBucketNameMissing, "bucket is required", "HeadBucket", "", s.region, "Set the delivery bucket name in storage settings.", false)
	}
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	if err != nil {
		return ClassifyS3Error(err, "HeadBucket", bucket, s.region)
	}
	return nil
}

// healthcheckObjectPrefix is the internal, bucket-level prefix LPBS owns for
// bounded readiness canaries. It is deliberately separate from the artifact
// prefix so validation never collides with a release object.
const healthcheckObjectPrefix = ".vrooli/healthchecks"

// VerifyOperations proves the bounded distribution contract with operations
// that are unique to this readiness attempt and are always cleaned up.
// HeadBucket alone is insufficient: a role can discover a bucket while lacking
// list, object write, read, or cleanup permissions. Each failure is classified
// so an operator can identify the exact missing permission.
func (s *s3Storage) VerifyOperations(ctx context.Context, bucket, _ string) error {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return newDiagnosticError(CodeBucketNameMissing, "bucket is required", "HeadBucket", "", s.region, "Set the delivery bucket name in storage settings.", false)
	}

	head, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	if err != nil {
		return ClassifyS3Error(err, "HeadBucket", bucket, s.region)
	}
	if head != nil && head.BucketRegion != nil {
		actual := strings.TrimSpace(*head.BucketRegion)
		if actual != "" && strings.TrimSpace(s.region) != "" && !strings.EqualFold(actual, s.region) {
			return newDiagnosticError(
				CodeBucketWrongRegion,
				fmt.Sprintf("bucket %q is in region %q but the configured region is %q", bucket, actual, s.region),
				"HeadBucket", bucket, s.region,
				fmt.Sprintf("Set the storage region to %q and retry.", actual), false,
			)
		}
	}

	if _, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(healthcheckObjectPrefix + "/"),
		MaxKeys: aws.Int32(1),
	}); err != nil {
		diagnostic := ClassifyS3Error(err, "ListBucket", bucket, s.region)
		diagnostic.Summary = "list readiness prefix: " + diagnostic.Summary
		return diagnostic
	}

	key, err := healthcheckObjectKey()
	if err != nil {
		return newDiagnosticError(CodeOperationFailed, fmt.Sprintf("generate readiness object identity: %v", err), "PutObject", bucket, s.region, "Retry; if the failure persists, inspect host randomness.", true)
	}
	payload := []byte("vrooli-s3-readiness-v1")
	if _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(payload),
		ContentType: aws.String("application/octet-stream"),
	}); err != nil {
		diagnostic := ClassifyS3Error(err, "PutObject", bucket, s.region)
		diagnostic.Summary = "write readiness object: " + diagnostic.Summary
		return diagnostic
	}

	response, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		diagnostic := ClassifyS3Error(err, "GetObject", bucket, s.region)
		diagnostic.Summary = "read readiness object: " + diagnostic.Summary
		if diagnostic.Code == CodeOperationFailed {
			diagnostic.Code = CodeWriteSucceededReadFailed
		}
		return s.cleanupAfterFailure(ctx, bucket, key, diagnostic)
	}
	read, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil {
		diagnostic := newDiagnosticError(CodeWriteSucceededReadFailed, fmt.Sprintf("read readiness object body: %v", readErr), "GetObject", bucket, s.region, "Retry the storage test; the canary object is recreated each run.", true)
		return s.cleanupAfterFailure(ctx, bucket, key, diagnostic)
	}
	if closeErr != nil {
		diagnostic := newDiagnosticError(CodeWriteSucceededReadFailed, fmt.Sprintf("close readiness object body: %v", closeErr), "GetObject", bucket, s.region, "Retry the storage test; the canary object is recreated each run.", true)
		return s.cleanupAfterFailure(ctx, bucket, key, diagnostic)
	}
	if !bytes.Equal(read, payload) {
		diagnostic := newDiagnosticError(CodeWriteSucceededReadFailed, "readiness object content did not round-trip", "GetObject", bucket, s.region, "The bucket may be transforming objects. Retry, then inspect bucket settings.", true)
		return s.cleanupAfterFailure(ctx, bucket, key, diagnostic)
	}

	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)}); err != nil {
		diagnostic := ClassifyS3Error(err, "DeleteObject", bucket, s.region)
		if diagnostic.Code != CodeDeleteObjectDenied {
			diagnostic.Code = CodeCleanupFailed
		}
		diagnostic.Summary = "write and read succeeded but cleanup failed: delete readiness object: " + diagnostic.Summary
		diagnostic.WithCanary(key)
		return diagnostic
	}
	return nil
}

// cleanupAfterFailure best-effort deletes a canary whose read failed. A cleanup
// failure retains the non-sensitive key so an operator can remove it later.
func (s *s3Storage) cleanupAfterFailure(ctx context.Context, bucket, key string, diagnostic *DiagnosticError) *DiagnosticError {
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)}); err != nil {
		diagnostic.WithCanary(key)
	}
	return diagnostic
}

// healthcheckObjectKey builds a collision-resistant RFC 4122 v4 identifier
// under the internal readiness prefix. Concurrent validations never share a key
// and never overwrite a fixed health-check object.
func healthcheckObjectKey() (string, error) {
	var token [16]byte
	if _, err := cryptorand.Read(token[:]); err != nil {
		return "", fmt.Errorf("generate readiness object identity: %w", err)
	}
	token[6] = (token[6] & 0x0f) | 0x40
	token[8] = (token[8] & 0x3f) | 0x80
	id := fmt.Sprintf("%x-%x-%x-%x-%x", token[0:4], token[4:6], token[6:8], token[8:10], token[10:16])
	return healthcheckObjectPrefix + "/" + id, nil
}

func (s *s3Storage) PresignGet(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	req, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)}, func(options *s3.PresignOptions) { options.Expires = ttl })
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (s *s3Storage) PresignPut(ctx context.Context, bucket, key string, ttl time.Duration, contentType string) (string, map[string]string, error) {
	input := &s3.PutObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)}
	if strings.TrimSpace(contentType) != "" {
		input.ContentType = aws.String(contentType)
	}
	req, err := s.presigner.PresignPutObject(ctx, input, func(options *s3.PresignOptions) { options.Expires = ttl })
	if err != nil {
		return "", nil, err
	}
	headers := make(map[string]string, len(req.SignedHeader))
	for key, values := range req.SignedHeader {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return req.URL, headers, nil
}

func (s *s3Storage) HeadObject(ctx context.Context, bucket, key string) (etag string, size int64, contentType string, err error) {
	response, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return "", 0, "", err
	}
	if response.ETag != nil {
		etag = strings.Trim(*response.ETag, "\"")
	}
	if response.ContentLength != nil {
		size = *response.ContentLength
	}
	if response.ContentType != nil {
		contentType = *response.ContentType
	}
	return etag, size, contentType, nil
}

func (s *s3Storage) ReadObject(ctx context.Context, bucket, key string) (io.ReadCloser, int64, string, error) {
	response, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return nil, 0, "", err
	}
	var size int64
	if response.ContentLength != nil {
		size = *response.ContentLength
	}
	var contentType string
	if response.ContentType != nil {
		contentType = *response.ContentType
	}
	return response.Body, size, contentType, nil
}
