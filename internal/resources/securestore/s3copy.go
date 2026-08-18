package securestore

// This file implements the small subset of the S3 protocol needed for the
// encrypted root copy. Keeping it here avoids making the control plane depend
// on one cloud vendor's SDK: any S3-compatible object store is a valid sink.

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

// ObjectStoreCredentials are resolved by the caller from the credential
// authority and live only for the duration of one upload.
type ObjectStoreCredentials struct {
	AccessKey    string
	SecretKey    string
	SessionToken string
}

// S3CopyOptions contains non-secret S3-compatible destination settings.
type S3CopyOptions struct {
	Region      string
	Endpoint    string
	Credentials ObjectStoreCredentials
	HTTPClient  *http.Client
	Now         func() time.Time
	// RepositorySinks contains non-secret s3://bucket/prefix locations for
	// registered Kopia repositories. A root-of-trust copy must not be placed
	// inside one of those repositories, even when both are object-store
	// locations rather than local filesystem paths.
	RepositorySinks []string
}

// CopyStoreS3 uploads the already encrypted store as one PUT object. Object
// PUT replacement is atomic from the perspective of S3 readers. The receipt
// is written only after the service acknowledges the object.
func CopyStoreS3(source, sink, receiptPath string, options S3CopyOptions) (CopyStatus, error) {
	sink = strings.TrimSpace(sink)
	if err := rejectS3RepositoryContainment(sink, options.RepositorySinks); err != nil {
		return CopyStatus{}, err
	}
	if strings.TrimSpace(options.Region) == "" {
		return CopyStatus{}, errors.New("object-store region is required")
	}
	if options.Credentials.AccessKey == "" || options.Credentials.SecretKey == "" {
		return CopyStatus{}, errors.New("object-store access and secret credentials are required")
	}
	if _, err := readSealedFile(source); err != nil {
		return CopyStatus{}, fmt.Errorf("read encrypted credential store: %w", err)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return CopyStatus{}, fmt.Errorf("read encrypted credential store: %w", err)
	}
	generation, err := StoreGeneration(source)
	if err != nil {
		return CopyStatus{}, err
	}
	objectURL, objectURI, err := s3ObjectURL(sink, options.Region, options.Endpoint)
	if err != nil {
		return CopyStatus{}, err
	}
	now := time.Now
	if options.Now != nil {
		now = options.Now
	}
	copiedAt := now().UTC()
	if err := putS3Object(options.HTTPClient, objectURL, data, options.Region, options.Credentials, copiedAt); err != nil {
		return CopyStatus{}, err
	}
	status := CopyStatus{Path: objectURI, Sink: sink, CopiedAt: copiedAt, Generation: generation}
	if err := writeCopyReceipt(receiptPath, status); err != nil {
		return CopyStatus{}, err
	}
	return status, nil
}

func rejectS3RepositoryContainment(sink string, repositories []string) error {
	sinkURL, err := parseS3Location(sink)
	if err != nil {
		return err
	}
	for _, repository := range repositories {
		repositoryURL, parseErr := parseS3Location(repository)
		if parseErr != nil {
			continue
		}
		if sinkURL.Host != repositoryURL.Host {
			continue
		}
		sinkPrefix := cleanS3Prefix(sinkURL.Path)
		repositoryPrefix := cleanS3Prefix(repositoryURL.Path)
		if repositoryPrefix == "" || sinkPrefix == repositoryPrefix || strings.HasPrefix(sinkPrefix, repositoryPrefix+"/") {
			return &SinkConflictError{
				Sink:       canonicalS3Location(sinkURL),
				Repository: canonicalS3Location(repositoryURL),
			}
		}
	}
	return nil
}

func parseS3Location(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "s3" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid object-store location %q; expected s3://bucket/prefix", raw)
	}
	return parsed, nil
}

func cleanS3Prefix(raw string) string {
	trimmed := strings.Trim(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return ""
	}
	return path.Clean("/" + trimmed)[1:]
}

func canonicalS3Location(value *url.URL) string {
	prefix := cleanS3Prefix(value.Path)
	if prefix == "" {
		return "s3://" + value.Host
	}
	return "s3://" + value.Host + "/" + prefix
}

func s3ObjectURL(raw, region, endpoint string) (string, string, error) {
	parsed, err := parseS3Location(raw)
	if err != nil {
		return "", "", fmt.Errorf("invalid object-store sink %q: %w", raw, err)
	}
	bucket := parsed.Host
	key := strings.Trim(parsed.Path, "/")
	if key == "" {
		key = "secrets.enc.json"
	} else {
		key = path.Join(key, "secrets.enc.json")
	}
	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if base == "" {
		base = "https://s3." + region + ".amazonaws.com"
	}
	baseURL, err := url.Parse(base)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return "", "", fmt.Errorf("invalid object-store endpoint %q", endpoint)
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/") + "/" + bucket + "/" + key
	return baseURL.String(), "s3://" + bucket + "/" + key, nil
}

func putS3Object(client *http.Client, objectURL string, data []byte, region string, credentials ObjectStoreCredentials, now time.Time) error {
	if client == nil {
		client = http.DefaultClient
	}
	payloadHash := sha256Hex(data)
	req, err := http.NewRequest(http.MethodPut, objectURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create object-store upload request: %w", err)
	}
	stamp := now.UTC()
	amzDate := stamp.Format("20060102T150405Z")
	date := stamp.Format("20060102")
	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	req.Header.Set("X-Amz-Date", amzDate)
	if credentials.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", credentials.SessionToken)
	}
	canonicalHeaders, signedHeaders := canonicalS3Headers(req)
	canonicalRequest := strings.Join([]string{http.MethodPut, canonicalS3Path(req.URL), "", canonicalHeaders, signedHeaders, payloadHash}, "\n")
	scope := date + "/" + region + "/s3/aws4_request"
	stringToSign := strings.Join([]string{"AWS4-HMAC-SHA256", amzDate, scope, sha256Hex([]byte(canonicalRequest))}, "\n")
	signingKey := hmacSHA256(hmacSHA256(hmacSHA256(hmacSHA256([]byte("AWS4"+credentials.SecretKey), []byte(date)), []byte(region)), []byte("s3")), []byte("aws4_request"))
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+credentials.AccessKey+"/"+scope+", SignedHeaders="+signedHeaders+", Signature="+signature)
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("upload encrypted credential store to object store: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("object-store upload returned %s: %s", response.Status, strings.TrimSpace(string(detail)))
	}
	return nil
}

func canonicalS3Headers(req *http.Request) (string, string) {
	values := map[string]string{
		"host":                 req.URL.Host,
		"x-amz-content-sha256": req.Header.Get("X-Amz-Content-Sha256"),
		"x-amz-date":           req.Header.Get("X-Amz-Date"),
	}
	if token := req.Header.Get("X-Amz-Security-Token"); token != "" {
		values["x-amz-security-token"] = token
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var canonical strings.Builder
	for _, key := range keys {
		canonical.WriteString(key)
		canonical.WriteByte(':')
		canonical.WriteString(strings.Join(strings.Fields(values[key]), " "))
		canonical.WriteByte('\n')
	}
	return strings.TrimSuffix(canonical.String(), "\n"), strings.Join(keys, ";")
}

func canonicalS3Path(value *url.URL) string {
	encoded := value.EscapedPath()
	if encoded == "" {
		return "/"
	}
	return encoded
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write(data)
	return h.Sum(nil)
}
