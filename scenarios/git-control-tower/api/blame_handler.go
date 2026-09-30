package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git-control-tower/internal/provenance"
)

func (s *Server) handleBlame(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 15*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()
	runner, ok := hctx.Git.(BlameRunner)
	if !ok {
		hctx.Resp.InternalError("configured git runner does not support blame")
		return
	}
	query := r.URL.Query()
	paths := append([]string(nil), query["path"]...)
	paths = append(paths, query["paths"]...)
	start, _ := strconv.Atoi(query.Get("start_line"))
	end, _ := strconv.Atoi(query.Get("end_line"))
	maxPaths, _ := strconv.Atoi(query.Get("max_paths"))
	maxLines, _ := strconv.Atoi(query.Get("max_lines"))
	maxBytes, _ := strconv.Atoi(query.Get("max_bytes"))
	result, err := ReadBlame(hctx.Ctx, runner, BlameRequest{
		RepoDir: hctx.RepoDir, Revision: strings.TrimSpace(query.Get("revision")), Paths: paths,
		StartLine: start, EndLine: end, MaxPaths: maxPaths, MaxLines: maxLines, MaxBytes: maxBytes,
		Enrich: query.Get("enrich") == "true",
	})
	if result != nil && query.Get("enrich") == "true" {
		s.enrichBlame(context.WithoutCancel(hctx.Ctx), hctx.RepoDir, result)
	}
	if err != nil {
		hctx.Resp.BadRequest(err.Error())
		return
	}
	hctx.Resp.OK(result)
}

func enrichedBlameFile(file BlameFile, evidence []provenance.Evidence) BlameFile {
	joined := JoinBlameEvidence(file, file.ContentDigest != "", evidence)
	file.Standing, file.DowngradeReasons, file.Evidence = joined.Standing, joined.DowngradeReasons, joined.Evidence
	return file
}
