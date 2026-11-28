package destinations

import "testing"

func TestParseS3Location(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want s3Location
	}{
		{name: "legacy bucket", in: "backup-bucket", want: s3Location{Bucket: "backup-bucket"}},
		{name: "standard bucket prefix and region", in: "s3://backup-bucket/vrooli/daily?region=us-east-2", want: s3Location{Bucket: "backup-bucket", Prefix: "vrooli/daily", Region: "us-east-2"}},
		{name: "provider TLS endpoint", in: "s3+https://objects.example:9443/vrooli-backups/host-a?region=west", want: s3Location{Bucket: "vrooli-backups", Endpoint: "objects.example:9443", Prefix: "host-a", Region: "west"}},
		{name: "local cleartext endpoint", in: "s3+http://minio:9000/backups", want: s3Location{Bucket: "backups", Endpoint: "minio:9000", DisableTLS: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseS3Location(tt.in)
			if err != nil {
				t.Fatalf("parseS3Location(%q): %v", tt.in, err)
			}
			if got != tt.want {
				t.Fatalf("parseS3Location(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseS3LocationRejectsCredentialsAndUnclearEndpoints(t *testing.T) {
	for _, input := range []string{
		"s3://user:password@bucket/prefix",
		"s3+https://endpoint",
		"s3+https://endpoint/bucket?token=secret",
		"ftp://endpoint/bucket",
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := parseS3Location(input); err == nil {
				t.Fatalf("parseS3Location(%q) unexpectedly succeeded", input)
			}
		})
	}
}
