package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/policygate"
	"git-control-tower/ssh"
	"github.com/vrooli/cli-core/cliutil"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
)

func (s *Server) getFilesConnect(ctx context.Context, req *repov1.GetFilesRequest) (*repov1.GetFilesResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connectFileError(err)
	}
	result, err := GetFileTree(ctx, FileDeps{Git: s.git, RepoDir: resolved.Path}, FileTreeRequest{
		Pattern: req.GetPattern(),
		Limit:   int(req.GetLimit()),
		Deep:    req.GetDeep(),
		Timeout: int(req.GetTimeoutMs()),
	})
	if err != nil {
		return nil, connectFileError(err)
	}
	files := make([]*repov1.RepoFileInfo, 0, len(result.Files))
	for _, file := range result.Files {
		files = append(files, &repov1.RepoFileInfo{Path: file.Path, Language: file.Language, Status: string(file.Status)})
	}
	return &repov1.GetFilesResponse{
		Files: files, Truncated: result.Truncated, Cancelled: result.Cancelled,
		SearchMode: result.SearchMode, Timestamp: result.Timestamp.UTC().Format(time.RFC3339Nano),
	}, nil
}

func (s *Server) getDirectoryContentsConnect(ctx context.Context, req *repov1.GetDirectoryContentsRequest) (*repov1.GetDirectoryContentsResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connectFileError(err)
	}
	result, err := GetDirectoryContents(ctx, FileDeps{Git: s.git, RepoDir: resolved.Path}, req.GetPath())
	if err != nil {
		return nil, connectFileError(err)
	}
	entries := make([]*repov1.DirectoryEntry, 0, len(result.Entries))
	for _, entry := range result.Entries {
		entries = append(entries, &repov1.DirectoryEntry{
			Name: entry.Name, Path: entry.Path, IsDir: entry.IsDir,
			Language: entry.Language, Tracked: entry.Tracked,
		})
	}
	return &repov1.GetDirectoryContentsResponse{
		Path: result.Path, Entries: entries, Timestamp: result.Timestamp.UTC().Format(time.RFC3339Nano),
	}, nil
}

func (s *Server) getRelatedFilesConnect(ctx context.Context, req *repov1.GetRelatedFilesRequest) (*repov1.GetRelatedFilesResponse, error) {
	if strings.TrimSpace(req.GetPath()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("path is required"))
	}
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connectFileError(err)
	}
	related, err := GetRelatedFiles(ctx, FileDeps{Git: s.git, RepoDir: resolved.Path}, req.GetPath())
	if err != nil {
		return nil, connectFileError(err)
	}
	result := make([]*repov1.RelatedFile, 0, len(related))
	for _, file := range related {
		result = append(result, &repov1.RelatedFile{Path: file.Path, RelationType: string(file.RelationType)})
	}
	return &repov1.GetRelatedFilesResponse{
		Path: req.GetPath(), Related: result, Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}, nil
}

func (s *Server) searchContentConnect(ctx context.Context, req *repov1.SearchContentRequest) (*repov1.SearchContentResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connectFileError(err)
	}
	result, err := SearchContent(ctx, ContentSearchDeps{Git: s.git, RepoDir: resolved.Path}, ContentSearchRequest{
		Query: req.GetQuery(), CaseSensitive: req.GetCaseSensitive(), WholeWord: req.GetWholeWord(),
		Regex: req.GetRegex(), Include: req.GetInclude(), Exclude: req.GetExclude(),
		ContextLines: int(req.GetContextLines()), Limit: int(req.GetLimit()), Timeout: int(req.GetTimeoutMs()),
	})
	if err != nil {
		return nil, connectFileError(err)
	}
	matches := make([]*repov1.ContentMatch, 0, len(result.Matches))
	for _, match := range result.Matches {
		matches = append(matches, &repov1.ContentMatch{
			Path: match.Path, LineNumber: int32(match.LineNumber), Content: match.Content,
			ContextBefore: match.ContextBefore, ContextAfter: match.ContextAfter,
		})
	}
	return &repov1.SearchContentResponse{
		Matches: matches, Total: int32(result.Total), Truncated: result.Truncated,
		Cancelled: result.Cancelled, Query: result.Query, Timestamp: result.Timestamp.UTC().Format(time.RFC3339Nano),
	}, nil
}

func (s *Server) deletePathConnect(ctx context.Context, req *repov1.DeletePathRequest) (*repov1.DeletePathResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.files.delete", "delete path", func(writeCtx context.Context, repoPath string) (any, error) {
		return DeletePath(writeCtx, FileDeps{Git: s.git, RepoDir: repoPath}, DeletePathRequest{Path: req.GetPath()})
	})
	if err != nil {
		return nil, err
	}
	value, _ := result.(*DeletePathResponse)
	if value == nil {
		return &repov1.DeletePathResponse{}, nil
	}
	return &repov1.DeletePathResponse{
		Success: value.Success, Path: value.Path, IsDir: value.IsDir, Error: value.Error,
		Timestamp: value.Timestamp.UTC().Format(time.RFC3339Nano),
	}, nil
}

func (s *Server) saveFileContentConnect(ctx context.Context, req *repov1.SaveFileContentRequest) (*repov1.SaveFileContentResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.files.content", "save file content", func(writeCtx context.Context, repoPath string) (any, error) {
		return SaveFileContent(writeCtx, FileContentDeps{FS: OSFileIO{}, RepoDir: repoPath}, SaveFileContentRequest{
			Path: req.GetPath(), Content: req.GetContent(), ExpectedHash: req.GetExpectedHash(),
		})
	})
	if err != nil {
		return nil, err
	}
	value, _ := result.(*SaveFileContentResponse)
	if value == nil {
		return &repov1.SaveFileContentResponse{}, nil
	}
	return &repov1.SaveFileContentResponse{
		Success: value.Success, Path: value.Path, ContentHash: value.ContentHash,
		BytesWritten: int32(value.BytesWritten), Timestamp: value.Timestamp.UTC().Format(time.RFC3339Nano),
	}, nil
}

func (s *Server) discardFilesConnect(ctx context.Context, req *repov1.DiscardFilesRequest) (*repov1.DiscardFilesResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.discard", "discard files", func(writeCtx context.Context, repoPath string) (any, error) {
		return DiscardFiles(writeCtx, DiscardDeps{Git: s.git, RepoDir: repoPath}, DiscardRequest{Paths: req.GetPaths(), Untracked: req.GetUntracked()})
	})
	if err != nil {
		return nil, err
	}
	value, _ := result.(*DiscardResponse)
	if value == nil {
		return &repov1.DiscardFilesResponse{}, nil
	}
	return &repov1.DiscardFilesResponse{Success: value.Success, Discarded: value.Discarded, Failed: value.Failed, Errors: value.Errors, Timestamp: value.Timestamp.UTC().Format(time.RFC3339Nano)}, nil
}

func (s *Server) ignorePathConnect(ctx context.Context, req *repov1.IgnorePathRequest) (*repov1.IgnorePathResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.ignore", "ignore path", func(writeCtx context.Context, repoPath string) (any, error) {
		return IgnorePath(writeCtx, IgnoreDeps{Git: s.git, FS: OSFileIO{}, RepoDir: repoPath}, IgnoreRequest{Path: req.GetPath(), Level: req.GetLevel(), GroupDir: req.GetGroupDir()})
	})
	if err != nil {
		return nil, err
	}
	value, _ := result.(*IgnoreResponse)
	if value == nil {
		return &repov1.IgnorePathResponse{}, nil
	}
	return &repov1.IgnorePathResponse{Success: value.Success, Ignored: value.Ignored, Failed: value.Failed, Errors: value.Errors, GitignorePath: value.GitignorePath, Timestamp: value.Timestamp.UTC().Format(time.RFC3339Nano)}, nil
}

func (s *Server) pushToRemoteConnect(ctx context.Context, req *repov1.PushToRemoteRequest) (*repov1.PushToRemoteResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.push", "push to remote", func(writeCtx context.Context, repoPath string) (any, error) {
		return PushToRemote(writeCtx, PushPullDeps{Git: s.git, RepoDir: repoPath, CredStore: s.credStore}, PushRequest{Remote: req.GetRemote(), Branch: req.GetBranch(), SetUpstream: req.GetSetUpstream()})
	})
	if err != nil {
		return nil, err
	}
	value, _ := result.(*PushResponse)
	if value == nil {
		return &repov1.PushToRemoteResponse{}, nil
	}
	return &repov1.PushToRemoteResponse{Success: value.Success, Remote: value.Remote, Branch: value.Branch, Pushed: value.Pushed, UpToDate: value.UpToDate, Verified: value.Verified, VerificationError: value.VerificationError, Error: value.Error, Timestamp: value.Timestamp.UTC().Format(time.RFC3339Nano)}, nil
}

