package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/credentials"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/reach"

	"github.com/gorilla/mux"
)

// DeploymentRepository provides access to deployment data.
type DeploymentRepository interface {
	GetDeployment(ctx context.Context, id string) (*domain.Deployment, error)
}

// ManagementDeps holds dependencies for secrets management handlers. The
// routes run on the credential binding model: Lifecycle binds one
// deployment's lifecycle; when nil, the default is bound through Reach
// from Repo (which must also be the credential ledger).
type ManagementDeps struct {
	Repo      DeploymentRepository
	Reach     reach.Reach
	Lifecycle func(ctx context.Context, deploymentID string) (BindingLifecycle, error)
}

// secretKeyRegex validates secret keys (uppercase + underscore format).
var secretKeyRegex = regexp.MustCompile(domain.SecretKeyValidationRegex)

// validateSecretKey validates that a key is in the correct format.
func validateSecretKey(key string) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}
	if len(key) > domain.MaxSecretKeyLength {
		return fmt.Errorf("key exceeds maximum length of %d characters", domain.MaxSecretKeyLength)
	}
	if !secretKeyRegex.MatchString(key) {
		return fmt.Errorf("key must be uppercase letters, numbers, and underscores (e.g., MY_API_KEY)")
	}
	for _, prefix := range domain.ReservedKeyPrefixes {
		if strings.HasPrefix(key, prefix) {
			return fmt.Errorf("key prefix %q is reserved", prefix)
		}
	}
	return nil
}

// validateSecretValue validates that a value is acceptable.
func validateSecretValue(value string) error {
	return credentials.ShapeProbe(context.Background(), value)
}

// lifecycleFor resolves the binding lifecycle for a deployment.
func (d ManagementDeps) lifecycleFor(ctx context.Context, deploymentID string) (BindingLifecycle, error) {
	if d.Lifecycle != nil {
		return d.Lifecycle(ctx, deploymentID)
	}
	store, ok := d.Repo.(credentials.Store)
	if !ok || d.Repo == nil {
		return nil, apierrors.New(apierrors.CodeInternal, "secrets management needs the credential ledger")
	}
	dep, err := d.Repo.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, apierrors.Internal("get deployment", err)
	}
	if dep == nil {
		return nil, apierrors.New(apierrors.CodeDeploymentNotFound, "deployment not found").WithDetail("deployment_id", deploymentID)
	}
	var m domain.CloudManifest
	if err := json.Unmarshal(dep.Manifest, &m); err != nil {
		return nil, apierrors.Internal("parse manifest", err)
	}
	if m.Target.VPS == nil {
		return nil, apierrors.New(apierrors.CodeUnsupportedCapability, "deployment has no VPS target")
	}
	target := credentials.Target{DeploymentID: dep.ID, Ref: dep.Target, Fence: dep.Fence}
	if strings.TrimSpace(target.Ref.Transport) == "" {
		target.Ref.Transport = identity.TransportSSH
	}
	if target.Ref.Locator.Host == "" {
		target.Ref.Locator = domain.TargetRefFromManifest(m).Locator
	}
	service, err := credentials.BindReach(credentials.BindOptions{Store: store}, d.Reach, target.Ref)
	if err != nil {
		return nil, apierrors.Internal("bind credential lifecycle", err)
	}
	return NewBindingLifecycle(service, target, m), nil
}

func writeManagementError(w http.ResponseWriter, err error) {
	if typed := apierrors.As(err); typed != nil {
		apierrors.Write(w, typed)
		return
	}
	apierrors.Write(w, apierrors.Internal("secrets management failed", err))
}

func requireDeploymentID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := mux.Vars(r)["id"]
	if id == "" {
		apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "Deployment ID is required"))
		return "", false
	}
	return id, true
}

// HandleListVPSSecrets lists the deployment's credential bindings as secret
// entries. Values are never returned.
//
// GET /api/v1/deployments/{id}/secrets
func HandleListVPSSecrets(deps ManagementDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deploymentID, ok := requireDeploymentID(w, r)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		lifecycle, err := deps.lifecycleFor(ctx, deploymentID)
		if err != nil {
			writeManagementError(w, err)
			return
		}
		entries, err := lifecycle.List(ctx)
		if err != nil {
			writeManagementError(w, err)
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		httputil.WriteJSON(w, http.StatusOK, domain.ListVPSSecretsResponse{
			Secrets:   entries,
			Metadata:  domain.VPSSecretsMetadata{Environment: "credential-binding", LastUpdated: now, ScenarioID: scenarioOf(lifecycle), GeneratedBy: "scenario-to-cloud"},
			Timestamp: now,
		})
	}
}

func scenarioOf(lifecycle BindingLifecycle) string {
	if l, ok := lifecycle.(*bindingLifecycle); ok {
		return l.manifest.Scenario.ID
	}
	return ""
}

