//go:build !windows

package lifecycle

import (
	"os"

	"github.com/vrooli/vrooli/internal/tuning"
)

// hostRecognizeArtifact (Unix): a regular file is a runnable build artifact iff
// it carries any execute bit. This is the only behavioral OS rule in the
// freshness path; the decision logic in isRunnableArtifact consumes its evidence
// without referencing runtime.GOOS.
func hostRecognizeArtifact(_ string, info os.FileInfo) artifactEvidence {
	return artifactEvidence{Known: true, Runnable: info.Mode()&tuning.PermExecuteMask != 0}
}

// hostVolumeCaseEvidence (Unix/Linux): native filesystems are case-sensitive.
// (macOS APFS/HFS+ default to case-insensitive, but the conservative
// case-sensitive verdict is correctness-safe there too — see caseEvidence — so a
// single Unix rule suffices without probing each mount.)
func hostVolumeCaseEvidence(_ string) caseEvidence {
	return caseEvidence{Known: true, Insensitive: false}
}
