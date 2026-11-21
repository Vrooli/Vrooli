package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

type credentialRequest struct {
	Identity string `json:"identity"`
	Field    string `json:"field"`
	Value    string `json:"value,omitempty"`
}

type credentialView struct {
	Identity   string `json:"identity"`
	Field      string `json:"field"`
	Configured bool   `json:"configured"`
	Required   bool   `json:"required"`
	Label      string `json:"label,omitempty"`
}

type credentialRecoveryExportRequest struct {
	Passphrase string `json:"passphrase"`
}

type credentialRecoveryRestoreRequest struct {
	Bundle     string `json:"bundle"`
	Passphrase string `json:"passphrase"`
}

func (s *Server) declaredCredential(identity, field string) (*manifest.Secret, bool) {
	parsed, err := credentialauthority.ParseIdentity(identity)
	if err != nil {
		return nil, false
	}
	field = strings.TrimSpace(field)
	for index := range s.runtime.Manifest().Secrets {
		secret := &s.runtime.Manifest().Secrets[index]
		if secret.LogicalID == string(parsed) && secret.CredentialField() == field {
			return secret, true
		}
	}
	return nil, false
}

func decodeCredentialRequest(w http.ResponseWriter, r *http.Request) (credentialRequest, bool) {
	var request credentialRequest
	if r.Method == http.MethodGet {
		request.Identity = r.URL.Query().Get("identity")
		request.Field = r.URL.Query().Get("field")
	} else if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid credential request", http.StatusBadRequest)
		return credentialRequest{}, false
	}
	request.Identity = strings.TrimSpace(request.Identity)
	request.Field = strings.TrimSpace(request.Field)
	if request.Field == "" {
		request.Field = "value"
	}
	return request, true
}

func (s *Server) requireDeclaredCredential(w http.ResponseWriter, request credentialRequest) bool {
	if _, ok := s.declaredCredential(request.Identity, request.Field); ok {
		return true
	}
	writeJSON(w, http.StatusForbidden, map[string]string{
		"error":  "credential_not_declared",
		"reason": "the requested identity and field are not declared by this bundle",
	})
	return false
}

func (s *Server) credentialViews() ([]credentialView, error) {
	values := s.runtime.SecretStore().Get()
	views := make([]credentialView, 0, len(s.runtime.Manifest().Secrets))
	for _, secret := range s.runtime.Manifest().Secrets {
		identity := strings.TrimSpace(secret.LogicalID)
		if identity == "" {
			continue
		}
		required := secret.Required == nil || *secret.Required
		views = append(views, credentialView{Identity: identity, Field: secret.CredentialField(), Configured: strings.TrimSpace(values[secret.ID]) != "", Required: required, Label: secret.Description})
	}
	return views, nil
}

func (s *Server) handleCredentialStatus(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeCredentialRequest(w, r)
	if !ok || !s.requireDeclaredCredential(w, request) {
		return
	}
	views, err := s.credentialViews()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credential provider unavailable"})
		return
	}
	for _, view := range views {
		if view.Identity == request.Identity && view.Field == request.Field {
			writeJSON(w, http.StatusOK, view)
			return
		}
	}
	http.Error(w, "credential_not_declared", http.StatusForbidden)
}

// handleCredentialResolve is an authenticated, one-shot read for scenario
// consumers. The runtime never persists a second copy; the caller owns the
// short-lived in-memory lifetime of the returned value.
func (s *Server) handleCredentialResolve(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeCredentialRequest(w, r)
	if !ok || !s.requireDeclaredCredential(w, request) {
		return
	}
	values := s.runtime.SecretStore().Get()
	secret, _ := s.declaredCredential(request.Identity, request.Field)
	value := strings.TrimSpace(values[secret.ID])
	if value == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "credential_unconfigured"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"value": value})
}

func (s *Server) handleCredentialList(w http.ResponseWriter, r *http.Request) {
	views, err := s.credentialViews()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credential provider unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, views)
}

func (s *Server) handleCredentialDoctor(w http.ResponseWriter, r *http.Request) {
	views, err := s.credentialViews()
	response := map[string]any{"credentials": views, "recovery": s.desktopRecoveryStatus()}
	if err != nil {
		response["provider"] = map[string]string{"condition": "unavailable"}
		writeJSON(w, http.StatusServiceUnavailable, response)
		return
	}
	response["provider"] = map[string]string{"condition": "available"}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) desktopRecoveryStatus() map[string]any {
	status := map[string]any{"receipt_exists": false, "exported_at": nil, "entry_count": 0, "uncovered": []string{}}
	data, err := os.ReadFile(filepath.Join(s.runtime.AppDataDir(), "runtime", "recovery-receipt.json"))
	if err != nil {
		return status
	}
	var receipt struct {
		ExportedAt string `json:"exported_at"`
		EntryCount int    `json:"entry_count"`
		Entries    []struct {
			Identity string `json:"identity"`
			Field    string `json:"field"`
		} `json:"entries"`
	}
	if json.Unmarshal(data, &receipt) == nil {
		status["receipt_exists"] = true
		status["exported_at"] = receipt.ExportedAt
		status["entry_count"] = receipt.EntryCount
		covered := make(map[string]bool, len(receipt.Entries))
		for _, entry := range receipt.Entries {
			covered[entry.Identity+":"+entry.Field] = true
		}
		uncovered := make([]string, 0)
		for _, secret := range s.runtime.Manifest().Secrets {
			if secret.Required != nil && !*secret.Required {
				continue
			}
			identity := strings.TrimSpace(secret.LogicalID)
			if identity == "" {
				continue
			}
			key := identity + ":" + secret.CredentialField()
			if !covered[key] {
				uncovered = append(uncovered, key)
			}
		}
		status["uncovered"] = uncovered
	}
	return status
}

