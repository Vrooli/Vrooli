package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git-control-tower/internal/policygate"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

const mutationOperationCommit = "repo.commit"

type AuthorityStatusResponse struct {
	Authenticated bool     `json:"authenticated"`
	PrincipalID   string   `json:"principal_id,omitempty"`
	Email         string   `json:"email,omitempty"`
	Realm         string   `json:"realm,omitempty"`
	CallerKind    string   `json:"caller_kind"`
	CanMutate     bool     `json:"can_mutate"`
	Reason        string   `json:"reason,omitempty"`
	Capabilities  []string `json:"capabilities,omitempty"`
	AuthSource    string   `json:"auth_source,omitempty"`
	AuthState     string   `json:"auth_state,omitempty"`
	RecoveryURL   string   `json:"recovery_url,omitempty"`
	FailureClass  string   `json:"failure_class,omitempty"`
	AuthSources   []string `json:"auth_sources,omitempty"`
}

type MutationPreviewRequest struct {
	RepositoryID   string `json:"repository_id,omitempty"`
	Operation      string `json:"operation,omitempty"`
	SubjectContext string `json:"subject_context,omitempty"`
}

type MutationPreviewResponse struct {
	RepositoryID     string    `json:"repository_id"`
	RepositoryPath   string    `json:"repository_path"`
	Operation        string    `json:"operation"`
	Branch           string    `json:"branch"`
	ExpectedRevision string    `json:"expected_revision"`
	SubjectDigest    string    `json:"subject_digest"`
	StagedFiles      []string  `json:"staged_files"`
	FileCount        int       `json:"file_count"`
	GeneratedAt      time.Time `json:"generated_at"`
	SubjectContext   string    `json:"subject_context,omitempty"`
}

type MutationIntentRequest struct {
	RepositoryID     string `json:"repository_id"`
	Operation        string `json:"operation"`
	ExpectedRevision string `json:"expected_revision"`
	SubjectDigest    string `json:"subject_digest"`
	SubjectContext   string `json:"subject_context,omitempty"`
}

type MutationIntentResponse struct {
	IntentID         string    `json:"intent_id"`
	PrincipalID      string    `json:"principal_id"`
	RepositoryID     string    `json:"repository_id"`
	Operation        string    `json:"operation"`
	ExpectedRevision string    `json:"expected_revision"`
	SubjectDigest    string    `json:"subject_digest"`
	ExpiresAt        time.Time `json:"expires_at"`
	SingleUse        bool      `json:"single_use"`
}

func (s *Server) handleAuthorityStatus(w http.ResponseWriter, r *http.Request) {
	resp := NewResponse(w)
	principal, ok := policygate.PrincipalFromContext(r.Context())
	status := authorityStatusFromContext(r.Context())
	if !ok {
		resp.OK(status)
		return
	}
	status.Authenticated = true
	status.PrincipalID, status.Email, status.Realm = principal.Subject, principal.Email, principal.Realm
	status.CallerKind = principal.Kind.String()
	status.CanMutate = principal.Kind == cliutil.CallerKindHuman
	if status.CanMutate {
		status.Capabilities = []string{"repo.commit"}
	} else {
		status.Reason = "agent callers may inspect and prepare changes, but cannot mutate repositories"
	}
	resp.OK(status)
}

func authorityStatusFromContext(ctx context.Context) AuthorityStatusResponse {
	status := AuthorityStatusResponse{CallerKind: cliutil.CallerKindUnknown.String()}
	if shared, ok := coreidentity.StatusFromContext(ctx); ok {
		status.AuthSource = string(shared.Source)
		status.AuthState = string(shared.State)
		status.RecoveryURL = shared.RecoveryURL
		status.FailureClass = string(shared.FailureClass)
		for _, source := range shared.ProviderSources {
			status.AuthSources = append(status.AuthSources, string(source))
		}
		switch shared.State {
		case coreidentity.StateExpired:
			status.Reason = "Cloudflare Access session expired; use the reauthentication action"
		case coreidentity.StateConflict:
			status.Reason = "multiple authentication providers identified different users; sign in again"
		case coreidentity.StateService:
			status.Reason = "service authentication cannot authorize human repository mutations"
		case coreidentity.StateAgent:
			status.Reason = "agent callers may inspect and prepare changes, but cannot mutate repositories"
		case coreidentity.StateError:
			status.Reason = "authentication failed or is unavailable; use the configured recovery action"
		default:
			status.Reason = "authenticate through the configured provider to authorize repository mutations"
		}
	}
	if failure := policygate.AuthFailureFromContext(ctx); failure != nil && status.Reason == "" {
		status.Reason = "authentication failed or is unavailable; sign in again"
	}
	return status
}

func (s *Server) handleMutationPreview(w http.ResponseWriter, r *http.Request) {
	resp := NewResponse(w)
	var req MutationPreviewRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}
	preview, err := s.prepareMutationWithContext(r.Context(), strings.TrimSpace(req.RepositoryID), strings.TrimSpace(req.Operation), strings.TrimSpace(req.SubjectContext))
	if err != nil {
		s.writeMutationError(resp, err)
		return
	}
	resp.OK(preview)
}

