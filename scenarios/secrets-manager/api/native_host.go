package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Native-host requests are a deliberately smaller authority surface than the
// ordinary vault API. The transport token authenticates the installed host,
// while the owner token in Authorization authenticates the human session. A
// host without both credentials remains unavailable.
type nativeHostEnrollmentInput struct {
	ExtensionID string `json:"extension_id"`
	Origin      string `json:"origin"`
}

type nativeHostUnlockInput struct {
	ExtensionID string `json:"extension_id"`
	WorkspaceID string `json:"workspace_id"`
	GrantID     string `json:"grant_id"`
	Origin      string `json:"origin"`
	TabID       string `json:"tab_id"`
	FrameID     string `json:"frame_id"`
	DocumentID  string `json:"document_id"`
	ItemID      string `json:"item_id"`
	Revision    int    `json:"item_revision"`
	Field       string `json:"field"`
}

type nativeHostFillInput = nativeHostUnlockInput

type nativeHostMetadataInput struct {
	ExtensionID string `json:"extension_id"`
	WorkspaceID string `json:"workspace_id"`
	Origin      string `json:"origin"`
	TabID       string `json:"tab_id"`
	FrameID     string `json:"frame_id"`
	DocumentID  string `json:"document_id"`
}

type nativeHostMetadataRecord struct {
	GrantID  string `json:"grant_id"`
	ItemID   string `json:"item_id"`
	VaultID  string `json:"vault_id,omitempty"`
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
	URI      string `json:"uri,omitempty"`
	Type     string `json:"type"`
	Revision int    `json:"revision"`
}

type nativeHostSaveInput struct {
	ExtensionID string `json:"extension_id"`
	WorkspaceID string `json:"workspace_id"`
	VaultID     string `json:"vault_id"`
	Origin      string `json:"origin"`
	TabID       string `json:"tab_id"`
	FrameID     string `json:"frame_id"`
	DocumentID  string `json:"document_id"`
	ItemID      string `json:"item_id"`
	Revision    int    `json:"item_revision"`
	Name        string `json:"name"`
	Username    string `json:"username"`
	URI         string `json:"uri"`
	Password    string `json:"password"`
	TOTP        string `json:"totp"`
}

func (h *passwordManagerHandlers) nativeHostRegisterRoutes(r *mux.Router) {
	r.HandleFunc("/native-host/enroll", h.nativeHostEnroll).Methods(http.MethodPost)
	r.HandleFunc("/native-host/unlock", h.nativeHostUnlock).Methods(http.MethodPost)
	r.HandleFunc("/native-host/fill", h.nativeHostFill).Methods(http.MethodPost)
	r.HandleFunc("/native-host/metadata", h.nativeHostMetadata).Methods(http.MethodPost)
	r.HandleFunc("/native-host/save", h.nativeHostSave).Methods(http.MethodPost)
	r.HandleFunc("/native-host/update", h.nativeHostUpdate).Methods(http.MethodPost)
	r.HandleFunc("/native-host/revoke", h.nativeHostRevoke).Methods(http.MethodPost)
}

func requireNativeHostTransport(r *http.Request) error {
	expected, _ := configuredSecretsManagerNativeHostTransport()
	provided := strings.TrimSpace(r.Header.Get("X-Secrets-Manager-Native-Host-Token"))
	if expected == "" || provided == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
		return errAccessDenied
	}
	return nil
}

func validateNativeHostIdentity(extensionID, origin string) error {
	if len(extensionID) < 1 || len(extensionID) > 128 {
		return errors.New("extension_id is invalid")
	}
	for _, char := range extensionID {
		if !(char >= 'A' && char <= 'Z') && !(char >= 'a' && char <= 'z') && !(char >= '0' && char <= '9') && char != '.' && char != '_' && char != '-' {
			return errors.New("extension_id is invalid")
		}
	}
	return validateNativeOrigin(origin)
}

func validateNativeOrigin(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("origin must be an absolute http(s) origin")
	}
	return nil
}

