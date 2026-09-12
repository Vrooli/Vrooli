package main

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/ssh"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
)

func (s *Server) listCredentialsConnect(ctx context.Context, req *repov1.ListCredentialsRequest) (*repov1.CredentialsListResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	result, err := ListCredentials(ctx, CredentialsDeps{Git: s.git, RepoDir: resolved.Path, Store: s.credStore})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	credentials := make([]*repov1.Credential, 0, len(result.Credentials))
	for _, credential := range result.Credentials {
		credentials = append(credentials, credentialProto(credential))
	}
	return &repov1.CredentialsListResponse{Credentials: credentials, Timestamp: result.Timestamp.UTC().Format(time.RFC3339Nano)}, nil
}

func (s *Server) saveCredentialConnect(ctx context.Context, req *repov1.SaveCredentialRequest) (*repov1.CredentialSaveResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.credentials", "save credential", func(writeCtx context.Context, repoPath string) (any, error) {
		return SaveCredential(writeCtx, CredentialsDeps{Git: s.git, RepoDir: repoPath, Store: s.credStore}, CredentialSaveRequest{
			Remote: req.GetRemote(), URL: req.GetUrl(), Username: req.GetUsername(), Token: req.GetToken(), SSHKeyPath: req.GetSshKeyPath(),
		})
	})
	if err != nil {
		return nil, err
	}
	value, _ := result.(*CredentialSaveResponse)
	if value == nil {
		return &repov1.CredentialSaveResponse{}, nil
	}
	response := &repov1.CredentialSaveResponse{Success: value.Success, Error: value.Error, Timestamp: value.Timestamp.UTC().Format(time.RFC3339Nano)}
	if value.Credential != nil {
		response.Credential = credentialProto(*value.Credential)
	}
	return response, nil
}

func (s *Server) deleteCredentialConnect(ctx context.Context, req *repov1.DeleteCredentialRequest) (*repov1.CredentialDeleteResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.credentials", "delete credential", func(writeCtx context.Context, repoPath string) (any, error) {
		return DeleteCredential(writeCtx, CredentialsDeps{Git: s.git, RepoDir: repoPath, Store: s.credStore}, CredentialDeleteRequest{ID: req.GetId()})
	})
	if err != nil {
		return nil, err
	}
	value, _ := result.(*CredentialDeleteResponse)
	if value == nil {
		return &repov1.CredentialDeleteResponse{}, nil
	}
	return &repov1.CredentialDeleteResponse{Success: value.Success, Error: value.Error, Timestamp: value.Timestamp.UTC().Format(time.RFC3339Nano)}, nil
}

func (s *Server) testCredentialConnect(ctx context.Context, req *repov1.TestCredentialRequest) (*repov1.CredentialTestResponse, error) {
	resolved, err := s.resolveRepoForConnect(ctx, req.GetRepositoryId(), "")
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	result, err := TestCredential(ctx, CredentialsDeps{Git: s.git, RepoDir: resolved.Path, Store: s.credStore}, CredentialTestRequest{Remote: req.GetRemote(), UseStored: req.GetUseStored()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &repov1.CredentialTestResponse{Success: result.Success, Reachable: result.Reachable, Authorized: result.Authorized, Error: result.Error, Timestamp: result.Timestamp.UTC().Format(time.RFC3339Nano)}, nil
}

func (s *Server) updateRemoteURLConnect(ctx context.Context, req *repov1.UpdateRemoteURLRequest) (*repov1.RemoteURLUpdateResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "repo.remote.url", "update remote URL", func(writeCtx context.Context, repoPath string) (any, error) {
		return UpdateRemoteURL(writeCtx, CredentialsDeps{Git: s.git, RepoDir: repoPath}, RemoteURLUpdateRequest{Remote: req.GetRemote(), URL: req.GetUrl()})
	})
	if err != nil {
		return nil, err
	}
	value, _ := result.(*RemoteURLUpdateResponse)
	if value == nil {
		return &repov1.RemoteURLUpdateResponse{}, nil
	}
	return &repov1.RemoteURLUpdateResponse{Success: value.Success, OldUrl: value.OldURL, NewUrl: value.NewURL, Error: value.Error, Timestamp: value.Timestamp.UTC().Format(time.RFC3339Nano)}, nil
}

func (s *Server) listSSHKeysConnect(ctx context.Context, _ *repov1.ListSSHKeysRequest) (*repov1.SSHListKeysResponse, error) {
	result, err := ssh.ListKeys(ctx, s.sshDeps)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	keys := make([]*repov1.SSHKeyInfo, 0, len(result.Keys))
	for _, key := range result.Keys {
		keys = append(keys, sshKeyProto(key))
	}
	return &repov1.SSHListKeysResponse{Keys: keys, SshDir: result.SSHDir, Timestamp: result.Timestamp}, nil
}

func (s *Server) generateSSHKeyConnect(ctx context.Context, req *repov1.GenerateSSHKeyRequest) (*repov1.SSHGenerateKeyResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "ssh.key.generate", "generate SSH key", func(writeCtx context.Context, _ string) (any, error) {
		return ssh.GenerateKeyService(writeCtx, s.sshDeps, ssh.GenerateKeyRequest{Type: ssh.KeyType(req.GetType()), Bits: int(req.GetBits()), Comment: req.GetComment(), Filename: req.GetFilename()})
	})
	if err != nil {
		return nil, err
	}
	value, _ := result.(*ssh.GenerateKeyResponse)
	if value == nil {
		return &repov1.SSHGenerateKeyResponse{}, nil
	}
	response := &repov1.SSHGenerateKeyResponse{Success: value.Success, PublicKey: value.PublicKey, Error: value.Error, Timestamp: value.Timestamp}
	if value.Key.Path != "" {
		response.Key = sshKeyProto(value.Key)
	}
	return response, nil
}

