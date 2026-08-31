package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
	"react-component-library/internal/components"
)

type entry struct {
	Path   string `json:"path"`
	Asset  string `json:"asset"`
	Status string `json:"status"`
}

type report struct {
	Candidates []entry  `json:"candidates"`
	Missing    []entry  `json:"missing"`
	Recorded   []string `json:"recorded,omitempty"`
	Applied    bool     `json:"applied"`
}

func main() {
	root := flag.String("root", "", "repository root (defaults to the current directory)")
	databasePath := flag.String("database", "", "SQLite database path (defaults to the Vrooli scenario database)")
	apply := flag.Bool("apply", false, "restore mirror bytes to the authored release files")
	includeRetired := flag.Bool("include-retired", false, "include all materialized retired versions whose rows are not in the live ledger")
	var selectedRetired []string
	flag.Func("retired", "include one retired version by asset@version (repeatable; requires --apply to restore)", func(value string) error {
		selectedRetired = append(selectedRetired, strings.TrimSpace(value))
		return nil
	})
	flag.Parse()
	selectedRetiredSet := make(map[string]struct{}, len(selectedRetired))
	for _, value := range selectedRetired {
		if value != "" {
			selectedRetiredSet[value] = struct{}{}
		}
	}

	resolvedRoot := *root
	if resolvedRoot == "" {
		var err error
		resolvedRoot, err = os.Getwd()
		if err != nil {
			fail(err)
		}
	}
	dbPath := *databasePath
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fail(err)
		}
		dbPath = filepath.Join(home, ".vrooli/data/vrooli/react-component-library/react-component-library.db")
	}
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		fail(err)
	}
	defer db.Close()

	ledgerPath := filepath.Join(resolvedRoot, "scenarios/react-component-library/library/released-version-hashes.json")
	ledgerRaw, err := os.ReadFile(ledgerPath)
	if err != nil {
		fail(err)
	}
	var ledger struct {
		SchemaVersion int `json:"schemaVersion"`
		Entries       []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(ledgerRaw, &ledger); err != nil {
		fail(err)
	}
	governed := make(map[string]struct{}, len(ledger.Entries))
	for _, row := range ledger.Entries {
		if components.IsAuthoredReleaseFile(row.Path) {
			governed[filepath.ToSlash(row.Path)] = struct{}{}
		}
	}
	recorded := make(map[string]string)

	libraryRoot := filepath.Join(resolvedRoot, "scenarios/react-component-library/library")
	includeRetiredRows := *includeRetired || len(selectedRetiredSet) > 0
	rows, err := db.Query(`SELECT v.source_path, f.path, f.content, f.content_sha256, v.status, v.presence
		FROM component_versions v
		JOIN component_version_files f ON f.version_id=v.id
		WHERE (v.status <> 'retired' AND v.presence = 'materialized') OR (v.status = 'retired' AND ?)
		ORDER BY v.source_path, f.path`, includeRetiredRows)
	if err != nil {
		fail(err)
	}
	defer rows.Close()
	result := report{Applied: *apply}
	for rows.Next() {
		var sourcePath, filePath, content, mirrorHash, status, presence string
		if err := rows.Scan(&sourcePath, &filePath, &content, &mirrorHash, &status, &presence); err != nil {
			fail(err)
		}
		relative := filepath.ToSlash(filepath.Join(filepath.Dir(sourcePath), filePath))
		_, governedPath := governed[relative]
		_, selected := selectedRetiredSet[assetFor(sourcePath)]
		retiredPath := status == "retired" && ((*includeRetired && presence == "materialized") || selected)
		if status == "retired" && !retiredPath {
			continue
		}
		// dependencies.json and parity.json are derived records. Every other
		// mirrored file is authored release content, even when an older ledger
		// omitted a file type such as CSS.
		derived := filePath == "dependencies.json" || filePath == "parity.json"
		if derived || (!governedPath && !retiredPath && status == "retired") {
			continue
		}
		path := filepath.Join(libraryRoot, filepath.FromSlash(relative))
		if !within(libraryRoot, path) {
			fail(fmt.Errorf("refusing path outside library root: %s", relative))
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			result.Missing = append(result.Missing, entry{Path: relative, Asset: assetFor(sourcePath), Status: "missing"})
			if !*apply || !retiredPath {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				fail(err)
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				fail(err)
			}
			if retiredPath && selected {
				recorded[relative] = mirrorHash
			}
			continue
		}
		digest := sha256.Sum256(raw)
		if hex.EncodeToString(digest[:]) == mirrorHash {
			if retiredPath && selected {
				recorded[relative] = mirrorHash
			}
			continue
		}
		result.Candidates = append(result.Candidates, entry{Path: relative, Asset: assetFor(sourcePath), Status: "restored-from-mirror"})
		if !*apply {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			fail(err)
		}
		tmp := path + ".mirror-restore.tmp"
		if err := os.WriteFile(tmp, []byte(content), info.Mode().Perm()); err != nil {
			fail(err)
		}
		if err := os.Rename(tmp, path); err != nil {
			_ = os.Remove(tmp)
			fail(err)
		}
		if retiredPath && selected {
			recorded[relative] = mirrorHash
		}
	}
	if err := rows.Err(); err != nil {
		fail(err)
	}
	if *apply && len(recorded) > 0 {
		known := make(map[string]struct{}, len(ledger.Entries))
		for _, item := range ledger.Entries {
			known[filepath.ToSlash(item.Path)] = struct{}{}
		}
		for relative, mirrorHash := range recorded {
			if _, ok := known[relative]; ok {
				continue
			}
			ledger.Entries = append(ledger.Entries, struct {
				Path   string `json:"path"`
				SHA256 string `json:"sha256"`
			}{Path: relative, SHA256: mirrorHash})
			result.Recorded = append(result.Recorded, relative)
		}
		sort.Slice(ledger.Entries, func(i, j int) bool { return ledger.Entries[i].Path < ledger.Entries[j].Path })
		updated, err := json.MarshalIndent(ledger, "", "  ")
		if err != nil {
			fail(err)
		}
		tmp := ledgerPath + ".mirror-restore.tmp"
		if err := os.WriteFile(tmp, updated, 0o644); err != nil {
			fail(err)
		}
		if err := os.Rename(tmp, ledgerPath); err != nil {
			_ = os.Remove(tmp)
			fail(err)
		}
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(encoded))
}

func assetFor(sourcePath string) string {
	parts := strings.Split(filepath.ToSlash(sourcePath), "/")
	for i, part := range parts {
		if part == "versions" && i > 0 && i+1 < len(parts) {
			return parts[i-1] + "@" + parts[i+1]
		}
	}
	return "__corpus__.released-version-hashes"
}

func within(root, path string) bool {
	root = filepath.Clean(root) + string(os.PathSeparator)
	return path == filepath.Clean(root[:len(root)-1]) || strings.HasPrefix(filepath.Clean(path), root)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