func nativeOriginMatches(approved, requested string) bool {
	if validateNativeOrigin(approved) != nil || validateNativeOrigin(requested) != nil {
		return false
	}
	return strings.EqualFold(strings.TrimRight(approved, "/"), strings.TrimRight(requested, "/"))
}

func validateNativeField(field string) error {
	switch strings.TrimSpace(field) {
	case "password", "username", "token", "api_key", "secret", "totp":
		return nil
	default:
		return errors.New("field is not fillable through the native host")
	}
}

func (h *passwordManagerHandlers) nativeHostEnroll(w http.ResponseWriter, r *http.Request) {
	if err := requireNativeHostTransport(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMemberManage); err != nil {
		writeVaultError(w, err)
		return
	}
	var input nativeHostEnrollmentInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := validateNativeHostIdentity(input.ExtensionID, input.Origin); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	if err := h.manager.enrollNativeHost(r.Context(), workspace, actor, input.ExtensionID, input.Origin); err != nil {
		writeVaultError(w, err)
		return
	}
	h.manager.recordAudit(r.Context(), auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "native_host.enroll", Outcome: "success", Detail: "extension origin enrolled"})
	writeJSON(w, http.StatusOK, map[string]any{"extension_id": input.ExtensionID, "origin": input.Origin, "workspace_id": workspace, "status": "enrolled"})
}

func (h *passwordManagerHandlers) nativeHostUnlock(w http.ResponseWriter, r *http.Request) {
	if err := requireNativeHostTransport(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilitySecretReveal); err != nil {
		writeVaultError(w, err)
		return
	}
	var input nativeHostUnlockInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := validateNativeHostInput(input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	if input.WorkspaceID != workspace || !h.manager.nativeHostEnrolled(r.Context(), workspace, input.ExtensionID, input.Origin) {
		writeVaultError(w, errAccessDenied)
		return
	}
	grant, err := h.manager.getGrant(r.Context(), workspace, input.GrantID)
	if err != nil || !h.manager.grantAllows(r.Context(), grant, actor, input.ItemID, "use", time.Now().UTC()) || !nativeOriginMatches(grant.Target, input.Origin) {
		writeVaultError(w, errAccessDenied)
		return
	}
	item, err := h.manager.metadata(r.Context(), workspace, grant.VaultID, input.ItemID)
	if err != nil || item.Revision != input.Revision {
		writeVaultError(w, errItemConflict)
		return
	}
	if err := h.manager.nativeVaultAvailable(r.Context(), workspace, grant.VaultID); err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "unlocked", "expires_at": time.Now().UTC().Add(2 * time.Minute), "item_id": input.ItemID, "revision": input.Revision, "field": input.Field})
}

func (h *passwordManagerHandlers) nativeHostFill(w http.ResponseWriter, r *http.Request) {
	if err := requireNativeHostTransport(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilitySecretReveal); err != nil {
		writeVaultError(w, err)
		return
	}
	var input nativeHostFillInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := validateNativeHostInput(input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	if input.WorkspaceID != workspace || !h.manager.nativeHostEnrolled(r.Context(), workspace, input.ExtensionID, input.Origin) {
		writeVaultError(w, errAccessDenied)
		return
	}
	grant, err := h.manager.getGrant(r.Context(), workspace, input.GrantID)
	if err != nil || !h.manager.grantAllows(r.Context(), grant, actor, input.ItemID, "use", time.Now().UTC()) || !nativeOriginMatches(grant.Target, input.Origin) {
		writeVaultError(w, errAccessDenied)
		return
	}
	item, err := h.manager.metadata(r.Context(), workspace, grant.VaultID, input.ItemID)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	if item.Revision != input.Revision {
		writeVaultError(w, errItemConflict)
		return
	}
	value, err := h.manager.nativeCredential(r.Context(), workspace, grant.VaultID, input.ItemID, input.Field)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	h.manager.recordAudit(r.Context(), auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "native_host.fill", ItemID: input.ItemID, Outcome: "success", Detail: "origin and revision bound field fill"})
	writeJSON(w, http.StatusOK, map[string]any{"credential": value, "field": input.Field, "item_id": input.ItemID, "revision": input.Revision})
}

