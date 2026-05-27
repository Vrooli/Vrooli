// run_artifacts.go enumerates and safely resolves a run's recorded artifacts
// (currently videos) so external consumers — git-control-tower's Workflows tab
// via WorkflowReplayService — can list and stream them without reaching into
// test-genie's filesystem directly.
//
// Videos are written by the playbooks FileWriter to
// coverage/runs/<runID>/automation/<workflow>/video/<file> (see
// playbooks/artifacts/workflow_writer.go).

package artifacts

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunVideo describes one recorded workflow video, addressable by a path
// relative to the run's artifact root (RunDir). RelPath is the stable handle a
// consumer passes back to ResolveRunArtifact to fetch the bytes.
type RunVideo struct {
	Workflow  string `json:"workflow"`
	RelPath   string `json:"rel_path"`
	SizeBytes int64  `json:"size_bytes"`
}

// videoSubdir is the per-workflow subdirectory diagnostic videos land in
// (WriteDiagnosticArtifact uses kind="video").
const videoSubdir = "video"

// ListRunVideos enumerates every recorded video under a run's automation tree,
// grouped by workflow directory. Returns an empty slice (no error) when the run
// has no automation artifacts, so callers can render an empty state.
func ListRunVideos(scenarioDir, runID string) ([]RunVideo, error) {
	automationDir := RunAutomationDir(scenarioDir, runID)
	entries, err := os.ReadDir(automationDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []RunVideo{}, nil
		}
		return nil, err
	}

	out := []RunVideo{}
	for _, wf := range entries {
		if !wf.IsDir() {
			continue
		}
		videoDir := filepath.Join(automationDir, wf.Name(), videoSubdir)
		files, ferr := os.ReadDir(videoDir)
		if ferr != nil {
			continue // no video subdir for this workflow
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			info, ierr := f.Info()
			if ierr != nil {
				continue
			}
			abs := filepath.Join(videoDir, f.Name())
			rel, rerr := filepath.Rel(RunDir(scenarioDir, runID), abs)
			if rerr != nil {
				continue
			}
			out = append(out, RunVideo{
				Workflow:  wf.Name(),
				RelPath:   filepath.ToSlash(rel),
				SizeBytes: info.Size(),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Workflow != out[j].Workflow {
			return out[i].Workflow < out[j].Workflow
		}
		return out[i].RelPath < out[j].RelPath
	})
	return out, nil
}

// ResolveRunArtifact maps a run-relative artifact path to an absolute path,
// rejecting any path that escapes the run's artifact root (traversal guard).
// The returned path is guaranteed to live under RunDir(scenarioDir, runID).
func ResolveRunArtifact(scenarioDir, runID, relPath string) (string, error) {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "", fmt.Errorf("artifact path is required")
	}
	if filepath.IsAbs(relPath) || strings.HasPrefix(relPath, "/") {
		return "", fmt.Errorf("artifact path must be relative")
	}
	runRoot := RunDir(scenarioDir, runID)
	// Clean against the run root, then verify containment.
	abs := filepath.Clean(filepath.Join(runRoot, filepath.FromSlash(relPath)))
	rootWithSep := runRoot + string(os.PathSeparator)
	if abs != runRoot && !strings.HasPrefix(abs, rootWithSep) {
		return "", fmt.Errorf("artifact path escapes run directory")
	}
	return abs, nil
}
