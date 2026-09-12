package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/backup"
	"scenario-to-cloud/bundle"
	"scenario-to-cloud/closure"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/vps"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/packages/recoverypoint"
)

// BackupSchemaVersion is the response schema version clients negotiate on.
const BackupSchemaVersion = "1"

func (s *Server) backupService() (*backup.Service, error) {
	if s == nil {
		return nil, apierrors.Internal("recovery service unavailable", nil)
	}
	return s.backupSvc, s.backupErr
}

// newDefaultBackupService binds the repository, the recovery-point store
// beside the bundle store, the credential authority (recovery keys and
// PostgreSQL credentials), the built-in providers, data-backup-manager as the
// backup owner and the frozen certification budgets.
func (s *Server) newDefaultBackupService() (*backup.Service, error) {
	if s.repo == nil {
		return nil, apierrors.New(apierrors.CodeInternal, "recovery service requires the deployment repository")
	}
	storeDir, err := s.localBundleDir(context.Background())
	if err != nil {
		return nil, err
	}
	budgets := backup.Budgets{}
	if repoRoot, rootErr := bundle.FindRepoRootFromCWD(); rootErr == nil {
		if loaded, loadErr := backup.LoadBudgets(filepath.Join(repoRoot, "scenarios", "scenario-to-cloud", "certification", "budgets.json")); loadErr == nil {
			budgets = loaded
		}
	}
	return &backup.Service{
		Repo: s.repo, Root: filepath.Join(filepath.Dir(storeDir), "recovery-points"),
		Keys: backup.AuthorityKeys{}, Providers: backup.DefaultProviders(),
		Boundary:  recoverypoint.ApplicationHooks{Runner: backup.OSRunner{}},
		Registrar: backup.DBMRegistrar{}, Budgets: budgets,
	}, nil
}

func (s *Server) localBundleDir(ctx context.Context) (string, error) {
	if s.fileRoots == nil {
		return bundle.GetLocalBundlesDir()
	}
	dir, err := fileRootPath(ctx, s.fileRoots, storage.ClassCache, "bundles")
	if err != nil {
		return "", fmt.Errorf("resolve routed bundle directory: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create routed bundle directory: %w", err)
	}
	return dir, nil
}

// registerBackupRoutes mounts the recovery-point surface. Mount line for
// main.go setupRoutes (owned by the operations agent):
//
//	s.registerBackupRoutes(api)
func (s *Server) registerBackupRoutes(api *mux.Router) {
	api.HandleFunc("/deployments/{id}/recovery-points", s.handleCaptureRecoveryPoint).Methods("POST")
	api.HandleFunc("/deployments/{id}/recovery-points", s.handleListRecoveryPoints).Methods("GET")
	api.HandleFunc("/deployments/{id}/recovery-points/rollback-admission", s.handleRollbackAdmission).Methods("POST")
	api.HandleFunc("/deployments/{id}/recovery-points/prune", s.handlePruneRecoveryPoints).Methods("POST")
	api.HandleFunc("/deployments/{id}/recovery-points/{rp}/restore", s.handleRestoreRecoveryPoint).Methods("POST")
	api.HandleFunc("/deployments/{id}/recovery-points/{rp}/verify", s.handleVerifyRecoveryPoint).Methods("GET")
}

// RecoveryPointCaptureRequest is the REST capture body. Bindings default to
// the deployment closure's declared persistent data.
type RecoveryPointCaptureRequest struct {
	backup.CaptureRequest
}

// RollbackAdmissionRequest is the REST rollback-admission body. The schema
// strategy and code-rollback contract default to the closure's declaration
// for the deployed scenario.
type RollbackAdmissionRequest struct {
	CurrentSchema  string   `json:"current_schema"`
	TargetSchema   string   `json:"target_schema"`
	SchemaStrategy string   `json:"schema_strategy,omitempty"`
	CodeRollback   string   `json:"code_rollback,omitempty"`
	ReadableBy     []string `json:"readable_by,omitempty"`
}

// PruneRequest is the REST prune body: the retention policy and the live
// references that protect points.
type PruneRequest struct {
	KeepLast      int   `json:"keep_last"`
	MaxAgeSeconds int64 `json:"max_age_seconds"`
	backup.References
}

// deploymentClosure derives the closure for a deployment (nil with a reason
// when the closure service is unavailable).
func (s *Server) deploymentClosure(ctx context.Context, dctx *DeploymentContext) (*domain.Closure, string) {
	svc, err := s.closureService()
	if err != nil || svc == nil {
		return nil, "closure service not configured"
	}
	derived, derr := svc.Derive(ctx, closure.Request{
		ScenarioID: dctx.Manifest.Scenario.ID, Environment: dctx.Deployment.Environment,
		OS: svc.DefaultPlatform.OS, Arch: svc.DefaultPlatform.Arch, Scope: string(closure.ScopeBundle),
	})
	if derr != nil {
		return nil, derr.Error()
	}
	return &derived, ""
}

