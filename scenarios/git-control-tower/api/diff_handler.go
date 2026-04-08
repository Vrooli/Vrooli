package main

import (
	"errors"
	"net/http"
	"time"
)

// [REQ:GCT-OT-P0-003] File diff endpoint
func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, s.repoLock, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	// Parse query parameters
	query := r.URL.Query()

	// Parse and validate view mode
	modeStr := query.Get("mode")
	var mode ViewMode
	switch modeStr {
	case "full_diff":
		mode = ViewModeFullDiff
	case "source":
		mode = ViewModeSource
	default:
		mode = ViewModeDiff
	}

	diff, err := GetDiff(hctx.Ctx, DiffDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, DiffRequest{
		Path:      query.Get("path"),
		Staged:    query.Get("staged") == "true",
		Untracked: query.Get("untracked") == "true",
		Base:      query.Get("base"),
		Commit:    query.Get("commit"),
		Mode:      mode,
		Any:       query.Get("any") == "true",
	})
	if err != nil {
		var tooLarge *FileTooLargeError
		if errors.As(err, &tooLarge) {
			hctx.Resp.PayloadTooLarge(tooLarge.Error())
			return
		}
		var unsupported *UnsupportedBinaryError
		if errors.As(err, &unsupported) {
			hctx.Resp.UnsupportedMediaType(unsupported.Error())
			return
		}
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(diff)
}
