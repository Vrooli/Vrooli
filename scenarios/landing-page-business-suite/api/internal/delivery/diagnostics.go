package delivery

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// DiagnosticCode is a stable, machine-readable identifier for a delivery
// storage failure. It exists so operators and automated callers can branch on
// the exact cause instead of matching a human sentence. The value is part of
// the admin API contract and must not change without a version bump.
type DiagnosticCode string

const (
	CodeReady                    DiagnosticCode = "ready"
	CodeBucketNameMissing        DiagnosticCode = "bucket_name_missing"
	CodeRegionMissing            DiagnosticCode = "region_missing"
	CodeAccessKeyIDMissing       DiagnosticCode = "access_key_id_missing"
	CodeSecretAccessKeyMissing   DiagnosticCode = "secret_access_key_missing"
	CodeSessionTokenMissing      DiagnosticCode = "session_token_missing"
	CodeCredentialConfigInvalid  DiagnosticCode = "credential_config_invalid"
	CodeAuthenticationRejected   DiagnosticCode = "authentication_rejected"
	CodeAccessKeyInactive        DiagnosticCode = "access_key_inactive"
	CodeSessionTokenInvalid      DiagnosticCode = "session_token_invalid"
	CodeBucketNotFound           DiagnosticCode = "bucket_not_found"
	CodeBucketWrongRegion        DiagnosticCode = "bucket_wrong_region"
	CodeNetworkFailure           DiagnosticCode = "network_failure"
	CodeRequestTimeout           DiagnosticCode = "request_timeout"
	CodeHeadBucketDenied         DiagnosticCode = "head_bucket_denied"
	CodeListBucketDenied         DiagnosticCode = "list_bucket_denied"
	CodeGetObjectDenied          DiagnosticCode = "get_object_denied"
	CodePutObjectDenied          DiagnosticCode = "put_object_denied"
	CodeDeleteObjectDenied       DiagnosticCode = "delete_object_denied"
	CodeEncryptionPermission     DiagnosticCode = "encryption_permission_denied"
	CodePublicAccessWarning      DiagnosticCode = "public_access_warning"
	CodeWriteSucceededReadFailed DiagnosticCode = "write_succeeded_read_failed"
	CodeCleanupFailed            DiagnosticCode = "cleanup_failed"
	CodeAuthorityUnavailable     DiagnosticCode = "authority_unavailable"
	CodeOperationFailed          DiagnosticCode = "operation_failed"
)

// Diagnostic is the safe, serializable explanation of one failed or partial
// storage operation. It deliberately carries no secret material: no access
// keys, secret keys, session tokens, authorization headers, signatures, or
// presigned URLs. The bucket, region, and canary key are non-sensitive.
type Diagnostic struct {
	Code           DiagnosticCode `json:"code"`
	Summary        string         `json:"summary"`
	Operation      string         `json:"operation,omitempty"`
	Bucket         string         `json:"bucket,omitempty"`
	Region         string         `json:"region,omitempty"`
	Remediation    string         `json:"remediation,omitempty"`
	Retryable      bool           `json:"retryable"`
	CleanupPending bool           `json:"cleanup_pending,omitempty"`
	CanaryKey      string         `json:"canary_key,omitempty"`
	RequestID      string         `json:"request_id,omitempty"`
	HTTPStatus     int            `json:"http_status,omitempty"`
}

// DiagnosticError carries a Diagnostic while satisfying the error contract.
// Handlers extract the structured form with FromDiagnostic.
type DiagnosticError struct {
	Diagnostic
}

func (e *DiagnosticError) Error() string {
	if e == nil {
		return ""
	}
	return e.Summary
}

// newDiagnosticError builds a DiagnosticError with the operation and target
// already stamped so no classification site forgets the addressing context.
func newDiagnosticError(code DiagnosticCode, summary, operation, bucket, region, remediation string, retryable bool) *DiagnosticError {
	return &DiagnosticError{Diagnostic: Diagnostic{
		Code:        code,
		Summary:     summary,
		Operation:   operation,
		Bucket:      strings.TrimSpace(bucket),
		Region:      strings.TrimSpace(region),
		Remediation: remediation,
		Retryable:   retryable,
	}}
}

// WithCanary annotates a cleanup diagnostic with the non-sensitive key that
// still exists in the bucket so an operator can remove it later.
func (e *DiagnosticError) WithCanary(key string) *DiagnosticError {
	if e == nil {
		return nil
	}
	e.CanaryKey = strings.TrimSpace(key)
	e.CleanupPending = true
	return e
}

// FromDiagnostic recovers the structured diagnostic from an error chain.
func FromDiagnostic(err error) (*Diagnostic, bool) {
	var diagnosticErr *DiagnosticError
	if errors.As(err, &diagnosticErr) && diagnosticErr != nil {
		return &diagnosticErr.Diagnostic, true
	}
	return nil, false
}