func (s *Server) handleMutationIntent(w http.ResponseWriter, r *http.Request) {
	resp := NewResponse(w)
	principal, ok := policygate.PrincipalFromContext(r.Context())
	if !ok {
		resp.Error(http.StatusUnauthorized, "authenticate through the configured provider before confirming a mutation")
		return
	}
	if principal.Kind != cliutil.CallerKindHuman {
		resp.Error(http.StatusForbidden, "agent callers cannot issue human mutation intents")
		return
	}
	var req MutationIntentRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}
	operation := strings.TrimSpace(req.Operation)
	if operation == "" {
		operation = mutationOperationCommit
	}
	preview, err := s.prepareMutationWithContext(r.Context(), req.RepositoryID, operation, strings.TrimSpace(req.SubjectContext))
	if err != nil {
		s.writeMutationError(resp, err)
		return
	}
	if preview.ExpectedRevision != strings.TrimSpace(req.ExpectedRevision) || preview.SubjectDigest != strings.TrimSpace(req.SubjectDigest) {
		resp.Error(http.StatusConflict, "repository changed; review the exact mutation preview again")
		return
	}
	intent, err := s.intentService.Issue(r.Context(), principal, policygate.IntentRequest{RepositoryID: preview.RepositoryID, Operation: operation, ExpectedRevision: preview.ExpectedRevision, SubjectDigest: preview.SubjectDigest, PolicyVersion: "gct-human-control-v1"})
	if err != nil {
		s.writeMutationError(resp, err)
		return
	}
	resp.OK(MutationIntentResponse{IntentID: intent.ID, PrincipalID: intent.PrincipalID, RepositoryID: intent.RepositoryID, Operation: intent.Operation, ExpectedRevision: intent.ExpectedRevision, SubjectDigest: intent.SubjectDigest, ExpiresAt: intent.ExpiresAt, SingleUse: true})
}

func (s *Server) prepareMutation(ctx context.Context, repositoryID, operation string) (MutationPreviewResponse, error) {
	return s.prepareMutationWithContext(ctx, repositoryID, operation, "")
}

func (s *Server) prepareMutationWithContext(ctx context.Context, repositoryID, operation, subjectContext string) (MutationPreviewResponse, error) {
	if operation == "" {
		operation = mutationOperationCommit
	}
	repo, err := s.mutationRepo(ctx, repositoryID)
	if err != nil {
		return MutationPreviewResponse{}, err
	}
	status, err := GetRepoStatus(ctx, RepoStatusDeps{Git: s.git, RepoDir: repo.Path, ConfigCache: s.configCache, StatusCache: NewRepoStatusCache(0)})
	if err != nil {
		return MutationPreviewResponse{}, err
	}
	revision := strings.TrimSpace(status.Branch.OID)
	if revision == "" {
		output, revErr := s.git.RevParse(ctx, repo.Path, "HEAD")
		if revErr == nil {
			revision = strings.TrimSpace(string(output))
		}
	}
	if revision == "" {
		revision = "empty-repository"
	}
	diff, err := s.git.Diff(ctx, repo.Path, "", true)
	if err != nil {
		return MutationPreviewResponse{}, err
	}
	digestInput := append([]byte(revision+"\x00"+subjectContext+"\x00"), diff...)
	hash := sha256.Sum256(digestInput)
	return MutationPreviewResponse{RepositoryID: repositoryIDFor(repo), RepositoryPath: repo.Path, Operation: operation, Branch: status.Branch.Head, ExpectedRevision: revision, SubjectDigest: hex.EncodeToString(hash[:]), StagedFiles: append([]string(nil), status.Files.Staged...), FileCount: len(status.Files.Staged), GeneratedAt: time.Now().UTC(), SubjectContext: subjectContext}, nil
}

func (s *Server) mutationRepo(ctx context.Context, repositoryID string) (*RepoRecord, error) {
	if strings.TrimSpace(repositoryID) != "" {
		id, err := strconv.ParseInt(strings.TrimSpace(repositoryID), 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("repository id is invalid")
		}
		repo, err := s.repos.store.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("repository not found: %w", err)
		}
		return repo, nil
	}
	repo, err := s.repos.GetActive(ctx)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, fmt.Errorf("no active repository")
	}
	return repo, nil
}

func repositoryIDFor(repo *RepoRecord) string {
	if repo == nil {
		return ""
	}
	if repo.ID > 0 {
		return strconv.FormatInt(repo.ID, 10)
	}
	return repo.Path
}

func (s *Server) writeMutationError(resp *HTTPResponse, err error) {
	switch {
	case errors.Is(err, policygate.ErrMutationAuthentication), errors.Is(err, policygate.ErrIntentRequired):
		resp.Error(http.StatusUnauthorized, err.Error())
	case errors.Is(err, policygate.ErrIntentExpired), errors.Is(err, policygate.ErrIntentReplay), errors.Is(err, policygate.ErrIntentMismatch):
		resp.Error(http.StatusConflict, err.Error())
	case errors.Is(err, policygate.ErrMutationPermission):
		resp.Error(http.StatusForbidden, err.Error())
	default:
		resp.InternalError(err.Error())
	}
}
