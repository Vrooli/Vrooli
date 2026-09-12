// Package retention exposes the operator-invoked maintenance commands for the
// storage budgets this scenario declares in .vrooli/service.json.
//
// These commands work directly on the SQLite file rather than through the API,
// because they exist for the case where the scenario must be stopped: a full
// VACUUM rewrites the entire database and cannot run alongside the ingest path.
// Scheduled, in-process enforcement is a different thing entirely — it needs no
// command and no operator, and it never rewrites the file.
package retention

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/api-core/retention"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	_ "modernc.org/sqlite"
)

// Register wires the retention subcommand group.
//
// NeedsAPI is false: the whole point is to run while the scenario is stopped.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "retention",
		Description: "Inspect and enforce the declared storage budgets (operator-invoked, run with the scenario stopped)",
		NeedsAPI:    false,
		Subcommands: []cliapp.Command{
			{
				Name:        "status",
				Description: "Report each declared budget and what it currently uses. Changes nothing.",
				Run:         func(args []string) error { return status(args) },
			},
			{
				Name:        "enforce",
				Description: "Prune every declared budget to its ceiling, and optionally compact the database",
				Run:         func(args []string) error { return enforce(args) },
			},
		},
	}
}

// budgetState is one budget's declaration plus its current measurement.
type budgetState struct {
	spec   retention.Spec
	path   string
	usage  retention.Usage
	pruner retention.Pruner
}

// open resolves the manifest, the storage paths, and the database handles. It is
// shared by status and enforce so both report the same numbers from the same
// source of truth.
func open(allowFullVacuum bool, batchSize int) ([]budgetState, func(), error) {
	manifest, err := readManifest()
	if err != nil {
		return nil, nil, err
	}
	specs, err := retention.ParseManifest(manifest)
	if err != nil {
		return nil, nil, err
	}
	specs = sqliteTableSpecs(specs)
	if len(specs) == 0 {
		return nil, func() {}, nil
	}

	scenarioID, err := retentionScenarioNamespace(os.Getenv)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve storage namespace: %w", err)
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return nil, nil, fmt.Errorf("create storage resolver: %w", err)
	}
	opts := storage.Options{ScenarioID: scenarioID}

	// One handle per database file, shared across the budgets that name it, so
	// two budgets on autoheal.sqlite do not fight each other for the write lock.
	handles := map[string]*sql.DB{}
	closeAll := func() {
		for _, db := range handles {
			_ = db.Close()
		}
	}

	states := make([]budgetState, 0, len(specs))
	for _, spec := range specs {
		path, err := spec.Target.Resolve(resolver, opts)
		if err != nil {
			closeAll()
			return nil, nil, fmt.Errorf("budget %q: %w", spec.Budget.Name, err)
		}
		db, ok := handles[path]
		if !ok {
			db, err = sql.Open("sqlite", path)
			if err != nil {
				closeAll()
				return nil, nil, fmt.Errorf("open %s: %w", path, err)
			}
			db.SetMaxOpenConns(1)
			handles[path] = db
		}
		pruner, err := retention.NewSQLiteTablePruner(retention.SQLiteTableConfig{
			DB:         db,
			Path:       path,
			Table:      spec.Target.Table,
			TimeColumn: spec.Target.TimeColumn,
			BatchSize:  batchSize,
			// An operator command has no wall-clock allowance: it is expected to
			// take as long as it takes, which on a 453 GB database is hours.
			AllowFullVacuum: allowFullVacuum,
		})
		if err != nil {
			closeAll()
			return nil, nil, err
		}
		states = append(states, budgetState{spec: spec, path: path, pruner: pruner})
	}
	return states, closeAll, nil
}

const retentionScenarioID = "vrooli-autoheal"

// retentionScenarioNamespace prevents a standalone installed CLI from
// inheriting another scenario's lifecycle namespace. This matters when the CLI
// is launched from a terminal embedded in a running scenario: blindly using
// VROOLI_STORAGE_NAMESPACE would open (and potentially prune) that scenario's
// database. An autoheal live/shadow namespace is honored; a foreign live
// namespace falls back to autoheal live, and a foreign non-live namespace
// fails rather than aliasing a shadow onto live.
func retentionScenarioNamespace(getenv func(string) string) (string, error) {
	root := strings.TrimSpace(getenv(storage.EnvStorageNamespace))
	variant := strings.ToLower(strings.TrimSpace(getenv(storage.EnvVariant)))
	if root == retentionScenarioID || strings.HasPrefix(root, retentionScenarioID+"_") {
		ns, err := storage.ResolveNamespace(storage.NamespaceConfig{Root: root, Variant: variant})
		if err != nil {
			return "", err
		}
		return ns.Root(), nil
	}
	if variant != "" && variant != "live" {
		return "", fmt.Errorf("refusing retention for %s: ambient namespace %q belongs to a non-live %q instance", retentionScenarioID, root, variant)
	}
	ns, err := storage.ResolveNamespace(storage.NamespaceConfig{Root: retentionScenarioID, Variant: "live"})
	if err != nil {
		return "", err
	}
	return ns.Root(), nil
}

