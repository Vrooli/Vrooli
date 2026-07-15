package opsrunner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/agentops"
)

// StoredOverride is one raw override document at an owner's layer plus its
// file-level provenance: the deterministic basename, the canonical content
// digest, and the file's last-modified time. The binding document schema stores
// no timestamps of its own (additionalProperties is false), so mtime is the
// only honest update marker.
type StoredOverride struct {
	Binding   agentops.OperationBinding
	File      string // basename within binding-overrides/
	Revision  string // canonical sha256 digest of the stored bytes
	UpdatedAt string // file mtime, RFC3339
}

// OverrideWriter is the WRITE side of the binding-override store. It is a
// sibling of FSOverrideStore (the resolver's read-only OverrideStore stays
// unchanged): the same on-disk layout, but administering documents instead of
// resolving them. Every write is fail-closed validated (schema +
// owner-kind↔layer via agentops.ValidateBinding) and atomic (temp+rename), and
// the filename is deterministic per operation+version so Put is an
// idempotent replace and Delete is precise.
//
// Binding resolution is SNAPSHOT-at-Invoke: a Put/Delete affects only
// operations started afterwards; a running operation keeps the provenance it
// pinned.
type OverrideWriter struct {
	loc DomainLocator
}

// NewOverrideWriter constructs a writer over a domain locator.
func NewOverrideWriter(loc DomainLocator) *OverrideWriter { return &OverrideWriter{loc: loc} }

// overrideFileName is the deterministic basename for one operation(+version)
// override: "<operation>.json" for a version-agnostic override,
// "<operation>@<version>.json" for an exact pin. Deterministic naming is what
// makes Put an idempotent replace and Delete precise.
func overrideFileName(operation agentops.OperationID, version string) string {
	if version == "" {
		return string(operation) + ".json"
	}
	return string(operation) + "@" + version + ".json"
}

// layerForOwnerKind derives the override layer an owner writes at. Deriving it
// server-side (instead of trusting a caller-supplied layer) means a client can
// never claim a mismatched layer for an owner.
func layerForOwnerKind(kind agentops.TargetKind) (agentops.BindingLayer, string, error) {
	switch kind {
	case agentops.TargetBacklogItem:
		return agentops.LayerBacklogItemOverride, "backlog-item", nil
	case agentops.TargetInitiative:
		return agentops.LayerInitiativeOverride, "initiative", nil
	default:
		return "", "", fmt.Errorf("target kind %q owns no binding-override layer (only backlog-item and initiative do)", kind)
	}
}

// BuildOverride constructs the owner-scoped binding document for a Put: the
// layer and owner are derived from the owner entity, never caller-supplied.
func BuildOverride(ownerKind agentops.TargetKind, ownerID string, operation agentops.OperationID, operationVersion, mode, modeRevision string, disabled bool) (agentops.OperationBinding, error) {
	layer, ownerKindName, err := layerForOwnerKind(ownerKind)
	if err != nil {
		return agentops.OperationBinding{}, err
	}
	return agentops.OperationBinding{
		Kind:             "agentops-operation-binding",
		Operation:        operation,
		OperationVersion: operationVersion,
		Layer:            layer,
		Owner:            &agentops.BindingOwner{Kind: ownerKindName, ID: ownerID},
		Mode:             mode,
		ModeRevision:     modeRevision,
		Disabled:         disabled,
	}, nil
}

func (w *OverrideWriter) overridesDir(kind agentops.TargetKind, id string) (string, error) {
	dir, err := w.loc.AgentOpsDir(kind, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, overridesSubdir), nil
}

// List returns every override document stored at an owner's layer, ordered by
// file name. A malformed document is a fail-closed error naming the offending
// file (mirroring the read store), never silently skipped.
func (w *OverrideWriter) List(kind agentops.TargetKind, id string) ([]StoredOverride, error) {
	dir, err := w.overridesDir(kind, id)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []StoredOverride
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if err := agentops.ValidateBinding(raw); err != nil {
			return nil, fmt.Errorf("invalid binding override %s: %w", path, err)
		}
		b, err := agentops.DecodeBinding(raw)
		if err != nil {
			return nil, err
		}
		rev, err := agentops.CanonicalDigest(raw)
		if err != nil {
			return nil, fmt.Errorf("digest binding override %s: %w", path, err)
		}
		var mtime string
		if info, err := e.Info(); err == nil {
			mtime = info.ModTime().UTC().Format(time.RFC3339)
		}
		out = append(out, StoredOverride{Binding: b, File: e.Name(), Revision: rev, UpdatedAt: mtime})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out, nil
}

