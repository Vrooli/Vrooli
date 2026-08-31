package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	root := flag.String("root", "", "repository root (defaults to the current directory)")
	databasePath := flag.String("database", "", "SQLite database path")
	libraryID := flag.String("library-id", "", "exact library id")
	version := flag.String("version", "", "exact version")
	confirm := flag.Bool("confirm", false, "accept current authored bytes")
	flag.Parse()
	if !*confirm || *libraryID == "" || *version == "" {
		fail("--library-id, --version, and --confirm are required")
	}
	resolvedRoot := *root
	if resolvedRoot == "" {
		var err error
		resolvedRoot, err = os.Getwd()
		if err != nil {
			fail(err.Error())
		}
	}
	dbPath := *databasePath
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fail(err.Error())
		}
		dbPath = filepath.Join(home, ".vrooli/data/vrooli/react-component-library/react-component-library.db")
	}
	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		fail(err.Error())
	}
	defer db.Close()
	var id, sourcePath string
	err = db.QueryRow(`SELECT id,source_path FROM component_versions WHERE library_id=? AND version=?`, *libraryID, *version).Scan(&id, &sourcePath)
	if err != nil {
		fail(err.Error())
	}
	versionDir := filepath.Dir(filepath.Join(resolvedRoot, "scenarios/react-component-library/library", filepath.FromSlash(sourcePath)))
	tx, err := db.Begin()
	if err != nil {
		fail(err.Error())
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.Query(`SELECT path FROM component_version_files WHERE version_id=?`, id)
	if err != nil {
		fail(err.Error())
	}
	updated := 0
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			fail(err.Error())
		}
		if name == "dependencies.json" || (!strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".tsx") && name != "story.json" && name != "experience-contract.json") {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Join(versionDir, name))
		if readErr != nil {
			continue
		}
		sum := sha256.Sum256(raw)
		digest := hex.EncodeToString(sum[:])
		if _, err := tx.Exec(`UPDATE component_version_files SET content=?,content_sha256=? WHERE version_id=? AND path=?`, string(raw), digest, id, name); err != nil {
			rows.Close()
			fail(err.Error())
		}
		if filepath.ToSlash(filepath.Join(filepath.Dir(sourcePath), name)) == sourcePath || name == filepath.Base(sourcePath) {
			if _, err := tx.Exec(`UPDATE component_versions SET content=?,content_sha256=? WHERE id=?`, string(raw), digest, id); err != nil {
				rows.Close()
				fail(err.Error())
			}
		}
		updated++
	}
	rows.Close()
	if err := tx.Commit(); err != nil {
		fail(err.Error())
	}
	fmt.Printf("reconciled %d current authored mirror file(s) for %s@%s\n", updated, *libraryID, *version)
}

func fail(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }
