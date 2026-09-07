// Package repo (handlers) implements the RepoService Connect-RPC
// service. It carries the complete repository-status contract consumed by
// the UI status model and CLI repo-status command, plus typed mutations.
package repo

import (
	"context"
	"errors"
	"log"
	"net/http"

	"connectrpc.com/connect"

	"git-control-tower/internal/repo"

	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
	repoconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo/repo_v1connect"
)

// Server implements repoconnect.RepoServiceHandler.
type Server struct {
	svc                  *repo.Service
	logger               *log.Logger
	listRepositories     func(context.Context, *repov1.ListRepositoriesRequest) (*repov1.ListRepositoriesResponse, error)
	getActiveRepository  func(context.Context, *repov1.GetActiveRepositoryRequest) (*repov1.GetActiveRepositoryResponse, error)
	setActiveRepository  func(context.Context, *repov1.SetActiveRepositoryRequest) (*repov1.RepoMutationResponse, error)
	openRepository       func(context.Context, *repov1.OpenRepositoryRequest) (*repov1.RepoMutationResponse, error)
	cloneRepository      func(context.Context, *repov1.CloneRepositoryRequest) (*repov1.RepoMutationResponse, error)
	removeRepository     func(context.Context, *repov1.RemoveRepositoryRequest) (*repov1.RemoveRepositoryResponse, error)
	getRepoStatus        func(context.Context, *repov1.GetRepoStatusRequest) (*repov1.GetRepoStatusResponse, error)
	getRepoDiff          func(context.Context, *repov1.GetRepoDiffRequest) (*repov1.GetRepoDiffResponse, error)
	getRepoGroups        func(context.Context, *repov1.GetRepoGroupsRequest) (*repov1.GetRepoGroupsResponse, error)
	getSyncStatus        func(context.Context, *repov1.GetSyncStatusRequest) (*repov1.GetSyncStatusResponse, error)
	getRepoHistory       func(context.Context, *repov1.GetRepoHistoryRequest) (*repov1.GetRepoHistoryResponse, error)
	getApprovedChanges   func(context.Context, *repov1.GetApprovedChangesRequest) (*repov1.GetApprovedChangesResponse, error)
	getProvenance        func(context.Context, *repov1.GetProvenanceRequest) (*repov1.GetProvenanceResponse, error)
	getBlame             func(context.Context, *repov1.GetBlameRequest) (*repov1.GetBlameResponse, error)
	searchProvenance     func(context.Context, *repov1.SearchProvenanceRequest) (*repov1.SearchProvenanceResponse, error)
	getFiles             func(context.Context, *repov1.GetFilesRequest) (*repov1.GetFilesResponse, error)
	getDirectoryContents func(context.Context, *repov1.GetDirectoryContentsRequest) (*repov1.GetDirectoryContentsResponse, error)
	getRelatedFiles      func(context.Context, *repov1.GetRelatedFilesRequest) (*repov1.GetRelatedFilesResponse, error)
	searchContent        func(context.Context, *repov1.SearchContentRequest) (*repov1.SearchContentResponse, error)
	deletePath           func(context.Context, *repov1.DeletePathRequest) (*repov1.DeletePathResponse, error)
	saveFileContent      func(context.Context, *repov1.SaveFileContentRequest) (*repov1.SaveFileContentResponse, error)
	discardFiles         func(context.Context, *repov1.DiscardFilesRequest) (*repov1.DiscardFilesResponse, error)
	ignorePath           func(context.Context, *repov1.IgnorePathRequest) (*repov1.IgnorePathResponse, error)
	pushToRemote         func(context.Context, *repov1.PushToRemoteRequest) (*repov1.PushToRemoteResponse, error)
	pullFromRemote       func(context.Context, *repov1.PullFromRemoteRequest) (*repov1.PullFromRemoteResponse, error)
	runUpstreamAction    func(context.Context, *repov1.RunUpstreamActionRequest) (*repov1.RunUpstreamActionResponse, error)
	getGroupingRules     func(context.Context, *repov1.GetGroupingRulesRequest) (*repov1.GroupingRulesResponse, error)
	saveGroupingRules    func(context.Context, *repov1.SaveGroupingRulesRequest) (*repov1.GroupingRulesResponse, error)
	getGitignoreHealth   func(context.Context, *repov1.GetGitignoreHealthRequest) (*repov1.GitignoreHealthResponse, error)
	moveGitignoreEntry   func(context.Context, *repov1.MoveGitignoreEntryRequest) (*repov1.MoveGitignoreEntryResponse, error)
	getTrackedBinaries   func(context.Context, *repov1.GetTrackedBinariesRequest) (*repov1.TrackedBinariesResponse, error)
	untrackBinary        func(context.Context, *repov1.UntrackBinaryRequest) (*repov1.UntrackBinaryResponse, error)
	getPrecommitConfig   func(context.Context, *repov1.GetPrecommitConfigRequest) (*repov1.PrecommitConfigResponse, error)
	savePrecommitConfig  func(context.Context, *repov1.SavePrecommitConfigRequest) (*repov1.PrecommitConfigResponse, error)
	runPrecommit         func(context.Context, *repov1.RunPrecommitRequest) (*repov1.PrecommitRunResponse, error)
	stageFiles           func(context.Context, *repov1.StageFilesRequest) (*repov1.StageFilesResponse, error)
	unstageFiles         func(context.Context, *repov1.UnstageFilesRequest) (*repov1.UnstageFilesResponse, error)
	createCommit         func(context.Context, *repov1.CreateCommitRequest) (*repov1.CreateCommitResponse, error)
	listCredentials      func(context.Context, *repov1.ListCredentialsRequest) (*repov1.CredentialsListResponse, error)
	saveCredential       func(context.Context, *repov1.SaveCredentialRequest) (*repov1.CredentialSaveResponse, error)
	deleteCredential     func(context.Context, *repov1.DeleteCredentialRequest) (*repov1.CredentialDeleteResponse, error)
	testCredential       func(context.Context, *repov1.TestCredentialRequest) (*repov1.CredentialTestResponse, error)
	updateRemoteURL      func(context.Context, *repov1.UpdateRemoteURLRequest) (*repov1.RemoteURLUpdateResponse, error)
	listSSHKeys          func(context.Context, *repov1.ListSSHKeysRequest) (*repov1.SSHListKeysResponse, error)
	generateSSHKey       func(context.Context, *repov1.GenerateSSHKeyRequest) (*repov1.SSHGenerateKeyResponse, error)
	getSSHPublicKey      func(context.Context, *repov1.GetSSHPublicKeyRequest) (*repov1.SSHGetPublicKeyResponse, error)
	testSSHConnection    func(context.Context, *repov1.TestSSHConnectionRequest) (*repov1.SSHTestConnectionResponse, error)
	deleteSSHKey         func(context.Context, *repov1.DeleteSSHKeyRequest) (*repov1.SSHDeleteKeyResponse, error)
}

// Deps wires dependencies.
type Deps struct {
	Service              *repo.Service
	Logger               *log.Logger
	ListRepositories     func(context.Context, *repov1.ListRepositoriesRequest) (*repov1.ListRepositoriesResponse, error)
	GetActiveRepository  func(context.Context, *repov1.GetActiveRepositoryRequest) (*repov1.GetActiveRepositoryResponse, error)
	SetActiveRepository  func(context.Context, *repov1.SetActiveRepositoryRequest) (*repov1.RepoMutationResponse, error)
	OpenRepository       func(context.Context, *repov1.OpenRepositoryRequest) (*repov1.RepoMutationResponse, error)
	CloneRepository      func(context.Context, *repov1.CloneRepositoryRequest) (*repov1.RepoMutationResponse, error)
	RemoveRepository     func(context.Context, *repov1.RemoveRepositoryRequest) (*repov1.RemoveRepositoryResponse, error)
	GetRepoStatus        func(context.Context, *repov1.GetRepoStatusRequest) (*repov1.GetRepoStatusResponse, error)
	GetRepoDiff          func(context.Context, *repov1.GetRepoDiffRequest) (*repov1.GetRepoDiffResponse, error)
	GetRepoGroups        func(context.Context, *repov1.GetRepoGroupsRequest) (*repov1.GetRepoGroupsResponse, error)
	GetSyncStatus        func(context.Context, *repov1.GetSyncStatusRequest) (*repov1.GetSyncStatusResponse, error)
	GetRepoHistory       func(context.Context, *repov1.GetRepoHistoryRequest) (*repov1.GetRepoHistoryResponse, error)
	GetApprovedChanges   func(context.Context, *repov1.GetApprovedChangesRequest) (*repov1.GetApprovedChangesResponse, error)
	GetProvenance        func(context.Context, *repov1.GetProvenanceRequest) (*repov1.GetProvenanceResponse, error)
	GetBlame             func(context.Context, *repov1.GetBlameRequest) (*repov1.GetBlameResponse, error)
	SearchProvenance     func(context.Context, *repov1.SearchProvenanceRequest) (*repov1.SearchProvenanceResponse, error)
	GetFiles             func(context.Context, *repov1.GetFilesRequest) (*repov1.GetFilesResponse, error)
	GetDirectoryContents func(context.Context, *repov1.GetDirectoryContentsRequest) (*repov1.GetDirectoryContentsResponse, error)
	GetRelatedFiles      func(context.Context, *repov1.GetRelatedFilesRequest) (*repov1.GetRelatedFilesResponse, error)
	SearchContent        func(context.Context, *repov1.SearchContentRequest) (*repov1.SearchContentResponse, error)
	DeletePath           func(context.Context, *repov1.DeletePathRequest) (*repov1.DeletePathResponse, error)
	SaveFileContent      func(context.Context, *repov1.SaveFileContentRequest) (*repov1.SaveFileContentResponse, error)
	DiscardFiles         func(context.Context, *repov1.DiscardFilesRequest) (*repov1.DiscardFilesResponse, error)
	IgnorePath           func(context.Context, *repov1.IgnorePathRequest) (*repov1.IgnorePathResponse, error)
	PushToRemote         func(context.Context, *repov1.PushToRemoteRequest) (*repov1.PushToRemoteResponse, error)
	PullFromRemote       func(context.Context, *repov1.PullFromRemoteRequest) (*repov1.PullFromRemoteResponse, error)
	RunUpstreamAction    func(context.Context, *repov1.RunUpstreamActionRequest) (*repov1.RunUpstreamActionResponse, error)
	GetGroupingRules     func(context.Context, *repov1.GetGroupingRulesRequest) (*repov1.GroupingRulesResponse, error)
	SaveGroupingRules    func(context.Context, *repov1.SaveGroupingRulesRequest) (*repov1.GroupingRulesResponse, error)
	GetGitignoreHealth   func(context.Context, *repov1.GetGitignoreHealthRequest) (*repov1.GitignoreHealthResponse, error)
	MoveGitignoreEntry   func(context.Context, *repov1.MoveGitignoreEntryRequest) (*repov1.MoveGitignoreEntryResponse, error)
	GetTrackedBinaries   func(context.Context, *repov1.GetTrackedBinariesRequest) (*repov1.TrackedBinariesResponse, error)
	UntrackBinary        func(context.Context, *repov1.UntrackBinaryRequest) (*repov1.UntrackBinaryResponse, error)
	GetPrecommitConfig   func(context.Context, *repov1.GetPrecommitConfigRequest) (*repov1.PrecommitConfigResponse, error)
	SavePrecommitConfig  func(context.Context, *repov1.SavePrecommitConfigRequest) (*repov1.PrecommitConfigResponse, error)
	RunPrecommit         func(context.Context, *repov1.RunPrecommitRequest) (*repov1.PrecommitRunResponse, error)
	StageFiles           func(context.Context, *repov1.StageFilesRequest) (*repov1.StageFilesResponse, error)
	UnstageFiles         func(context.Context, *repov1.UnstageFilesRequest) (*repov1.UnstageFilesResponse, error)
	CreateCommit         func(context.Context, *repov1.CreateCommitRequest) (*repov1.CreateCommitResponse, error)
	ListCredentials      func(context.Context, *repov1.ListCredentialsRequest) (*repov1.CredentialsListResponse, error)
	SaveCredential       func(context.Context, *repov1.SaveCredentialRequest) (*repov1.CredentialSaveResponse, error)
	DeleteCredential     func(context.Context, *repov1.DeleteCredentialRequest) (*repov1.CredentialDeleteResponse, error)
	TestCredential       func(context.Context, *repov1.TestCredentialRequest) (*repov1.CredentialTestResponse, error)
	UpdateRemoteURL      func(context.Context, *repov1.UpdateRemoteURLRequest) (*repov1.RemoteURLUpdateResponse, error)
	ListSSHKeys          func(context.Context, *repov1.ListSSHKeysRequest) (*repov1.SSHListKeysResponse, error)
	GenerateSSHKey       func(context.Context, *repov1.GenerateSSHKeyRequest) (*repov1.SSHGenerateKeyResponse, error)
	GetSSHPublicKey      func(context.Context, *repov1.GetSSHPublicKeyRequest) (*repov1.SSHGetPublicKeyResponse, error)
	TestSSHConnection    func(context.Context, *repov1.TestSSHConnectionRequest) (*repov1.SSHTestConnectionResponse, error)
	DeleteSSHKey         func(context.Context, *repov1.DeleteSSHKeyRequest) (*repov1.SSHDeleteKeyResponse, error)
}

// NewServer constructs a Server.
func NewServer(d Deps) *Server {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &Server{svc: d.Service, logger: d.Logger, listRepositories: d.ListRepositories, getActiveRepository: d.GetActiveRepository, setActiveRepository: d.SetActiveRepository, openRepository: d.OpenRepository, cloneRepository: d.CloneRepository, removeRepository: d.RemoveRepository, getRepoStatus: d.GetRepoStatus, getRepoDiff: d.GetRepoDiff, getRepoGroups: d.GetRepoGroups, getSyncStatus: d.GetSyncStatus, getRepoHistory: d.GetRepoHistory, getApprovedChanges: d.GetApprovedChanges, getProvenance: d.GetProvenance, getBlame: d.GetBlame, searchProvenance: d.SearchProvenance, getFiles: d.GetFiles, getDirectoryContents: d.GetDirectoryContents, getRelatedFiles: d.GetRelatedFiles, searchContent: d.SearchContent, deletePath: d.DeletePath, saveFileContent: d.SaveFileContent, discardFiles: d.DiscardFiles, ignorePath: d.IgnorePath, pushToRemote: d.PushToRemote, pullFromRemote: d.PullFromRemote, runUpstreamAction: d.RunUpstreamAction, getGroupingRules: d.GetGroupingRules, saveGroupingRules: d.SaveGroupingRules, getGitignoreHealth: d.GetGitignoreHealth, moveGitignoreEntry: d.MoveGitignoreEntry, getTrackedBinaries: d.GetTrackedBinaries, untrackBinary: d.UntrackBinary, getPrecommitConfig: d.GetPrecommitConfig, savePrecommitConfig: d.SavePrecommitConfig, runPrecommit: d.RunPrecommit, stageFiles: d.StageFiles, unstageFiles: d.UnstageFiles, createCommit: d.CreateCommit, listCredentials: d.ListCredentials, saveCredential: d.SaveCredential, deleteCredential: d.DeleteCredential, testCredential: d.TestCredential, updateRemoteURL: d.UpdateRemoteURL, listSSHKeys: d.ListSSHKeys, generateSSHKey: d.GenerateSSHKey, getSSHPublicKey: d.GetSSHPublicKey, testSSHConnection: d.TestSSHConnection, deleteSSHKey: d.DeleteSSHKey}
}

