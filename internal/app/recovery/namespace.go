package recovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// storageAppID mirrors the api-core storage convention: per-scenario class roots
// live under "<class-root>/<app>/<scenario>", and every Vrooli scenario uses the
// default app id "vrooli" (packages/api-core/storage/resolver.go defaultAppID).
// This package does not import api-core/storage, so the convention is mirrored
// here with this comment as the cross-reference. data-backup-manager's
// scenariospec.Inspector resolves the live data dir the same way, so the shadow
// data dir this command emits is a sibling of the live one it is restored from.
const storageAppID = "vrooli"

// NamespaceRequest resolves the storage namespaces of one scenario instance.
type NamespaceRequest struct {
	Scenario string
	// Variant is the instance variant; empty resolves the canonical "live"
	// instance (the CLI defaults it to "shadow", this command's primary use).
	Variant string
}

// NamespaceOutput is the SSOT storage addressing for a (scenario, variant),
// derived from scenarioruntime.InstanceKey.Namespace(). It is the floor query
// `git-control-tower baseline start` consumes to build the data-backup-manager
// `safety populate-shadow` mappings: a module-level scenario CLI cannot import
// the platform InstanceKey SSOT, so the recovery floor emits it here. The field
// names match the safety-target names register-targets assigns ("postgres" →
// PostgresDB, "data" → DataDir), so the caller maps target → location directly.
type NamespaceOutput struct {
	// Scenario is the resolved scenario slug.
	Scenario string `json:"scenario"`
	// Variant is the normalized variant ("live", "shadow", ...).
	Variant string `json:"variant"`
	// InstanceKey is the canonical "<scenario>" / "<scenario>@<variant>" identifier
	// — the registry record slug and the lifecycle `--instance` target.
	InstanceKey string `json:"instanceKey"`
	// PostgresDb is the variant-aware Postgres database name: "vrooli_<scenario>"
	// for live, "vrooli_<scenario>_<variant>" otherwise. The shadow location for a
	// registered "postgres" backup target.
	PostgresDb string `json:"postgresDb"`
	// DataDir is the absolute variant-aware durable-data directory, or "" when the
	// home/data root cannot be resolved. The shadow location for a registered
	// "data" backup target.
	DataDir string `json:"dataDir"`
	// DataDirName is the per-variant data subdirectory name ("<scenario>" for live,
	// "<scenario>@<variant>" otherwise — "@" is filesystem-safe).
	DataDirName string `json:"dataDirName"`
	// StorageNamespace is the variant-aware Redis/Qdrant namespace root
	// ("<scenario>" for live, "<scenario>_<variant>" otherwise) the api-core
	// storage helpers compose per-domain names from.
	StorageNamespace string `json:"storageNamespace"`
}

// Namespace resolves a scenario instance's SSOT storage namespaces. It is pure
// addressing — it reads no manifest and touches no running process; the strings
// come straight from scenarioruntime.InstanceKey.Namespace(), the one place that
// composes them, so no caller re-derives them.
func (s Service) Namespace(req NamespaceRequest) (NamespaceOutput, error) {
	scenario := strings.TrimSpace(req.Scenario)
	if scenario == "" {
		return NamespaceOutput{}, fmt.Errorf("recovery: --scenario is required")
	}

	ns := scenarioruntime.InstanceKey{
		Scenario: scenario,
		Variant:  strings.TrimSpace(req.Variant),
	}.Namespace()

	out := NamespaceOutput{
		Scenario:         scenario,
		Variant:          ns.Variant,
		InstanceKey:      ns.RecordSlug,
		PostgresDb:       ns.PostgresDB,
		DataDirName:      ns.DataDirName,
		StorageNamespace: ns.StorageNamespace,
	}
	if dir, ok := s.scenarioDataDir(ns.DataDirName); ok {
		out.DataDir = dir
	}
	return out, nil
}

// scenarioDataDir resolves the absolute durable-data directory for a variant's
// data subdirectory name, mirroring the api-core storage convention
// (<data-root>/<app>/<dataDirName>). Returns ok=false when the home/data root is
// unresolvable — the data location is then simply omitted, never a hard error
// (Postgres alone still gives a usable mapping).
func (s Service) scenarioDataDir(dataDirName string) (string, bool) {
	home, err := s.homeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", false
	}
	dataRoot, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyData)
	if err != nil || strings.TrimSpace(dataRoot) == "" {
		return "", false
	}
	return filepath.Join(dataRoot, storageAppID, dataDirName), true
}

// homeDir resolves the operator home, honoring the optional test seam.
func (s Service) homeDir() (string, error) {
	if s.HomeDir != nil {
		return s.HomeDir()
	}
	return os.UserHomeDir()
}
