package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"swarm-manager/internal/pathredact"

	"github.com/vrooli/api-core/preflight"
	repocontract "github.com/vrooli/repo-contract-go"
)

type repeatFlag []string

func (f *repeatFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *repeatFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value != "" {
		*f = append(*f, value)
	}
	return nil
}

type fileChange struct {
	Path     string `json:"path"`
	BytesIn  int    `json:"bytes_in"`
	BytesOut int    `json:"bytes_out"`
}

type summary struct {
	Root             string       `json:"root"`
	Write            bool         `json:"write"`
	ScannedFiles     int          `json:"scanned_files"`
	ChangedFiles     int          `json:"changed_files"`
	SkippedBinary    int          `json:"skipped_binary"`
	RemainingMatches int          `json:"remaining_matches"`
	Changes          []fileChange `json:"changes,omitempty"`
	RemainingPaths   []string     `json:"remaining_paths,omitempty"`
}

func main() {
	if preflight.Run(preflight.Config{
		ScenarioName:          "swarm-manager",
		DisableLifecycleGuard: true,
		DisableStaleness:      true,
	}) {
		return
	}

	var oldRepoRoots repeatFlag
	var oldHomes repeatFlag
	var identityTerms repeatFlag
	rootFlag := flag.String("root", "", "repo root (defaults to repo-contract discovery)")
	write := flag.Bool("write", false, "rewrite generated state in place")
	jsonOut := flag.Bool("json", false, "emit a machine-readable summary")
	includeArchiveDocs := flag.Bool("include-archive-docs", false, "include archive documentation under generated artifact folders")
	flag.Var(&oldRepoRoots, "old-repo-root", "old repo root to sanitize; repeatable")
	flag.Var(&oldHomes, "old-home", "old home directory to sanitize; repeatable")
	flag.Var(&identityTerms, "identity-term", "additional standalone identity string to sanitize; repeatable")
	flag.Parse()

	root, err := resolveRoot(*rootFlag)
	if err != nil {
		fatal(err)
	}
	redactor := pathredact.NewFromEnvironment(root)
	redactor.RepoRoots = append(redactor.RepoRoots, oldRepoRoots...)
	redactor.HomeDirs = append(redactor.HomeDirs, oldHomes...)
	for _, home := range oldHomes {
		if base := strings.TrimSpace(filepath.Base(home)); base != "" && base != "." && base != string(filepath.Separator) {
			redactor.Usernames = append(redactor.Usernames, base)
		}
	}
	redactor.IdentityTerms = append(redactor.IdentityTerms, identityTerms...)

	result, err := sanitize(root, redactor, *write, *includeArchiveDocs)
	if err != nil {
		fatal(err)
	}
	if *jsonOut {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fatal(err)
		}
		fmt.Println(string(data))
	} else {
		printSummary(result)
	}
	if *write && result.RemainingMatches > 0 {
		os.Exit(2)
	}
}

func resolveRoot(raw string) (string, error) {
	if strings.TrimSpace(raw) != "" {
		return filepath.Abs(raw)
	}
	return repocontract.FindRepoRootFromEnvOrCWD()
}

func sanitize(root string, redactor pathredact.Redactor, write bool, includeArchiveDocs bool) (summary, error) {
	result := summary{Root: root, Write: write}
	base := filepath.Join(root, "scenarios", "swarm-manager")
	topDirs := []string{"ideas", "research", "fix", "execute", "chore", "initiatives"}
	for _, top := range topDirs {
		start := filepath.Join(base, top)
		if _, err := os.Stat(start); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return result, err
		}
		err := filepath.WalkDir(start, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if entry.IsDir() {
				if !includeArchiveDocs && strings.Contains(rel, "/archive/") {
					return filepath.SkipDir
				}
				return nil
			}
			if !isGeneratedSurface(rel) {
				return nil
			}
			return sanitizeFile(root, rel, path, redactor, write, &result)
		})
		if err != nil {
			return result, err
		}
	}
	sort.Slice(result.Changes, func(i, j int) bool { return result.Changes[i].Path < result.Changes[j].Path })
	sort.Strings(result.RemainingPaths)
	return result, nil
}

