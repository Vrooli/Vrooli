package cleanup

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"

	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/cleanup"
)

func TestReadAuthorizationDecodesOpaqueFields(t *testing.T) {
	input := map[string]string{
		"sealed_passphrase": base64.RawStdEncoding.EncodeToString([]byte("sealed-envelope")),
		"capability":        base64.StdEncoding.EncodeToString([]byte("opaque-capability")),
	}
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	sealed, capability, err := readAuthorization(bytes.NewReader(raw), &cleanupv1.CleanupOperation{})
	if err != nil {
		t.Fatal(err)
	}
	if string(sealed) != "sealed-envelope" || string(capability) != "opaque-capability" {
		t.Fatalf("sealed=%q capability=%q", sealed, capability)
	}
}
