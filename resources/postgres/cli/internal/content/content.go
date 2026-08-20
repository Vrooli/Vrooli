package content

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/vrooli/cli-core/cliapp"
)

type DatabaseEnsureRequest struct {
	Host, Port, User, Password, Database, Owner string
}

type (
	DatabaseEnsurer func(context.Context, DatabaseEnsureRequest) (created bool, err error)
	DatabaseDropper func(context.Context, DatabaseEnsureRequest) error
)

// EnsureCommandGroup registers the typed dependency-config reconciliation
// surface used by the control plane after PostgreSQL is healthy.
func EnsureCommandGroup(h *Handlers) cliapp.CommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.CommandGroup{
		Title: "Dependency Ensurance",
		Commands: []cliapp.Command{{
			Name:        "ensure",
			Description: "Create the database explicitly declared by a scenario dependency",
			Usage:       "resource-postgres ensure --config-base64 <base64-json>",
			Run:         h.Ensure,
		}},
	}
}

// Ensure applies the resource-owned portion of a scenario's PostgreSQL
// dependency declaration. Database creation is idempotent; schema migrations
// remain owned by the scenario API.
func (h *Handlers) Ensure(args []string) error {
	fs := flag.NewFlagSet("ensure", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	configB64 := fs.String("config-base64", "", "Base64-encoded PostgreSQL dependency config")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	if strings.TrimSpace(*configB64) == "" {
		return fmt.Errorf("--config-base64 is required")
	}
	raw, err := base64.StdEncoding.DecodeString(*configB64)
	if err != nil {
		return fmt.Errorf("decode --config-base64: %w", err)
	}
	var cfg struct {
		Database string `json:"database"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("parse PostgreSQL dependency config: %w", err)
	}
	database := strings.TrimSpace(cfg.Database)
	if database == "" {
		fmt.Fprintln(h.Stdout, "postgres ensure: no database declared; nothing to do")
		return nil
	}
	if err := validateIdentifier(database); err != nil {
		return err
	}
	owner := h.resolveUser()
	if err := validateIdentifier(owner); err != nil {
		return err
	}
	ensurer := h.EnsureDatabase
	if ensurer == nil {
		ensurer = ensureManagedDatabase
	}
	created, err := ensurer(context.Background(), DatabaseEnsureRequest{
		Host: h.resolveHost(defaultInstance), Port: h.resolvePort(), User: owner,
		Password: h.resolvePassword(), Database: database, Owner: owner,
	})
	if err != nil {
		return err
	}
	state := "already exists"
	if created {
		state = "created"
	}
	fmt.Fprintf(h.Stdout, "database %q %s\n", database, state)
	return nil
}

func ensureManagedDatabase(parent context.Context, req DatabaseEnsureRequest) (bool, error) {
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	db, err := openManagedAdminDatabase(ctx, req)
	if err != nil {
		return false, err
	}
	defer db.Close()
	var exists bool
	if err := db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", req.Database).Scan(&exists); err != nil {
		return false, fmt.Errorf("check PostgreSQL database %q: %w", req.Database, err)
	}
	if exists {
		return false, nil
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s OWNER %s", req.Database, req.Owner)); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "42P04" {
			return false, nil
		}
		return false, fmt.Errorf("create PostgreSQL database %q: %w", req.Database, err)
	}
	return true, nil
}

func dropManagedDatabase(parent context.Context, req DatabaseEnsureRequest) error {
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	db, err := openManagedAdminDatabase(ctx, req)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", req.Database)); err != nil {
		return fmt.Errorf("drop PostgreSQL database %q: %w", req.Database, err)
	}
	return nil
}

func openManagedAdminDatabase(ctx context.Context, req DatabaseEnsureRequest) (*sql.DB, error) {
	dsn := (&url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(req.User, req.Password),
		Host:   net.JoinHostPort(req.Host, req.Port),
		Path:   "postgres",
		RawQuery: url.Values{
			"sslmode": []string{"disable"},
		}.Encode(),
	}).String()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL admin connection: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to PostgreSQL admin database: %w", err)
	}
	return db, nil
}

const (
	defaultInstance = "main"
	containerPrefix = "vrooli-postgres-"
)

// Handlers owns the runtime dependencies (Runner, env lookup, stdout) for the
// content subcommand group. Tests inject fakes.
type Handlers struct {
	Runner         Runner
	EnsureDatabase DatabaseEnsurer
	DropDatabase   DatabaseDropper
	GetEnv         func(string) string
	Stdout         io.Writer
	Stderr         io.Writer
	Stdin          io.Reader
	LookupDir      func(path string) ([]string, error) // hook for listing .sql files
}

// Default returns Handlers wired to real OS/docker.
func Default() *Handlers {
	return &Handlers{
		Runner:    NewDockerRunner(),
		GetEnv:    os.Getenv,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		Stdin:     os.Stdin,
		LookupDir: listSQLFiles,
	}
}

// Commands returns the `content` subcommand group for registration with
// cli-core.
func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "content",
		Description: "Execute SQL and manage databases inside the postgres resource container",
		Subcommands: []cliapp.Command{
			{
				Name:        "execute",
				Description: "Execute SQL from --file or --sql against an instance",
				Usage:       "resource-postgres content execute [--instance <name>] [--database <db>] (--file <path> | --sql <sql>)",
				Run:         h.Execute,
			},
			{
				Name:        "create-database",
				Description: "Create a database (idempotent: exits 0 if it already exists)",
				Usage:       "resource-postgres content create-database [--instance <name>] [--owner <user>] <db-name>",
				Run:         h.CreateDatabase,
			},
			{
				Name:        "add",
				Description: "Load a SQL file or directory; use --init to create the database first",
				Usage:       "resource-postgres content add [--instance <name>] [--database <db>] [--init] <file-or-dir>",
				Run:         h.Add,
			},
			{
				Name:        "get",
				Description: "Print connection info for a database (use --as-env for shell export lines)",
				Usage:       "resource-postgres content get [--instance <name>] [--database <db>] [--as-env]",
				Run:         h.Get,
			},
			{
				Name:        "list",
				Description: "List databases in the postgres instance",
				Usage:       "resource-postgres content list [--instance <name>]",
				Run:         h.List,
			},
			{
				Name:        "remove",
				Description: "Drop a database",
				Usage:       "resource-postgres content remove [--instance <name>] <db-name>",
				Run:         h.Remove,
			},
		},
	}
}

// --- Execute -----------------------------------------------------------------

// Execute runs `content execute`.
func (h *Handlers) Execute(args []string) error {
	fs := flag.NewFlagSet("content execute", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	instance := fs.String("instance", defaultInstance, "Postgres instance name")
	database := fs.String("database", "", "Target database (defaults to resource's POSTGRES_DB)")
	file := fs.String("file", "", "Path to SQL file")
	sqlFlag := fs.String("sql", "", "Inline SQL statement")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" && *sqlFlag == "" {
		return fmt.Errorf("one of --file or --sql is required")
	}
	if *file != "" && *sqlFlag != "" {
		return fmt.Errorf("--file and --sql are mutually exclusive")
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	db := h.resolveDatabase(*database)
	if *file != "" {
		return h.executeFile(*instance, db, *file)
	}
	return h.executeSQL(*instance, db, *sqlFlag)
}

func (h *Handlers) executeFile(instance, database, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open SQL file: %w", err)
	}
	defer f.Close()
	return h.runPSQL(instance, database, nil, f)
}

func (h *Handlers) executeSQL(instance, database, sql string) error {
	return h.runPSQL(instance, database, []string{"-c", sql}, nil)
}

// --- CreateDatabase ----------------------------------------------------------

// CreateDatabase runs `content create-database`.
func (h *Handlers) CreateDatabase(args []string) error {
	fs := flag.NewFlagSet("content create-database", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	instance := fs.String("instance", defaultInstance, "Postgres instance name")
	owner := fs.String("owner", "", "Database owner (defaults to POSTGRES_USER)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: create-database [--instance <name>] [--owner <user>] <db-name>")
	}
	return h.ensureDatabase(*instance, fs.Arg(0), *owner)
}

func (h *Handlers) ensureDatabase(instance, dbName, owner string) error {
	if owner == "" {
		owner = h.resolveUser()
	}
	if err := validateIdentifier(dbName); err != nil {
		return err
	}
	if err := validateIdentifier(owner); err != nil {
		return err
	}
	// We connect to POSTGRES_DB (not the target) to issue the CREATE.
	adminDB := h.resolveDatabase("")
	sql := fmt.Sprintf("CREATE DATABASE %s OWNER %s;", dbName, owner)
	err := h.runPSQL(instance, adminDB, []string{"-c", sql}, nil)
	if err == nil {
		fmt.Fprintf(h.Stdout, "database %q created\n", dbName)
		return nil
	}
	// Treat "already exists" (SQLSTATE 42P04) as success.
	if strings.Contains(err.Error(), "already exists") {
		fmt.Fprintf(h.Stdout, "database %q already exists\n", dbName)
		return nil
	}
	return err
}

// --- Add ---------------------------------------------------------------------

// Add runs `content add`.
func (h *Handlers) Add(args []string) error {
	fs := flag.NewFlagSet("content add", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	instance := fs.String("instance", defaultInstance, "Postgres instance name")
	database := fs.String("database", "", "Target database (defaults to POSTGRES_DB)")
	initDB := fs.Bool("init", false, "Create the target database first if it does not exist")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: add [--instance <name>] [--database <db>] [--init] <file-or-dir>")
	}
	target := fs.Arg(0)

	db := h.resolveDatabase(*database)
	if *initDB {
		if err := h.ensureDatabase(*instance, db, ""); err != nil {
			return err
		}
	}

	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("stat %s: %w", target, err)
	}
	if info.IsDir() {
		files, err := h.LookupDir(target)
		if err != nil {
			return fmt.Errorf("list SQL files in %s: %w", target, err)
		}
		if len(files) == 0 {
			return fmt.Errorf("no .sql files found in %s", target)
		}
		for _, f := range files {
			if err := h.executeFile(*instance, db, f); err != nil {
				return err
			}
		}
		return nil
	}
	return h.executeFile(*instance, db, target)
}

// --- Get ---------------------------------------------------------------------

// Get runs `content get`.
func (h *Handlers) Get(args []string) error {
	fs := flag.NewFlagSet("content get", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	instance := fs.String("instance", defaultInstance, "Postgres instance name")
	database := fs.String("database", "", "Target database (defaults to POSTGRES_DB)")
	asEnv := fs.Bool("as-env", false, "Print shell export statements")
	if err := fs.Parse(args); err != nil {
		return err
	}
	// Allow a single positional arg as a shorthand for --database.
	if fs.NArg() == 1 && *database == "" {
		*database = fs.Arg(0)
	} else if fs.NArg() > 0 && (*database != "" || fs.NArg() > 1) {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	db := h.resolveDatabase(*database)
	user := h.resolveUser()
	password := h.resolvePassword()
	host := h.resolveHost(*instance)
	port := h.resolvePort()
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, db)

	if *asEnv {
		fmt.Fprintf(h.Stdout, "export POSTGRES_HOST=%s\n", shellQuote(host))
		fmt.Fprintf(h.Stdout, "export POSTGRES_PORT=%s\n", shellQuote(port))
		fmt.Fprintf(h.Stdout, "export POSTGRES_USER=%s\n", shellQuote(user))
		fmt.Fprintf(h.Stdout, "export POSTGRES_PASSWORD=%s\n", shellQuote(password))
		fmt.Fprintf(h.Stdout, "export POSTGRES_DB=%s\n", shellQuote(db))
		fmt.Fprintf(h.Stdout, "export POSTGRES_URL=%s\n", shellQuote(connStr))
		return nil
	}

	fmt.Fprintf(h.Stdout, "Instance:          %s\n", *instance)
	fmt.Fprintf(h.Stdout, "Container:         %s%s\n", containerPrefix, *instance)
	fmt.Fprintf(h.Stdout, "Database:          %s\n", db)
	fmt.Fprintf(h.Stdout, "Host:              %s\n", host)
	fmt.Fprintf(h.Stdout, "Port:              %s\n", port)
	fmt.Fprintf(h.Stdout, "User:              %s\n", user)
	fmt.Fprintf(h.Stdout, "Connection String: %s\n", connStr)
	return nil
}

// --- List --------------------------------------------------------------------

// List runs `content list`.
func (h *Handlers) List(args []string) error {
	fs := flag.NewFlagSet("content list", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	instance := fs.String("instance", defaultInstance, "Postgres instance name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	adminDB := h.resolveDatabase("")
	// -t = tuples only, -A = unaligned output
	return h.runPSQL(*instance, adminDB, []string{"-At", "-c", "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname;"}, nil)
}

// --- Remove ------------------------------------------------------------------

// Remove runs `content remove`.
func (h *Handlers) Remove(args []string) error {
	fs := flag.NewFlagSet("content remove", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	instance := fs.String("instance", defaultInstance, "Postgres instance name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: remove [--instance <name>] <db-name>")
	}
	dbName := fs.Arg(0)
	if err := validateIdentifier(dbName); err != nil {
		return err
	}
	owner := h.resolveUser()
	dropper := h.DropDatabase
	if dropper == nil {
		dropper = dropManagedDatabase
	}
	if err := dropper(context.Background(), DatabaseEnsureRequest{
		Host: h.resolveHost(*instance), Port: h.resolvePort(), User: owner,
		Password: h.resolvePassword(), Database: dbName, Owner: owner,
	}); err != nil {
		return err
	}
	fmt.Fprintf(h.Stdout, "database %q dropped\n", dbName)
	return nil
}

// --- shared ------------------------------------------------------------------

func (h *Handlers) runPSQL(instance, database string, psqlArgs []string, stdin io.Reader) error {
	container := containerPrefix + instance
	user := h.resolveUser()
	env := []string{
		"PGPASSWORD=" + h.resolvePassword(),
	}
	args := []string{"psql", "-U", user, "-d", database, "-v", "ON_ERROR_STOP=1", "--no-psqlrc", "-w"}
	args = append(args, psqlArgs...)
	stdout, stderr, err := h.Runner.Run(context.Background(), container, args, stdin, env)
	if len(stdout) > 0 {
		_, _ = h.Stdout.Write(stdout)
	}
	if len(stderr) > 0 {
		_, _ = h.Stderr.Write(stderr)
	}
	if err != nil {
		// Prefer stderr text for the wrapped error so callers see the psql message.
		if msg := strings.TrimSpace(string(stderr)); msg != "" {
			return fmt.Errorf("%s: %w", firstLine(msg), err)
		}
		return err
	}
	return nil
}

func (h *Handlers) resolveDatabase(override string) string {
	if override != "" {
		return override
	}
	if v := h.GetEnv("POSTGRES_DB"); v != "" {
		return v
	}
	return "vrooli"
}

func (h *Handlers) resolveUser() string {
	if v := h.GetEnv("POSTGRES_USER"); v != "" {
		return v
	}
	return "vrooli"
}

func (h *Handlers) resolvePassword() string {
	if v := h.GetEnv("POSTGRES_PASSWORD"); v != "" {
		return v
	}
	return "vrooli"
}

func (h *Handlers) resolveHost(instance string) string {
	// From the scenario's POV the resource is reachable on the host; the
	// CLI-side `docker exec` path doesn't need this, but `get --as-env`
	// reports the host-side connection string so scenarios can connect without
	// going through docker exec.
	if v := h.GetEnv("POSTGRES_HOST"); v != "" {
		return v
	}
	return "localhost"
}

func (h *Handlers) resolvePort() string {
	if v := h.GetEnv("POSTGRES_PORT"); v != "" {
		return v
	}
	return "5433"
}

func validateIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("identifier is required")
	}
	for i, r := range name {
		isAlpha := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
		isDigit := r >= '0' && r <= '9'
		if i == 0 && !isAlpha {
			return fmt.Errorf("invalid identifier %q: must start with a letter or underscore", name)
		}
		if !isAlpha && !isDigit {
			return fmt.Errorf("invalid identifier %q: only letters, digits, and underscores are allowed", name)
		}
	}
	return nil
}

func listSQLFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".sql") {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func shellQuote(s string) string {
	// Single-quote for POSIX shells; escape embedded quotes.
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
