package operatorapi

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/shared"
)

// FileRoute serves research-home files read-only, so a review's REVIEW.html
// and the media it links by relative path open in the browser. A link from
// reviews/<id>/REVIEW.html to ../../topics/<t>/runs/… resolves under the
// same route.
const FileRoute = "/api/v1/home/"

// fileHandler serves one regular file under the research home. Hidden path
// parts are refused, and a symlink must resolve inside the home.
func fileHandler(researchDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(r.URL.Path, FileRoute)
		path, status, err := resolveHomeFile(researchDir, rel)
		if err != nil {
			writeError(w, status, err)
			return
		}
		f, err := os.Open(path)
		if err != nil {
			writeError(w, http.StatusNotFound, errors.New("the file cannot be opened"))
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || !info.Mode().IsRegular() {
			writeError(w, http.StatusNotFound, errors.New("not a regular file"))
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-cache")
		// Agent-written HTML is shown, never run: no scripts, no outside fetches.
		w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; img-src 'self' data:; media-src 'self'; style-src 'unsafe-inline'; font-src data:")
		http.ServeContent(w, r, info.Name(), info.ModTime(), f)
	}
}

func resolveHomeFile(researchDir, rel string) (string, int, error) {
	clean := filepath.Clean("/" + rel)[1:]
	if clean == "" {
		return "", http.StatusBadRequest, errors.New("name a file under the research home, e.g. reviews/<id>/REVIEW.html")
	}
	for _, part := range strings.Split(clean, "/") {
		if strings.HasPrefix(part, ".") {
			return "", http.StatusNotFound, errors.New("hidden files are not served")
		}
	}
	root, err := filepath.EvalSymlinks(researchDir)
	if err != nil {
		return "", http.StatusServiceUnavailable, errors.New("the research home cannot be read")
	}
	path, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(clean)))
	if err != nil {
		return "", http.StatusNotFound, errors.New("no such file under the research home")
	}
	if !strings.HasPrefix(path, root+string(filepath.Separator)) {
		return "", http.StatusNotFound, errors.New("the file resolves outside the research home")
	}
	return path, http.StatusOK, nil
}

func writeError(w http.ResponseWriter, status int, err error) {
	code := "not_found"
	switch status {
	case http.StatusBadRequest:
		code = "invalid_request"
	case http.StatusServiceUnavailable:
		code = "internal"
	}
	body, _ := protojson.Marshal(&sharedv1.ErrorEnvelope{Code: code, Message: err.Error()})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
