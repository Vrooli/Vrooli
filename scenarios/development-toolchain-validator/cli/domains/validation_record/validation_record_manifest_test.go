package validation_record

import (
	"os"
	"path/filepath"
	"testing"

	vrecv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record"

	"github.com/vrooli/cli-core/cliapp"
)

func TestValidationRecordManifestCoversValidationRecordService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, vrecv1.File_development_toolchain_validator_v1_validation_record_validation_record_proto, "ValidationRecordService")
}
