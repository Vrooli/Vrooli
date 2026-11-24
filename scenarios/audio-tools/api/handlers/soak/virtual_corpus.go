//go:build audio_lab

package soak

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// serveVirtualCorpus is intentionally a fixed, test-only corpus surface. The
// browser accelerated lane must obtain its samples through an explicit
// qualification route; it must never accept an arbitrary local path from a
// page query string. The fixture is the same canonical speech sample used by
// the realtime BAS lane, and its reference is carried as immutable metadata
// for the independent quality assertion.
func serveVirtualCorpus(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("clip") != "quick-brown-fox" {
		writeError(w, http.StatusNotFound, "unknown virtual corpus clip")
		return
	}
	roots := make([]string, 0, 6)
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		roots = append(roots, filepath.Join(root, "scenarios", "audio-tools"))
	}
	for _, name := range []string{"SCENARIO_ROOT", "BUNDLE_ROOT"} {
		if root := strings.TrimSpace(os.Getenv(name)); root != "" {
			roots = append(roots, root)
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			roots = append(roots, dir)
		}
	}
	candidates := make([]string, 0, len(roots))
	for _, root := range roots {
		candidates = append(candidates, filepath.Join(root, "bas", "fixtures", "dictation-reference.wav"))
	}
	for _, path := range candidates {
		// #nosec G703 -- path contains only the fixed fixture suffix under operator-configured roots; no request path is used.
		if _, err := os.Stat(path); err != nil {
			continue
		}
		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("X-Vrooli-Corpus-Id", "quick-brown-fox")
		w.Header().Set("X-Vrooli-Corpus-Reference", "the quick brown fox jumps.")
		// #nosec G703 -- path is the fixed fixture candidate selected above; the request cannot choose a filesystem path.
		http.ServeFile(w, r, path)
		return
	}
	writeError(w, http.StatusNotFound, "virtual corpus fixture is unavailable")
}
