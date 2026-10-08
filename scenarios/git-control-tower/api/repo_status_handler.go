package main

import (
	"net/http"
	"time"
)

func (s *Server) handleRepoStatus(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 5*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	includeHotspots := r.URL.Query().Get("hotspots") == "true"

	status, err := GetRepoStatus(hctx.Ctx, RepoStatusDeps{
		Git:             hctx.Git,
		RepoDir:         hctx.RepoDir,
		ConfigCache:     s.configCache,
		StatusCache:     s.statusCache,
		IncludeHotspots: includeHotspots,
	})
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(status)
}

// capHistoryLimit applies limit caps based on whether grep is active.
func capHistoryLimit(limit int, grepPattern string, hasExplicitLimit bool) int {
	if grepPattern != "" {
		if !hasExplicitLimit {
			return 1000
		}
		if limit > 1000 {
			return 1000
		}
		return limit
	}
	if limit > 200 {
		return 200
	}
	return limit
}
