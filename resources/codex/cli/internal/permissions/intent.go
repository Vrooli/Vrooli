package permissions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/vrooli/agentharness"
)

// IntentSchemaVersion identifies the intent sidecar format.
const IntentSchemaVersion = 1

// Intent is Vrooli's permission intent for Codex. Codex has no native seam for
// command-pattern rules or for the portable execution document, so the record
// lives in a Vrooli-owned sidecar beside config.toml and config.toml carries
// only keys Codex reads: the projected `[permissions.vrooli]` profile and, when
// activated, its selection and approval settings. The hook projection in
// hooks.json is derived from the deny rules here.
type Intent struct {
	SchemaVersion   int                                `json:"schemaVersion"`
	SettingsPath    string                             `json:"settingsPath"`
	BashDeny        []string                           `json:"bashDeny,omitempty"`
	BashAsk         []string                           `json:"bashAsk,omitempty"`
	BashAllow       []string                           `json:"bashAllow,omitempty"`
	Execution       *agentharness.ExecutionPermissions `json:"execution,omitempty"`
	ExecutionActive bool                               `json:"executionActive,omitempty"`
}

// IntentPath is the sidecar that holds the intent for this adapter's scope.
func (a *Adapter) IntentPath() string {
	name := ".vrooli-permissions-intent.json"
	if a.Scope == ScopeAdmin {
		name = ".vrooli-permissions-intent.admin.json"
	}
	return filepath.Join(filepath.Dir(a.SettingsPath), name)
}

// decodeIntent reads a sidecar snapshot; a missing sidecar is an empty intent.
func decodeIntent(snapshot agentharness.PermissionFileSnapshot) (Intent, error) {
	if !snapshot.Exists {
		return Intent{}, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(snapshot.Data))
	decoder.DisallowUnknownFields()
	var intent Intent
	if err := decoder.Decode(&intent); err != nil {
		return Intent{}, fmt.Errorf("parse %s: %w", snapshot.Path, err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return Intent{}, fmt.Errorf("parse %s: trailing data", snapshot.Path)
	}
	if intent.SchemaVersion != IntentSchemaVersion {
		return Intent{}, fmt.Errorf("%s has schema %d, want %d; use the resource version that wrote it", snapshot.Path, intent.SchemaVersion, IntentSchemaVersion)
	}
	if intent.Execution != nil {
		if err := agentharness.ValidateExecutionPermissions(intent.Execution); err != nil {
			return Intent{}, fmt.Errorf("invalid recorded execution intent in %s: %w", snapshot.Path, err)
		}
	}
	return intent, nil
}

// encodeIntent renders a deterministic sidecar so unchanged intent is
// byte-identical across writes.
func encodeIntent(intent Intent) ([]byte, error) {
	intent.SchemaVersion = IntentSchemaVersion
	intent.BashDeny = sortedCopy(intent.BashDeny)
	intent.BashAsk = sortedCopy(intent.BashAsk)
	intent.BashAllow = sortedCopy(intent.BashAllow)
	if intent.Execution == nil {
		intent.ExecutionActive = false
	}
	data, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
