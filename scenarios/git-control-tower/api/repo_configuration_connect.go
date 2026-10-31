package main

import (
	"context"
	"fmt"
	"path/filepath"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/storage"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
)

func (s *Server) groupingConfigPath(ctx context.Context, repoID int64) (string, error) {
	if s.fileRoots != nil {
		root, err := s.fileRoots.Pick(ctx, storage.ClassConfig)
		if err != nil {
			return "", err
		}
		return filepath.Join(root, fmt.Sprintf("%d/grouping-rules.json", repoID)), nil
	}
	return s.storageResolver.Path(
		storage.Options{ScenarioID: "git-control-tower"},
		storage.ClassConfig,
		fmt.Sprintf("%d/grouping-rules.json", repoID),
	)
}

func (s *Server) getGroupingRulesConnect(ctx context.Context, req *repov1.GetGroupingRulesRequest) (*repov1.GroupingRulesResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connectFileError(err)
	}
	configPath, err := s.groupingConfigPath(ctx, resolved.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	config, err := LoadGroupingRules(GroupingDeps{FS: OSFileIO{}, ConfigPath: configPath})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return groupingRulesProto(*config), nil
}

func (s *Server) saveGroupingRulesConnect(ctx context.Context, req *repov1.SaveGroupingRulesRequest) (*repov1.GroupingRulesResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.grouping-rules", "save grouping rules", func(writeCtx context.Context, repoPath string) (any, error) {
		resolved, resolveErr := s.resolveRepoForConnect(writeCtx, req.GetRepositoryId(), repoPath)
		if resolveErr != nil {
			return nil, resolveErr
		}
		configPath, pathErr := s.groupingConfigPath(writeCtx, resolved.ID)
		if pathErr != nil {
			return nil, pathErr
		}
		config := groupingRulesFromProto(req)
		if saveErr := SaveGroupingRules(GroupingDeps{AuthContext: writeCtx, FS: OSFileIO{}, ConfigPath: configPath}, config); saveErr != nil {
			return nil, saveErr
		}
		return &config, nil
	})
	if err != nil {
		return nil, err
	}
	config, _ := result.(*GroupingRulesConfig)
	if config == nil {
		return &repov1.GroupingRulesResponse{}, nil
	}
	return groupingRulesProto(*config), nil
}

func (s *Server) getGitignoreHealthConnect(ctx context.Context, req *repov1.GetGitignoreHealthRequest) (*repov1.GitignoreHealthResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connectFileError(err)
	}
	configPath, err := s.groupingConfigPath(ctx, resolved.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result, err := AnalyzeGitignoreHealth(HealthDeps{
		FS: OSFileIO{}, RepoDir: resolved.Path,
		GroupingDeps: GroupingDeps{FS: OSFileIO{}, ConfigPath: configPath},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return gitignoreHealthProto(*result), nil
}

func (s *Server) moveGitignoreEntryConnect(ctx context.Context, req *repov1.MoveGitignoreEntryRequest) (*repov1.MoveGitignoreEntryResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.gitignore.move", "move gitignore entry", func(writeCtx context.Context, repoPath string) (any, error) {
		resolved, resolveErr := s.resolveRepoForConnect(writeCtx, req.GetRepositoryId(), repoPath)
		if resolveErr != nil {
			return nil, resolveErr
		}
		configPath, pathErr := s.groupingConfigPath(writeCtx, resolved.ID)
		if pathErr != nil {
			return nil, pathErr
		}
		return MoveGitignoreEntry(HealthDeps{
			AuthContext: writeCtx, FS: OSFileIO{}, RepoDir: resolved.Path,
			GroupingDeps: GroupingDeps{FS: OSFileIO{}, ConfigPath: configPath},
		}, GitignoreMoveRequest{
			Line: int(req.GetLine()), Pattern: req.GetPattern(), GroupDir: req.GetGroupDir(), TargetPattern: req.GetTargetPattern(),
		})
	})
	if err != nil {
		return nil, err
	}
	value, _ := result.(*GitignoreMoveResponse)
	if value == nil {
		return &repov1.MoveGitignoreEntryResponse{}, nil
	}
	return &repov1.MoveGitignoreEntryResponse{Success: value.Success, RemovedFrom: value.RemovedFrom, AddedTo: value.AddedTo, Error: value.Error}, nil
}

