package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/resources/sqlite/internal/config"
	"github.com/vrooli/resources/sqlite/internal/sqlite"
)

// CLI dispatches resource commands.
type CLI struct {
	Service     *sqlite.Service
	Config      config.Config
	RuntimePath string
	RuntimeData []byte
	Now         func() time.Time
}

func New(cfg config.Config, runtimePath string, runtimeData []byte) *CLI {
	return &CLI{
		Service:     sqlite.NewService(cfg),
		Config:      cfg,
		RuntimePath: runtimePath,
		RuntimeData: runtimeData,
		Now:         time.Now,
	}
}

func (c *CLI) Run(args []string) int {
	if len(args) == 0 {
		c.printUsage()
		return 0
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "help", "--help", "-h":
		c.printUsage()
		return 0
	case "info":
		return c.handleInfo()
	case "status":
		return c.handleStatus()
	case "logs":
		fmt.Println("SQLite is serverless; no dedicated logs available.")
		return 0
	case "manage":
		return c.handleManage(rest)
	case "content":
		return c.handleContent(rest)
	case "replicate", "migrate", "query", "stats", "test":
		switch cmd {
		case "replicate":
			return c.handleReplicate(rest)
		case "migrate":
			return c.handleMigrate(rest)
		case "query":
			return c.handleQuery(rest)
		case "stats":
			return c.handleStats(rest)
		case "test":
			return c.handleTest(rest)
		}
		return 1
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		c.printUsage()
		return 1
	}
}

func (c *CLI) handleManage(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "manage requires a subcommand (install|uninstall|start|stop|restart)")
		return 1
	}
	sub := args[0]
	switch sub {
	case "install":
		if err := c.Config.EnsureDirectories(); err != nil {
			return fail(err)
		}
		fmt.Println("SQLite resource ready (Go binary, no external installer needed).")
		return 0
	case "uninstall":
		fmt.Println("Uninstall is a no-op for the portable Go binary. Remove the binary and data directory to clean up.")
		return 0
	case "start", "stop", "restart":
		if err := c.Config.EnsureDirectories(); err != nil {
			return fail(err)
		}
		fmt.Printf("SQLite is serverless; %s acknowledged.\n", sub)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown manage subcommand: %s\n", sub)
		return 1
	}
}

