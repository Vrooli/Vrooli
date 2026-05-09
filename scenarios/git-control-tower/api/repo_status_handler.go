package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
		IncludeHotspots: includeHotspots,
	})
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(status)
}

func (s *Server) handleRepoHistory(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 5*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	params, err := parseHistoryParams(r)
	if err != nil {
		hctx.Resp.BadRequest(err.Error())
		return
	}

	history, err := GetRepoHistory(hctx.Ctx, RepoHistoryDeps{
		Git:           hctx.Git,
		RepoDir:       hctx.RepoDir,
		Limit:         params.limit,
		IncludeFiles:  params.includeFiles,
		IncludeChecks: params.includeChecks,
		CommitChecks:  s.commitChecks,
		GrepPattern:   params.grepPattern,
	})
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(history)
}

// historyParams holds parsed and validated query parameters for the history endpoint.
type historyParams struct {
	limit         int
	includeFiles  bool
	includeChecks bool
	grepPattern   string
}

// parseHistoryParams extracts and validates query parameters for handleRepoHistory.
func parseHistoryParams(r *http.Request) (historyParams, error) {
	p := historyParams{limit: 30}
	includeParam := strings.TrimSpace(r.URL.Query().Get("include"))
	for _, token := range strings.Split(includeParam, ",") {
		switch strings.TrimSpace(token) {
		case "files", "details":
			p.includeFiles = true
		case "checks":
			p.includeChecks = true
		}
	}
	p.grepPattern = strings.TrimSpace(r.URL.Query().Get("grep"))

	if strings.ContainsAny(p.grepPattern, "\x00\n\r") {
		return p, fmt.Errorf("grep pattern contains invalid characters")
	}

	hasExplicitLimit := false
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		hasExplicitLimit = true
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return p, fmt.Errorf("limit must be a positive integer")
		}
		p.limit = parsed
	}

	p.limit = capHistoryLimit(p.limit, p.grepPattern, hasExplicitLimit)
	return p, nil
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