func sanitizeFile(root, rel, path string, redactor pathredact.Redactor, write bool, result *summary) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !pathredact.IsTextArtifact(path, data) {
		result.SkippedBinary++
		return nil
	}
	result.ScannedFiles++
	redacted, err := redactFile(path, data, redactor)
	if err != nil {
		return fmt.Errorf("%s: %w", rel, err)
	}
	if !bytes.Equal(data, redacted) {
		result.ChangedFiles++
		result.Changes = append(result.Changes, fileChange{Path: rel, BytesIn: len(data), BytesOut: len(redacted)})
		if write {
			if err := os.WriteFile(path, redacted, 0o600); err != nil {
				return err
			}
		}
	}
	if redactor.RedactString(string(redacted)) != string(redacted) {
		result.RemainingMatches++
		result.RemainingPaths = append(result.RemainingPaths, rel)
	}
	_ = root
	return nil
}

func redactFile(path string, data []byte, redactor pathredact.Redactor) ([]byte, error) {
	if strings.EqualFold(filepath.Ext(path), ".json") {
		var decoded any
		if err := json.Unmarshal(data, &decoded); err == nil {
			redacted, changed, err := redactor.RedactJSONValue(decoded)
			if err != nil {
				return nil, err
			}
			if !changed {
				return data, nil
			}
			out, err := json.MarshalIndent(redacted, "", "  ")
			if err != nil {
				return nil, err
			}
			return append(out, '\n'), nil
		}
	}
	out, _ := redactor.RedactBytes(path, data)
	return out, nil
}

// generatedSuffixes are path suffixes whose presence alone marks a file as
// generated state (checked before the base-name rules below).
var generatedSuffixes = []string{
	"/.swarm/last-research-prompt-trace.json",
}

// generatedBaseNames are file names that are generated state regardless of
// their containing directory.
var generatedBaseNames = map[string]bool{
	"acceptance-validation.json": true,
}

// roundJSONDirs are directory segments that contain "round-NNN.json" files
// that are generated state.
var roundJSONDirs = []string{
	"/workshop/",
	"/review/",
	"/operating-mode/",
}

// isRoundJSON reports whether base looks like a generated round file.
func isRoundJSON(base string) bool {
	return strings.HasPrefix(base, "round-") && strings.HasSuffix(base, ".json")
}

// handoffBases are the file names inside /handoff/ that are generated state.
var handoffBases = map[string]bool{
	"manifest.json":     true,
	"source-index.json": true,
	"brief.md":          true,
}

func isGeneratedSurface(rel string) bool {
	base := filepath.Base(rel)

	for _, suffix := range generatedSuffixes {
		if strings.HasSuffix(rel, suffix) {
			return true
		}
	}
	if generatedBaseNames[base] {
		return true
	}
	for _, dir := range roundJSONDirs {
		if strings.Contains(rel, dir) && isRoundJSON(base) {
			return true
		}
	}
	if strings.Contains(rel, "/review/captures/") {
		return true
	}
	if strings.Contains(rel, "/review/decisions/") && strings.HasSuffix(base, ".json") {
		return true
	}
	if strings.Contains(rel, "/evidence/") {
		return true
	}
	if strings.Contains(rel, "/handoff/") && handoffBases[base] {
		return true
	}
	if strings.Contains(rel, "/feedback/") && base == "feedback.json" {
		return true
	}
	return false
}

func printSummary(result summary) {
	mode := "dry-run"
	if result.Write {
		mode = "write"
	}
	fmt.Printf("sanitize-generated-state %s\n", mode)
	fmt.Printf("root: %s\n", result.Root)
	fmt.Printf("scanned: %d, changed: %d, skipped_binary: %d, remaining_matches: %d\n",
		result.ScannedFiles, result.ChangedFiles, result.SkippedBinary, result.RemainingMatches)
	for _, change := range result.Changes {
		fmt.Printf("changed %s (%d -> %d bytes)\n", change.Path, change.BytesIn, change.BytesOut)
	}
	for _, path := range result.RemainingPaths {
		fmt.Printf("remaining %s\n", path)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