func (s *Server) handleCredentialRecoveryExport(w http.ResponseWriter, r *http.Request) {
	var request credentialRecoveryExportRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || strings.TrimSpace(request.Passphrase) == "" {
		http.Error(w, "recovery passphrase is required in the request body", http.StatusBadRequest)
		return
	}
	store, ok := s.runtime.SecretStore().(interface {
		ExportRecovery(string) ([]byte, int, error)
	})
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "desktop recovery is unavailable"})
		return
	}
	bundle, count, err := store.ExportRecovery(request.Passphrase)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "desktop recovery export failed"})
		return
	}
	if err := s.writeDesktopRecoveryReceipt(count); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "desktop recovery receipt could not be recorded"})
		return
	}
	s.runtime.RecordTelemetry("recovery_exported", map[string]interface{}{"count": count})
	writeJSON(w, http.StatusOK, map[string]any{"bundle": base64.StdEncoding.EncodeToString(bundle), "entry_count": count})
}

func (s *Server) handleCredentialRecoveryRestore(w http.ResponseWriter, r *http.Request) {
	var request credentialRecoveryRestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || strings.TrimSpace(request.Bundle) == "" || strings.TrimSpace(request.Passphrase) == "" {
		http.Error(w, "recovery bundle and passphrase are required in the request body", http.StatusBadRequest)
		return
	}
	bundle, err := base64.StdEncoding.DecodeString(request.Bundle)
	if err != nil {
		http.Error(w, "recovery bundle is not valid base64", http.StatusBadRequest)
		return
	}
	manifest, err := credentialauthority.InspectRecovery(bundle, request.Passphrase)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "desktop recovery restore failed"})
		return
	}
	for _, entry := range manifest.Entries {
		if _, declared := s.declaredCredential(string(entry.Identity), entry.Field); !declared {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "credential_not_declared", "reason": "the recovery bundle contains an identity and field not declared by this bundle"})
			return
		}
	}
	store, ok := s.runtime.SecretStore().(interface {
		RestoreRecovery([]byte, string) error
	})
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "desktop recovery is unavailable"})
		return
	}
	if err := store.RestoreRecovery(bundle, request.Passphrase); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "desktop recovery restore failed"})
		return
	}
	s.runtime.RecordTelemetry("recovery_restored", map[string]interface{}{"status": "completed"})
	writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
}

func (s *Server) writeDesktopRecoveryReceipt(count int) error {
	path := filepath.Join(s.runtime.AppDataDir(), "runtime", "recovery-receipt.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	entries := make([]map[string]string, 0, len(s.runtime.Manifest().Secrets))
	for _, secret := range s.runtime.Manifest().Secrets {
		if identity := strings.TrimSpace(secret.LogicalID); identity != "" {
			entries = append(entries, map[string]string{"identity": identity, "field": secret.CredentialField()})
		}
	}
	payload := map[string]any{"exported_at": time.Now().UTC().Format(time.RFC3339), "entry_count": count, "entries": entries}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func (s *Server) handleCredentialProvision(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeCredentialRequest(w, r)
	if !ok || !s.requireDeclaredCredential(w, request) {
		return
	}
	if strings.TrimSpace(request.Value) == "" {
		http.Error(w, "credential value is required in the request body", http.StatusBadRequest)
		return
	}
	values := s.runtime.SecretStore().Get()
	secret, _ := s.declaredCredential(request.Identity, request.Field)
	values[secret.ID] = request.Value
	if err := s.runtime.SecretStore().Persist(values); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credential provider rejected provisioning"})
		return
	}
	s.runtime.RecordTelemetry("secrets_updated", map[string]interface{}{"count": 1})
	writeJSON(w, http.StatusCreated, map[string]string{"status": "provisioned", "identity": request.Identity, "field": request.Field})
}

func (s *Server) handleCredentialDelete(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeCredentialRequest(w, r)
	if !ok || !s.requireDeclaredCredential(w, request) {
		return
	}
	values := s.runtime.SecretStore().Get()
	secret, _ := s.declaredCredential(request.Identity, request.Field)
	delete(values, secret.ID)
	if err := s.runtime.SecretStore().Persist(values); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credential provider rejected deletion"})
		return
	}
	s.runtime.RecordTelemetry("secrets_updated", map[string]interface{}{"count": 1})
	w.WriteHeader(http.StatusNoContent)
}