func (c *CLI) handleContent(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "content requires a subcommand")
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.Config.CLITimeout)
	defer cancel()

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "create":
		if len(rest) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: content create <database_name>")
			return 1
		}
		path, err := c.Service.CreateDatabase(ctx, rest[0])
		if err != nil {
			return fail(err)
		}
		fmt.Printf("Created %s\n", path)
		return 0
	case "execute":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: content execute <database_name> \"<sql>\"")
			return 1
		}
		dbName, query := rest[0], strings.Join(rest[1:], " ")
		result, err := c.Service.Execute(ctx, dbName, query)
		if err != nil {
			return fail(err)
		}
		if result != nil {
			printTable(result)
		} else {
			fmt.Println("OK")
		}
		return 0
	case "list":
		dbs, err := c.Service.ListDatabases()
		if err != nil {
			return fail(err)
		}
		if len(dbs) == 0 {
			fmt.Println("No databases found.")
			return 0
		}
		for _, db := range dbs {
			fmt.Printf("%s\t%d bytes\tmodified %s\n", db.Name, db.Size, db.Modified.Format(time.RFC3339))
		}
		return 0
	case "get":
		if len(rest) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: content get <database_name> [select query]")
			return 1
		}
		dbName := rest[0]
		query := strings.Join(rest[1:], " ")
		result, err := c.Service.GetDatabaseInfo(ctx, dbName, query)
		if err != nil {
			return fail(err)
		}
		if result.Result != nil {
			printTable(result.Result)
		} else {
			for k, v := range result.Info {
				fmt.Printf("%s: %s\n", k, v)
			}
		}
		return 0
	case "backup":
		if len(rest) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: content backup <database_name>")
			return 1
		}
		path, err := c.Service.BackupDatabase(ctx, rest[0])
		if err != nil {
			return fail(err)
		}
		fmt.Printf("Backup created at %s\n", path)
		return 0
	case "restore":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: content restore <database_name> <backup_file> [--force]")
			return 1
		}
		dbName, backup := rest[0], rest[1]
		force := contains(rest[2:], "--force")
		path, err := c.Service.RestoreDatabase(ctx, dbName, backup, force)
		if err != nil {
			return fail(err)
		}
		fmt.Printf("Restored to %s\n", path)
		return 0
	case "remove":
		if len(rest) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: content remove <database_name> --force")
			return 1
		}
		force := contains(rest[1:], "--force")
		if !force {
			fmt.Fprintln(os.Stderr, "Refusing to remove without --force")
			return 1
		}
		if err := c.Service.RemoveDatabase(rest[0], force); err != nil {
			return fail(err)
		}
		fmt.Println("Removed.")
		return 0
	case "batch":
		if len(rest) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: content batch <database_name> [sql_file]")
			return 1
		}
		dbName := rest[0]
		var reader io.Reader
		if len(rest) > 1 && rest[1] != "-" {
			file, err := os.Open(rest[1])
			if err != nil {
				return fail(err)
			}
			defer file.Close()
			reader = file
		} else {
			reader = os.Stdin
		}
		result, err := c.Service.Batch(ctx, dbName, reader)
		if err != nil {
			return fail(err)
		}
		fmt.Printf("Batch executed in %s\n", result.Duration)
		return 0
	case "import_csv":
		if len(rest) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: content import_csv <database_name> <table> <csv_file> [--no-header]")
			return 1
		}
		dbName, table, file := rest[0], rest[1], rest[2]
		hasHeader := true
		if contains(rest[3:], "--no-header") {
			hasHeader = false
		}
		count, err := c.Service.ImportCSV(ctx, dbName, table, file, hasHeader)
		if err != nil {
			return fail(err)
		}
		fmt.Printf("Imported %d rows\n", count)
		return 0
	case "export_csv":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: content export_csv <database_name> <table> [output_file]")
			return 1
		}
		dbName, table := rest[0], rest[1]
		out := ""
		if len(rest) > 2 {
			out = rest[2]
		}
		count, err := c.Service.ExportCSV(ctx, dbName, table, out)
		if err != nil {
			return fail(err)
		}
		fmt.Printf("Exported %d rows\n", count)
		return 0
	case "encrypt":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: content encrypt <database_name> <password>")
			return 1
		}
		if err := c.Service.EncryptDatabase(rest[0], rest[1]); err != nil {
			return fail(err)
		}
		fmt.Println("Encrypted.")
		return 0
	case "decrypt":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: content decrypt <database_name> <password>")
			return 1
		}
		if err := c.Service.DecryptDatabase(rest[0], rest[1]); err != nil {
			return fail(err)
		}
		fmt.Println("Decrypted.")
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown content subcommand: %s\n", sub)
		return 1
	}
}

func (c *CLI) handleInfo() int {
	data, err := os.ReadFile(c.RuntimePath)
	if err != nil && len(c.RuntimeData) == 0 {
		return fail(err)
	}
	if err == nil {
		fmt.Println(string(data))
		return 0
	}
	fmt.Println(string(c.RuntimeData))
	return 0
}

func (c *CLI) handleStatus() int {
	ctx, cancel := context.WithTimeout(context.Background(), c.Config.CLITimeout)
	defer cancel()

	status, err := c.Service.Status(ctx)
	if err != nil {
		return fail(err)
	}
	fmt.Printf("SQLite Status\n=============\nVersion: %s\nDatabase Path: %s\nDatabases: %d\n", status.Version, status.DatabasePath, status.DatabaseCount)
	for _, db := range status.Databases {
		fmt.Printf("- %s (%d bytes, modified %s)\n", db.Name, db.Size, db.Modified.Format(time.RFC3339))
	}
	return 0
}

func (c *CLI) notImplemented(cmd string) int {
	fmt.Fprintf(os.Stderr, "%s commands are not implemented yet in the Go port.\n", cmd)
	fmt.Fprintln(os.Stderr, "See README for the remaining parity work.")
	return 1
}

func (c *CLI) printUsage() {
	fmt.Println(`SQLite Resource (Go)
Usage:
  resource-sqlite <command> [subcommand] [options]

Commands:
  status                     Show resource status
  info                       Show runtime info (config/runtime.json)
  logs                       Show logs (serverless no-op)
  manage <sub>               install|uninstall|start|stop|restart
  content <sub>              create|execute|list|get|backup|restore|remove|batch|import_csv|export_csv|encrypt|decrypt
  replicate <sub>            add|remove|list|sync|verify|toggle
  migrate <sub>              init|create|up|status
  query <sub>                select|insert|update
  stats <sub>                enable|show|analyze|vacuum
  test <sub>                 smoke|integration|unit (runs go test ./...)`)
}

