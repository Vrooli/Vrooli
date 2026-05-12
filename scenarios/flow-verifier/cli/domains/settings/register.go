// Package settings is the CLI's UI/CLI-preferences command surface,
// backed by the SQLite store the settings API domain owns. Opens the
// same database file the API uses (resolved via
// internal/database.DefaultDSN), reads/writes in-process via
// settings.Service. Mirrors the runs CLI shape; default output is
// human text, --format json is opt-in per the scenario's CLI
// conventions.
package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/cli-core/cliapp"
	// modernc.org/sqlite registers itself as the "sqlite" driver via init();
	// the CLI opens the user_settings DB directly and needs the driver
	// loaded even though it doesn't reference an exported symbol.
	_ "modernc.org/sqlite"

	"flow-verifier/internal/clock"
	localdb "flow-verifier/internal/database"
	"flow-verifier/internal/settings"
)

// allowedKeys is the authoritative set of `<key>=<value>` keys the
// `settings set` subcommand accepts. Sorted so the help / error
// message order is stable.
var allowedKeys = []string{
	"defaultRoot",
	"density",
	"fontScale",
	"reducedMotion",
	"rtl",
	"sidebarWidth",
	"theme",
}

// ExitUnknownKey is the exit code the set subcommand returns when the
// user passes a `<key>=<value>` pair whose key isn't in allowedKeys.
// Surfaced as a typed sentinel so tests can pin it without parsing the
// error message.
const ExitUnknownKey = 2

// UnknownKeyError is returned by ParseAssignments when an input pair
// has a key that isn't in allowedKeys. The CLI dispatcher translates
// it to exit code 2 + a stderr message listing allowedKeys.
type UnknownKeyError struct {
	Key string
}

func (e UnknownKeyError) Error() string {
	return fmt.Sprintf("unknown setting %q; allowed keys: %s", e.Key, strings.Join(allowedKeys, ", "))
}

// Register returns the `settings` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	_ = core
	return cliapp.SubcommandGroup{
		Name:        "settings",
		Description: "Get or update UI/CLI preferences (theme, font scale, density, …)",
		Subcommands: []cliapp.Command{
			{
				Name:        "get",
				Description: "Print the local principal's UI/CLI preferences",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "format", Description: "Output format: text (default) or json"},
					},
				},
				RunCtx: runGet,
			},
			{
				Name:        "set",
				Description: "Update one or more preferences via <key>=<value> pairs",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "format", Description: "Output format: text (default) or json"},
					},
					Positionals: []cliapp.Positional{
						{Name: "assignment", Required: true, Repeated: true, Description: "key=value (e.g. theme=dark)"},
					},
				},
				RunCtx: runSet,
			},
		},
	}
}

func openService(ctx context.Context) (*settings.Service, *sql.DB, error) {
	dsn, err := localdb.DefaultDSN()
	if err != nil {
		return nil, nil, err
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := apidb.EnsureSchemas(ctx, db,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(settings.Schema),
	); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("ensure settings schema: %w", err)
	}
	return settings.NewService(settings.NewSQLiteRepository(db, clock.System{})), db, nil
}

func runGet(ctx cliapp.RunContext) error {
	bg := context.Background()
	svc, db, err := openService(bg)
	if err != nil {
		return err
	}
	defer db.Close()

	got, err := svc.Get(bg)
	if err != nil {
		return err
	}
	return renderSettings(ctx, got)
}

func runSet(ctx cliapp.RunContext) error {
	pairs := ctx.Positionals("assignment")
	patch, err := ParseAssignments(pairs)
	if err != nil {
		return err
	}

	bg := context.Background()
	svc, db, err := openService(bg)
	if err != nil {
		return err
	}
	defer db.Close()

	updated, err := svc.Upsert(bg, patch)
	if err != nil {
		return err
	}
	return renderSettings(ctx, updated)
}

func renderSettings(ctx cliapp.RunContext, s settings.Settings) error {
	out := ctx.Stdout()
	format := strings.ToLower(strings.TrimSpace(ctx.Flag("format")))
	if format == "" {
		format = "text"
	}
	switch format {
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(s)
	case "text":
		return renderText(out, s)
	default:
		return fmt.Errorf("unknown --format %q; allowed: text, json", format)
	}
}