func (h *passwordManagerHandlers) nativeHostMetadata(w http.ResponseWriter, r *http.Request) {
	if err := requireNativeHostTransport(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	var input nativeHostMetadataInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := validateNativeHostMetadataInput(input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	if input.WorkspaceID != workspace || !h.manager.nativeHostEnrolled(r.Context(), workspace, input.ExtensionID, input.Origin) {
		writeVaultError(w, errAccessDenied)
		return
	}
	grants, err := h.manager.listGrants(r.Context(), workspace)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	items := make([]nativeHostMetadataRecord, 0)
	seen := make(map[string]struct{})
	now := time.Now().UTC()
	for _, grant := range grants {
		if !h.manager.grantAllows(r.Context(), grant, actor, grant.ItemID, "use", now) || !nativeOriginMatches(grant.Target, input.Origin) {
			continue
		}
		candidates := make([]VaultItem, 0, 1)
		if grant.ItemID != "" {
			item, metadataErr := h.manager.metadata(r.Context(), workspace, grant.VaultID, grant.ItemID)
			if metadataErr == nil {
				candidates = append(candidates, item)
			}
		} else {
			candidates, err = h.manager.listItems(r.Context(), workspace, grant.VaultID, "", false, 1, 200)
			if err != nil {
				writeVaultError(w, err)
				return
			}
		}
		for _, item := range candidates {
			if item.Trashed {
				continue
			}
			key := grant.ID + "\x00" + item.ID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			items = append(items, nativeHostMetadataRecord{GrantID: grant.ID, ItemID: item.ID, VaultID: item.VaultID, Name: item.Name, Username: item.Username, URI: item.URI, Type: item.Type, Revision: item.Revision})
		}
	}
	h.manager.recordAudit(r.Context(), auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "native_host.metadata", Outcome: "success", Detail: "credential-free origin-bound account metadata"})
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "origin": input.Origin})
}

func (h *passwordManagerHandlers) nativeHostSave(w http.ResponseWriter, r *http.Request) {
	h.nativeHostWriteItem(w, r, false)
}

func (h *passwordManagerHandlers) nativeHostUpdate(w http.ResponseWriter, r *http.Request) {
	h.nativeHostWriteItem(w, r, true)
}

func (h *passwordManagerHandlers) nativeHostWriteItem(w http.ResponseWriter, r *http.Request, update bool) {
	if err := requireNativeHostTransport(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityItemWrite); err != nil {
		writeVaultError(w, err)
		return
	}
	var input nativeHostSaveInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := validateNativeHostSaveInput(input, update); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	if input.WorkspaceID != workspace || !h.manager.nativeHostEnrolled(r.Context(), workspace, input.ExtensionID, input.Origin) {
		writeVaultError(w, errAccessDenied)
		return
	}
	itemInput := createItemInput{Type: "login", Name: input.Name, Username: input.Username, URI: input.URI, Fields: map[string]string{"password": input.Password}}
	if input.TOTP != "" {
		itemInput.Fields["totp"] = input.TOTP
	}
	var item VaultItem
	var err error
	if update {
		item, err = h.manager.updateItem(r.Context(), workspace, input.VaultID, actor, input.ItemID, input.Revision, itemInput)
	} else {
		item, err = h.manager.createItem(r.Context(), workspace, input.VaultID, actor, "", itemInput)
	}
	if err != nil {
		writeVaultError(w, err)
		return
	}
	h.manager.recordAudit(r.Context(), auditEvent{WorkspaceID: workspace, ActorID: actor, Action: map[bool]string{true: "native_host.update", false: "native_host.save"}[update], ItemID: item.ID, Outcome: "success", Detail: "explicit origin-bound browser write"})
	writeJSON(w, map[bool]int{true: http.StatusOK, false: http.StatusCreated}[update], nativeHostMetadataRecord{ItemID: item.ID, VaultID: item.VaultID, Name: item.Name, Username: item.Username, URI: item.URI, Type: item.Type, Revision: item.Revision})
}