// CaptureRecoveryPoint is the transport-neutral capture path shared by REST
// and Connect.
func (s *Server) CaptureRecoveryPoint(ctx context.Context, deploymentID string, req backup.CaptureRequest) (*domain.RecoveryPoint, error) {
	svc, err := s.backupService()
	if err != nil {
		return nil, apierrors.Internal("recovery service unavailable", err)
	}
	dctx, derr := s.loadDeploymentContext(ctx, deploymentID)
	if derr != nil {
		return nil, derr
	}
	req.DeploymentID = deploymentID
	if req.ReleaseDigest == "" && dctx.Deployment.BundleSHA256 != nil {
		req.ReleaseDigest = "sha256:" + *dctx.Deployment.BundleSHA256
	}
	if req.ConfigurationDigest == "" {
		if digest, digestErr := vps.ConfigurationDigest(dctx.Manifest); digestErr == nil {
			req.ConfigurationDigest = digest
		}
	}
	if len(req.Bindings) == 0 {
		derived, reason := s.deploymentClosure(ctx, dctx)
		if derived == nil {
			return nil, apierrors.New(apierrors.CodeClosureUnavailable, "cannot resolve data bindings: "+reason)
		}
		workdir := domain.DefaultVPSWorkdir
		if dctx.Manifest.Target.VPS != nil && strings.TrimSpace(dctx.Manifest.Target.VPS.Workdir) != "" {
			workdir = dctx.Manifest.Target.VPS.Workdir
		}
		bindings, berr := backup.ResolveBindings(derived, backup.BindingInputs{Workdir: workdir})
		if berr != nil {
			return nil, berr
		}
		req.Bindings = bindings
	}
	if svc.Owner == "" {
		svc.Owner = dctx.Manifest.Scenario.ID
	}
	return svc.Capture(ctx, req)
}

func (s *Server) handleCaptureRecoveryPoint(w http.ResponseWriter, r *http.Request) {
	var req RecoveryPointCaptureRequest
	if !httputil.DecodeRequestBody(w, r, &req) {
		return
	}
	rp, err := s.CaptureRecoveryPoint(r.Context(), mux.Vars(r)["id"], req.CaptureRequest)
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, map[string]any{"schema_version": BackupSchemaVersion, "recovery_point": rp})
}

// ListRecoveryPoints is the transport-neutral list path.
func (s *Server) ListRecoveryPoints(ctx context.Context, deploymentID string) ([]domain.RecoveryPoint, error) {
	svc, err := s.backupService()
	if err != nil {
		return nil, apierrors.Internal("recovery service unavailable", err)
	}
	if _, derr := s.loadDeploymentContext(ctx, deploymentID); derr != nil {
		return nil, derr
	}
	return svc.Repo.ListRecoveryPoints(ctx, deploymentID)
}

func (s *Server) handleListRecoveryPoints(w http.ResponseWriter, r *http.Request) {
	points, err := s.ListRecoveryPoints(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"schema_version": BackupSchemaVersion, "recovery_points": points})
}

// RestoreRecoveryPoint is the transport-neutral restore path.
func (s *Server) RestoreRecoveryPoint(ctx context.Context, deploymentID, recoveryPointID string, req backup.RestoreRequest) (*domain.RestoreReceipt, error) {
	svc, err := s.backupService()
	if err != nil {
		return nil, apierrors.Internal("recovery service unavailable", err)
	}
	if _, derr := s.loadDeploymentContext(ctx, deploymentID); derr != nil {
		return nil, derr
	}
	req.DeploymentID, req.RecoveryPointID = deploymentID, recoveryPointID
	return svc.Restore(ctx, req)
}

func (s *Server) handleRestoreRecoveryPoint(w http.ResponseWriter, r *http.Request) {
	var req backup.RestoreRequest
	if !httputil.DecodeRequestBody(w, r, &req) {
		return
	}
	vars := mux.Vars(r)
	receipt, err := s.RestoreRecoveryPoint(r.Context(), vars["id"], vars["rp"], req)
	if err != nil {
		if receipt != nil {
			apierrors.Write(w, apierrors.As(err).WithDetail("restore_receipt", receipt))
			return
		}
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"schema_version": BackupSchemaVersion, "receipt": receipt})
}