func renderText(out interface{ Write([]byte) (int, error) }, s settings.Settings) error {
	rows := [][2]string{
		{"theme", string(s.Theme)},
		{"fontScale", string(s.FontScale)},
		{"reducedMotion", strconv.FormatBool(s.ReducedMotion)},
		{"rtl", strconv.FormatBool(s.RTL)},
		{"defaultRoot", s.DefaultRoot},
		{"density", string(s.Density)},
		{"sidebarWidth", strconv.Itoa(s.SidebarWidth)},
		{"inventoryFilters.language", s.InventoryFilters.Language},
		{"inventoryFilters.sort.key", s.InventoryFilters.Sort.Key},
		{"inventoryFilters.sort.dir", s.InventoryFilters.Sort.Dir},
	}
	width := 0
	for _, r := range rows {
		if len(r[0]) > width {
			width = len(r[0])
		}
	}
	for _, r := range rows {
		line := fmt.Sprintf("%-*s  %s\n", width, r[0], r[1])
		if _, err := out.Write([]byte(line)); err != nil {
			return err
		}
	}
	return nil
}

// ParseAssignments converts a slice of "<key>=<value>" strings into a
// settings.Patch. Unknown keys yield UnknownKeyError; invalid values
// for known keys yield a typed error from the value-coercion step.
// Pure function so it can be unit-tested without a database handle.
func ParseAssignments(pairs []string) (settings.Patch, error) {
	var patch settings.Patch
	allowed := allowedKeySet()
	for _, raw := range pairs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		idx := strings.IndexByte(raw, '=')
		if idx <= 0 {
			return settings.Patch{}, fmt.Errorf("expected <key>=<value>, got %q", raw)
		}
		key := strings.TrimSpace(raw[:idx])
		value := strings.TrimSpace(raw[idx+1:])
		if _, ok := allowed[key]; !ok {
			return settings.Patch{}, UnknownKeyError{Key: key}
		}
		if err := applyAssignment(&patch, key, value); err != nil {
			return settings.Patch{}, err
		}
	}
	return patch, nil
}

func applyAssignment(p *settings.Patch, key, value string) error {
	switch key {
	case "theme":
		t := settings.Theme(value)
		p.Theme = &t
	case "fontScale":
		f := settings.FontScale(value)
		p.FontScale = &f
	case "density":
		d := settings.Density(value)
		p.Density = &d
	case "defaultRoot":
		v := value
		p.DefaultRoot = &v
	case "reducedMotion":
		b, err := parseBool(value)
		if err != nil {
			return fmt.Errorf("reducedMotion: %w", err)
		}
		p.ReducedMotion = &b
	case "rtl":
		b, err := parseBool(value)
		if err != nil {
			return fmt.Errorf("rtl: %w", err)
		}
		p.RTL = &b
	case "sidebarWidth":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("sidebarWidth: must be a non-negative integer, got %q", value)
		}
		p.SidebarWidth = &n
	default:
		// Unreachable: ParseAssignments rejects unknown keys before
		// applyAssignment is called. Kept as a defensive fallback so a
		// future allowedKeys addition that forgets to wire a case here
		// fails fast rather than silently dropping the value.
		return fmt.Errorf("internal: no apply handler for key %q", key)
	}
	return nil
}

// parseBool accepts the same vocabulary the plan documents: true|false
// |1|0|on|off (case-insensitive). strconv.ParseBool covers true/false/1
// /0/t/f/TRUE/etc. but not on/off, so we wrap it.
func parseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "on":
		return true, nil
	case "off":
		return false, nil
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return false, fmt.Errorf("expected one of true|false|1|0|on|off, got %q", s)
	}
	return v, nil
}

func allowedKeySet() map[string]struct{} {
	m := make(map[string]struct{}, len(allowedKeys))
	for _, k := range allowedKeys {
		m[k] = struct{}{}
	}
	return m
}

// AllowedKeys returns a sorted copy of the keys `settings set` accepts.
// Useful for tests and downstream help-text rendering.
func AllowedKeys() []string {
	out := append([]string(nil), allowedKeys...)
	sort.Strings(out)
	return out
}
