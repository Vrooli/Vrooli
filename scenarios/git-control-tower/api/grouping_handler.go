package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/vrooli/api-core/storage"
)

func (s *Server) groupingConfigPath(repoID int64) (string, error) {
	return s.storageResolver.Path(
		storage.Options{ScenarioID: "git-control-tower"},
		storage.ClassConfig,
		fmt.Sprintf("%d/grouping-rules.json", repoID),
	)
}

func (s *Server) handleGetGroupingRules(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 5*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	configPath, err := s.groupingConfigPath(hctx.RepoID)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	cfg, err := LoadGroupingRules(GroupingDeps{FS: OSFileIO{}, ConfigPath: configPath})
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(cfg)
}

func (s *Server) handleSaveGroupingRules(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 5*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var cfg GroupingRulesConfig
	if !ParseJSONBody(w, r, &cfg) {
		return
	}

	configPath, err := s.groupingConfigPath(hctx.RepoID)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	if err := SaveGroupingRules(GroupingDeps{FS: OSFileIO{}, ConfigPath: configPath}, cfg); err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(cfg)
}

func (s *Server) handleGitignoreHealth(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	configPath, err := s.groupingConfigPath(hctx.RepoID)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	result, err := AnalyzeGitignoreHealth(HealthDeps{
		FS:      OSFileIO{},
		RepoDir: hctx.RepoDir,
		GroupingDeps: GroupingDeps{
			FS:         OSFileIO{},
			ConfigPath: configPath,
		},
	})
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

func (s *Server) handleGitignoreMove(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req GitignoreMoveRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	configPath, err := s.groupingConfigPath(hctx.RepoID)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	result, err := MoveGitignoreEntry(HealthDeps{
		FS:      OSFileIO{},
		RepoDir: hctx.RepoDir,
		GroupingDeps: GroupingDeps{
			FS:         OSFileIO{},
			ConfigPath: configPath,
		},
	}, req)
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
