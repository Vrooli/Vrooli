package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var supportedSourceKinds = map[string]bool{
	"native": true, "onepassword": true, "bitwarden_secrets": true, "bitwarden_vault": true,
}

type sourceCapabilities struct {
	Metadata bool `json:"metadata"`
	Read     bool `json:"read"`
	Write    bool `json:"write"`
	Verify   bool `json:"verify"`
	Rotate   bool `json:"rotate"`
	Revoke   bool `json:"revoke"`
	Export   bool `json:"export"`
}

type sourceRecord struct {
	ID, WorkspaceID, Kind, Label, Endpoint, BootstrapRef, Status, LastError string
	Capabilities                                                            sourceCapabilities
	CreatedAt, UpdatedAt                                                    time.Time
}

type sourceInput struct {
	Kind         string `json:"kind"`
	Label        string `json:"label"`
	Endpoint     string `json:"endpoint"`
	BootstrapRef string `json:"bootstrap_ref"`
}

type sourceBinding struct {
	ItemID         string `json:"item_id"`
	SourceID       string `json:"source_id"`
	ExternalRef    string `json:"external_ref"`
	SourceRevision string `json:"source_revision,omitempty"`
}

func capabilitiesForSource(kind string) sourceCapabilities {
	switch kind {
	case "native":
		return sourceCapabilities{Metadata: true, Read: true, Write: true, Verify: true, Rotate: true, Revoke: true, Export: true}
	case "onepassword":
		return sourceCapabilities{Metadata: true, Read: true, Verify: true}
	case "bitwarden_secrets":
		return sourceCapabilities{Metadata: true, Read: true, Write: true, Verify: true, Rotate: true, Revoke: true}
	case "bitwarden_vault":
		return sourceCapabilities{Metadata: true, Read: true, Write: true, Verify: true, Export: true}
	default:
		return sourceCapabilities{}
	}
}

func validateSourceEndpoint(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("source endpoint must be an https origin without credentials or query state")
	}
	return nil
}

