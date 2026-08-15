package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type credentialReadiness struct {
	Resource    string `json:"resource"`
	LogicalID   string `json:"logical_id"`
	Field       string `json:"field"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	ObtainURL   string `json:"obtain_url,omitempty"`
	Required    bool   `json:"required"`
	Status      string `json:"status"`
	Detail      string `json:"detail,omitempty"`
}

type readinessCredentialDescriptor struct {
	LogicalID   string `json:"logical_id"`
	Field       string `json:"field"`
	Label       string `json:"label"`
	Description string `json:"description"`
	ObtainURL   string `json:"obtain_url"`
	Required    bool   `json:"required"`
}

type readinessResponse struct {
	Status              string                `json:"status"`
	Scenarios           []string              `json:"scenarios"`
	Resources           []string              `json:"resources"`
	Credentials         []credentialReadiness `json:"credentials"`
	Hosts               []hostReadiness       `json:"hosts"`
	Integrations        []readinessItem       `json:"integrations"`
	CheckedAt           string                `json:"checked_at"`
	CredentialDiagnosis json.RawMessage       `json:"credential_diagnosis,omitempty"`
	Recovery            recoveryReadiness     `json:"recovery"`
}

type recoveryReadiness struct {
	ReceiptExists bool     `json:"receipt_exists"`
	ExportedAt    string   `json:"exported_at,omitempty"`
	EntryCount    int      `json:"entry_count"`
	Uncovered     []string `json:"uncovered"`
}

type credentialDiagnosisResponse struct {
	Recovery recoveryReadiness `json:"recovery"`
}

// readinessItem is a metadata-safe, actionable validation result. Its status
// is one of ready, degraded, missing, unsupported, or deferred.
type readinessItem struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Detail      string `json:"detail,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

type releaseAuthorityStatus struct {
	Configured       bool   `json:"configured"`
	TrustAnchorMatch bool   `json:"trust_anchor_match"`
	Provider         string `json:"provider"`
}

type hostReadiness struct {
	readinessItem
	Kind     string `json:"kind"`
	Required bool   `json:"required"`
}

var credentialStatusCommand = func(ctx context.Context, logicalID, field string) ([]byte, error) {
	return onboardingStatusJSON(ctx, logicalID, field)
}

var releaseAuthorityStatusCommand = func(ctx context.Context, root string) ([]byte, error) {
	command := exec.CommandContext(ctx, "vrooli", "release-authority", "status", "--format", "json")
	command.Dir = root
	return command.Output()
}

func releaseAuthorityReadiness(root string) readinessItem {
	item := readinessItem{
		Name:        "release-authority",
		Status:      "unsupported",
		Detail:      "release authority status is unavailable",
		Remediation: "Run `vrooli release-authority init` after the native secure store is available.",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := releaseAuthorityStatusCommand(ctx, root)
	if err != nil {
		return item
	}
	var status releaseAuthorityStatus
	if err := json.Unmarshal(output, &status); err != nil {
		item.Detail = "release authority returned an invalid status response"
		return item
	}
	if !status.Configured {
		item.Status = "missing"
		item.Detail = "no managed release key is configured"
		return item
	}
	if !status.TrustAnchorMatch {
		item.Status = "degraded"
		item.Detail = "managed release key exists but the repository trust anchor is not synchronized"
		item.Remediation = "Run `vrooli release-authority init --replace-trust-anchor` after reviewing the trust-root change."
		return item
	}
	item.Status = "ready"
	item.Detail = "managed release key and repository trust anchor are synchronized"
	item.Remediation = ""
	return item
}

func selectedScenarioModels() ([]ScenarioReadModel, error) {
	models, err := loadScenarioReadModels()
	if err != nil {
		return nil, err
	}
	selected := make([]ScenarioReadModel, 0, len(models))
	for _, model := range models {
		if model.Enabled {
			selected = append(selected, model)
		}
	}
	return selected, nil
}

func loadCredentialReadiness(resource string) ([]credentialReadiness, error) {
	root, err := manifestRoot()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(root, "resources", resource, "resource.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var manifest struct {
		Credentials struct {
			Descriptors []readinessCredentialDescriptor `json:"descriptors"`
		} `json:"credentials"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode %s credential descriptors: %w", resource, err)
	}
	return credentialReadinessForDescriptors(resource, manifest.Credentials.Descriptors), nil
}

func loadScenarioCredentialReadiness(scenario string) ([]credentialReadiness, error) {
	root, err := manifestRoot()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(root, "scenarios", scenario, ".vrooli", "service.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var manifest struct {
		Credentials struct {
			Descriptors []readinessCredentialDescriptor `json:"descriptors"`
		} `json:"credentials"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode %s scenario credential descriptors: %w", scenario, err)
	}
	return credentialReadinessForDescriptors(scenario, manifest.Credentials.Descriptors), nil
}

func credentialReadinessForDescriptors(owner string, descriptors []readinessCredentialDescriptor) []credentialReadiness {
	items := make([]credentialReadiness, 0, len(descriptors))
	for _, descriptor := range descriptors {
		field := strings.TrimSpace(descriptor.Field)
		if field == "" {
			field = "value"
		}
		item := credentialReadiness{Resource: owner, LogicalID: descriptor.LogicalID, Field: field, Label: descriptor.Label, Description: descriptor.Description, ObtainURL: descriptor.ObtainURL, Required: descriptor.Required, Status: "unconfigured"}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		output, statusErr := credentialStatusCommand(ctx, descriptor.LogicalID, field)
		cancel()
		if statusErr != nil {
			item.Status = "unsupported"
			item.Detail = "native credential authority unavailable"
		} else {
			var status struct {
				Configured bool `json:"configured"`
			}
			if err := json.Unmarshal(output, &status); err != nil {
				item.Status = "unsupported"
				item.Detail = "credential authority returned an invalid status response"
			} else if status.Configured {
				item.Status = "configured"
			}
		}
		items = append(items, item)
	}
	return items
}

func (s *Server) handleV2Readiness(w http.ResponseWriter, r *http.Request) {
	models, err := selectedScenarioModels()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	resourceSet := map[string]struct{}{}
	response := readinessResponse{Status: "ready", Scenarios: make([]string, 0, len(models)), Credentials: []credentialReadiness{}, Hosts: []hostReadiness{}, Integrations: []readinessItem{{Name: "integration-hub", Status: "deferred", Detail: "Integration Hub is not yet available; no bindings were created.", Remediation: "Configure integrations after Integration Hub is installed."}}, Recovery: recoveryReadiness{Uncovered: []string{}}, CheckedAt: operatorStateNow().UTC().Format(time.RFC3339)}
	for _, model := range models {
		response.Scenarios = append(response.Scenarios, model.Name)
		scenarioCredentials, err := loadScenarioCredentialReadiness(model.Name)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		response.Credentials = append(response.Credentials, scenarioCredentials...)
		for _, resource := range model.Resources {
			resourceSet[resource] = struct{}{}
		}
	}
	for resource := range resourceSet {
		response.Resources = append(response.Resources, resource)
		credentials, err := loadCredentialReadiness(resource)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		response.Credentials = append(response.Credentials, credentials...)
	}
	sort.Strings(response.Scenarios)
	sort.Strings(response.Resources)
	sort.Slice(response.Credentials, func(i, j int) bool {
		return response.Credentials[i].Resource+response.Credentials[i].Field < response.Credentials[j].Resource+response.Credentials[j].Field
	})
	root, err := manifestRoot()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	state, err := loadOperatorState()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	hosts, err := deriveV2HostRequirements(root, state, models)
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	for _, tool := range hosts.Tools {
		response.Hosts = append(response.Hosts, inspectToolReadiness(tool))
	}
	for _, safeguard := range hosts.Safeguards {
		response.Hosts = append(response.Hosts, inspectSafeguardReadiness(root, safeguard))
	}
	sort.Slice(response.Hosts, func(i, j int) bool {
		return response.Hosts[i].Kind+response.Hosts[i].Name < response.Hosts[j].Kind+response.Hosts[j].Name
	})
	for _, credential := range response.Credentials {
		switch {
		case credential.Status == "unsupported":
			response.Status = lessReady(response.Status, "unsupported")
		case credential.Required && credential.Status != "configured":
			response.Status = lessReady(response.Status, "missing")
		case !credential.Required && credential.Status != "configured":
			response.Status = lessReady(response.Status, "degraded")
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	if output, err := credentialDoctorCommand(ctx); err == nil && json.Valid(output) {
		response.CredentialDiagnosis = append(response.CredentialDiagnosis[:0], output...)
		var diagnosis credentialDiagnosisResponse
		if json.Unmarshal(output, &diagnosis) == nil {
			response.Recovery = diagnosis.Recovery
			if response.Recovery.Uncovered == nil {
				response.Recovery.Uncovered = []string{}
			}
		}
	}
	cancel()
	for _, host := range response.Hosts {
		response.Status = lessReady(response.Status, host.Status)
	}
	releaseItem := releaseAuthorityReadiness(root)
	response.Integrations = append(response.Integrations, releaseItem)
	response.Status = lessReady(response.Status, releaseItem.Status)
	writeJSON(w, http.StatusOK, response)
}

func inspectToolReadiness(item hostItem) hostReadiness {
	result := hostReadiness{readinessItem: readinessItem{Name: item.Name, Status: "deferred", Detail: item.Notes, Remediation: "Select this optional tool in onboarding when its capability is needed."}, Kind: "tool", Required: item.Required}
	selected := item.Status == "required" || item.Status == "opted_in"
	if !selected {
		return result
	}
	if !hostPlatformSupported(item.Platforms) {
		result.Status = "unsupported"
		result.Detail = "This tool is not declared for the current operating system."
		result.Remediation = "Use a supported deployment target or deselect the optional capability."
		return result
	}
	if len(item.Commands) == 0 {
		result.Status = "unsupported"
		result.Detail = "The tool manifest has no executable probe command."
		result.Remediation = "Add a declared command to the tool manifest before enabling this capability."
		return result
	}
	if _, err := exec.LookPath(item.Commands[0]); err != nil {
		result.Status = "missing"
		result.Detail = "The selected tool was not found on this host."
		result.Remediation = "Install the declared host tool, then rerun validation."
		return result
	}
	result.Status = "ready"
	result.Detail = "The declared executable is available on this host."
	result.Remediation = ""
	return result
}

func inspectSafeguardReadiness(root string, item hostItem) hostReadiness {
	result := hostReadiness{readinessItem: readinessItem{Name: item.Name, Status: "deferred", Detail: item.Notes, Remediation: "Select this optional safeguard only after reviewing its host-change risk."}, Kind: "safeguard", Required: item.Required}
	selected := item.Status == "required" || item.Status == "opted_in"
	if !selected {
		return result
	}
	if !hostPlatformSupported(item.Platforms) {
		result.Status = "unsupported"
		result.Detail = "This safeguard is not declared for the current operating system."
		result.Remediation = "Use a supported deployment target or deselect the optional safeguard."
		return result
	}
	data, err := hostManifest(root, "safeguards", item.Name)
	if err != nil {
		result.Status = "unsupported"
		result.Detail = "The safeguard manifest could not be resolved."
		result.Remediation = "Repair the safeguard declaration before continuing."
		return result
	}
	var manifest struct {
		VerificationCheck struct {
			Files []string `json:"files"`
		} `json:"verificationCheck"`
	}
	if json.Unmarshal(data, &manifest) != nil || len(manifest.VerificationCheck.Files) == 0 {
		result.Status = "unsupported"
		result.Detail = "The safeguard has no declarative verification probe."
		result.Remediation = "Add a verification check before enabling this safeguard."
		return result
	}
	for _, path := range manifest.VerificationCheck.Files {
		if _, err := os.Stat(path); err != nil {
			result.Status = "missing"
			result.Detail = "The safeguard has not been applied on this host."
			result.Remediation = "Apply the declared safeguard with the required host privilege, then rerun validation."
			return result
		}
	}
	result.Status = "ready"
	result.Detail = "The declared safeguard verification files are present."
	result.Remediation = ""
	return result
}

func lessReady(current, candidate string) string {
	priority := map[string]int{"ready": 0, "deferred": 0, "degraded": 1, "missing": 2, "unsupported": 3}
	if priority[candidate] > priority[current] {
		return candidate
	}
	return current
}
