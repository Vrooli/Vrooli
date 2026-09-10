package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/credentials"
	"scenario-to-cloud/credentialsvc"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/secrets"
)

// credentialLifecycleOverride lets tests substitute the lifecycle behind the
// REST routes and the Connect CredentialsService.
var credentialLifecycleOverride credentialsvc.Lifecycle

// credentialRewrapVerifier is the backup domain's proof that a recovery
// point rewrapped and restored under a new encryption/recovery key version.
// Nil means encryption-key rotation refuses to retire the predecessor, which
// is the fail-closed default until the backup owner wires its verifier.
var credentialRewrapVerifier func(ctx context.Context, ref domain.CredentialVersionRef) error

// credentialLifecycle returns the lifecycle the routes run on.
func (s *Server) credentialLifecycle() credentialsvc.Lifecycle {
	if credentialLifecycleOverride != nil {
		return credentialLifecycleOverride
	}
	return credentialAdapter{s: s}
}

// CredentialsService returns the Connect service over this server.
func (s *Server) CredentialsService() *credentialsvc.Service {
	return credentialsvc.New(s.credentialLifecycle())
}

// registerCredentialRoutes mounts the REST credential surface beside the
// Connect CredentialsService. Mount line for main.go setupRoutes:
//
//	s.registerCredentialRoutes(api)
func (s *Server) registerCredentialRoutes(api *mux.Router) {
	api.HandleFunc("/deployments/{id}/credentials", s.handleListCredentialBindings).Methods("GET")
	api.HandleFunc("/deployments/{id}/credentials/recover", s.handleRecoverCredentials).Methods("POST")
	api.HandleFunc("/deployments/{id}/credentials/rotations/{rotation}", s.handleGetCredentialRotation).Methods("GET")
	api.HandleFunc("/deployments/{id}/credentials/rotations/{rotation}/resume", s.handleResumeCredentialRotation).Methods("POST")
	api.HandleFunc("/deployments/{id}/credentials/{binding}/rotate", s.handleRotateCredential).Methods("POST")
	api.HandleFunc("/deployments/{id}/credentials/{binding}/revoke", s.handleRevokeCredential).Methods("POST")
	api.HandleFunc("/deployments/{id}/credentials/{binding}/break-glass", s.handleBreakGlassCredential).Methods("POST")
	path, handler := s.CredentialsService().Handler()
	s.router.PathPrefix(path).Handler(handler)
}

func (s *Server) handleListCredentialBindings(w http.ResponseWriter, r *http.Request) {
	views, err := s.credentialLifecycle().ListBindings(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		apierrors.Write(w, credentialsvc.OperationError(err, nil))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"schema_version": credentialsvc.SchemaVersion, "bindings": views})
}

// rotateBody is the REST rotation body. Value is inbound only.
type rotateBody struct {
	Value      string `json:"value,omitempty"`
	RequestKey string `json:"request_key,omitempty"`
}

func (s *Server) handleRotateCredential(w http.ResponseWriter, r *http.Request) {
	var body rotateBody
	if r.ContentLength != 0 && !httputil.DecodeRequestBody(w, r, &body) {
		return
	}
	vars := mux.Vars(r)
	rotation, err := s.credentialLifecycle().Rotate(r.Context(), credentialsvc.RotateRequest{DeploymentID: vars["id"], BindingID: vars["binding"], Value: body.Value, RequestKey: body.RequestKey})
	writeCredentialOperation(w, rotation, err)
}

type revokeBody struct {
	RequestKey string `json:"request_key,omitempty"`
}

func (s *Server) handleRevokeCredential(w http.ResponseWriter, r *http.Request) {
	var body revokeBody
	if r.ContentLength != 0 && !httputil.DecodeRequestBody(w, r, &body) {
		return
	}
	vars := mux.Vars(r)
	rotation, err := s.credentialLifecycle().Revoke(r.Context(), credentialsvc.RevokeRequest{DeploymentID: vars["id"], BindingID: vars["binding"], RequestKey: body.RequestKey})
	writeCredentialOperation(w, rotation, err)
}

type recoverBody struct {
	BundleRef  string `json:"bundle_ref"`
	Passphrase string `json:"passphrase,omitempty"`
	RequestKey string `json:"request_key,omitempty"`
}

func (s *Server) handleRecoverCredentials(w http.ResponseWriter, r *http.Request) {
	var body recoverBody
	if !httputil.DecodeRequestBody(w, r, &body) {
		return
	}
	rotation, err := s.credentialLifecycle().Recover(r.Context(), credentialsvc.RecoverRequest{DeploymentID: mux.Vars(r)["id"], BundleRef: body.BundleRef, Passphrase: body.Passphrase, RequestKey: body.RequestKey})
	writeCredentialOperation(w, rotation, err)
}

