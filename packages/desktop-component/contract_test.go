package desktopcomponent

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"testing"

	release "github.com/vrooli/vrooli/packages/resource-deployment"
)

func featureEntry(owner, service, feature string) Entry {
	return Entry{SchemaVersion: 1, ID: owner, Owner: owner, Version: "0.0.1", OS: "linux", Arch: "amd64", HostRange: ">=0.0.1 <1.0.0", Provides: []Feature{{ID: feature, Service: owner + "--" + service, Port: "api"}}, StateClass: "stateless", License: "Vrooli source license", ArtifactURL: "https://example.invalid/component.tar.gz", SHA256: strings.Repeat("a", 64), TreeSHA256: strings.Repeat("b", 64), Size: 1024, Manifest: json.RawMessage(fmt.Sprintf(`{"services":[{"type":"api-binary","id":%q,"env":{"Z":"last","A":"first"},"ports":{"requested":[{"name":"api"}]},"health":{"type":"http","path":"/health","port_name":"api"},"readiness":{"type":"health_success"}}],"app":{"version":"0.0.1","scenario":%q}}`, owner+"--"+service, owner))}
}

// Sign even invalid declarations using the wire canonicalization, independently
// of admission. Rejection must not depend on the producer refusing to sign them.
func signFeatureEntry(t *testing.T, entry Entry, key *rsa.PrivateKey) SignedEntry {
	t.Helper()
	var manifest any
	decoder := json.NewDecoder(strings.NewReader(string(entry.Manifest)))
	decoder.UseNumber()
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatal(err)
	}
	entry.Manifest, _ = json.Marshal(manifest)
	canonical, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	signed := SignedEntry{Entry: entry, Signature: release.ReleaseSignature{SchemaVersion: "v1", KeyID: "test-release", Algorithm: "rsa-pkcs1v15-sha256", Signature: base64.StdEncoding.EncodeToString(signature)}}
	public, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err := release.VerifyReleaseSignature(canonical, signed.Signature, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public})); err != nil {
		t.Fatalf("test entry must have a valid signature: %v", err)
	}
	return signed
}