func (p *PasswordManager) createSource(ctx context.Context, workspace, actor string, input sourceInput) (sourceRecord, error) {
	input.Kind = strings.TrimSpace(strings.ToLower(input.Kind))
	input.Label = strings.TrimSpace(input.Label)
	input.Endpoint = strings.TrimRight(strings.TrimSpace(input.Endpoint), "/")
	input.BootstrapRef = strings.TrimSpace(input.BootstrapRef)
	if !supportedSourceKinds[input.Kind] {
		return sourceRecord{}, errors.New("unsupported source kind")
	}
	if input.Label == "" || len([]rune(input.Label)) > 160 {
		return sourceRecord{}, errors.New("source label is required and bounded")
	}
	if err := validateSourceEndpoint(input.Endpoint); err != nil {
		return sourceRecord{}, err
	}
	if input.Kind != "native" && input.BootstrapRef == "" {
		return sourceRecord{}, errors.New("external source bootstrap_ref is required")
	}
	now := time.Now().UTC()
	status := "unverified"
	if input.Kind == "native" {
		status = "configured"
	}
	source := sourceRecord{ID: uuid.NewString(), WorkspaceID: workspace, Kind: input.Kind, Label: input.Label, Endpoint: input.Endpoint, BootstrapRef: input.BootstrapRef, Capabilities: capabilitiesForSource(input.Kind), Status: status, CreatedAt: now, UpdatedAt: now}
	if p.memory {
		p.mu.Lock()
		p.sources[source.ID] = source
		p.mu.Unlock()
		p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "source.create", Outcome: "success", Detail: source.Kind + " source metadata registered"})
		return source, nil
	}
	capabilities, _ := json.Marshal(source.Capabilities)
	_, err := p.db.ExecContext(ctx, `INSERT INTO pm_sources(id,workspace_id,kind,label,endpoint,bootstrap_ref,capabilities,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, source.ID, source.WorkspaceID, source.Kind, source.Label, source.Endpoint, source.BootstrapRef, string(capabilities), source.Status)
	if err == nil {
		p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "source.create", Outcome: "success", Detail: source.Kind + " source metadata registered"})
	}
	return source, err
}

func (p *PasswordManager) listSources(ctx context.Context, workspace string) ([]sourceRecord, error) {
	if p.memory {
		p.mu.RLock()
		defer p.mu.RUnlock()
		result := make([]sourceRecord, 0)
		for _, source := range p.sources {
			if source.WorkspaceID == workspace {
				result = append(result, source)
			}
		}
		return result, nil
	}
	rows, err := p.db.QueryContext(ctx, `SELECT id,workspace_id,kind,label,COALESCE(endpoint,''),COALESCE(bootstrap_ref,''),capabilities,status,COALESCE(last_error,''),created_at,updated_at FROM pm_sources WHERE workspace_id=$1 ORDER BY created_at DESC LIMIT 100`, workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]sourceRecord, 0)
	for rows.Next() {
		var source sourceRecord
		var capabilities, created, updated string
		if err := rows.Scan(&source.ID, &source.WorkspaceID, &source.Kind, &source.Label, &source.Endpoint, &source.BootstrapRef, &capabilities, &source.Status, &source.LastError, &created, &updated); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(capabilities), &source.Capabilities)
		source.CreatedAt, source.UpdatedAt = parseTime(created), parseTime(updated)
		result = append(result, source)
	}
	return result, rows.Err()
}

func (p *PasswordManager) getSource(ctx context.Context, workspace, id string) (sourceRecord, error) {
	if p.memory {
		p.mu.RLock()
		source, ok := p.sources[id]
		p.mu.RUnlock()
		if !ok || source.WorkspaceID != workspace {
			return sourceRecord{}, sql.ErrNoRows
		}
		return source, nil
	}
	var source sourceRecord
	var capabilities, created, updated string
	err := p.db.QueryRowContext(ctx, `SELECT id,workspace_id,kind,label,COALESCE(endpoint,''),COALESCE(bootstrap_ref,''),capabilities,status,COALESCE(last_error,''),created_at,updated_at FROM pm_sources WHERE id=$1 AND workspace_id=$2`, id, workspace).Scan(&source.ID, &source.WorkspaceID, &source.Kind, &source.Label, &source.Endpoint, &source.BootstrapRef, &capabilities, &source.Status, &source.LastError, &created, &updated)
	if err != nil {
		return sourceRecord{}, err
	}
	_ = json.Unmarshal([]byte(capabilities), &source.Capabilities)
	source.CreatedAt, source.UpdatedAt = parseTime(created), parseTime(updated)
	return source, nil
}

func (p *PasswordManager) bindSource(ctx context.Context, workspace, actor, vaultID, itemID string, binding sourceBinding) error {
	if binding.ExternalRef == "" || len(binding.ExternalRef) > 512 {
		return errors.New("source external_ref is required and bounded")
	}
	if _, err := p.metadata(ctx, workspace, vaultID, itemID); err != nil {
		return err
	}
	source, err := p.getSource(ctx, workspace, binding.SourceID)
	if err != nil {
		return err
	}
	if source.Status == "unconfigured" {
		return errors.New("source bootstrap is unavailable")
	}
	binding.ItemID, binding.SourceID = itemID, source.ID
	if p.memory {
		p.mu.Lock()
		p.bindings[itemID] = binding
		p.mu.Unlock()
		p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "source.bind", ItemID: itemID, Outcome: "success", Detail: "explicit source reference"})
		return nil
	}
	_, err = p.db.ExecContext(ctx, `INSERT INTO pm_source_bindings(item_id,source_id,external_ref,source_revision) VALUES($1,$2,$3,$4) ON CONFLICT(item_id) DO UPDATE SET source_id=excluded.source_id,external_ref=excluded.external_ref,source_revision=excluded.source_revision`, itemID, source.ID, binding.ExternalRef, binding.SourceRevision)
	if err == nil {
		p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "source.bind", ItemID: itemID, Outcome: "success", Detail: "explicit source reference"})
	}
	return err
}

func (p *PasswordManager) sourceHealth(ctx context.Context, workspace, id string) (sourceRecord, error) {
	source, err := p.getSource(ctx, workspace, id)
	if err != nil {
		return sourceRecord{}, err
	}
	// Provider calls are deliberately not guessed here. Until an owner adapter
	// produces a capability receipt, report the source as unavailable and never
	// read from the native vault or another configured source as a fallback.
	if source.Kind != "native" {
		source.Status = "unavailable"
		if source.BootstrapRef == "" {
			source.LastError = "authority bootstrap reference is unavailable"
		} else {
			source.LastError = "provider adapter is unavailable; no fallback source was consulted"
		}
	}
	return source, nil
}

func sourcePublic(source sourceRecord) map[string]any {
	return map[string]any{"id": source.ID, "workspace_id": source.WorkspaceID, "kind": source.Kind, "label": source.Label, "endpoint": source.Endpoint, "status": source.Status, "last_error": source.LastError, "capabilities": source.Capabilities, "created_at": source.CreatedAt, "updated_at": source.UpdatedAt}
}

func (h *passwordManagerHandlers) createSource(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityItemWrite); err != nil {
		writeVaultError(w, err)
		return
	}
	var input sourceInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	source, err := h.manager.createSource(r.Context(), workspace, actor, input)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sourcePublic(source))
}

func (h *passwordManagerHandlers) listSources(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	sources, err := h.manager.listSources(r.Context(), workspace)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	result := make([]map[string]any, 0, len(sources))
	for _, source := range sources {
		result = append(result, sourcePublic(source))
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": result})
}

func (h *passwordManagerHandlers) sourceHealth(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	source, err := h.manager.sourceHealth(r.Context(), workspace, mux.Vars(r)["id"])
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sourcePublic(source))
}

func (h *passwordManagerHandlers) bindSource(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityItemWrite); err != nil {
		writeVaultError(w, err)
		return
	}
	var input sourceBinding
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	vars := mux.Vars(r)
	if err := h.manager.bindSource(r.Context(), workspace, actor, vars["vault"], vars["id"], input); err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item_id": vars["id"], "source_id": input.SourceID, "external_ref": input.ExternalRef, "binding": "explicit"})
}
