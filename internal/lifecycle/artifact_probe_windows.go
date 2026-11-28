//go:build windows

package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
)

// hostRecognizeArtifact (Windows): NTFS carries no Unix exec bit, so runnability
// is decided by executable extension. Go build artifacts are produced as `.exe`.
func hostRecognizeArtifact(path string, _ os.FileInfo) artifactEvidence {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".exe", ".com", ".bat", ".cmd":
		return artifactEvidence{Known: true, Runnable: true}
	default:
		return artifactEvidence{Known: true, Runnable: false}
	}
}

// hostVolumeCaseEvidence (Windows): NTFS/FAT volumes are case-insensitive.
func hostVolumeCaseEvidence(_ string) caseEvidence {
	return caseEvidence{Known: true, Insensitive: true}
}
