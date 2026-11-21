package annotations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// goldRoot is the private directory gold files live in, as the API resolved it
// from runtime.gold_root and the scenario's durable state root. Gold rows carry
// the operator's own words, so the CLI reads and writes them here, never in a
// tracked path. An empty root means the server could not resolve one.
type goldRoot string

// fetchGoldRoot asks the API where gold lives, so the CLI never guesses a home.
func fetchGoldRoot(ctx cliapp.RunContext) (goldRoot, error) {
	body, err := ctx.Core().Request("GET", "annotations/packs", nil, nil)
	if err != nil {
		return "", err
	}
	var response struct {
		Packs []struct {
			GoldRoot string `json:"gold_root"`
		} `json:"packs"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("decode packs: %w", err)
	}
	if len(response.Packs) == 0 {
		return "", nil
	}
	return goldRoot(response.Packs[0].GoldRoot), nil
}

// resolve turns a --gold style argument into a filesystem path: an absolute
// path as written, a relative one against the gold root.
func (r goldRoot) resolve(path string) (string, error) {
	path = strings.TrimSpace(path)
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	if r == "" {
		return "", fmt.Errorf("%q is relative but the server reports no gold root; pass an absolute path or set runtime.gold_root", path)
	}
	return filepath.Join(string(r), path), nil
}

// name is how a measurement records a gold file: relative to the root when it
// lives there, so the record survives moving the state directory, else absolute.
func (r goldRoot) name(resolved string) string {
	if r == "" {
		return resolved
	}
	if rel, err := filepath.Rel(string(r), resolved); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return resolved
}

// create opens a gold file for writing, owner-only, creating the private
// directory it lives in. A file that already exists is tightened to 0600.
func (r goldRoot) create(path string) (*os.File, string, error) {
	resolved, err := r.resolve(path)
	if err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o700); err != nil {
		return nil, "", err
	}
	file, err := os.OpenFile(resolved, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, "", err
	}
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return nil, "", err
	}
	return file, resolved, nil
}

// readRows resolves path against the root and reads its gold rows.
func (r goldRoot) readRows(path string) ([]goldFileRow, error) {
	resolved, err := r.resolve(path)
	if err != nil {
		return nil, err
	}
	return readGoldRows(resolved)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
