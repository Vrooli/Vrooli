package delivery

import (
	"context"
	"errors"
	"fmt"
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
	return &s3Storage{client: client, presigner: s3.NewPresignClient(client)}, nil
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
		return fmt.Errorf("bucket is required")
	}
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	return err
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