// VerifyRecoveryPoint is the transport-neutral verify path.
func (s *Server) VerifyRecoveryPoint(ctx context.Context, deploymentID, recoveryPointID string, open bool, expect map[string]domain.BindingChecksum) (*backup.VerifyReport, error) {
	svc, err := s.backupService()
	if err != nil {
		return nil, apierrors.Internal("recovery service unavailable", err)
	}
	if _, derr := s.loadDeploymentContext(ctx, deploymentID); derr != nil {
		return nil, derr
	}
	return svc.Verify(ctx, backup.VerifyRequest{DeploymentID: deploymentID, RecoveryPointID: recoveryPointID, OpenArtifacts: open, Expect: expect})
}

func (s *Server) handleVerifyRecoveryPoint(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	open := r.URL.Query().Get("open") == "1" || strings.EqualFold(r.URL.Query().Get("open"), "true")
	report, err := s.VerifyRecoveryPoint(r.Context(), vars["id"], vars["rp"], open, nil)
	if err != nil {
		if report != nil {
			apierrors.Write(w, apierrors.As(err).WithDetail("report", report))
			return
		}
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"schema_version": BackupSchemaVersion, "report": report})
}

// EvaluateRollbackAdmission is the transport-neutral admission path. The
// closure's declared recovery contract for the deployed scenario supplies
// the schema strategy and rollback policy unless the request overrides them.
func (s *Server) EvaluateRollbackAdmission(ctx context.Context, deploymentID string, req RollbackAdmissionRequest) (domain.RollbackVerdict, error) {
	svc, err := s.backupService()
	if err != nil {
		return domain.RollbackVerdict{}, apierrors.Internal("recovery service unavailable", err)
	}
	dctx, derr := s.loadDeploymentContext(ctx, deploymentID)
	if derr != nil {
		return domain.RollbackVerdict{}, derr
	}
	facts := backup.RollbackFacts{CurrentSchema: req.CurrentSchema, TargetSchema: req.TargetSchema, SchemaStrategy: req.SchemaStrategy, CodeRollback: req.CodeRollback, ReadableBy: req.ReadableBy}
	if facts.SchemaStrategy == "" || facts.CodeRollback == "" {
		if derived, _ := s.deploymentClosure(ctx, dctx); derived != nil {
			for _, component := range derived.ComponentsOfKind(domain.ClosureKindScenario) {
				if component.ID == dctx.Manifest.Scenario.ID && component.Recovery != nil {
					if facts.SchemaStrategy == "" {
						facts.SchemaStrategy = component.Recovery.SchemaStrategy
					}
					if facts.CodeRollback == "" {
						facts.CodeRollback = component.Recovery.CodeRollback
					}
				}
			}
		}
	}
	points, err := svc.Repo.ListRecoveryPoints(ctx, deploymentID)
	if err != nil {
		return domain.RollbackVerdict{}, err
	}
	facts.RecoveryPoints = points
	return backup.EvaluateRollback(facts), nil
}

func (s *Server) handleRollbackAdmission(w http.ResponseWriter, r *http.Request) {
	var req RollbackAdmissionRequest
	if !httputil.DecodeRequestBody(w, r, &req) {
		return
	}
	verdict, err := s.EvaluateRollbackAdmission(r.Context(), mux.Vars(r)["id"], req)
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	if rollbackErr := backup.RollbackError(verdict); rollbackErr != nil {
		apierrors.Write(w, rollbackErr.WithDetail("verdict", verdict))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"schema_version": BackupSchemaVersion, "verdict": verdict})
}

// PruneRecoveryPoints is the transport-neutral retention path.
func (s *Server) PruneRecoveryPoints(ctx context.Context, deploymentID string, req PruneRequest) (backup.PrunePlan, error) {
	svc, err := s.backupService()
	if err != nil {
		return backup.PrunePlan{}, apierrors.Internal("recovery service unavailable", err)
	}
	dctx, derr := s.loadDeploymentContext(ctx, deploymentID)
	if derr != nil {
		return backup.PrunePlan{}, derr
	}
	refs := req.References
	if dctx.Deployment.BundleSHA256 != nil {
		refs.ActiveReleases = append(refs.ActiveReleases, "sha256:"+*dctx.Deployment.BundleSHA256)
	}
	return svc.Prune(ctx, deploymentID, refs, backup.Policy{KeepLast: req.KeepLast, MaxAge: time.Duration(req.MaxAgeSeconds) * time.Second})
}

func (s *Server) handlePruneRecoveryPoints(w http.ResponseWriter, r *http.Request) {
	var req PruneRequest
	if !httputil.DecodeRequestBody(w, r, &req) {
		return
	}
	plan, err := s.PruneRecoveryPoints(r.Context(), mux.Vars(r)["id"], req)
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"schema_version": BackupSchemaVersion, "plan": plan})
}
