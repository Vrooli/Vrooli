package main

import (
	"net/http"
	"time"
)

// handleTrackedBinaries reports compiled executables that git is tracking.
//
// Scanning reads every tracked file's first bytes, so it gets a longer budget
// than the 10s repo-read default: a large repository has tens of thousands of
// entries.
func (s *Server) handleTrackedBinaries(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 60*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	result, err := AnalyzeTrackedBinaries(hctx.Ctx, HealthDeps{
		FS:      OSFileIO{},
		RepoDir: hctx.RepoDir,
	}, s.git)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}
	hctx.Resp.OK(result)
}

// handleUntrackBinary removes one binary from the index and ignores it.
//
// Uses RepoWrite: this mutates .git/index, so it must take the per-repo lock
// that serializes index commands against a concurrent stage or commit.
func (s *Server) handleUntrackBinary(w http.ResponseWriter, r *http.Request) {
	hctx := RepoWrite(w, r, s.git, s.repos, s.repoLock, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req UntrackBinaryRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := UntrackBinary(hctx.Ctx, HealthDeps{
		FS:      OSFileIO{},
		RepoDir: hctx.RepoDir,
	}, s.git, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}
	if !result.Success {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
}
