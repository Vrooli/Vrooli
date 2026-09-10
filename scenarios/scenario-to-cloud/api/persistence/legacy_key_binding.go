package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/credentials"
)

// legacyKeyRow is the predecessor shape that carried the operator's SSH key
// path on the wire: manifest.target.vps.key_path, and the persisted
// ssh_identity (explicit_key auth mode) derived from it.
type legacyKeyRow struct {
	id          string
	manifest    []byte
	sshIdentity []byte
}

// convertLegacyKeyPathBindings moves a legacy row's key path reference into
// the credential binding `vrooli/scenario-to-cloud:ssh-key`. The binding
// carries the file locator only; the manifest column is left untouched (the
// domain type no longer reads key_path, so the binding is the single
// authority from here on). Rows that already hold the binding are never
// rewritten, so the conversion is idempotent and safe to run at every start.
func convertLegacyKeyPathBindings(ctx context.Context, db DB) error {
	rows, err := db.QueryContext(ctx, `SELECT id, manifest, ssh_identity FROM deployments`)
	if err != nil {
		return fmt.Errorf("list deployments for key binding conversion: %w", err)
	}
	var legacy []legacyKeyRow
	for rows.Next() {
		var row legacyKeyRow
		var manifest, ident []byte
		if err := rows.Scan(&row.id, &manifest, &ident); err != nil {
			rows.Close()
			return fmt.Errorf("scan deployment for key binding conversion: %w", err)
		}
		row.manifest, row.sshIdentity = manifest, ident
		legacy = append(legacy, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate deployments for key binding conversion: %w", err)
	}
	rows.Close()

	now := time.Now().UTC()
	for _, row := range legacy {
		keyPath := legacyKeyPath(row)
		if keyPath == "" {
			continue
		}
		binding := credentials.NewSSHKeyBinding(row.id, keyPath, now)
		consumers, _ := json.Marshal(binding.ConsumerRefs)
		if _, err := db.ExecContext(ctx, `
			INSERT INTO cloud_credential_bindings (`+credentialBindingColumns+`)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NULL, $12, '', '', $13, $14, $15)
			ON CONFLICT (deployment_id, logical_id, field) DO NOTHING`,
			binding.ID, binding.DeploymentID, binding.Descriptor.LogicalID, binding.Descriptor.Field, string(binding.Class), binding.SourceClass,
			binding.Target.Type, binding.Target.Name, binding.Version.Number, binding.Version.ContentRef, now, string(consumers),
			string(binding.State), now, now,
		); err != nil {
			return fmt.Errorf("convert deployment %s key path into a credential binding: %w", row.id, err)
		}
	}
	return nil
}

// legacyKeyPath reads the key path a legacy row carried: the manifest's
// target.vps.key_path first, then an explicit_key ssh_identity.
func legacyKeyPath(row legacyKeyRow) string {
	if len(row.manifest) > 0 {
		var m struct {
			Target struct {
				VPS *struct {
					KeyPath string `json:"key_path"`
				} `json:"vps"`
			} `json:"target"`
		}
		if json.Unmarshal(row.manifest, &m) == nil && m.Target.VPS != nil && strings.TrimSpace(m.Target.VPS.KeyPath) != "" {
			return strings.TrimSpace(m.Target.VPS.KeyPath)
		}
	}
	if len(row.sshIdentity) > 0 {
		var ident struct {
			KeyPath  string `json:"key_path"`
			AuthMode string `json:"auth_mode"`
		}
		if json.Unmarshal(row.sshIdentity, &ident) == nil && ident.AuthMode == "explicit_key" && strings.TrimSpace(ident.KeyPath) != "" {
			return strings.TrimSpace(ident.KeyPath)
		}
	}
	return ""
}
