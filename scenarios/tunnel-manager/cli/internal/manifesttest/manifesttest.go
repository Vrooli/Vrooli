package manifesttest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// RequireServiceCoverage asserts every RPC declared on serviceName has a
// manifest binding or an explicit omission.
func RequireServiceCoverage(t *testing.T, file protoreflect.FileDescriptor, serviceName string) {
	t.Helper()
	manifestPath := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read %s: %v", manifestPath, err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, file, serviceName)
}