func (s *Server) getSSHPublicKeyConnect(ctx context.Context, req *repov1.GetSSHPublicKeyRequest) (*repov1.SSHGetPublicKeyResponse, error) {
	result, err := ssh.GetPublicKeyService(ctx, s.sshDeps, ssh.GetPublicKeyRequest{KeyPath: req.GetKeyPath()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &repov1.SSHGetPublicKeyResponse{Success: result.Success, PublicKey: result.PublicKey, Fingerprint: result.Fingerprint, Error: result.Error, Timestamp: result.Timestamp}, nil
}

func (s *Server) testSSHConnectionConnect(ctx context.Context, req *repov1.TestSSHConnectionRequest) (*repov1.SSHTestConnectionResponse, error) {
	result, err := ssh.TestGitHubConnectionService(ctx, s.sshDeps, ssh.TestConnectionRequest{KeyPath: req.GetKeyPath()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &repov1.SSHTestConnectionResponse{Success: result.Success, Status: result.Status, Message: result.Message, Hint: result.Hint, GithubUser: result.GitHubUser, Fingerprint: result.Fingerprint, LatencyMs: result.LatencyMs, Timestamp: result.Timestamp}, nil
}

func (s *Server) deleteSSHKeyConnect(ctx context.Context, req *repov1.DeleteSSHKeyRequest) (*repov1.SSHDeleteKeyResponse, error) {
	result, err := s.runFileMutation(ctx, req.GetRepositoryId(), req.GetIntentId(), "ssh.key.delete", "delete SSH key", func(writeCtx context.Context, _ string) (any, error) {
		return ssh.DeleteKeyService(writeCtx, s.sshDeps, ssh.DeleteKeyRequest{KeyPath: req.GetKeyPath()})
	})
	if err != nil {
		return nil, err
	}
	value, _ := result.(*ssh.DeleteKeyResponse)
	if value == nil {
		return &repov1.SSHDeleteKeyResponse{}, nil
	}
	return &repov1.SSHDeleteKeyResponse{Success: value.Success, Message: value.Message, Error: value.Error, PrivateDeleted: value.PrivateDeleted, PublicDeleted: value.PublicDeleted, Timestamp: value.Timestamp}, nil
}

func credentialProto(value Credential) *repov1.Credential {
	return &repov1.Credential{Id: value.ID, Remote: value.Remote, Url: value.URL, Type: string(value.Type), Username: value.Username, TokenMasked: value.TokenMasked, SshKeyPath: value.SSHKeyPath, IsConfigured: value.IsConfigured, CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: value.UpdatedAt.UTC().Format(time.RFC3339Nano)}
}

func sshKeyProto(value ssh.KeyInfo) *repov1.SSHKeyInfo {
	return &repov1.SSHKeyInfo{Path: value.Path, Filename: value.Filename, Type: string(value.Type), Bits: int32(value.Bits), Fingerprint: value.Fingerprint, Comment: value.Comment, CreatedAt: value.CreatedAt, HasPublic: value.HasPublic}
}