func (s *Server) pullFromRemoteConnect(ctx context.Context, req *repov1.PullFromRemoteRequest) (*repov1.PullFromRemoteResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.pull", "pull from remote", func(writeCtx context.Context, repoPath string) (any, error) {
		return PullFromRemote(writeCtx, PushPullDeps{Git: s.git, RepoDir: repoPath, CredStore: s.credStore}, PullRequest{Remote: req.GetRemote(), Branch: req.GetBranch()})
	})
	if err != nil {
		return nil, err
	}
	value, _ := result.(*PullResponse)
	if value == nil {
		return &repov1.PullFromRemoteResponse{}, nil
	}
	return &repov1.PullFromRemoteResponse{Success: value.Success, Remote: value.Remote, Branch: value.Branch, Error: value.Error, HasConflicts: value.HasConflicts, Timestamp: value.Timestamp.UTC().Format(time.RFC3339Nano)}, nil
}

func (s *Server) runUpstreamActionConnect(ctx context.Context, req *repov1.RunUpstreamActionRequest) (*repov1.RunUpstreamActionResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.push", "upstream action", func(writeCtx context.Context, repoPath string) (any, error) {
		return RunUpstreamAction(writeCtx, PushPullDeps{Git: s.git, RepoDir: repoPath, CredStore: s.credStore}, UpstreamActionRequest{Action: req.GetAction(), Remote: req.GetRemote(), Branch: req.GetBranch(), Upstream: req.GetUpstream()})
	})
	if err != nil {
		return nil, err
	}
	value, _ := result.(*UpstreamActionResponse)
	if value == nil {
		return &repov1.RunUpstreamActionResponse{}, nil
	}
	return &repov1.RunUpstreamActionResponse{Success: value.Success, Action: value.Action, Remote: value.Remote, Branch: value.Branch, Upstream: value.Upstream, Error: value.Error, Timestamp: value.Timestamp.UTC().Format(time.RFC3339Nano)}, nil
}

func (s *Server) getPrecommitConfigConnect(ctx context.Context, req *repov1.GetPrecommitConfigRequest) (*repov1.PrecommitConfigResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connectFileError(err)
	}
	config, err := s.precommit.Get(ctx, resolved.Path)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return precommitConfigProto(config), nil
}

func (s *Server) savePrecommitConfigConnect(ctx context.Context, req *repov1.SavePrecommitConfigRequest) (*repov1.PrecommitConfigResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.precommit", "save precommit configuration", func(writeCtx context.Context, repoPath string) (any, error) {
		return s.precommit.Save(writeCtx, repoPath, PrecommitConfig{
			Enabled: req.GetEnabled(), Command: req.GetCommand(), WorkingDirectory: req.GetWorkingDirectory(),
			TimeoutSeconds: int(req.GetTimeoutSeconds()), RunBeforeCommit: req.GetRunBeforeCommit(), AllowOverride: req.GetAllowOverride(),
		})
	})
	if err != nil {
		return nil, err
	}
	value, _ := result.(*PrecommitConfig)
	if value == nil {
		return &repov1.PrecommitConfigResponse{}, nil
	}
	return precommitConfigProto(*value), nil
}

func (s *Server) runPrecommitConnect(ctx context.Context, req *repov1.RunPrecommitRequest) (*repov1.PrecommitRunResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connectFileError(err)
	}
	result, err := s.precommit.Run(ctx, resolved.Path, PrecommitRunRequest{
		Command: req.GetCommand(), WorkingDirectory: req.GetWorkingDirectory(), TimeoutSeconds: int(req.GetTimeoutSeconds()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return &repov1.PrecommitRunResponse{Success: result.Status == "passed", Result: precommitRunResultProto(result)}, nil
}

func precommitConfigProto(config PrecommitConfig) *repov1.PrecommitConfigResponse {
	result := &repov1.PrecommitConfigResponse{
		Enabled: config.Enabled, Command: config.Command, WorkingDirectory: config.WorkingDirectory,
		TimeoutSeconds: int32(config.TimeoutSeconds), RunBeforeCommit: config.RunBeforeCommit, AllowOverride: config.AllowOverride,
	}
	if config.LastResult != nil {
		result.LastResult = precommitRunResultProto(*config.LastResult)
	}
	if config.Hook != nil {
		result.Hook = &repov1.PrecommitHookState{
			Status: config.Hook.Status, Reason: config.Hook.Reason, ExistingKind: config.Hook.ExistingKind,
			ExistingHookPreview: config.Hook.ExistingHookPreview, Path: config.Hook.Path, HooksPath: config.Hook.HooksPath,
			InstalledAt: config.Hook.InstalledAt.UTC().Format(time.RFC3339Nano),
		}
	}
	return result
}

func precommitRunResultProto(result PrecommitRunResult) *repov1.PrecommitRunResult {
	return &repov1.PrecommitRunResult{
		Status: result.Status, Command: result.Command, ExitCode: int32(result.ExitCode), Summary: result.Summary,
		Stdout: result.Stdout, Stderr: result.Stderr, DurationMs: result.DurationMs,
		OverrideAllowed: result.OverrideAllowed, Timestamp: result.Timestamp.UTC().Format(time.RFC3339Nano),
	}
}

func (s *Server) runFileMutation(ctx context.Context, repositoryID, intentID, operation, writerName string, run func(context.Context, string) (any, error)) (any, error) {
	principal, ok := policygate.PrincipalFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authenticate through the configured provider before changing repository files"))
	}
	if principal.Kind != cliutil.CallerKindHuman {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("agent callers cannot change repository files"))
	}
	if strings.TrimSpace(intentID) == "" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("review the exact file change and confirm it before writing"))
	}

	writeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	repo, err := s.mutationRepo(writeCtx, repositoryID)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	cleanStaleLock(repo.Path)
	unlock, err := s.repoLock.Acquire(writeCtx, repo.Path)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("repository is busy, please retry"))
	}
	defer func() {
		unlock()
		s.repos.InvalidateStatus(repo.Path)
	}()

	preview, err := s.prepareMutation(writeCtx, repositoryIDFor(repo), operation)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	consumedIntent, err := s.intentService.Consume(writeCtx, principal, intentID, preview.RepositoryID, operation, preview.ExpectedRevision, preview.SubjectDigest)
	if err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	writeCtx = policygate.WithIntent(writeCtx, consumedIntent.AsHumanIntent())

	result, err := run(writeCtx, repo.Path)
	if err != nil {
		return nil, connectFileMutationError(writerName, err)
	}
	s.auditLogAsync(AuditEntry{
		Operation: fileMutationAuditOperation(writerName), RepoDir: repo.Path,
		Success: fileMutationSucceeded(result), Timestamp: time.Now().UTC(),
	})
	return result, nil
}