func (c *CLI) handleReplicate(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: replicate <add|remove|list|sync|verify|toggle> ...")
		return 1
	}
	sub := args[0]
	switch sub {
	case "add":
		if len(args) < 5 {
			fmt.Fprintln(os.Stderr, "Usage: replicate add --database <name> --target <path> [--interval <seconds>]")
			return 1
		}
		db, target, interval := "", "", 60
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--database", "-d":
				i++
				db = args[i]
			case "--target", "-t":
				i++
				target = args[i]
			case "--interval", "-i":
				i++
				fmt.Sscanf(args[i], "%d", &interval)
			}
		}
		if err := c.Service.AddReplica(db, target, time.Duration(interval)*time.Second); err != nil {
			return fail(err)
		}
		fmt.Println("Replica added.")
		return 0
	case "remove":
		if len(args) < 5 {
			fmt.Fprintln(os.Stderr, "Usage: replicate remove --database <name> --target <path>")
			return 1
		}
		db, target := "", ""
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--database", "-d":
				i++
				db = args[i]
			case "--target", "-t":
				i++
				target = args[i]
			}
		}
		if err := c.Service.RemoveReplica(db, target); err != nil {
			return fail(err)
		}
		fmt.Println("Replica removed.")
		return 0
	case "list":
		replicas, err := c.Service.ListReplicas()
		if err != nil {
			return fail(err)
		}
		if len(replicas) == 0 {
			fmt.Println("No replicas configured.")
			return 0
		}
		for _, r := range replicas {
			fmt.Printf("[%s] -> %s (interval %s, enabled=%v)\n", r.Database, r.Target, r.Interval, r.Enabled)
		}
		return 0
	case "sync":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: replicate sync --database <name> [--force]")
			return 1
		}
		db := ""
		force := false
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--database", "-d":
				i++
				db = args[i]
			case "--force", "-f":
				force = true
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), c.Config.CLITimeout)
		defer cancel()
		ok, failCount, err := c.Service.SyncReplica(ctx, db, force)
		if err != nil {
			return fail(err)
		}
		fmt.Printf("Replication complete: %d succeeded, %d failed\n", ok, failCount)
		if failCount > 0 {
			return 1
		}
		return 0
	case "verify":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: replicate verify --database <name>")
			return 1
		}
		db := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "--database" || args[i] == "-d" {
				i++
				db = args[i]
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), c.Config.CLITimeout)
		defer cancel()
		issues, err := c.Service.VerifyReplicas(ctx, db)
		if err != nil {
			return fail(err)
		}
		if len(issues) == 0 {
			fmt.Println("All replicas consistent.")
			return 0
		}
		for _, i := range issues {
			fmt.Printf("- %s\n", i)
		}
		return 1
	case "toggle":
		if len(args) < 5 {
			fmt.Fprintln(os.Stderr, "Usage: replicate toggle --database <name> --target <path> (--enable|--disable)")
			return 1
		}
		db, target := "", ""
		enabled := true
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--database", "-d":
				i++
				db = args[i]
			case "--target", "-t":
				i++
				target = args[i]
			case "--disable":
				enabled = false
			case "--enable":
				enabled = true
			}
		}
		if err := c.Service.ToggleReplica(db, target, enabled); err != nil {
			return fail(err)
		}
		fmt.Println("Replica updated.")
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown replicate subcommand: %s\n", sub)
		return 1
	}
}

func (c *CLI) handleMigrate(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: migrate <init|create|up|status> ...")
		return 1
	}
	sub := args[0]
	switch sub {
	case "init":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: migrate init <database>")
			return 1
		}
		if err := c.Service.InitMigrations(args[1]); err != nil {
			return fail(err)
		}
		fmt.Println("Migration tracking initialized.")
		return 0
	case "create":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: migrate create \"name\"")
			return 1
		}
		path, err := c.Service.CreateMigration(strings.Join(args[1:], " "))
		if err != nil {
			return fail(err)
		}
		fmt.Printf("Migration created at %s\n", path)
		return 0
	case "up":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: migrate up <database> [target_version]")
			return 1
		}
		db := args[1]
		target := ""
		if len(args) > 2 {
			target = args[2]
		}
		ctx, cancel := context.WithTimeout(context.Background(), c.Config.CLITimeout)
		defer cancel()
		count, err := c.Service.ApplyMigrations(ctx, db, target)
		if err != nil {
			return fail(err)
		}
		fmt.Printf("Applied %d migrations\n", count)
		return 0
	case "status":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: migrate status <database>")
			return 1
		}
		ctx, cancel := context.WithTimeout(context.Background(), c.Config.CLITimeout)
		defer cancel()
		applied, pending, err := c.Service.MigrationStatus(ctx, args[1])
		if err != nil {
			return fail(err)
		}
		fmt.Println("Applied:")
		for _, a := range applied {
			fmt.Printf("- %s\n", a)
		}
		fmt.Println("Pending:")
		for _, p := range pending {
			fmt.Printf("- %s\n", p)
		}
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown migrate subcommand: %s\n", sub)
		return 1
	}
}