// HandleGetVPSSecret returns one binding's metadata. Values are never
// returned, including when reveal is requested.
//
// GET /api/v1/deployments/{id}/secrets/{key}
func HandleGetVPSSecret(deps ManagementDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deploymentID, ok := requireDeploymentID(w, r)
		if !ok {
			return
		}
		key := mux.Vars(r)["key"]
		if key == "" {
			apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "Secret key is required"))
			return
		}
		if r.URL.Query().Get("reveal") == "true" {
			apierrors.Write(w, apierrors.New(apierrors.CodeForbiddenScope, "Credential values cannot be revealed through the management API").WithDetail("key", key))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		lifecycle, err := deps.lifecycleFor(ctx, deploymentID)
		if err != nil {
			writeManagementError(w, err)
			return
		}
		entry, err := lifecycle.Get(ctx, key)
		if err != nil {
			writeManagementError(w, err)
			return
		}
		httputil.WriteJSON(w, http.StatusOK, domain.GetVPSSecretResponse{Secret: *entry, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	}
}

// HandleCreateVPSSecret binds a new key and materialises its first version
// from the operator value.
//
// POST /api/v1/deployments/{id}/secrets
func HandleCreateVPSSecret(deps ManagementDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deploymentID, ok := requireDeploymentID(w, r)
		if !ok {
			return
		}
		var req domain.CreateSecretRequest
		if !httputil.DecodeRequestBody(w, r, &req) {
			return
		}
		if err := validateSecretKey(req.Key); err != nil {
			apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, err.Error()).WithDetail("field", "key"))
			return
		}
		if err := validateSecretValue(req.Value); err != nil {
			apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "invalid value: "+err.Error()).WithDetail("field", "value"))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		lifecycle, err := deps.lifecycleFor(ctx, deploymentID)
		if err != nil {
			writeManagementError(w, err)
			return
		}
		binding, err := lifecycle.Create(ctx, req.Key, req.Value)
		if err != nil {
			writeManagementError(w, err)
			return
		}
		response := domain.NewSecretOperationResponse(true, req.Key, "created", "Credential version "+fmt.Sprint(binding.Version.Number)+" materialised")
		response.BindingID = binding.ID
		response.Version = binding.Version.Number
		finishWithRestart(ctx, w, lifecycle, req.RestartScenario, response, http.StatusCreated)
	}
}

// HandleUpdateVPSSecret rotates the key to the operator value.
//
// PUT /api/v1/deployments/{id}/secrets/{key}
func HandleUpdateVPSSecret(deps ManagementDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deploymentID, ok := requireDeploymentID(w, r)
		if !ok {
			return
		}
		key := mux.Vars(r)["key"]
		var req domain.UpdateSecretRequest
		if !httputil.DecodeRequestBody(w, r, &req) {
			return
		}
		if err := validateSecretValue(req.Value); err != nil {
			apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "invalid value: "+err.Error()).WithDetail("field", "value"))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		lifecycle, err := deps.lifecycleFor(ctx, deploymentID)
		if err != nil {
			writeManagementError(w, err)
			return
		}
		rotation, err := lifecycle.Replace(ctx, key, req.Value)
		if err != nil {
			writeOperationError(w, err, rotation)
			return
		}
		response := domain.NewSecretOperationResponse(true, key, "updated", "Rotation "+rotation.ID+" is "+string(rotation.State))
		response.BindingID = rotation.BindingID
		response.Version = rotation.ToVersion
		response.RotationID = rotation.ID
		response.OperationState = string(rotation.State)
		status := http.StatusOK
		if rotation.Incomplete() {
			status = http.StatusAccepted
		}
		finishWithRestart(ctx, w, lifecycle, req.RestartScenario, response, status)
	}
}

// HandleDeleteVPSSecret revokes the key's active version.
//
// DELETE /api/v1/deployments/{id}/secrets/{key}
func HandleDeleteVPSSecret(deps ManagementDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deploymentID, ok := requireDeploymentID(w, r)
		if !ok {
			return
		}
		key := mux.Vars(r)["key"]
		var req domain.DeleteSecretRequest
		if !httputil.DecodeRequestBody(w, r, &req) {
			return
		}
		if req.Confirmation != "DELETE" {
			apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "Confirmation must be exactly 'DELETE'").WithDetail("field", "confirmation"))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		lifecycle, err := deps.lifecycleFor(ctx, deploymentID)
		if err != nil {
			writeManagementError(w, err)
			return
		}
		rotation, err := lifecycle.Delete(ctx, key)
		if err != nil {
			writeOperationError(w, err, rotation)
			return
		}
		response := domain.NewSecretOperationResponse(true, key, "deleted", "Revocation "+rotation.ID+" is "+string(rotation.State))
		response.BindingID = rotation.BindingID
		response.RotationID = rotation.ID
		response.OperationState = string(rotation.State)
		finishWithRestart(ctx, w, lifecycle, req.RestartScenario, response, http.StatusOK)
	}
}

func writeOperationError(w http.ResponseWriter, err error, rotation *domain.CredentialRotation) {
	typed := apierrors.As(err)
	if typed == nil {
		typed = apierrors.Internal("credential lifecycle failed", err)
	}
	if rotation != nil {
		typed = typed.WithDetail("operation", rotation)
	}
	apierrors.Write(w, typed)
}

func finishWithRestart(ctx context.Context, w http.ResponseWriter, lifecycle BindingLifecycle, restart bool, response domain.SecretOperationResponse, status int) {
	if restart {
		if err := lifecycle.Restart(ctx); err != nil {
			response.Message += "; scenario restart failed: " + err.Error()
			response.ScenarioRestart = false
		} else {
			response.ScenarioRestart = true
			response.Message += "; scenario restarted"
		}
	}
	httputil.WriteJSON(w, status, response)
}
