package main

import (
	"net/http"
	"time"
)

func (s *Server) handlePrecommitGet(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()
	cfg, err := s.precommit.Get(hctx.Ctx, hctx.RepoDir)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}
	hctx.Resp.OK(cfg)
}

func (s *Server) handlePrecommitSave(w http.ResponseWriter, r *http.Request) {
	hctx := RepoWrite(w, r, s.git, s.repos, s.repoLock, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()
	var req PrecommitConfig
	if !ParseJSONBody(w, r, &req) {
		return
	}
	cfg, err := s.precommit.Save(hctx.Ctx, hctx.RepoDir, req)
	if err != nil {
		hctx.Resp.BadRequest(err.Error())
		return
	}
	hctx.Resp.OK(cfg)
}

func (s *Server) handlePrecommitRun(w http.ResponseWriter, r *http.Request) {
	hctx := RepoWrite(w, r, s.git, s.repos, s.repoLock, maxPrecommitTimeoutSeconds*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()
	var req PrecommitRunRequest
	if r.Body != nil && r.ContentLength != 0 {
		if !ParseJSONBody(w, r, &req) {
			return
		}
	}
	result, err := s.precommit.Run(hctx.Ctx, hctx.RepoDir, req)
	if err != nil {
		hctx.Resp.BadRequest(err.Error())
		return
	}
	resp := PrecommitRunResponse{Success: result.Status == "passed", Result: result}
	if !resp.Success {
		hctx.Resp.UnprocessableEntity(resp)
		return
	}
	hctx.Resp.OK(resp)
}