func (s *Server) handleGetCredentialRotation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rotation, err := s.credentialLifecycle().GetRotation(r.Context(), vars["id"], vars["rotation"])
	writeCredentialOperation(w, rotation, err)
}

type resumeBody struct {
	OperatorConfirmed bool `json:"operator_confirmed"`
}

func (s *Server) handleResumeCredentialRotation(w http.ResponseWriter, r *http.Request) {
	var body resumeBody
	if r.ContentLength != 0 && !httputil.DecodeRequestBody(w, r, &body) {
		return
	}
	vars := mux.Vars(r)
	rotation, err := s.credentialLifecycle().Resume(r.Context(), credentialsvc.ResumeRequest{DeploymentID: vars["id"], RotationID: vars["rotation"], OperatorConfirmed: body.OperatorConfirmed})
	writeCredentialOperation(w, rotation, err)
}

type breakGlassBody struct {
	Scope         string `json:"scope"`
	WindowSeconds int64  `json:"window_seconds"`
	Confirmation  string `json:"confirmation"`
	Operator      string `json:"operator"`
	RequestKey    string `json:"request_key,omitempty"`
}

func (s *Server) handleBreakGlassCredential(w http.ResponseWriter, r *http.Request) {
	var body breakGlassBody
	if !httputil.DecodeRequestBody(w, r, &body) {
		return
	}
	vars := mux.Vars(r)
	rotation, err := s.credentialLifecycle().BreakGlass(r.Context(), credentialsvc.BreakGlassRequest{DeploymentID: vars["id"], BindingID: vars["binding"], Scope: body.Scope, WindowSeconds: body.WindowSeconds, Confirmation: body.Confirmation, Operator: body.Operator, RequestKey: body.RequestKey})
	writeCredentialOperation(w, rotation, err)
}

// writeCredentialOperation renders an operation. A typed refusal carries the
// operation in its details so the standing is never lost; an operation that
// stopped short of its terminal state without a refusal is 202.
func writeCredentialOperation(w http.ResponseWriter, rotation *domain.CredentialRotation, err error) {
	if err != nil {
		apierrors.Write(w, credentialsvc.OperationError(err, rotation))
		return
	}
	status := http.StatusOK
	if rotation != nil && rotation.Incomplete() {
		status = http.StatusAccepted
	}
	httputil.WriteJSON(w, status, map[string]any{"schema_version": credentialsvc.SchemaVersion, "operation": rotation})
}

// credentialAdapter resolves a deployment's target and runs the credential
// lifecycle over the deployment's transport. It is the single lifecycle
// construction path shared by REST, Connect and the post-deploy secrets
// management routes.
type credentialAdapter struct{ s *Server }

// boundLifecycle is the lifecycle bound to one deployment's target.
type boundLifecycle struct {
	service  *credentials.Service
	target   credentials.Target
	manifest domain.CloudManifest
}

func (a credentialAdapter) bind(ctx context.Context, deploymentID string) (*boundLifecycle, error) {
	if a.s.repo == nil {
		return nil, apierrors.New(apierrors.CodeInternal, "credential lifecycle needs the deployment repository")
	}
	dctx, terr := a.s.loadDeploymentContext(ctx, deploymentID)
	if terr != nil {
		return nil, terr
	}
	target := credentials.Target{DeploymentID: dctx.Deployment.ID, Ref: dctx.Deployment.Target, Fence: dctx.Deployment.Fence}
	if strings.TrimSpace(target.Ref.Transport) == "" {
		target.Ref.Transport = identity.TransportSSH
	}
	if target.Ref.Locator.Host == "" && dctx.Manifest.Target.VPS != nil {
		target.Ref.Locator = identity.TargetLocator{Host: dctx.Manifest.Target.VPS.Host, Port: dctx.Manifest.Target.VPS.Port, User: dctx.Manifest.Target.VPS.User, Workdir: dctx.Manifest.Target.VPS.Workdir}
	}
	opts := credentials.BindOptions{Store: a.s.repo, Logger: log.Default(), RewrapAndRestore: credentialRewrapVerifier}
	var service *credentials.Service
	switch target.Ref.Transport {
	case identity.TransportBridge:
		client, err := credentials.NewBridgeClient(credentials.BridgeConfig{
			ResolveURL: func(ctx context.Context) (string, error) {
				return discovery.ResolveScenarioURLDefault(ctx, "vrooli-bridge")
			},
			TokenProvider: bridgeOwnerToken,
		})
		if err != nil {
			return nil, apierrors.Internal("bridge credential client", err)
		}
		service = credentials.BindBridge(opts, client, target.NodeLabel())
	default:
		bound, err := credentials.BindReach(opts, a.s.reachFor(dctx), target.Ref)
		if err != nil {
			return nil, apierrors.Internal("credential client", err)
		}
		service = bound
	}
	return &boundLifecycle{service: service, target: target, manifest: dctx.Manifest}, nil
}

