package destinations

import (
	"fmt"
	"net/url"
	"strings"
)

// s3Location maps an operator-facing location to the non-secret parameters
// resource-kopia needs. Legacy plain bucket names remain valid. Provider
// endpoints use s3+https://endpoint/bucket/prefix (or s3+http:// for an
// explicitly cleartext local-compatible service); standard S3 uses
// s3://bucket/prefix. Credentials are never accepted in the URI.
type s3Location struct {
	Bucket     string
	Endpoint   string
	Prefix     string
	Region     string
	DisableTLS bool
}

func parseS3Location(raw string) (s3Location, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return s3Location{}, fmt.Errorf("S3 location is required")
	}
	if !strings.Contains(raw, "://") {
		if strings.ContainsAny(raw, "?#") {
			return s3Location{}, fmt.Errorf("plain S3 location cannot contain a query or fragment")
		}
		parts := strings.SplitN(strings.Trim(raw, "/"), "/", 2)
		if parts[0] == "" {
			return s3Location{}, fmt.Errorf("plain S3 location must contain a bucket name")
		}
		location := s3Location{Bucket: parts[0]}
		if len(parts) == 2 {
			location.Prefix = strings.Trim(parts[1], "/")
		}
		return location, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return s3Location{}, fmt.Errorf("parse S3 location: %w", err)
	}
	if u.User != nil || u.Fragment != "" {
		return s3Location{}, fmt.Errorf("S3 location cannot contain credentials or a fragment")
	}
	region := strings.TrimSpace(u.Query().Get("region"))
	if len(u.Query()) > 0 {
		for key := range u.Query() {
			if key != "region" {
				return s3Location{}, fmt.Errorf("unsupported S3 location query parameter %q", key)
			}
		}
	}
	path := strings.Trim(u.EscapedPath(), "/")
	if path != "" {
		path, err = url.PathUnescape(path)
		if err != nil {
			return s3Location{}, fmt.Errorf("decode S3 location path: %w", err)
		}
	}
	if u.Scheme == "s3" {
		if u.Host == "" {
			return s3Location{}, fmt.Errorf("s3:// location requires a bucket name")
		}
		return s3Location{Bucket: u.Host, Prefix: strings.Trim(path, "/"), Region: region}, nil
	}
	if u.Scheme != "s3+https" && u.Scheme != "s3+http" {
		return s3Location{}, fmt.Errorf("unsupported S3 location scheme %q", u.Scheme)
	}
	if u.Host == "" || path == "" {
		return s3Location{}, fmt.Errorf("provider S3 location requires endpoint and bucket path: s3+https://endpoint/bucket/prefix")
	}
	parts := strings.SplitN(path, "/", 2)
	if strings.TrimSpace(parts[0]) == "" {
		return s3Location{}, fmt.Errorf("provider S3 location requires a bucket path")
	}
	return s3Location{
		Bucket: parts[0], Endpoint: u.Host, Prefix: func() string {
			if len(parts) == 2 {
				return strings.Trim(parts[1], "/")
			}
			return ""
		}(), Region: region, DisableTLS: u.Scheme == "s3+http",
	}, nil
}