func fileMutationAuditOperation(writerName string) AuditOperation {
	switch writerName {
	case "discard files":
		return AuditOpDiscard
	case "ignore path":
		return AuditOpIgnore
	case "push to remote", "upstream action":
		return AuditOpPush
	case "pull from remote":
		return AuditOpPull
	case "move gitignore entry":
		return AuditOperation("repo_gitignore_move")
	case "untrack binary":
		return AuditOperation("repo_tracked_binaries_untrack")
	case "save grouping rules":
		return AuditOperation("repo_grouping_rules")
	default:
		return AuditOperation(strings.ReplaceAll(writerName, " ", "_"))
	}
}

func fileMutationSucceeded(result any) bool {
	switch value := result.(type) {
	case *DeletePathResponse:
		return value != nil && value.Success
	case *SaveFileContentResponse:
		return value != nil && value.Success
	case *DiscardResponse:
		return value != nil && value.Success
	case *IgnoreResponse:
		return value != nil && value.Success
	case *PushResponse:
		return value != nil && value.Success
	case *PullResponse:
		return value != nil && value.Success
	case *UpstreamActionResponse:
		return value != nil && value.Success
	case *PrecommitConfig:
		return value != nil
	case *GitignoreMoveResponse:
		return value != nil && value.Success
	case *UntrackBinaryResponse:
		return value != nil && value.Success
	case *GroupingRulesConfig:
		return value != nil
	case *CredentialSaveResponse:
		return value != nil && value.Success
	case *CredentialDeleteResponse:
		return value != nil && value.Success
	case *RemoteURLUpdateResponse:
		return value != nil && value.Success
	case *ssh.GenerateKeyResponse:
		return value != nil && value.Success
	case *ssh.DeleteKeyResponse:
		return value != nil && value.Success
	default:
		return false
	}
}

func (s *Server) auditLogAsync(entry AuditEntry) {
	if s.audit == nil {
		return
	}
	go func() {
		logCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.audit.Log(logCtx, entry)
	}()
}

func connectFileError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "required") || strings.Contains(message, "invalid") || strings.Contains(message, "query") || strings.Contains(message, "regex") {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if strings.Contains(message, "not found") || strings.Contains(message, "directory") {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func connectFileMutationError(writerName string, err error) error {
	if err == nil {
		return nil
	}
	var tooLarge *FileTooLargeError
	if errors.As(err, &tooLarge) {
		return connect.NewError(connect.CodeResourceExhausted, err)
	}
	var unsupported *UnsupportedBinaryError
	if errors.As(err, &unsupported) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	var conflict *FileContentConflictError
	if errors.As(err, &conflict) {
		connectErr := connect.NewError(connect.CodeAborted, err)
		connectErr.Meta().Set("X-GCT-Current-Hash", conflict.CurrentHash)
		return connectErr
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "required") || strings.Contains(message, "invalid") {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if strings.Contains(message, "not found") || strings.Contains(message, "directory") {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, errors.New(writerName+": "+err.Error()))
}