// Put validates and atomically writes one override document at the owner's
// layer, replacing any existing document for the same operation+version. It
// fails closed BEFORE writing when:
//
//   - the binding document is invalid (schema or owner-kind↔layer mismatch);
//   - the binding's owner does not match the owner entity being written under; or
//   - the write would introduce same-layer ambiguity: another stored override
//     already covers the same operation under a DIFFERENT version key (the
//     resolver treats two same-layer candidates for one operation as an
//     ambiguous authoring error and fails every resolution for it).
//
// Catalog- and mode-existence checks (operation declared at the pinned version,
// mode revision exists, mode compatible) are the caller's responsibility — the
// writer has no catalog or mode registry by design.
func (w *OverrideWriter) Put(ownerKind agentops.TargetKind, ownerID string, b agentops.OperationBinding) (StoredOverride, error) {
	layer, ownerKindName, err := layerForOwnerKind(ownerKind)
	if err != nil {
		return StoredOverride{}, err
	}
	if b.Layer != layer {
		return StoredOverride{}, fmt.Errorf("binding layer %q does not match owner kind %q (expected %q)", b.Layer, ownerKind, layer)
	}
	if b.Owner == nil || b.Owner.Kind != ownerKindName || b.Owner.ID != ownerID {
		return StoredOverride{}, fmt.Errorf("binding owner must be %s/%s", ownerKindName, ownerID)
	}
	raw, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return StoredOverride{}, err
	}
	if err := agentops.ValidateBinding(raw); err != nil {
		return StoredOverride{}, err
	}

	file := overrideFileName(b.Operation, b.OperationVersion)
	existing, err := w.List(ownerKind, ownerID)
	if err != nil {
		return StoredOverride{}, fmt.Errorf("scan existing overrides: %w", err)
	}
	for _, st := range existing {
		if st.Binding.Operation == b.Operation && st.File != file {
			return StoredOverride{}, fmt.Errorf("override for operation %q already exists as %s: two same-layer overrides for one operation are ambiguous and fail every resolution — delete it first", b.Operation, st.File)
		}
	}

	dir, err := w.overridesDir(ownerKind, ownerID)
	if err != nil {
		return StoredOverride{}, err
	}
	path := filepath.Join(dir, file)
	if err := atomicWrite(path, raw); err != nil {
		return StoredOverride{}, err
	}
	rev, err := agentops.CanonicalDigest(raw)
	if err != nil {
		return StoredOverride{}, err
	}
	return StoredOverride{
		Binding: b, File: file, Revision: rev,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// Delete removes the owner's override for an operation(+version). It returns
// found=false (not an error) when no matching document exists. Beyond the
// deterministic filename it also removes any legacy-named document whose
// decoded binding matches the same operation+version, so Delete stays precise
// against hand-authored files.
func (w *OverrideWriter) Delete(ownerKind agentops.TargetKind, ownerID string, operation agentops.OperationID, operationVersion string) (bool, error) {
	if _, _, err := layerForOwnerKind(ownerKind); err != nil {
		return false, err
	}
	dir, err := w.overridesDir(ownerKind, ownerID)
	if err != nil {
		return false, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	canonical := overrideFileName(operation, operationVersion)
	found := false
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		match := e.Name() == canonical
		if !match {
			raw, err := os.ReadFile(path)
			if err != nil {
				return found, err
			}
			b, err := agentops.DecodeBinding(raw)
			if err != nil {
				continue // an undecodable stray file is not a match
			}
			match = b.Operation == operation && b.OperationVersion == operationVersion
		}
		if !match {
			continue
		}
		if err := os.Remove(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return found, err
		}
		found = true
	}
	return found, nil
}
