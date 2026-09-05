package recoverycli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	recoveryapp "github.com/vrooli/vrooli/internal/app/recovery"
	"github.com/vrooli/vrooli/internal/baselinefloor"
	"github.com/vrooli/vrooli/internal/cliout"
)

func RenderCapture(w io.Writer, format cliout.Format, resp recoveryapp.CaptureOutput) error {
	return renderTransfer(w, format,
		func(w io.Writer) error { return cliout.WriteProtoJSON(w, RecoveryCaptureResponse(resp)) },
		fmt.Sprintf("captured %s/%s", resp.Scenario, resp.Slug),
		[]string{"source: " + filepath.ToSlash(resp.Source), "restore point: " + filepath.ToSlash(resp.RestorePointPath)},
		resp.Stats)
}

func RenderRestore(w io.Writer, format cliout.Format, resp recoveryapp.RestoreOutput) error {
	return renderTransfer(w, format,
		func(w io.Writer) error { return cliout.WriteProtoJSON(w, RecoveryRestoreResponse(resp)) },
		fmt.Sprintf("restored %s/%s", resp.Scenario, resp.Slug),
		[]string{"restore point: " + filepath.ToSlash(resp.RestorePointPath), "dest: " + filepath.ToSlash(resp.Dest)},
		resp.Stats)
}

func renderTransfer(w io.Writer, format cliout.Format, writeJSON func(io.Writer) error, headline string, detail []string, stats baselinefloor.CopyStats) error {
	return cliout.RenderJSONOr(w, format, writeJSON, func(w io.Writer) error {
		_, _ = fmt.Fprintln(w, headline)
		for _, line := range detail {
			_, _ = fmt.Fprintln(w, "  "+line)
		}
		printStats(w, stats)
		return nil
	})
}

func RenderEngagement(w io.Writer, format cliout.Format, resp recoveryapp.EngagementView) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, RecoveryEngagementResponse(resp)) }, func(w io.Writer) error { printEngagement(w, resp); return nil })
}

func RenderList(w io.Writer, format cliout.Format, resp recoveryapp.ListOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, RecoveryListResponse(resp)) }, func(w io.Writer) error {
		rows := make([][]string, 0, len(resp.Engagements))
		for _, e := range resp.Engagements {
			rows = append(rows, []string{engagementSlug(e.Manifest), string(e.Mode), e.Variant, ttlLabel(e), expiryLabel(e)})
		}
		return cliout.WriteSection(w, cliout.Section{Empty: cliout.EmptyLabel("active engagements"), Rows: rows})
	})
}

func RenderClean(w io.Writer, format cliout.Format, resp recoveryapp.CleanOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, RecoveryCleanResponse(resp)) }, func(w io.Writer) error {
		_, _ = fmt.Fprintf(w, "cleaned %s/%s\n", resp.Scenario, resp.Slug)
		_, _ = fmt.Fprintf(w, "  removed: %s\n", filepath.ToSlash(resp.EngagementDir))
		return nil
	})
}

func RenderMigrate(w io.Writer, format cliout.Format, resp recoveryapp.MigrateOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, RecoveryMigrateResponse(resp)) }, func(w io.Writer) error {
		if resp.FastPath {
			_, _ = fmt.Fprintf(w, "no migrations for %s/%s — shape-unchanged fast path (DB handling skipped)\n", resp.Scenario, resp.Slug)
			return nil
		}
		verb := "migrated"
		if resp.DryRun {
			verb = "dry-ran migrations for"
		}
		_, _ = fmt.Fprintf(w, "%s %s/%s (engine=%s)\n", verb, resp.Scenario, resp.Slug, resp.Engine)
		if resp.Database != "" {
			_, _ = fmt.Fprintf(w, "  database: %s\n", filepath.ToSlash(resp.Database))
		}
		_, _ = fmt.Fprintf(w, "  scripts: %d seen, %d applied, %d already-applied\n", resp.ScriptsSeen, len(resp.Applied), len(resp.Skipped))
		if len(resp.Applied) > 0 {
			_, _ = fmt.Fprintf(w, "  applied: %s\n", strings.Join(resp.Applied, ", "))
		}
		return nil
	})
}

func RenderNamespace(w io.Writer, format cliout.Format, resp recoveryapp.NamespaceOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, RecoveryNamespaceResponse(resp)) }, func(w io.Writer) error {
		_, _ = fmt.Fprintf(w, "namespace: %s\n", resp.InstanceKey)
		_, _ = fmt.Fprintf(w, "  variant: %s\n", resp.Variant)
		_, _ = fmt.Fprintf(w, "  postgres db: %s\n", resp.PostgresDb)
		if resp.DataDir != "" {
			_, _ = fmt.Fprintf(w, "  data dir: %s\n", filepath.ToSlash(resp.DataDir))
		}
		_, _ = fmt.Fprintf(w, "  storage namespace: %s\n", resp.StorageNamespace)
		return nil
	})
}

func printEngagement(w io.Writer, e recoveryapp.EngagementView) {
	_, _ = fmt.Fprintf(w, "engagement: %s\n", engagementSlug(e.Manifest))
	_, _ = fmt.Fprintf(w, "  mode: %s\n", e.Mode)
	_, _ = fmt.Fprintf(w, "  variant: %s\n", e.Variant)
	_, _ = fmt.Fprintf(w, "  restore point: %s\n", filepath.ToSlash(e.RestorePointPath))
	if e.ShadowInstanceKey != "" {
		_, _ = fmt.Fprintf(w, "  shadow instance: %s\n", e.ShadowInstanceKey)
	}
	if e.AmbientVar != "" {
		_, _ = fmt.Fprintf(w, "  ambient var: %s\n", e.AmbientVar)
	}
	if e.AnchorBaselineName != "" {
		_, _ = fmt.Fprintf(w, "  anchor: %s\n", e.AnchorBaselineName)
	}
	_, _ = fmt.Fprintf(w, "  ttl: %s\n", ttlLabel(e))
	_, _ = fmt.Fprintf(w, "  expires: %s\n", expiryLabel(e))
}

func printStats(w io.Writer, s baselinefloor.CopyStats) {
	_, _ = fmt.Fprintf(w, "  copied: %d dirs, %d files (%d reflink, %d deep-copy), %d symlinks, %d bytes; %d excluded\n",
		s.Dirs, s.ReflinkFiles+s.DeepCopyFiles, s.ReflinkFiles, s.DeepCopyFiles, s.Symlinks, s.BytesCopied, s.Excluded)
}

func engagementSlug(m baselinefloor.Manifest) string {
	return m.Scenario + "/" + m.Slug
}

func ttlLabel(e recoveryapp.EngagementView) string {
	if e.TTL <= 0 {
		return "none (heartbeat)"
	}
	return e.TTL.String()
}

func expiryLabel(e recoveryapp.EngagementView) string {
	if e.ExpiresAt == nil {
		return "never (idle)"
	}
	state := "ok"
	if e.Expired {
		state = "EXPIRED"
	}
	return fmt.Sprintf("%s (%s)", e.ExpiresAt.Format(time.RFC3339), state)
}
