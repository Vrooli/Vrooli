package lifecycle

import "os"

// artifactEvidence is capability-flagged OS evidence about whether a stat'd path
// is a runnable compiled build artifact on the host. Known=false means the host
// probe cannot judge this OS, so freshness decision logic degrades to "assume
// runnable" (reuse) rather than condemning the artifact as missing. It mirrors
// scenarioruntime.ListenerEvidence's {value, known} shape so the freshness
// decision path carries no runtime.GOOS branch — the per-OS rule lives only in
// the build-tagged hostRecognizeArtifact seam below.
type artifactEvidence struct {
	Known    bool
	Runnable bool
}

// recognizeArtifact is the default artifact-recognition seam. A directory or nil
// info is never a runnable artifact (Known, not runnable); otherwise the per-OS
// hostRecognizeArtifact rule decides (exec bit on Unix, executable extension on
// Windows). Tests inject a different func to exercise the probe-unavailable
// (Known:false) degrade path.
func recognizeArtifact(path string, info os.FileInfo) artifactEvidence {
	if info == nil || info.IsDir() {
		return artifactEvidence{Known: true, Runnable: false}
	}
	return hostRecognizeArtifact(path, info)
}

// artifactRecognizer returns the configured artifact-recognition seam, or the
// default host recognizer when unset (e.g. a hostProbeDeps a test built without
// wiring the seam). Decision logic always has a usable recognizer.
func (d hostProbeDeps) artifactRecognizer() func(string, os.FileInfo) artifactEvidence {
	if d.recognizeArtifact != nil {
		return d.recognizeArtifact
	}
	return recognizeArtifact
}

// volumeCaseInsensitive reports whether the volume holding path is
// case-insensitive, via the configured seam (default host rule when unset). An
// unknown verdict degrades to false (case-sensitive comparison).
func (d hostProbeDeps) volumeCaseInsensitive(path string) bool {
	probe := d.volumeCaseEvidence
	if probe == nil {
		probe = hostVolumeCaseEvidence
	}
	ev := probe(path)
	return ev.Known && ev.Insensitive
}

// caseEvidence is capability-flagged OS evidence about whether the volume holding
// a path treats names case-insensitively. Known=false means the host cannot tell;
// callers then fall back to case-sensitive comparison (the correctness-safe
// default — it can only over-report staleness, never silently merge two distinct
// files). Like artifactEvidence, the per-OS rule lives in a build-tagged seam.
type caseEvidence struct {
	Known       bool
	Insensitive bool
}