func (h *passwordManagerHandlers) nativeHostRevoke(w http.ResponseWriter, r *http.Request) {
	if err := requireNativeHostTransport(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMemberManage); err != nil {
		writeVaultError(w, err)
		return
	}
	var input nativeHostEnrollmentInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := validateNativeHostIdentity(input.ExtensionID, input.Origin); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	if err := h.manager.revokeNativeHost(r.Context(), workspace, input.ExtensionID, input.Origin); err != nil {
		writeVaultError(w, err)
		return
	}
	h.manager.recordAudit(r.Context(), auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "native_host.revoke", Outcome: "success", Detail: "extension enrollment revoked"})
	writeJSON(w, http.StatusOK, map[string]any{"extension_id": input.ExtensionID, "status": "revoked"})
}

func validateNativeHostInput(input nativeHostUnlockInput) error {
	if err := validateNativeHostIdentity(input.ExtensionID, input.Origin); err != nil {
		return err
	}
	if input.WorkspaceID == "" || input.GrantID == "" || input.ItemID == "" || input.TabID == "" || input.FrameID == "" || input.DocumentID == "" || input.Revision < 0 {
		return errors.New("native host scope is incomplete")
	}
	return validateNativeField(input.Field)
}

func validateNativeHostMetadataInput(input nativeHostMetadataInput) error {
	if err := validateNativeHostIdentity(input.ExtensionID, input.Origin); err != nil {
		return err
	}
	if input.WorkspaceID == "" || input.TabID == "" || input.FrameID == "" || input.DocumentID == "" {
		return errors.New("native host metadata scope is incomplete")
	}
	return nil
}

func validateNativeHostSaveInput(input nativeHostSaveInput, update bool) error {
	if err := validateNativeHostIdentity(input.ExtensionID, input.Origin); err != nil {
		return err
	}
	if input.WorkspaceID == "" || input.VaultID == "" || input.TabID == "" || input.FrameID == "" || input.DocumentID == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.URI) == "" || strings.TrimSpace(input.Password) == "" {
		return errors.New("native host save fields are incomplete")
	}
	if update && (input.ItemID == "" || input.Revision < 1) {
		return errors.New("native host update requires item_id and item_revision")
	}
	parsed, err := url.Parse(strings.TrimSpace(input.URI))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || !nativeOriginMatches(parsed.Scheme+"://"+parsed.Host, input.Origin) {
		return errors.New("uri must belong to the current origin")
	}
	if len(input.Name) > 512 || len(input.Username) > 1024 || len(input.URI) > 4096 || len(input.Password) > 32768 || len(input.TOTP) > 128 {
		return errors.New("native host save field exceeds its bound")
	}
	return nil
}

func (p *PasswordManager) enrollNativeHost(ctx context.Context, workspace, actor, extensionID, origin string) error {
	key := workspace + "\x00" + extensionID
	if p.memory {
		p.mu.Lock()
		defer p.mu.Unlock()
		if existing, ok := p.nativeHosts[key]; ok && existing.Status == "active" && existing.Origin != origin {
			return errAccessDenied
		}
		p.nativeHosts[key] = nativeHostEnrollment{WorkspaceID: workspace, ExtensionID: extensionID, Origin: origin, Status: "active", EnrolledBy: actor, EnrolledAt: time.Now().UTC()}
		return nil
	}
	var existingOrigin, status string
	err := p.db.QueryRowContext(ctx, `SELECT origin,status FROM pm_native_host_enrollments WHERE workspace_id=$1 AND extension_id=$2`, workspace, extensionID).Scan(&existingOrigin, &status)
	if err == nil && status == "active" && existingOrigin != origin {
		return errAccessDenied
	}
	_, err = p.db.ExecContext(ctx, `INSERT INTO pm_native_host_enrollments(workspace_id,extension_id,origin,enrolled_by,status,revoked_at) VALUES($1,$2,$3,$4,'active',NULL) ON CONFLICT (workspace_id,extension_id) DO UPDATE SET origin=EXCLUDED.origin,enrolled_by=EXCLUDED.enrolled_by,status='active',revoked_at=NULL`, workspace, extensionID, origin, actor)
	return err
}