// sqliteTableSpecs narrows the scenario-wide retention declaration to the
// database targets this offline operator command owns. ParseManifest also
// returns storage.entries directory/file budgets; those are enforced by the
// framework manager and must not make database maintenance unusable.
func sqliteTableSpecs(specs []retention.Spec) []retention.Spec {
	out := make([]retention.Spec, 0, len(specs))
	for _, spec := range specs {
		if spec.Target.Kind == retention.TargetSQLiteTable {
			out = append(out, spec)
		}
	}
	return out
}

// writeJSON renders a value as indented JSON on stdout.
func writeJSON(v any) error {
	encoded, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(encoded))
	return nil
}

// readManifest loads .vrooli/service.json from wherever the CLI is being run.
func readManifest() ([]byte, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	dir := wd
	for {
		candidate := filepath.Join(dir, ".vrooli", "service.json")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return os.ReadFile(candidate)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("no .vrooli/service.json found from %s upward; run this from the scenario directory", wd)
		}
		dir = parent
	}
}

func status(args []string) error {
	fs := support.NewFlagSet("retention status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	states, closeAll, err := open(false, 0)
	if err != nil {
		return err
	}
	defer closeAll()

	ctx := context.Background()
	for i := range states {
		usage, err := states[i].pruner.Measure(ctx)
		if err != nil {
			return fmt.Errorf("measure %q: %w", states[i].spec.Budget.Name, err)
		}
		states[i].usage = usage
	}

	if *jsonOutput {
		return writeJSON(jsonReport(states))
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d declared budget(s). Nothing was changed.", len(states))},
		ResultsHeading: "Budgets",
		Results:        statusLines(states),
		RetrievalHints: []string{
			"vrooli-autoheal retention enforce --dry-run",
			"vrooli-autoheal retention enforce --compact",
		},
	})
}

func statusLines(states []budgetState) []string {
	lines := make([]string, 0, len(states)*2)
	for _, s := range states {
		ceiling := "none"
		if s.spec.Budget.HasByteBound() {
			ceiling = retention.FormatBytes(s.spec.Budget.MaxBytes)
		}
		horizon := "none"
		if s.spec.Budget.HasAgeBound() {
			horizon = s.spec.Budget.MaxAge.String()
		}
		over := ""
		if s.spec.Budget.HasByteBound() && s.usage.Bytes > s.spec.Budget.MaxBytes {
			over = "  OVER CEILING"
		}
		lines = append(lines,
			fmt.Sprintf("%s: %s across %d rows (max_bytes=%s max_age=%s)%s",
				s.spec.Budget.Name, retention.FormatBytes(s.usage.Bytes), s.usage.Items, ceiling, horizon, over),
			fmt.Sprintf("    %s", s.path),
		)
	}
	return lines
}

func jsonReport(states []budgetState) any {
	type entry struct {
		Budget    string `json:"budget"`
		Path      string `json:"path"`
		UsedBytes int64  `json:"used_bytes"`
		Rows      int64  `json:"rows"`
		MaxBytes  int64  `json:"max_bytes"`
		MaxAge    string `json:"max_age"`
		Over      bool   `json:"over_ceiling"`
	}
	out := make([]entry, 0, len(states))
	for _, s := range states {
		out = append(out, entry{
			Budget:    s.spec.Budget.Name,
			Path:      s.path,
			UsedBytes: s.usage.Bytes,
			Rows:      s.usage.Items,
			MaxBytes:  s.spec.Budget.MaxBytes,
			MaxAge:    s.spec.Budget.MaxAge.String(),
			Over:      s.spec.Budget.HasByteBound() && s.usage.Bytes > s.spec.Budget.MaxBytes,
		})
	}
	return map[string]any{"budgets": out}
}