func (c *CLI) handleQuery(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: query <select|insert|update> ...")
		return 1
	}
	sub := args[0]
	switch sub {
	case "select":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: query select <db> <table> [--where expr] [--order expr] [--limit n]")
			return 1
		}
		db, table := args[1], args[2]
		where, order := "", ""
		limit := 0
		for i := 3; i < len(args); i++ {
			switch args[i] {
			case "--where":
				i++
				where = args[i]
			case "--order":
				i++
				order = args[i]
			case "--limit":
				i++
				fmt.Sscanf(args[i], "%d", &limit)
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), c.Config.CLITimeout)
		defer cancel()
		result, err := c.Service.QuerySelect(ctx, db, table, where, order, limit)
		if err != nil {
			return fail(err)
		}
		printTable(result)
		return 0
	case "insert":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: query insert <db> <table> col=val [col=val] ...")
			return 1
		}
		db, table := args[1], args[2]
		ctx, cancel := context.WithTimeout(context.Background(), c.Config.CLITimeout)
		defer cancel()
		if err := c.Service.QueryInsert(ctx, db, table, args[3:]); err != nil {
			return fail(err)
		}
		fmt.Println("Inserted.")
		return 0
	case "update":
		if len(args) < 5 {
			fmt.Fprintln(os.Stderr, "Usage: query update <db> <table> col=val [col=val] --where expr")
			return 1
		}
		db, table := args[1], args[2]
		where := ""
		pairs := []string{}
		for i := 3; i < len(args); i++ {
			if args[i] == "--where" {
				i++
				where = args[i]
			} else {
				pairs = append(pairs, args[i])
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), c.Config.CLITimeout)
		defer cancel()
		if err := c.Service.QueryUpdate(ctx, db, table, pairs, where); err != nil {
			return fail(err)
		}
		fmt.Println("Updated.")
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown query subcommand: %s\n", sub)
		return 1
	}
}

func (c *CLI) handleStats(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: stats <enable|show|analyze|vacuum> <db>")
		return 1
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "stats requires database name")
		return 1
	}
	sub := args[0]
	db := args[1]
	ctx, cancel := context.WithTimeout(context.Background(), c.Config.CLITimeout)
	defer cancel()
	switch sub {
	case "enable":
		if err := c.Service.EnableStats(ctx, db); err != nil {
			return fail(err)
		}
		fmt.Println("Stats enabled.")
		return 0
	case "show":
		res, err := c.Service.ShowStats(ctx, db)
		if err != nil {
			return fail(err)
		}
		printTable(res)
		return 0
	case "analyze":
		if err := c.Service.Analyze(ctx, db); err != nil {
			return fail(err)
		}
		fmt.Println("Analyze complete.")
		return 0
	case "vacuum":
		if err := c.Service.Vacuum(ctx, db); err != nil {
			return fail(err)
		}
		fmt.Println("Vacuum complete.")
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown stats subcommand: %s\n", sub)
		return 1
	}
}

func (c *CLI) handleTest(args []string) int {
	ctx, cancel := context.WithTimeout(context.Background(), c.Config.CLITimeout)
	defer cancel()

	if _, err := exec.LookPath("go"); err != nil {
		fmt.Fprintln(os.Stderr, "Go toolchain not found. Install Go or run `go test ./...` inside resources/sqlite.")
		return 1
	}

	// Run tests from the resource root (parent of config directory).
	root := filepath.Dir(filepath.Dir(c.RuntimePath))
	cmd := exec.CommandContext(ctx, "go", "test", "./...")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return 1
	}
	return 0
}

func printTable(result *sqlite.QueryResult) {
	if result == nil {
		return
	}
	fmt.Println(strings.Join(result.Columns, "\t"))
	for _, row := range result.Rows {
		fmt.Println(strings.Join(row, "\t"))
	}
}

func contains(args []string, token string) bool {
	for _, a := range args {
		if a == token {
			return true
		}
	}
	return false
}

func fail(err error) int {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		fmt.Fprintf(os.Stderr, "Path error: %v\n", pathErr)
		return 1
	}
	fmt.Fprintf(os.Stderr, "%v\n", err)
	return 1
}

// Debug helper to pretty-print JSON structures if needed in the future.
func toJSON(v any) string {
	data, _ := json.MarshalIndent(v, "", "  ")
	return string(data)
}
