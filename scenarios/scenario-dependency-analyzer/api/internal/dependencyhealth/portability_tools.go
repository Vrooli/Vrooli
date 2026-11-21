package dependencyhealth

import (
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/packages/deployability"
)

// validateToolAcquisitionCoverage applies the control plane's single
// acquisition-route rule to the repository's tool manifests while the
// portability provider is running. The rule, its manifest mapping, and the
// repository-data test all live in internal/deployability and internal/tools;
// this provider only supplies the directory.
func validateToolAcquisitionCoverage(repoRoot string) error {
	return deployability.ValidateToolManifestAcquisitionCoverage(os.DirFS(filepath.Join(repoRoot, "internal", "tools")))
}