func enforce(args []string) error {
	fs := support.NewFlagSet("retention enforce")
	jsonOutput := cliutil.JSONFlag(fs)
	compact := fs.Bool("compact", false, "Also rewrite the database to reclaim freed pages (slow; needs free space for a full copy)")
	dryRun := fs.Bool("dry-run", false, "Report what would be removed without removing anything")
	rebuild := fs.Bool("rebuild", false, "Rebuild each byte-bounded table around the rows that survive instead of deleting the rest. Orders of magnitude faster when most of a table goes; implies --compact")
	batchSize := fs.Int("batch-size", 0, "Rows per delete statement (0 uses the framework default)")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	states, closeAll, err := open(*compact || *rebuild, *batchSize)
	if err != nil {
		return err
	}
	defer closeAll()

	ctx := context.Background()
	if *dryRun {
		for i := range states {
			usage, err := states[i].pruner.Measure(ctx)
			if err != nil {
				return err
			}
			states[i].usage = usage
		}
		return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
			Summary:        []string{"Dry run. Nothing was removed."},
			ResultsHeading: "Budgets",
			Results:        statusLines(states),
		})
	}

	// The projection is printed BEFORE any rewrite, so an operator can stop a
	// compaction that is about to need more space than the disk has. Pruning
	// always precedes compaction: a full VACUUM writes a complete copy of the
	// RESULT, so its cost is the size after pruning, not before.
	if *compact {
		if err := printCompactionProjection(ctx, states); err != nil {
			return err
		}
	}

	summary := []string{}
	results := []string{}
	for _, s := range states {
		// Byte measurements only, never Measure: counting rows is a full index
		// scan, and on the table this command exists for it takes longer than the
		// delete it would be reporting on.
		sqlitePruner, ok := s.pruner.(*retention.SQLiteTablePruner)
		if !ok {
			return fmt.Errorf("budget %q: expected a sqlite_table pruner", s.spec.Budget.Name)
		}
		before, err := sqlitePruner.DatabaseBytes(ctx)
		if err != nil {
			return err
		}
		started := time.Now()
		// Rebuild only applies to a byte-bounded budget: it sizes the surviving
		// set from the ceiling. Age-bounded budgets still prune normally.
		var result retention.Result
		if *rebuild && s.spec.Budget.HasByteBound() {
			result, err = sqlitePruner.RebuildToBudget(ctx, s.spec.Budget)
		} else {
			result, err = sqlitePruner.Prune(ctx, s.spec.Budget)
		}
		if err != nil {
			return fmt.Errorf("prune %q: %w", s.spec.Budget.Name, err)
		}
		after, err := sqlitePruner.DatabaseBytes(ctx)
		if err != nil {
			return err
		}
		live, err := sqlitePruner.LiveBytes(ctx)
		if err != nil {
			return err
		}
		results = append(results, fmt.Sprintf("%s: removed %d rows in %s; file %s -> %s, live payload %s (bound by %s)",
			s.spec.Budget.Name, result.Deleted, time.Since(started).Round(time.Second),
			retention.FormatBytes(before), retention.FormatBytes(after), retention.FormatBytes(live), result.BoundBy))
		if result.CompactSkipped {
			results = append(results, "    compaction skipped: "+result.CompactSkipReason)
		}
		if result.Incomplete {
			results = append(results, "    incomplete: run again to continue")
		}
	}
	if !*compact && !*rebuild {
		summary = append(summary, "Ran without --compact: rows are gone but freed pages stay in the file.")
	}
	summary = append(summary, fmt.Sprintf("Enforced %d budget(s).", len(states)))

	if *jsonOutput {
		return writeJSON(map[string]any{"results": results})
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Results",
		Results:        results,
		RetrievalHints: []string{"vrooli-autoheal retention status"},
	})
}

// printCompactionProjection reports the projected copy size against available
// free space before anything is rewritten.
func printCompactionProjection(ctx context.Context, states []budgetState) error {
	seen := map[string]bool{}
	for _, s := range states {
		if seen[s.path] {
			continue
		}
		seen[s.path] = true
		sqlitePruner, ok := s.pruner.(*retention.SQLiteTablePruner)
		if !ok {
			continue
		}
		allocated, err := sqlitePruner.DatabaseBytes(ctx)
		if err != nil {
			return err
		}
		live, err := sqlitePruner.LiveBytes(ctx)
		if err != nil {
			return err
		}
		free, err := retention.FreeSpace(filepath.Dir(s.path))
		if err != nil {
			return fmt.Errorf("measure free space for %s: %w", s.path, err)
		}
		fmt.Fprintf(os.Stdout,
			"%s occupies %s on disk holding %s of live payload. Pruning runs first; the compaction copy is the live payload that remains, not the file size. Free space available: %s.\n",
			s.path, retention.FormatBytes(allocated), retention.FormatBytes(live), retention.FormatBytes(free))
	}
	return nil
}
