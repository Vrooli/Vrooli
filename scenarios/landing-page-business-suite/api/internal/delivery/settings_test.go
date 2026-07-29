package delivery

import "testing"

func TestValidateSettingsRejectsPartialCredentials(t *testing.T) {
	err := ValidateSettings(StorageSettings{SignedURLTTLSeconds: 900, AccessKeyID: "key"})
	if err == nil {
		t.Fatal("expected partial credentials to be rejected")
	}
}

func TestApplySettingsUpdateNormalizesValues(t *testing.T) {
	bucket := "  releases  "
	ttl := 120
	settings := ApplySettingsUpdate(StorageSettings{}, StorageSettingsUpdate{Bucket: &bucket, SignedURLTTLSeconds: &ttl})
	if settings.Bucket != "releases" || settings.SignedURLTTLSeconds != 120 {
		t.Fatalf("unexpected settings: %#v", settings)
	}
}
