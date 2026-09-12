package verify

import (
	"os"
	"path/filepath"
	"testing"

	verificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/verifications"

	"github.com/vrooli/cli-core/cliapp"
)

func TestVerifyManifestCoversVerificationsService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, verificationsv1.File_flow_verifier_v1_verifications_verifications_proto, "VerificationsService")
}
