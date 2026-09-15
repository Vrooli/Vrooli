package delivery

import "testing"

func TestStableS3URI(t *testing.T) {
	if got, want := StableS3URI("release-bucket", "releases/app.zip"), "s3://release-bucket/releases/app.zip"; got != want {
		t.Fatalf("StableS3URI() = %q, want %q", got, want)
	}
}