// bridgeOwnerToken supplies the Bridge owner credential per call from the
// environment the control plane started the API with.
func bridgeOwnerToken(context.Context) (string, error) {
	for _, name := range []string{"VROOLI_BRIDGE_API_TOKEN", "VROOLI_BRIDGE_TOKEN", "VROOLI_API_TOKEN"} {
		if token := strings.TrimSpace(os.Getenv(name)); token != "" {
			return token, nil
		}
	}
	return "", errors.New("no Bridge owner token is configured (VROOLI_BRIDGE_API_TOKEN)")
}

func (a credentialAdapter) ListBindings(ctx context.Context, deploymentID string) ([]credentials.BindingView, error) {
	if a.s.repo == nil {
		return nil, apierrors.New(apierrors.CodeInternal, "credential lifecycle needs the deployment repository")
	}
	if _, terr := a.s.loadDeploymentContext(ctx, deploymentID); terr != nil {
		return nil, terr
	}
	return (&credentials.Service{Store: a.s.repo}).ListBindings(ctx, deploymentID)
}

func (a credentialAdapter) Rotate(ctx context.Context, req credentialsvc.RotateRequest) (*domain.CredentialRotation, error) {
	bound, err := a.bind(ctx, req.DeploymentID)
	if err != nil {
		return nil, err
	}
	return bound.service.Rotate(ctx, credentials.RotateRequest{Target: bound.target, DeploymentID: req.DeploymentID, BindingID: req.BindingID, Value: req.Value, RequestKey: req.RequestKey})
}

func (a credentialAdapter) Revoke(ctx context.Context, req credentialsvc.RevokeRequest) (*domain.CredentialRotation, error) {
	bound, err := a.bind(ctx, req.DeploymentID)
	if err != nil {
		return nil, err
	}
	return bound.service.Revoke(ctx, credentials.RevokeBindingRequest{Target: bound.target, DeploymentID: req.DeploymentID, BindingID: req.BindingID, RequestKey: req.RequestKey})
}

func (a credentialAdapter) Recover(ctx context.Context, req credentialsvc.RecoverRequest) (*domain.CredentialRotation, error) {
	bound, err := a.bind(ctx, req.DeploymentID)
	if err != nil {
		return nil, err
	}
	return bound.service.Recover(ctx, credentials.RecoverRequest{DeploymentID: req.DeploymentID, NewTarget: bound.target, BundleRef: req.BundleRef, Passphrase: req.Passphrase, RequestKey: req.RequestKey})
}

func (a credentialAdapter) GetRotation(ctx context.Context, deploymentID, rotationID string) (*domain.CredentialRotation, error) {
	if a.s.repo == nil {
		return nil, apierrors.New(apierrors.CodeInternal, "credential lifecycle needs the deployment repository")
	}
	rotation, err := (&credentials.Service{Store: a.s.repo}).GetRotation(ctx, rotationID)
	if err != nil {
		return nil, err
	}
	if rotation.DeploymentID != deploymentID {
		return nil, apierrors.New(credentials.CodeRotationNotFound, "no such credential operation on this deployment").WithDetail("rotation_id", rotationID)
	}
	return rotation, nil
}

func (a credentialAdapter) Resume(ctx context.Context, req credentialsvc.ResumeRequest) (*domain.CredentialRotation, error) {
	if _, err := a.GetRotation(ctx, req.DeploymentID, req.RotationID); err != nil {
		return nil, err
	}
	bound, err := a.bind(ctx, req.DeploymentID)
	if err != nil {
		return nil, err
	}
	return bound.service.Resume(ctx, bound.target, req.RotationID, credentials.ResumeInput{OperatorConfirmed: req.OperatorConfirmed})
}

func (a credentialAdapter) BreakGlass(ctx context.Context, req credentialsvc.BreakGlassRequest) (*domain.CredentialRotation, error) {
	bound, err := a.bind(ctx, req.DeploymentID)
	if err != nil {
		return nil, err
	}
	return bound.service.BreakGlass(ctx, credentials.BreakGlassRequest{Target: bound.target, DeploymentID: req.DeploymentID, BindingID: req.BindingID, Scope: req.Scope, Window: time.Duration(req.WindowSeconds) * time.Second, Confirmation: req.Confirmation, Operator: req.Operator, RequestKey: req.RequestKey})
}

// managementLifecycle adapts the bound lifecycle to the post-deploy secrets
// management routes, which create, replace and delete operator-supplied
// values through the same binding model.
func (a credentialAdapter) Management(ctx context.Context, deploymentID string) (secrets.BindingLifecycle, error) {
	bound, err := a.bind(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	return secrets.NewBindingLifecycle(bound.service, bound.target, bound.manifest), nil
}
