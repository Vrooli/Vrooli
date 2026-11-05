package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	maxPortableItems   = 1000
	maxPortablePayload = 8 * 1024 * 1024
)

type portableItem struct {
	Metadata   VaultItem         `json:"metadata"`
	Ciphertext string            `json:"ciphertext,omitempty"`
	Fields     map[string]string `json:"fields,omitempty"`
}

type portableBundle struct {
	Version int            `json:"version"`
	Format  string         `json:"format"`
	VaultID string         `json:"vault_id"`
	Items   []portableItem `json:"items"`
}

type exportHandle struct {
	ID, Workspace, VaultID, Format, CapabilityDigest, EncryptedPayload string
	Expires                                                            time.Time
	Used                                                               bool
}

type exportInput struct {
	Format        string   `json:"format"`
	ItemIDs       []string `json:"item_ids"`
	RequestDigest string   `json:"request_digest,omitempty"`
}

type importPreviewInput struct {
	Bundle string `json:"bundle"`
}

type importCommitInput struct {
	Bundle          string `json:"bundle"`
	DuplicatePolicy string `json:"duplicate_policy"`
}

func randomCapability() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(token))
	return token, base64.RawURLEncoding.EncodeToString(digest[:]), nil
}

func (p *PasswordManager) exportBundle(ctx context.Context, workspace, actor, vaultID string, input exportInput, assurance string) (exportHandle, string, error) {
	vault, err := p.ensureVault(ctx, workspace, vaultID)
	if err != nil {
		return exportHandle{}, "", err
	}
	if vault.Status == "locked" {
		return exportHandle{}, "", errVaultLocked
	}
	if input.Format == "" {
		input.Format = "native"
	}
	if input.Format != "native" && input.Format != "plaintext" {
		return exportHandle{}, "", errors.New("export format must be native or plaintext")
	}
	if input.Format == "plaintext" {
		if err := p.consumeAssurance(ctx, workspace, actor, "export:"+vaultID, input.RequestDigest, assurance); err != nil {
			return exportHandle{}, "", err
		}
	}
	items, err := p.exportItems(ctx, workspace, vaultID, input.ItemIDs, input.Format == "plaintext")
	if err != nil {
		return exportHandle{}, "", err
	}
	payload, err := json.Marshal(portableBundle{Version: 1, Format: input.Format, VaultID: vaultID, Items: items})
	if err != nil {
		return exportHandle{}, "", err
	}
	if len(payload) > maxPortablePayload {
		return exportHandle{}, "", errors.New("export exceeds the 8 MiB safety bound")
	}
	token, digest, err := randomCapability()
	if err != nil {
		return exportHandle{}, "", err
	}
	handle := exportHandle{ID: uuid.NewString(), Workspace: workspace, VaultID: vaultID, Format: input.Format, CapabilityDigest: digest, Expires: time.Now().UTC().Add(5 * time.Minute)}
	if p.memory {
		handle.EncryptedPayload = base64.RawStdEncoding.EncodeToString(payload)
		p.mu.Lock()
		p.exports[handle.ID] = handle
		p.mu.Unlock()
		return handle, token, nil
	}
	sealed, err := p.encrypt(vaultID, "export/"+handle.ID, map[string]string{"payload": string(payload)})
	if err != nil {
		return exportHandle{}, "", err
	}
	_, err = p.db.ExecContext(ctx, `INSERT INTO pm_export_handles(id,workspace_id,vault_id,format,capability_digest,encrypted_payload,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, handle.ID, workspace, vaultID, input.Format, digest, sealed, handle.Expires)
	if err != nil {
		return exportHandle{}, "", err
	}
	return handle, token, nil
}

func (p *PasswordManager) exportItems(ctx context.Context, workspace, vaultID string, ids []string, plaintext bool) ([]portableItem, error) {
	if len(ids) > maxPortableItems {
		return nil, errors.New("export contains too many items")
	}
	selected := make(map[string]bool, len(ids))
	for _, id := range ids {
		selected[id] = true
	}
	result := make([]portableItem, 0)
	add := func(record vaultRecord) error {
		if record.Workspace != workspace || record.Meta.VaultID != vaultID || record.Meta.Trashed || (len(selected) > 0 && !selected[record.Meta.ID]) {
			return nil
		}
		item := portableItem{Metadata: record.Meta}
		if plaintext {
			fields, err := p.decrypt(vaultID, record.Meta.ID, record.Ciphertext)
			if err != nil {
				return err
			}
			item.Fields = fields
		} else {
			item.Ciphertext = record.Ciphertext
		}
		result = append(result, item)
		return nil
	}
	if p.memory {
		p.mu.RLock()
		defer p.mu.RUnlock()
		for _, record := range p.items {
			if err := add(record); err != nil {
				return nil, err
			}
		}
		return result, nil
	}
	rows, err := p.db.QueryContext(ctx, `SELECT id,vault_id,item_type,name,username,uri,folder,tags,favorite,trashed,version,encrypted_payload,created_at,updated_at FROM pm_items WHERE workspace_id=$1 AND vault_id=$2 AND trashed=FALSE ORDER BY updated_at DESC LIMIT $3`, workspace, vaultID, maxPortableItems)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item VaultItem
		var tags, created, updated, ciphertext string
		if err := rows.Scan(&item.ID, &item.VaultID, &item.Type, &item.Name, &item.Username, &item.URI, &item.Folder, &tags, &item.Favorite, &item.Trashed, &item.Revision, &ciphertext, &created, &updated); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(tags), &item.Tags)
		item.CreatedAt, item.UpdatedAt = parseTime(created), parseTime(updated)
		if err := add(vaultRecord{Meta: item, Workspace: workspace, Ciphertext: ciphertext}); err != nil {
			return nil, err
		}
	}
	return result, rows.Err()
}

func (p *PasswordManager) redeemExport(ctx context.Context, workspace, handleID, token string) (portableBundle, error) {
	digest := sha256.Sum256([]byte(token))
	digestText := base64.RawURLEncoding.EncodeToString(digest[:])
	var vaultID, encrypted string
	if p.memory {
		p.mu.Lock()
		handle, ok := p.exports[handleID]
		if !ok || handle.Workspace != workspace || handle.Used || time.Now().After(handle.Expires) || handle.CapabilityDigest != digestText {
			p.mu.Unlock()
			return portableBundle{}, errAccessDenied
		}
		handle.Used = true
		p.exports[handleID] = handle
		p.mu.Unlock()
		payload, err := base64.RawStdEncoding.DecodeString(handle.EncryptedPayload)
		if err != nil {
			return portableBundle{}, errAccessDenied
		}
		var bundle portableBundle
		if err := json.Unmarshal(payload, &bundle); err != nil {
			return portableBundle{}, errAccessDenied
		}
		return bundle, nil
	}
	var expires string
	if err := p.db.QueryRowContext(ctx, `SELECT vault_id,encrypted_payload,expires_at FROM pm_export_handles WHERE id=$1 AND workspace_id=$2 AND capability_digest=$3 AND used_at IS NULL`, handleID, workspace, digestText).Scan(&vaultID, &encrypted, &expires); err != nil {
		return portableBundle{}, errAccessDenied
	}
	if time.Now().After(parseTime(expires)) {
		return portableBundle{}, errAccessDenied
	}
	fields, err := p.decrypt(vaultID, "export/"+handleID, encrypted)
	if err != nil {
		return portableBundle{}, errAccessDenied
	}
	result, err := p.db.ExecContext(ctx, `UPDATE pm_export_handles SET used_at=CURRENT_TIMESTAMP WHERE id=$1 AND workspace_id=$2 AND used_at IS NULL`, handleID, workspace)
	if err != nil {
		return portableBundle{}, err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return portableBundle{}, errAccessDenied
	}
	var bundle portableBundle
	if err := json.Unmarshal([]byte(fields["payload"]), &bundle); err != nil {
		return portableBundle{}, errAccessDenied
	}
	return bundle, nil
}

func (p *PasswordManager) previewImport(ctx context.Context, workspace, vaultID string, bundle portableBundle) (map[string]any, error) {
	if bundle.Version != 1 || bundle.Format != "native" || len(bundle.Items) > maxPortableItems {
		return nil, errors.New("unsupported or oversized native import")
	}
	duplicates := make([]string, 0)
	unsupported := make([]string, 0)
	for _, item := range bundle.Items {
		if item.Metadata.Name == "" || !supportedVaultItemTypes[item.Metadata.Type] || item.Ciphertext == "" {
			unsupported = append(unsupported, item.Metadata.ID)
			continue
		}
		items, err := p.listItems(ctx, workspace, vaultID, item.Metadata.Name, false, 1, 100)
		if err != nil {
			return nil, err
		}
		if len(items) > 0 {
			duplicates = append(duplicates, item.Metadata.Name)
		}
	}
	return map[string]any{"version": bundle.Version, "format": bundle.Format, "item_count": len(bundle.Items), "duplicate_candidates": duplicates, "unsupported_items": unsupported, "requires_duplicate_policy": len(duplicates) > 0}, nil
}

func (p *PasswordManager) importNative(ctx context.Context, workspace, vaultID, actor string, bundle portableBundle, policy string) (map[string]any, error) {
	if policy != "skip" && policy != "rename" && policy != "replace" {
		return nil, errors.New("duplicate_policy must be skip, rename, or replace")
	}
	if bundle.Version != 1 || bundle.Format != "native" || len(bundle.Items) > maxPortableItems {
		return nil, errors.New("unsupported or oversized native import")
	}
	created, skipped, renamed := 0, 0, 0
	for _, imported := range bundle.Items {
		if imported.Metadata.Name == "" || !supportedVaultItemTypes[imported.Metadata.Type] || imported.Ciphertext == "" {
			skipped++
			continue
		}
		fields, err := p.decrypt(bundle.VaultID, imported.Metadata.ID, imported.Ciphertext)
		if err != nil {
			skipped++
			continue
		}
		name := imported.Metadata.Name
		items, err := p.listItems(ctx, workspace, vaultID, name, false, 1, 100)
		if err != nil {
			return nil, err
		}
		if len(items) > 0 && policy == "skip" {
			skipped++
			continue
		}
		if len(items) > 0 && policy == "replace" {
			if _, err := p.updateItem(ctx, workspace, vaultID, actor, items[0].ID, items[0].Revision, createItemInput{Type: imported.Metadata.Type, Name: name, Username: imported.Metadata.Username, URI: imported.Metadata.URI, Folder: imported.Metadata.Folder, Tags: imported.Metadata.Tags, Favorite: imported.Metadata.Favorite, Fields: fields}); err != nil {
				return nil, err
			}
			created++
			continue
		}
		if len(items) > 0 {
			name += " (imported)"
			renamed++
		}
		if _, err := p.createItem(ctx, workspace, vaultID, actor, uuid.NewString(), createItemInput{Type: imported.Metadata.Type, Name: name, Username: imported.Metadata.Username, URI: imported.Metadata.URI, Folder: imported.Metadata.Folder, Tags: imported.Metadata.Tags, Favorite: imported.Metadata.Favorite, Fields: fields}); err != nil {
			return nil, err
		}
		created++
	}
	return map[string]any{"created": created, "skipped": skipped, "renamed": renamed, "source_vault_id": bundle.VaultID}, nil
}

func (h *passwordManagerHandlers) export(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilitySecretReveal); err != nil {
		writeVaultError(w, err)
		return
	}
	var input exportInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	handle, token, err := h.manager.exportBundle(r.Context(), workspace, actor, mux.Vars(r)["vault"], input, r.Header.Get("X-Assurance-Token"))
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"download_handle": handle.ID, "download_token": token, "expires_at": handle.Expires, "format": handle.Format, "secret_bearing": handle.Format == "plaintext"})
}

func (h *passwordManagerHandlers) redeemExport(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilitySecretReveal); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	bundle, err := h.manager.redeemExport(r.Context(), workspace, mux.Vars(r)["id"], r.Header.Get("X-Download-Token"))
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bundle)
}

func (h *passwordManagerHandlers) previewImport(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityMetadataRead); err != nil {
		writeVaultError(w, err)
		return
	}
	var input importPreviewInput
	if err := decodeJSON(r, &input); err != nil || len(input.Bundle) > maxPortablePayload {
		writeVaultError(w, errors.New("bounded import payload is required"))
		return
	}
	var bundle portableBundle
	if err := json.Unmarshal([]byte(input.Bundle), &bundle); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, _ := requestIdentity(r)
	preview, err := h.manager.previewImport(r.Context(), workspace, mux.Vars(r)["vault"], bundle)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (h *passwordManagerHandlers) commitImport(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilityItemWrite); err != nil {
		writeVaultError(w, err)
		return
	}
	var input importCommitInput
	if err := decodeJSON(r, &input); err != nil || len(input.Bundle) > maxPortablePayload {
		writeVaultError(w, errors.New("bounded import payload is required"))
		return
	}
	var bundle portableBundle
	if err := json.Unmarshal([]byte(input.Bundle), &bundle); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	result, err := h.manager.importNative(r.Context(), workspace, mux.Vars(r)["vault"], actor, bundle, input.DuplicatePolicy)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
