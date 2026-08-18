package coverage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	maxAge   = 14 * 24 * time.Hour
	maxBytes = int64(2 << 30)
)

type candidate struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	Reason  string    `json:"reason"`
	Deleted bool      `json:"deleted"`
	ModTime time.Time `json:"-"`
}

type pruneReport struct {
	Root             string      `json:"root"`
	DryRun           bool        `json:"dryRun"`
	MaxAge           string      `json:"maxAge"`
	MaxBytes         int64       `json:"maxBytes"`
	TotalBytes       int64       `json:"totalBytes"`
	SelectedBytes    int64       `json:"selectedBytes"`
	Selected         []candidate `json:"selected"`
	RetainedArtifact []string    `json:"retainedArtifacts"`
}

// Commands exposes the local cache-retention command. It is intentionally
// local: coverage is generated evidence and must not require the API to be
// running merely to report or prune its cache.
func Commands() cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Coverage",
		Commands: []cliapp.Command{{
			Name:        "coverage",
			NeedsAPI:    false,
			Description: "Inspect and prune the local validation coverage cache",
			Run:         run,
		}},
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] != "prune" {
		return fmt.Errorf("usage: react-component-library coverage prune [--apply] [--json] [--root <path>]")
	}

	apply, jsonOutput := false, false
	root := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--apply":
			apply = true
		case "--json":
			jsonOutput = true
		case "--root":
			if i+1 >= len(args) {
				return fmt.Errorf("--root requires a path")
			}
			i++
			root = args[i]
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}
	if root == "" {
		root = findCoverageRoot()
	}
	report, err := plan(root, !apply)
	if err != nil {
		return err
	}
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	fmt.Fprintf(os.Stdout, "Coverage cache %s: %d bytes total, %d bytes selected (max age %s, max %d bytes).\n", report.Root, report.TotalBytes, report.SelectedBytes, report.MaxAge, report.MaxBytes)
	if len(report.Selected) == 0 {
		fmt.Fprintln(os.Stdout, "No files selected.")
		return nil
	}
	fmt.Fprintln(os.Stdout, "Selected before deletion:")
	for _, item := range report.Selected {
		fmt.Fprintf(os.Stdout, "  %s (%d bytes, %s)\n", item.Path, item.Size, item.Reason)
	}
	if !apply {
		fmt.Fprintln(os.Stdout, "Dry run: pass --apply to delete the selected cache files.")
	}
	return nil
}

func plan(root string, dryRun bool) (pruneReport, error) {
	report := pruneReport{Root: root, DryRun: dryRun, MaxAge: maxAge.String(), MaxBytes: maxBytes}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return report, err
	}
	if !info.IsDir() {
		return report, fmt.Errorf("coverage root %s is not a directory", root)
	}
	now := time.Now()
	var files []candidate
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		stat, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		report.TotalBytes += stat.Size()
		if isRetained(rel) {
			report.RetainedArtifact = append(report.RetainedArtifact, rel)
			return nil
		}
		if now.Sub(stat.ModTime()) > maxAge {
			files = append(files, candidate{Path: path, Size: stat.Size(), ModTime: stat.ModTime(), Reason: "older than retention window"})
		}
		return nil
	})
	if err != nil {
		return report, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	selectedBytes := int64(0)
	for _, item := range files {
		selectedBytes += item.Size
	}
	// If age-based pruning does not bring the cache below its byte limit,
	// select the oldest non-retained files until the limit is met.
	if report.TotalBytes-selectedBytes > maxBytes {
		var extra []candidate
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() {
				return walkErr
			}
			rel, _ := filepath.Rel(root, path)
			if isRetained(rel) {
				return nil
			}
			for _, existing := range files {
				if existing.Path == path {
					return nil
				}
			}
			stat, statErr := entry.Info()
			if statErr == nil {
				extra = append(extra, candidate{Path: path, Size: stat.Size(), ModTime: stat.ModTime(), Reason: "oldest files required to meet byte limit"})
			}
			return nil
		})
		sort.Slice(extra, func(i, j int) bool {
			if extra[i].ModTime.Equal(extra[j].ModTime) {
				return extra[i].Path < extra[j].Path
			}
			return extra[i].ModTime.Before(extra[j].ModTime)
		})
		for _, item := range extra {
			if report.TotalBytes-selectedBytes <= maxBytes {
				break
			}
			files = append(files, item)
			selectedBytes += item.Size
		}
	}
	report.Selected = files
	report.SelectedBytes = selectedBytes
	if !dryRun {
		for i := range report.Selected {
			if err := os.Remove(report.Selected[i].Path); err != nil {
				return report, fmt.Errorf("remove %s: %w", report.Selected[i].Path, err)
			}
			report.Selected[i].Deleted = true
		}
	}
	return report, nil
}

func isRetained(path string) bool {
	base := filepath.Base(path)
	return base == "verdicts.json" || strings.Contains(path, "verdict") || strings.Contains(path, "evaluator") || strings.Contains(path, "calibration") || strings.Contains(path, "hash") || strings.Contains(path, "timing")
}

func findCoverageRoot() string {
	if value := strings.TrimSpace(os.Getenv("VROOLI_REACT_COMPONENT_LIBRARY_COVERAGE")); value != "" {
		return value
	}
	wd, _ := os.Getwd()
	for current := wd; current != filepath.Dir(current); current = filepath.Dir(current) {
		candidate := filepath.Join(current, "scenarios", "react-component-library", "coverage")
		if _, err := os.Stat(filepath.Join(current, "scenarios", "react-component-library")); err == nil {
			return candidate
		}
	}
	return filepath.Join(wd, "coverage")
}
