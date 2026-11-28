package credentialclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type MigrationReport struct {
	SourcePath   string   `json:"source_path"`
	Mapped       []string `json:"mapped"`
	Unmapped     []string `json:"unmapped"`
	Reclassified []string `json:"reclassified,omitempty"`
	Provisioned  []string `json:"provisioned"`
	BundlePath   string   `json:"bundle_path,omitempty"`
	Deleted      bool     `json:"deleted"`
}

// MigrateLegacyJSON moves a legacy plaintext object into the authority. It
// refuses to delete the source until every key has a descriptor and a newly
// exported bundle has been opened successfully.
func MigrateLegacyJSON(ctx context.Context, client Client, path, bundlePath, passphrase string) (MigrationReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return MigrationReport{SourcePath: path}, fmt.Errorf("read legacy credential file: %w", err)
	}
	return migrateLegacyJSONData(ctx, client, path, bundlePath, passphrase, data, os.Remove)
}

// migrateLegacyJSONData contains the migration policy independently from the
// host filesystem. The production wrapper above supplies os.ReadFile and
// os.Remove; keeping the policy injectable lets tests prove deletion ordering
// without creating a plaintext credential file even under a temporary path.
func migrateLegacyJSONData(ctx context.Context, client Client, path, bundlePath, passphrase string, data []byte, removeSource func(string) error) (MigrationReport, error) {
	report := MigrationReport{SourcePath: path, Mapped: []string{}, Unmapped: []string{}, Reclassified: []string{}, Provisioned: []string{}}
	if strings.TrimSpace(bundlePath) == "" || strings.TrimSpace(passphrase) == "" {
		return report, fmt.Errorf("migration requires a recovery bundle path and passphrase")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return report, fmt.Errorf("parse legacy credential file: %w", err)
	}
	refs, err := client.List(ctx)
	if err != nil {
		return report, fmt.Errorf("list credential descriptors: %w", err)
	}
	byKey := make(map[string]CredentialRef)
	for _, ref := range refs {
		if ref.Env != "" {
			byKey[strings.ToUpper(ref.Env)] = ref
		}
		byKey[strings.ToUpper(ref.LogicalID)] = ref
	}
	values := make(map[string]string)
	for key, encoded := range raw {
		if key == "_metadata" {
			continue
		}
		if isLegacyConfigurationKey(key) {
			report.Reclassified = append(report.Reclassified, key)
			continue
		}
		var value string
		if err := json.Unmarshal(encoded, &value); err != nil {
			return report, fmt.Errorf("legacy credential %q is not a string", key)
		}
		ref, ok := byKey[strings.ToUpper(key)]
		if !ok {
			report.Unmapped = append(report.Unmapped, key)
			continue
		}
		label := ref.LogicalID + ":" + ref.Field
		report.Mapped = append(report.Mapped, label)
		values[label] = value
	}
	if len(report.Unmapped) > 0 {
		return report, nil
	}
	for label, value := range values {
		identity, field, ok := strings.Cut(label, ":")
		if !ok {
			return report, fmt.Errorf("mapped credential %q has no field", label)
		}
		status, err := client.Status(ctx, identity, field)
		if err != nil {
			return report, fmt.Errorf("check mapped credential %s: %w", label, err)
		}
		if status.Configured {
			continue
		}
		if _, err := client.Provision(ctx, ProvisionRequest{Identity: identity, Field: field, Value: value}); err != nil {
			return report, fmt.Errorf("provision mapped credential %s: %w", label, err)
		}
		report.Provisioned = append(report.Provisioned, label)
	}
	entries := make([]CredentialRef, 0, len(values))
	for label := range values {
		identity, field, _ := strings.Cut(label, ":")
		entries = append(entries, CredentialRef{LogicalID: identity, Field: field, Required: true})
	}
	export, err := client.RecoveryExport(ctx, RecoveryExportRequest{Entries: entries, Passphrase: passphrase, OutputPath: bundlePath})
	if err != nil {
		return report, fmt.Errorf("export migrated credentials: %w", err)
	}
	report.BundlePath = export.Path
	if _, err := client.RecoveryVerify(ctx, RecoveryVerifyRequest{InputPath: bundlePath, Passphrase: passphrase}); err != nil {
		return report, fmt.Errorf("verify migrated credentials: %w", err)
	}
	if removeSource == nil {
		return report, fmt.Errorf("remove legacy credential file: remover is required")
	}
	if err := removeSource(path); err != nil {
		return report, fmt.Errorf("remove legacy credential file: %w", err)
	}
	report.Deleted = true
	return report, nil
}

func isLegacyConfigurationKey(key string) bool {
	switch strings.ToUpper(strings.TrimSpace(key)) {
	case "POSTGRES_DB", "POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "MAILINABOX_PRIMARY_HOSTNAME", "N8N_EMAIL", "MAILINABOX_ADMIN_EMAIL":
		return true
	default:
		return false
	}
}