func TestSignedFeatureAdmissionRequiresOwnUsableService(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	public, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	publicPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public}))
	for _, fixture := range []struct{ owner, service, feature string }{{"audio-tools", "api", "audio.processing"}, {"agent-manager", "usage", "usage.live"}} {
		t.Run(fixture.feature, func(t *testing.T) {
			entry := featureEntry(fixture.owner, fixture.service, fixture.feature)
			if err := entry.ValidateFeatures(); err != nil {
				t.Fatalf("stored provider admission rejected own service: %v", err)
			}
			if err := signFeatureEntry(t, entry, key).Verify(publicPEM); err != nil {
				t.Fatalf("own projected provider rejected: %v", err)
			}
		})
	}
	cases := []struct {
		name string
		edit func(*Entry)
	}{
		{"host root service", func(e *Entry) { e.Provides[0].Service = "api" }},
		{"other component service", func(e *Entry) { e.Provides[0].Service = "agent-manager--usage" }},
		{"missing own service", func(e *Entry) { e.Provides[0].Service = "audio-tools--missing" }},
		{"missing requested port", func(e *Entry) { e.Provides[0].Port = "ui" }},
		{"duplicate feature", func(e *Entry) { e.Provides = append(e.Provides, e.Provides[0]) }},
		{"manifest owner mismatch", func(e *Entry) { e.Owner = "other-owner" }},
		{"manifest version mismatch", func(e *Entry) { e.Version = "0.0.2" }},
		{"host service inside manifest", func(e *Entry) {
			e.Manifest = json.RawMessage(strings.Replace(string(e.Manifest), `"services":[`, `"services":[{"id":"api"},`, 1))
		}},
		{"missing health", func(e *Entry) {
			e.Manifest = json.RawMessage(strings.Replace(string(e.Manifest), `"type":"http"`, `"type":""`, 1))
		}},
		{"disabled health", func(e *Entry) {
			e.Manifest = json.RawMessage(strings.Replace(string(e.Manifest), `"type":"http"`, `"type":"none"`, 1))
		}},
		{"unknown health", func(e *Entry) {
			e.Manifest = json.RawMessage(strings.Replace(string(e.Manifest), `"type":"http"`, `"type":"unknown"`, 1))
		}},
		{"undeclared health port", func(e *Entry) {
			e.Manifest = json.RawMessage(strings.Replace(string(e.Manifest), `"port_name":"api"`, `"port_name":"missing"`, 1))
		}},
		{"missing readiness", func(e *Entry) {
			e.Manifest = json.RawMessage(strings.Replace(string(e.Manifest), `"type":"health_success"`, `"type":""`, 1))
		}},
		{"unknown readiness", func(e *Entry) {
			e.Manifest = json.RawMessage(strings.Replace(string(e.Manifest), `"type":"health_success"`, `"type":"unknown"`, 1))
		}},
		{"undeclared readiness port", func(e *Entry) {
			e.Manifest = json.RawMessage(strings.Replace(string(e.Manifest), `"type":"health_success"`, `"type":"port_open","port_name":"missing"`, 1))
		}},
		{"ambiguous service", func(e *Entry) {
			e.Manifest = json.RawMessage(strings.Replace(string(e.Manifest), `"services":[`, `"services":[{"id":"audio-tools--api"},`, 1))
		}},
		{"ambiguous requested port", func(e *Entry) {
			e.Manifest = json.RawMessage(strings.Replace(string(e.Manifest), `"requested":[`, `"requested":[{"name":"api"},`, 1))
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := featureEntry("audio-tools", "api", "audio.processing")
			tc.edit(&entry)
			if err := entry.ValidateFeatures(); err == nil {
				t.Error("stored provider admission accepted invalid feature declaration")
			}
			if _, err := entry.CanonicalBytes(); err == nil {
				t.Error("producer admitted invalid feature declaration")
			}
			if err := signFeatureEntry(t, entry, key).Verify(publicPEM); err == nil {
				t.Error("consumer admitted correctly signed invalid feature declaration")
			}
		})
	}
}

func TestReleaseSignatureSurvivesCatalogJSONNormalization(t *testing.T) {
	entry := featureEntry("audio-tools", "api", "audio.processing")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	public, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public})
	canonical, err := entry.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	signed := SignedEntry{Entry: entry, Signature: release.ReleaseSignature{SchemaVersion: "v1", KeyID: "test-release", Algorithm: "rsa-pkcs1v15-sha256", Signature: base64.StdEncoding.EncodeToString(signature)}}
	if err = signed.Verify(string(publicPEM)); err != nil {
		t.Fatal(err)
	}
	// Exercise the generic metadata codec used around a JSONB catalog, which
	// reorders nested objects without changing release facts.
	raw, _ := json.Marshal(signed)
	var metadata map[string]any
	if err = json.Unmarshal(raw, &metadata); err != nil {
		t.Fatal(err)
	}
	raw, _ = json.MarshalIndent(metadata, "", "  ")
	var stored SignedEntry
	if err = json.Unmarshal(raw, &stored); err != nil {
		t.Fatal(err)
	}
	if err = stored.Verify(string(publicPEM)); err != nil {
		t.Fatalf("same release facts lost their signature through catalog JSON normalization: %v", err)
	}
	t.Run("changed nested release facts remain rejected", func(t *testing.T) {
		changed := stored
		changed.Entry.Manifest = json.RawMessage(strings.Replace(string(stored.Entry.Manifest), "api-binary", "different-code", 1))
		if err := changed.Verify(string(publicPEM)); err == nil {
			t.Fatal("changed nested manifest accepted with old release signature")
		}
	})
	t.Run("changed archive remains rejected", func(t *testing.T) {
		changed := stored
		changed.Entry.SHA256 = strings.Repeat("c", 64)
		if err := changed.Verify(string(publicPEM)); err == nil {
			t.Fatal("changed artifact digest accepted with old release signature")
		}
	})
}
