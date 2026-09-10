package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/manifest"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/vps"

	"github.com/gorilla/mux"
)

// Type aliases for backward compatibility.
type (
	VPSActionRequest  = vps.ActionRequest
	VPSActionResponse = vps.ActionResponse
)

// handleVPSAction runs a VPS management action through the target owner's
// typed verbs. Stopping goes through the lifecycle owner (scoped stops keep
// the demand other deployments hold on shared resources); Docker cleanup
// goes through the privilege broker's prune actions. Actions with no owner
// verb (host reboot, deleting the installation) are refused with the owner
// that must exist first; the retirement plan is the supported way to remove
// a deployment.
// POST /api/v1/deployments/{id}/actions/vps
func (s *Server) handleVPSAction(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	req, err := httputil.DecodeJSON[VPSActionRequest](r.Body, 1<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{Code: "invalid_json", Message: "Request body must be valid JSON", Hint: err.Error()})
		return
	}
	validActions := map[string]bool{"reboot": true, "stop_vrooli": true, "cleanup": true}
	if !validActions[req.Action] {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{Code: "invalid_action", Message: "Action must be 'reboot', 'stop_vrooli', or 'cleanup'"})
		return
	}
	if req.Action == "cleanup" && (req.CleanupLevel < 1 || req.CleanupLevel > 5) {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{Code: "invalid_cleanup_level", Message: "Cleanup level must be between 1 and 5"})
		return
	}
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Code: "get_failed", Message: "Failed to get deployment", Hint: err.Error()})
		return
	}
	if deployment == nil {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{Code: "not_found", Message: "Deployment not found"})
		return
	}
	var m domain.CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &m); err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{Code: "manifest_parse_failed", Message: "Failed to parse deployment manifest", Hint: err.Error()})
		return
	}
	normalized, _ := manifest.ValidateAndNormalize(m)
	if normalized.Target.VPS == nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{Code: "no_vps_target", Message: "Deployment does not have a VPS target"})
		return
	}
	if err := vps.ValidateActionConfirmation(req.Action, req.CleanupLevel, req.Confirmation, deployment.Name); err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{Code: "invalid_confirmation", Message: err.Error()})
		return
	}

	timeout := 2 * time.Minute
	if req.Action == "cleanup" && req.CleanupLevel >= 3 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	rt := s.executionRuntime(normalized, id, "")
	response := VPSActionResponse{Action: req.Action, Timestamp: time.Now().UTC().Format(time.RFC3339)}

	var (
		outputs []string
		actErr  *apierrors.Error
	)
	switch req.Action {
	case "reboot":
		actErr = apierrors.New(apierrors.CodeUnsupportedCapability, "Host reboot has no target owner action; reboot the host through its provider console").
			WithDetail("required_owner", "privilegebroker host.reboot")
	case "stop_vrooli":
		outputs, actErr = s.stopDeploymentWorkloads(ctx, rt, normalized)
		if actErr == nil {
			if err := s.repo.UpdateDesiredState(ctx, id, domain.DesiredStopped); err != nil {
				s.log("failed to record desired state", map[string]interface{}{"deployment_id": id, "error": err.Error()})
			}
		}
		response.Message = "Vrooli workloads stopped through the lifecycle owner; desired state recorded as stopped"
	case "cleanup":
		switch req.CleanupLevel {
		case 3:
			for _, action := range []string{"docker.prune.unused-images", "docker.prune.unused-volumes"} {
				out, err := managementRepair(ctx, rt, action, map[string]any{})
				outputs = append(outputs, action+": "+out)
				if err != nil {
					actErr = err
					break
				}
			}
			response.Message = "Docker prune actions ran through the privilege broker (named volumes and in-use images are kept)"
		default:
			actErr = apierrors.Newf(apierrors.CodeUnsupportedCapability, "Cleanup level %d deletes owned files in place; retire the deployment through its retirement plan instead", req.CleanupLevel).
				WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "retire", Reference: "/api/v1/deployments/" + id + "/retire/plan", Label: "Preview the retirement plan (retained vs deleted objects) and apply it"})
		}
	}
	if actErr != nil {
		apierrors.Write(w, actErr)
		return
	}
	response.OK = true
	response.Output = strings.Join(outputs, "\n")
	httputil.WriteJSON(w, http.StatusOK, response)
}

// stopDeploymentWorkloads stops the target scenario and its dependent
// scenarios through scoped lifecycle stops, then its resources through the
// resource owner. Shared resources another deployment demands stay up: the
// lifecycle owner releases only this deployment's demand.
func (s *Server) stopDeploymentWorkloads(ctx context.Context, rt vps.Runtime, m domain.CloudManifest) ([]string, *apierrors.Error) {
	var outputs []string
	scenarios := []string{m.Scenario.ID}
	for _, dep := range m.Dependencies.Scenarios {
		if dep != m.Scenario.ID {
			scenarios = append(scenarios, dep)
		}
	}
	for _, scenario := range scenarios {
		out, err := managementRepair(ctx, rt, "process.stop.scoped", map[string]any{"process": map[string]any{"scenario": scenario, "workdir": m.Target.VPS.Workdir}})
		outputs = append(outputs, "stop "+scenario+": "+out)
		if err != nil {
			return outputs, err
		}
	}
	for _, resource := range m.Dependencies.Resources {
		res, err := rt.Reach.Exec(ctx, rt.Target, reach.Command{Verb: "resource stop", Args: []string{resource, "--json"}, RequiredScope: "vrooli:write", Effectful: true, Timeout: 2 * time.Minute})
		if err != nil {
			return outputs, reach.APIError(err)
		}
		if res.ExitCode != 0 {
			outputs = append(outputs, "resource "+resource+": kept (exit "+fmt.Sprint(res.ExitCode)+"; another deployment may still demand it)")
			continue
		}
		outputs = append(outputs, "resource "+resource+": stopped")
	}
	return outputs, nil
}

// managementRepair runs one privilege broker action through the target
// owner without an operation (management actions are operator-driven).
func managementRepair(ctx context.Context, rt vps.Runtime, action string, subject any) (string, *apierrors.Error) {
	raw, _ := json.Marshal(subject)
	encoded := vps.JSONArgPrefix + base64.RawURLEncoding.EncodeToString(raw)
	res, err := rt.Reach.Exec(ctx, rt.Target, reach.Command{Verb: "cloud-target host repair", Args: []string{"--action", action, "--subject", encoded, "--json"}, RequiredScope: "vrooli:write", Effectful: true, Timeout: 5 * time.Minute})
	if err != nil {
		return "", reach.APIError(err)
	}
	var reply struct {
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Result struct {
			Changed bool   `json:"changed"`
			Status  string `json:"status"`
		} `json:"result"`
	}
	_ = json.Unmarshal([]byte(strings.TrimSpace(res.Stdout)), &reply)
	if reply.Error != nil {
		return "", apierrors.New(reply.Error.Code, reply.Error.Message).WithDetail("action", action)
	}
	if res.ExitCode != 0 {
		return "", apierrors.Newf(apierrors.CodeInternal, "host repair %s exited %d", action, res.ExitCode)
	}
	return fmt.Sprintf("%s (changed=%t)", reply.Result.Status, reply.Result.Changed), nil
}