// NewHandler returns the procedure prefix and http.Handler for mounting.
// Extra connect.HandlerOption values (e.g. interceptors wired by main.go)
// are passed through to NewRepoServiceHandler.
func NewHandler(d Deps, opts ...connect.HandlerOption) (string, http.Handler) {
	return repoconnect.NewRepoServiceHandler(NewServer(d), opts...)
}

func (s *Server) ListRepositories(ctx context.Context, req *connect.Request[repov1.ListRepositoriesRequest]) (*connect.Response[repov1.ListRepositoriesResponse], error) {
	if s.listRepositories == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo registry service is not configured"))
	}
	result, err := s.listRepositories(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetActiveRepository(ctx context.Context, req *connect.Request[repov1.GetActiveRepositoryRequest]) (*connect.Response[repov1.GetActiveRepositoryResponse], error) {
	if s.getActiveRepository == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo registry service is not configured"))
	}
	result, err := s.getActiveRepository(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) SetActiveRepository(ctx context.Context, req *connect.Request[repov1.SetActiveRepositoryRequest]) (*connect.Response[repov1.RepoMutationResponse], error) {
	if s.setActiveRepository == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo registry service is not configured"))
	}
	result, err := s.setActiveRepository(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) OpenRepository(ctx context.Context, req *connect.Request[repov1.OpenRepositoryRequest]) (*connect.Response[repov1.RepoMutationResponse], error) {
	if s.openRepository == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo registry service is not configured"))
	}
	result, err := s.openRepository(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) CloneRepository(ctx context.Context, req *connect.Request[repov1.CloneRepositoryRequest]) (*connect.Response[repov1.RepoMutationResponse], error) {
	if s.cloneRepository == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo registry service is not configured"))
	}
	result, err := s.cloneRepository(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) RemoveRepository(ctx context.Context, req *connect.Request[repov1.RemoveRepositoryRequest]) (*connect.Response[repov1.RemoveRepositoryResponse], error) {
	if s.removeRepository == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo registry service is not configured"))
	}
	result, err := s.removeRepository(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetRepoStatus(ctx context.Context, req *connect.Request[repov1.GetRepoStatusRequest]) (*connect.Response[repov1.GetRepoStatusResponse], error) {
	if s.getRepoStatus != nil {
		result, err := s.getRepoStatus(ctx, req.Msg)
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(result), nil
	}
	status, err := s.svc.GetRepoStatus(ctx, req.Msg.GetRepoPath())
	if err != nil {
		connectErr := repo.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			s.logger.Printf("repo.GetRepoStatus: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&repov1.GetRepoStatusResponse{
		Branch:   status.Branch,
		Detached: status.Detached,
		Worktree: &repov1.WorktreeIdentity{
			IsLinkedWorktree:    status.Identity.IsLinkedWorktree,
			CommonRepoRoot:      status.Identity.CommonRepoRoot,
			WorktreeName:        status.Identity.WorktreeName,
			WorktreeHead:        status.Identity.WorktreeHead,
			LinkedWorktreeCount: int32(status.Identity.LinkedWorktreeCount),
		},
	}), nil
}

func (s *Server) GetRepoDiff(ctx context.Context, req *connect.Request[repov1.GetRepoDiffRequest]) (*connect.Response[repov1.GetRepoDiffResponse], error) {
	if s.getRepoDiff == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo diff service is not configured"))
	}
	result, err := s.getRepoDiff(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetRepoGroups(ctx context.Context, req *connect.Request[repov1.GetRepoGroupsRequest]) (*connect.Response[repov1.GetRepoGroupsResponse], error) {
	if s.getRepoGroups == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo groups service is not configured"))
	}
	result, err := s.getRepoGroups(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetSyncStatus(ctx context.Context, req *connect.Request[repov1.GetSyncStatusRequest]) (*connect.Response[repov1.GetSyncStatusResponse], error) {
	if s.getSyncStatus == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo sync status service is not configured"))
	}
	result, err := s.getSyncStatus(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetFiles(ctx context.Context, req *connect.Request[repov1.GetFilesRequest]) (*connect.Response[repov1.GetFilesResponse], error) {
	if s.getFiles == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo files service is not configured"))
	}
	result, err := s.getFiles(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetDirectoryContents(ctx context.Context, req *connect.Request[repov1.GetDirectoryContentsRequest]) (*connect.Response[repov1.GetDirectoryContentsResponse], error) {
	if s.getDirectoryContents == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo directory service is not configured"))
	}
	result, err := s.getDirectoryContents(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetRelatedFiles(ctx context.Context, req *connect.Request[repov1.GetRelatedFilesRequest]) (*connect.Response[repov1.GetRelatedFilesResponse], error) {
	if s.getRelatedFiles == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo related-files service is not configured"))
	}
	result, err := s.getRelatedFiles(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) SearchContent(ctx context.Context, req *connect.Request[repov1.SearchContentRequest]) (*connect.Response[repov1.SearchContentResponse], error) {
	if s.searchContent == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo content-search service is not configured"))
	}
	result, err := s.searchContent(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) DeletePath(ctx context.Context, req *connect.Request[repov1.DeletePathRequest]) (*connect.Response[repov1.DeletePathResponse], error) {
	if s.deletePath == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo delete service is not configured"))
	}
	result, err := s.deletePath(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) SaveFileContent(ctx context.Context, req *connect.Request[repov1.SaveFileContentRequest]) (*connect.Response[repov1.SaveFileContentResponse], error) {
	if s.saveFileContent == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo file-content service is not configured"))
	}
	result, err := s.saveFileContent(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) DiscardFiles(ctx context.Context, req *connect.Request[repov1.DiscardFilesRequest]) (*connect.Response[repov1.DiscardFilesResponse], error) {
	if s.discardFiles == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo discard service is not configured"))
	}
	result, err := s.discardFiles(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) IgnorePath(ctx context.Context, req *connect.Request[repov1.IgnorePathRequest]) (*connect.Response[repov1.IgnorePathResponse], error) {
	if s.ignorePath == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo ignore service is not configured"))
	}
	result, err := s.ignorePath(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) PushToRemote(ctx context.Context, req *connect.Request[repov1.PushToRemoteRequest]) (*connect.Response[repov1.PushToRemoteResponse], error) {
	if s.pushToRemote == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo push service is not configured"))
	}
	result, err := s.pushToRemote(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) PullFromRemote(ctx context.Context, req *connect.Request[repov1.PullFromRemoteRequest]) (*connect.Response[repov1.PullFromRemoteResponse], error) {
	if s.pullFromRemote == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo pull service is not configured"))
	}
	result, err := s.pullFromRemote(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) RunUpstreamAction(ctx context.Context, req *connect.Request[repov1.RunUpstreamActionRequest]) (*connect.Response[repov1.RunUpstreamActionResponse], error) {
	if s.runUpstreamAction == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo upstream service is not configured"))
	}
	result, err := s.runUpstreamAction(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetGroupingRules(ctx context.Context, req *connect.Request[repov1.GetGroupingRulesRequest]) (*connect.Response[repov1.GroupingRulesResponse], error) {
	if s.getGroupingRules == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo grouping rules service is not configured"))
	}
	result, err := s.getGroupingRules(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) SaveGroupingRules(ctx context.Context, req *connect.Request[repov1.SaveGroupingRulesRequest]) (*connect.Response[repov1.GroupingRulesResponse], error) {
	if s.saveGroupingRules == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo grouping rules service is not configured"))
	}
	result, err := s.saveGroupingRules(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetGitignoreHealth(ctx context.Context, req *connect.Request[repov1.GetGitignoreHealthRequest]) (*connect.Response[repov1.GitignoreHealthResponse], error) {
	if s.getGitignoreHealth == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo gitignore health service is not configured"))
	}
	result, err := s.getGitignoreHealth(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) MoveGitignoreEntry(ctx context.Context, req *connect.Request[repov1.MoveGitignoreEntryRequest]) (*connect.Response[repov1.MoveGitignoreEntryResponse], error) {
	if s.moveGitignoreEntry == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo gitignore move service is not configured"))
	}
	result, err := s.moveGitignoreEntry(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetTrackedBinaries(ctx context.Context, req *connect.Request[repov1.GetTrackedBinariesRequest]) (*connect.Response[repov1.TrackedBinariesResponse], error) {
	if s.getTrackedBinaries == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo tracked binaries service is not configured"))
	}
	result, err := s.getTrackedBinaries(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) UntrackBinary(ctx context.Context, req *connect.Request[repov1.UntrackBinaryRequest]) (*connect.Response[repov1.UntrackBinaryResponse], error) {
	if s.untrackBinary == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo untrack binary service is not configured"))
	}
	result, err := s.untrackBinary(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetPrecommitConfig(ctx context.Context, req *connect.Request[repov1.GetPrecommitConfigRequest]) (*connect.Response[repov1.PrecommitConfigResponse], error) {
	if s.getPrecommitConfig == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo precommit service is not configured"))
	}
	result, err := s.getPrecommitConfig(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) SavePrecommitConfig(ctx context.Context, req *connect.Request[repov1.SavePrecommitConfigRequest]) (*connect.Response[repov1.PrecommitConfigResponse], error) {
	if s.savePrecommitConfig == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo precommit service is not configured"))
	}
	result, err := s.savePrecommitConfig(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) RunPrecommit(ctx context.Context, req *connect.Request[repov1.RunPrecommitRequest]) (*connect.Response[repov1.PrecommitRunResponse], error) {
	if s.runPrecommit == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo precommit service is not configured"))
	}
	result, err := s.runPrecommit(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) CreateCommit(ctx context.Context, req *connect.Request[repov1.CreateCommitRequest]) (*connect.Response[repov1.CreateCommitResponse], error) {
	if s.createCommit == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo commit service is not configured"))
	}
	result, err := s.createCommit(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) StageFiles(ctx context.Context, req *connect.Request[repov1.StageFilesRequest]) (*connect.Response[repov1.StageFilesResponse], error) {
	if s.stageFiles == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo stage service is not configured"))
	}
	result, err := s.stageFiles(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) UnstageFiles(ctx context.Context, req *connect.Request[repov1.UnstageFilesRequest]) (*connect.Response[repov1.UnstageFilesResponse], error) {
	if s.unstageFiles == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo unstage service is not configured"))
	}
	result, err := s.unstageFiles(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) ListCredentials(ctx context.Context, req *connect.Request[repov1.ListCredentialsRequest]) (*connect.Response[repov1.CredentialsListResponse], error) {
	if s.listCredentials == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo credentials service is not configured"))
	}
	result, err := s.listCredentials(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) SaveCredential(ctx context.Context, req *connect.Request[repov1.SaveCredentialRequest]) (*connect.Response[repov1.CredentialSaveResponse], error) {
	if s.saveCredential == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo credential save service is not configured"))
	}
	result, err := s.saveCredential(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) DeleteCredential(ctx context.Context, req *connect.Request[repov1.DeleteCredentialRequest]) (*connect.Response[repov1.CredentialDeleteResponse], error) {
	if s.deleteCredential == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo credential delete service is not configured"))
	}
	result, err := s.deleteCredential(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) TestCredential(ctx context.Context, req *connect.Request[repov1.TestCredentialRequest]) (*connect.Response[repov1.CredentialTestResponse], error) {
	if s.testCredential == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo credential test service is not configured"))
	}
	result, err := s.testCredential(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) UpdateRemoteURL(ctx context.Context, req *connect.Request[repov1.UpdateRemoteURLRequest]) (*connect.Response[repov1.RemoteURLUpdateResponse], error) {
	if s.updateRemoteURL == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo remote URL service is not configured"))
	}
	result, err := s.updateRemoteURL(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) ListSSHKeys(ctx context.Context, req *connect.Request[repov1.ListSSHKeysRequest]) (*connect.Response[repov1.SSHListKeysResponse], error) {
	if s.listSSHKeys == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo SSH key service is not configured"))
	}
	result, err := s.listSSHKeys(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GenerateSSHKey(ctx context.Context, req *connect.Request[repov1.GenerateSSHKeyRequest]) (*connect.Response[repov1.SSHGenerateKeyResponse], error) {
	if s.generateSSHKey == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo SSH key generation service is not configured"))
	}
	result, err := s.generateSSHKey(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetSSHPublicKey(ctx context.Context, req *connect.Request[repov1.GetSSHPublicKeyRequest]) (*connect.Response[repov1.SSHGetPublicKeyResponse], error) {
	if s.getSSHPublicKey == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo SSH public key service is not configured"))
	}
	result, err := s.getSSHPublicKey(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) TestSSHConnection(ctx context.Context, req *connect.Request[repov1.TestSSHConnectionRequest]) (*connect.Response[repov1.SSHTestConnectionResponse], error) {
	if s.testSSHConnection == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo SSH connection service is not configured"))
	}
	result, err := s.testSSHConnection(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) DeleteSSHKey(ctx context.Context, req *connect.Request[repov1.DeleteSSHKeyRequest]) (*connect.Response[repov1.SSHDeleteKeyResponse], error) {
	if s.deleteSSHKey == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo SSH key deletion service is not configured"))
	}
	result, err := s.deleteSSHKey(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetRepoHistory(ctx context.Context, req *connect.Request[repov1.GetRepoHistoryRequest]) (*connect.Response[repov1.GetRepoHistoryResponse], error) {
	if s.getRepoHistory == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("repo history service is not configured"))
	}
	result, err := s.getRepoHistory(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetApprovedChanges(ctx context.Context, req *connect.Request[repov1.GetApprovedChangesRequest]) (*connect.Response[repov1.GetApprovedChangesResponse], error) {
	if s.getApprovedChanges == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("approved changes service is not configured"))
	}
	result, err := s.getApprovedChanges(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetProvenance(ctx context.Context, req *connect.Request[repov1.GetProvenanceRequest]) (*connect.Response[repov1.GetProvenanceResponse], error) {
	if s.getProvenance == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("provenance service is not configured"))
	}
	result, err := s.getProvenance(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) GetBlame(ctx context.Context, req *connect.Request[repov1.GetBlameRequest]) (*connect.Response[repov1.GetBlameResponse], error) {
	if s.getBlame == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("blame service is not configured"))
	}
	result, err := s.getBlame(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (s *Server) SearchProvenance(ctx context.Context, req *connect.Request[repov1.SearchProvenanceRequest]) (*connect.Response[repov1.SearchProvenanceResponse], error) {
	if s.searchProvenance == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("provenance search service is not configured"))
	}
	result, err := s.searchProvenance(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}
