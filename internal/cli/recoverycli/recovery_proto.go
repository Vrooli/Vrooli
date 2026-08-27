package recoverycli

import (
	"time"

	recoveryapp "github.com/vrooli/vrooli/internal/app/recovery"
	"github.com/vrooli/vrooli/internal/baselinefloor"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// formatTime renders an RFC3339Nano timestamp; the zero time maps to "".
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

// recoveryCopyStats maps the internal copy-ladder tallies onto the wire type.
func recoveryCopyStats(s baselinefloor.CopyStats) *cliv1.RecoveryCopyStats {
	return &cliv1.RecoveryCopyStats{
		Dirs:          int32(s.Dirs),
		Symlinks:      int32(s.Symlinks),
		ReflinkFiles:  int32(s.ReflinkFiles),
		DeepCopyFiles: int32(s.DeepCopyFiles),
		BytesCopied:   s.BytesCopied,
		Excluded:      int32(s.Excluded),
	}
}

// RecoveryCaptureResponse maps the internal capture output onto the wire type.
func RecoveryCaptureResponse(resp recoveryapp.CaptureOutput) *cliv1.RecoveryCaptureOutput {
	return &cliv1.RecoveryCaptureOutput{
		Scenario:         resp.Scenario,
		Slug:             resp.Slug,
		Source:           resp.Source,
		RestorePointPath: resp.RestorePointPath,
		Stats:            recoveryCopyStats(resp.Stats),
	}
}

// RecoveryRestoreResponse maps the internal restore output onto the wire type.
func RecoveryRestoreResponse(resp recoveryapp.RestoreOutput) *cliv1.RecoveryRestoreOutput {
	return &cliv1.RecoveryRestoreOutput{
		Scenario:         resp.Scenario,
		Slug:             resp.Slug,
		RestorePointPath: resp.RestorePointPath,
		Dest:             resp.Dest,
		Stats:            recoveryCopyStats(resp.Stats),
	}
}

// recoveryEngagementView maps the internal engagement view (embedded Manifest +
// derived idle-expiry) onto the flattened wire type.
func recoveryEngagementView(e recoveryapp.EngagementView) *cliv1.RecoveryEngagementView {
	expiresAt := ""
	if e.ExpiresAt != nil {
		expiresAt = formatTime(*e.ExpiresAt)
	}
	return &cliv1.RecoveryEngagementView{
		Scenario:           e.Manifest.Scenario,
		Slug:               e.Manifest.Slug,
		Variant:            e.Manifest.Variant,
		Mode:               string(e.Manifest.Mode),
		RestorePointPath:   e.Manifest.RestorePointPath,
		AnchorBaselineName: e.Manifest.AnchorBaselineName,
		AmbientVar:         e.Manifest.AmbientVar,
		ShadowInstanceKey:  e.Manifest.ShadowInstanceKey,
		CreatedAt:          formatTime(e.Manifest.CreatedAt),
		LastTouchedAt:      formatTime(e.Manifest.LastTouchedAt),
		Ttl:                e.Manifest.TTL.String(),
		ExpiresAt:          expiresAt,
		Expired:            e.Expired,
	}
}

// RecoveryEngagementResponse maps an engagement view onto the wire type.
func RecoveryEngagementResponse(resp recoveryapp.EngagementView) *cliv1.RecoveryEngagementView {
	return recoveryEngagementView(resp)
}

// RecoveryListResponse maps the engagement list onto the wire type.
func RecoveryListResponse(resp recoveryapp.ListOutput) *cliv1.RecoveryListOutput {
	out := &cliv1.RecoveryListOutput{}
	for _, e := range resp.Engagements {
		out.Engagements = append(out.Engagements, recoveryEngagementView(e))
	}
	return out
}

// RecoveryCleanResponse maps the internal clean output onto the wire type.
func RecoveryCleanResponse(resp recoveryapp.CleanOutput) *cliv1.RecoveryCleanOutput {
	return &cliv1.RecoveryCleanOutput{
		Scenario:      resp.Scenario,
		Slug:          resp.Slug,
		EngagementDir: resp.EngagementDir,
	}
}

// RecoveryMigrateResponse maps the internal migrate output (embedded
// MigrationResult) onto the flattened wire type.
func RecoveryMigrateResponse(resp recoveryapp.MigrateOutput) *cliv1.RecoveryMigrateOutput {
	return &cliv1.RecoveryMigrateOutput{
		Scenario:           resp.Scenario,
		Slug:               resp.Slug,
		MigrationsDir:      resp.MigrationsDir,
		DbPathAutoResolved: resp.DBPathAutoResolved,
		Engine:             string(resp.MigrationResult.Engine),
		Database:           resp.MigrationResult.Database,
		DryRun:             resp.MigrationResult.DryRun,
		FastPath:           resp.MigrationResult.FastPath,
		ScriptsSeen:        int32(resp.MigrationResult.ScriptsSeen),
		Applied:            resp.MigrationResult.Applied,
		Skipped:            resp.MigrationResult.Skipped,
	}
}

// RecoveryNamespaceResponse maps the internal namespace output onto the wire type.
func RecoveryNamespaceResponse(resp recoveryapp.NamespaceOutput) *cliv1.RecoveryNamespaceOutput {
	return &cliv1.RecoveryNamespaceOutput{
		Scenario:         resp.Scenario,
		Variant:          resp.Variant,
		InstanceKey:      resp.InstanceKey,
		PostgresDb:       resp.PostgresDb,
		DataDir:          resp.DataDir,
		DataDirName:      resp.DataDirName,
		StorageNamespace: resp.StorageNamespace,
	}
}