// awsAPIError and awsHTTPStatus are the minimal fragments of the AWS SDK error
// surface the classifier needs. Declaring them locally avoids a direct
// dependency on the (already transitively present) smithy module.
type awsAPIError interface {
	ErrorCode() string
	ErrorMessage() string
}

type awsHTTPStatus interface {
	HTTPStatusCode() int
}

type awsRequestID interface {
	ServiceRequestID() string
}

// ClassifyS3Error maps a raw AWS/S3/transport error into a structured
// diagnostic. operation is one of HeadBucket, ListBucket, PutObject,
// GetObject, or DeleteObject so an AccessDenied can be attributed to the exact
// missing permission instead of a generic "S3 unavailable".
func ClassifyS3Error(err error, operation, bucket, region string) *DiagnosticError {
	if err == nil {
		return nil
	}
	bucket = strings.TrimSpace(bucket)
	region = strings.TrimSpace(region)

	var timeout interface{ Timeout() bool }
	var dnsErr *net.DNSError
	var urlErr *url.Error
	var httpErr awsHTTPStatus
	var apiErr awsAPIError
	var requestID awsRequestID

	errors.As(err, &httpErr)
	errors.As(err, &apiErr)
	errors.As(err, &requestID)

	status := 0
	if httpErr != nil {
		status = httpErr.HTTPStatusCode()
	}

	code := ""
	message := ""
	if apiErr != nil {
		code = strings.TrimSpace(apiErr.ErrorCode())
		message = strings.TrimSpace(apiErr.ErrorMessage())
	}
	lowerMessage := strings.ToLower(message)
	lowerCode := strings.ToLower(code)

	diagnostic := &DiagnosticError{Diagnostic: Diagnostic{
		Operation:  operation,
		Bucket:     bucket,
		Region:     region,
		HTTPStatus: status,
	}}
	if requestID != nil {
		diagnostic.RequestID = strings.TrimSpace(requestID.ServiceRequestID())
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded) || (timeout != nil && timeout.Timeout()) || status == 408 || status == 504:
		diagnostic.Code = CodeRequestTimeout
		diagnostic.Summary = fmt.Sprintf("%s timed out contacting the delivery bucket", operation)
		diagnostic.Remediation = "Verify the bucket endpoint is reachable from this host and that no proxy or firewall drops the request, then retry."
		diagnostic.Retryable = true
	case errors.As(err, &dnsErr):
		diagnostic.Code = CodeNetworkFailure
		diagnostic.Summary = fmt.Sprintf("%s failed: DNS lookup for the delivery endpoint failed", operation)
		diagnostic.Remediation = "Confirm the bucket region and endpoint are correct and that this host can resolve the S3 endpoint DNS name."
		diagnostic.Retryable = true
	case errors.As(err, &urlErr) || isNetworkError(err):
		diagnostic.Code = CodeNetworkFailure
		diagnostic.Summary = fmt.Sprintf("%s failed: the delivery endpoint could not be reached", operation)
		diagnostic.Remediation = "Confirm outbound HTTPS/DNS access from this host to the bucket region, then retry."
		diagnostic.Retryable = true
	case code == "NoSuchBucket" || (operation == "HeadBucket" && (code == "NotFound" || code == "NoSuchBucket" || status == 404)):
		diagnostic.Code = CodeBucketNotFound
		diagnostic.Summary = fmt.Sprintf("delivery bucket %q does not exist", bucket)
		diagnostic.Remediation = "Confirm the bucket name and region in the storage settings, or create the bucket, then retry."
		diagnostic.Retryable = false
	case code == "PermanentRedirect" || code == "AuthorizationHeaderMalformed" || code == "IncorrectEndpoint" || strings.Contains(lowerMessage, "wrong region") || status == 301:
		diagnostic.Code = CodeBucketWrongRegion
		diagnostic.Summary = fmt.Sprintf("delivery bucket %q is not in the configured region %q", bucket, region)
		diagnostic.Remediation = "Set the storage region to the bucket's actual region and retry."
		diagnostic.Retryable = false
	case strings.Contains(lowerCode, "expiredtoken") || strings.Contains(lowerCode, "invalidtoken") || strings.Contains(lowerMessage, "session token") && strings.Contains(lowerMessage, "expired"):
		diagnostic.Code = CodeSessionTokenInvalid
		diagnostic.Summary = fmt.Sprintf("%s failed: the session token is invalid or expired", operation)
		diagnostic.Remediation = "Provision a fresh set of temporary credentials (access key ID, secret, and session token) and retry."
		diagnostic.Retryable = true
	case strings.Contains(lowerCode, "invalidaccesskeyid") || strings.Contains(lowerMessage, "inactive"):
		diagnostic.Code = CodeAccessKeyInactive
		diagnostic.Summary = fmt.Sprintf("%s failed: the access key ID does not exist or is inactive", operation)
		diagnostic.Remediation = "Provision a current access key ID for the delivery IAM user, or reactivate the key in IAM."
		diagnostic.Retryable = false
	case strings.Contains(lowerCode, "signaturedoesnotmatch") || strings.Contains(lowerCode, "authfailure") || strings.Contains(lowerCode, "invalidsecurity") || code == "InvalidAccessKeyId" || code == "AccessDenied" && operation == "":
		diagnostic.Code = CodeAuthenticationRejected
		diagnostic.Summary = fmt.Sprintf("%s failed: AWS rejected the credential signature", operation)
		diagnostic.Remediation = "Re-provision the access key ID and secret access key together. A mismatched pair or a rotated key is the usual cause."
		diagnostic.Retryable = false
	case isEncryptionError(lowerCode, lowerMessage):
		diagnostic.Code = CodeEncryptionPermission
		diagnostic.Summary = fmt.Sprintf("%s failed: the identity lacks the KMS permission required by the bucket encryption", operation)
		diagnostic.Remediation = "Grant the IAM identity the required KMS encrypt/decrypt permissions for the bucket's default key, or use SSE-S3 bucket encryption."
		diagnostic.Retryable = false
	case code == "AccessDenied" || code == "Forbidden" || status == 403:
		diagnostic.Code = deniedCodeForOperation(operation)
		if diagnostic.Code == CodeHeadBucketDenied {
			// AWS answers the same 403 for a missing bucket, a bucket in
			// another account, and a missing s3:ListBucket grant, so the
			// diagnostic states all three instead of guessing one.
			diagnostic.Summary = fmt.Sprintf("HeadBucket was denied for bucket %q. AWS returns this same denial whether the bucket does not exist, belongs to another AWS account, or this identity lacks s3:GetBucketLocation/s3:ListBucket.", bucket)
			if status != 0 {
				diagnostic.Summary = fmt.Sprintf("%s (HTTP %d)", diagnostic.Summary, status)
			}
		} else {
			diagnostic.Summary = fmt.Sprintf("%s denied on bucket %q", operation, bucket)
		}
		diagnostic.Remediation = remediationForDeniedOperation(operation)
		diagnostic.Retryable = false
	default:
		diagnostic.Code = CodeOperationFailed
		if message != "" {
			diagnostic.Summary = fmt.Sprintf("%s failed: %s", operation, message)
		} else {
			diagnostic.Summary = fmt.Sprintf("%s failed: %v", operation, err)
		}
		diagnostic.Remediation = "Review the delivery storage settings and IAM policy, then retry."
		diagnostic.Retryable = true
	}
	return diagnostic
}