func (p *PasswordManager) nativeHostEnrolled(ctx context.Context, workspace, extensionID, origin string) bool {
	key := workspace + "\x00" + extensionID
	if p.memory {
		p.mu.RLock()
		enrollment, ok := p.nativeHosts[key]
		p.mu.RUnlock()
		return ok && enrollment.Status == "active" && enrollment.Origin == origin
	}
	var status, storedOrigin string
	if err := p.db.QueryRowContext(ctx, `SELECT status,origin FROM pm_native_host_enrollments WHERE workspace_id=$1 AND extension_id=$2`, workspace, extensionID).Scan(&status, &storedOrigin); err != nil {
		return false
	}
	return status == "active" && storedOrigin == origin
}

func (p *PasswordManager) revokeNativeHost(ctx context.Context, workspace, extensionID, origin string) error {
	key := workspace + "\x00" + extensionID
	if p.memory {
		p.mu.Lock()
		defer p.mu.Unlock()
		enrollment, ok := p.nativeHosts[key]
		if !ok || enrollment.Status != "active" || enrollment.Origin != origin {
			return errAccessDenied
		}
		enrollment.Status = "revoked"
		enrollment.RevokedAt = time.Now().UTC()
		p.nativeHosts[key] = enrollment
		return nil
	}
	result, err := p.db.ExecContext(ctx, `UPDATE pm_native_host_enrollments SET status='revoked',revoked_at=CURRENT_TIMESTAMP WHERE workspace_id=$1 AND extension_id=$2 AND origin=$3 AND status='active'`, workspace, extensionID, origin)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return errAccessDenied
	}
	return nil
}

func (p *PasswordManager) nativeVaultAvailable(ctx context.Context, workspace, vaultID string) error {
	vault, err := p.ensureVault(ctx, workspace, vaultID)
	if err != nil {
		return err
	}
	if vault.Status == "locked" {
		return errVaultLocked
	}
	return nil
}

func (p *PasswordManager) nativeCredential(ctx context.Context, workspace, vaultID, itemID, field string) (string, error) {
	if err := validateNativeField(field); err != nil {
		return "", err
	}
	if err := p.nativeVaultAvailable(ctx, workspace, vaultID); err != nil {
		return "", err
	}
	var record vaultRecord
	if p.memory {
		p.mu.RLock()
		record = p.items[itemID]
		p.mu.RUnlock()
		if record.Workspace != workspace || record.Meta.VaultID != vaultID || record.Meta.Trashed {
			return "", errAccessDenied
		}
	} else {
		var itemType, name, username, uri, folder, tags, created, updated string
		var favorite, trashed bool
		if err := p.db.QueryRowContext(ctx, `SELECT id,vault_id,item_type,name,username,uri,folder,tags,favorite,trashed,version,encrypted_payload,created_at,updated_at FROM pm_items WHERE id=$1 AND workspace_id=$2 AND vault_id=$3`, itemID, workspace, vaultID).Scan(&record.Meta.ID, &record.Meta.VaultID, &itemType, &name, &username, &uri, &folder, &tags, &favorite, &trashed, &record.Meta.Revision, &record.Ciphertext, &created, &updated); err != nil {
			return "", err
		}
		record.Meta.Type, record.Meta.Name, record.Meta.Username, record.Meta.URI, record.Meta.Folder = itemType, name, username, uri, folder
		record.Meta.Favorite, record.Meta.Trashed, record.Workspace = favorite, trashed, workspace
	}
	fields, err := p.decrypt(vaultID, itemID, record.Ciphertext)
	if err != nil {
		return "", err
	}
	value, ok := fields[field]
	if !ok || value == "" {
		return "", errors.New("requested fill field is empty")
	}
	return value, nil
}