func (s *Server) getTrackedBinariesConnect(ctx context.Context, req *repov1.GetTrackedBinariesRequest) (*repov1.TrackedBinariesResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connectFileError(err)
	}
	result, err := AnalyzeTrackedBinaries(ctx, HealthDeps{FS: OSFileIO{}, RepoDir: resolved.Path}, s.git)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	binaries := make([]*repov1.TrackedBinary, 0, len(result.Binaries))
	for _, binary := range result.Binaries {
		binaries = append(binaries, &repov1.TrackedBinary{
			Path: binary.Path, Bytes: binary.Bytes, Format: binary.Format, OwnerDir: binary.OwnerDir,
			IgnorePattern: binary.IgnorePattern, AlreadyIgnored: binary.AlreadyIgnored,
		})
	}
	return &repov1.TrackedBinariesResponse{Binaries: binaries, TotalBytes: result.TotalBytes, HistoryWarning: result.HistoryWarning}, nil
}

func (s *Server) untrackBinaryConnect(ctx context.Context, req *repov1.UntrackBinaryRequest) (*repov1.UntrackBinaryResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.tracked-binaries.untrack", "untrack binary", func(writeCtx context.Context, repoPath string) (any, error) {
		return UntrackBinary(writeCtx, HealthDeps{FS: OSFileIO{}, RepoDir: repoPath}, s.git, UntrackBinaryRequest{
			Path: req.GetPath(), OwnerDir: req.GetOwnerDir(), IgnorePattern: req.GetIgnorePattern(),
		})
	})
	if err != nil {
		return nil, err
	}
	value, _ := result.(*UntrackBinaryResponse)
	if value == nil {
		return &repov1.UntrackBinaryResponse{}, nil
	}
	return &repov1.UntrackBinaryResponse{
		Success: value.Success, RemovedFromIndex: value.RemovedFromIndex, IgnoreAddedTo: value.IgnoreAddedTo, Error: value.Error,
	}, nil
}

func groupingRulesProto(config GroupingRulesConfig) *repov1.GroupingRulesResponse {
	rules := make([]*repov1.GroupingRule, 0, len(config.Rules))
	for _, rule := range config.Rules {
		rules = append(rules, &repov1.GroupingRule{Id: rule.ID, Label: rule.Label, Prefixes: rule.Prefixes, Mode: rule.Mode})
	}
	return &repov1.GroupingRulesResponse{Enabled: config.Enabled, Rules: rules}
}

func groupingRulesFromProto(req *repov1.SaveGroupingRulesRequest) GroupingRulesConfig {
	rules := make([]GroupingRule, 0, len(req.GetRules()))
	for _, rule := range req.GetRules() {
		rules = append(rules, GroupingRule{ID: rule.GetId(), Label: rule.GetLabel(), Prefixes: rule.GetPrefixes(), Mode: rule.GetMode()})
	}
	return GroupingRulesConfig{Enabled: req.GetEnabled(), Rules: rules}
}

func gitignoreHealthProto(result GitignoreHealthResponse) *repov1.GitignoreHealthResponse {
	suggestions := make([]*repov1.GitignoreSuggestion, 0, len(result.Suggestions))
	for _, suggestion := range result.Suggestions {
		suggestions = append(suggestions, &repov1.GitignoreSuggestion{
			Line: int32(suggestion.Line), Pattern: suggestion.Pattern, Type: suggestion.Type,
			GroupLabel: suggestion.GroupLabel, GroupDir: suggestion.GroupDir, TargetPattern: suggestion.TargetPattern,
			HasGitignore: suggestion.HasGitignore,
		})
	}
	return &repov1.GitignoreHealthResponse{RootEntryCount: int32(result.RootEntryCount), Suggestions: suggestions}
}