func deniedCodeForOperation(operation string) DiagnosticCode {
	switch operation {
	case "HeadBucket", "GetBucketLocation":
		return CodeHeadBucketDenied
	case "ListBucket":
		return CodeListBucketDenied
	case "PutObject":
		return CodePutObjectDenied
	case "GetObject":
		return CodeGetObjectDenied
	case "DeleteObject":
		return CodeDeleteObjectDenied
	default:
		return CodeOperationFailed
	}
}

func remediationForDeniedOperation(operation string) string {
	switch operation {
	case "HeadBucket", "GetBucketLocation":
		return "Confirm the bucket name and region are correct and that the bucket exists in the same AWS account as this access key. If it does, attach VrooliDeliveryBucketAccess (s3:GetBucketLocation, s3:ListBucket, s3:ListBucketMultipartUploads) to this IAM user; a bucket in another account requires a bucket policy that grants these actions."
	case "ListBucket":
		return "Add s3:ListBucket for the bucket ARN to the VrooliDeliveryBucketAccess policy."
	case "PutObject":
		return "Add s3:PutObject for the bucket object ARN to the VrooliDeliveryBucketAccess policy."
	case "GetObject":
		return "Add s3:GetObject for the bucket object ARN to the VrooliDeliveryBucketAccess policy."
	case "DeleteObject":
		return "Add s3:DeleteObject for the bucket object ARN to the VrooliDeliveryBucketAccess policy."
	default:
		return "Confirm the VrooliDeliveryBucketAccess policy grants the operation for this bucket."
	}
}

func isEncryptionError(lowerCode, lowerMessage string) bool {
	if strings.HasPrefix(lowerCode, "kms") || strings.Contains(lowerCode, "kmsaccessdenied") {
		return true
	}
	return strings.Contains(lowerMessage, "kms") && (strings.Contains(lowerMessage, "denied") || strings.Contains(lowerMessage, "not authorized"))
}

func isNetworkError(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "tls handshake") ||
		strings.Contains(lower, "certificate") ||
		strings.Contains(lower, "dial tcp")
}
