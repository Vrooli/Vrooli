package pinnedrelease

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/vrooli/binaryfetch"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// Profile shares native release mechanics. Tool owners retain their
// embedded manifest, reviewed version, license verification and SDA admission.
type Profile struct {
	Name, Version, License string
	Manifest               []byte
	MaxTreeBytes           int64
	VerifyLicense          func(string) error
}

type Receipt struct {
	Version       string `json:"version"`
	ArchiveSHA256 string `json:"archive_sha256"`
	TreeSHA256    string `json:"tree_sha256"`
	TreeBytes     int64  `json:"tree_bytes"`
	License       string `json:"license"`
}

type FetchFunc func(context.Context, binaryfetch.Target, string, binaryfetch.ProgressFunc) (string, error)

func (p Profile) ValidateSource(source string) error {
	b, err := os.ReadFile(filepath.Join(source, "tool.json"))
	if err != nil {
		return err
	}
	if !bytes.Equal(b, p.Manifest) {
		return fmt.Errorf("%s source manifest differs from pinned owner closure", p.Name)
	}
	return nil
}

func (p Profile) Target(goos, goarch string) (binaryfetch.Target, error) {
	if goos != "linux" || goarch != "amd64" {
		return binaryfetch.Target{}, fmt.Errorf("%s profile unsupported on %s/%s; qualified Linux amd64 only", p.Name, goos, goarch)
	}
	var m hostreqkit.ToolManifest
	if err := json.Unmarshal(p.Manifest, &m); err != nil {
		return binaryfetch.Target{}, err
	}
	if p.Name == "" || p.Version == "" || m.Name != p.Name || m.Version != p.Version || p.License == "" || p.MaxTreeBytes <= 0 || p.VerifyLicense == nil {
		return binaryfetch.Target{}, fmt.Errorf("%s manifest identity or release policy differs from owner", p.Name)
	}
	t, ok := hostreqkit.TargetFor(m.Acquisition, goos, goarch)
	if !ok {
		return binaryfetch.Target{}, fmt.Errorf("%s release target missing", p.Name)
	}
	return binaryfetch.Target{Name: p.Name, URL: t.URL, SHA256: t.SHA256, Archive: t.Archive, Layout: t.Layout, BinPath: t.BinPath}, nil
}

func (p Profile) installation() (string, string, error) {
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return "", "", err
	}
	root, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return "", "", err
	}
	return filepath.Join(root, "opt", p.Name), home, nil
}

func (p Profile) SelectedInstallation(ctx context.Context) (string, Receipt, error) {
	if ctx == nil {
		return "", Receipt{}, fmt.Errorf("nil %s selection context", p.Name)
	}
	if _, err := p.Target(runtime.GOOS, runtime.GOARCH); err != nil {
		return "", Receipt{}, err
	}
	root, home, err := p.installation()
	if err != nil {
		return "", Receipt{}, err
	}
	unlock, err := artifactlock.AcquireContext(ctx, p.Name)
	if err != nil {
		return "", Receipt{}, err
	}
	defer unlock()
	return p.SelectAt(ctx, root, home)
}

// SelectAt checks an attributed immutable installation; it never adopts PATH.
func (p Profile) SelectAt(ctx context.Context, root, home string) (string, Receipt, error) {
	var r Receipt
	if ctx == nil {
		return "", r, fmt.Errorf("nil %s selection context", p.Name)
	}
	if err := ctx.Err(); err != nil {
		return "", r, err
	}
	t, err := p.Target(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", r, err
	}
	info, err := os.Lstat(root)
	if err != nil {
		return "", r, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", r, fmt.Errorf("%s selection must be an owned regular directory", p.Name)
	}
	ledger, err := cliinstall.LoadInstallRecord(home)
	if err != nil {
		return "", r, err
	}
	owned := false
	for _, entry := range ledger.Entries {
		if entry.Scope == cliinstall.ScopeRuntime && entry.Kind == cliinstall.EntryDirectory && filepath.Clean(entry.Path) == filepath.Clean(root) {
			owned = true
			break
		}
	}
	if !owned {
		return "", r, fmt.Errorf("%s selection is not attributed in the native install ledger", p.Name)
	}
	b, err := os.ReadFile(filepath.Join(root, "release.json"))
	if err != nil {
		return "", r, err
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return "", r, err
	}
	if r.Version != p.Version || r.ArchiveSHA256 != t.SHA256 || r.License != p.License || r.TreeBytes <= 0 || r.TreeBytes > p.MaxTreeBytes {
		return "", r, fmt.Errorf("%s receipt differs from pinned release identity", p.Name)
	}
	tree := filepath.Join(root, "tree")
	if err := p.VerifyLicense(tree); err != nil {
		return "", r, err
	}
	if err := binaryfetch.VerifyArtifactContext(ctx, tree, "dir", r.TreeSHA256); err != nil {
		return "", r, err
	}
	entry := filepath.Join(tree, t.BinPath)
	st, err := os.Lstat(entry)
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0o111 == 0 {
		return "", r, fmt.Errorf("%s selected executable is missing, linked or not executable", p.Name)
	}
	return entry, r, nil
}

// Provision is called only after the tool's closed SDA resolver admits it.
func (p Profile) Provision(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("nil %s acquisition context", p.Name)
	}
	target, err := p.Target(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	root, home, err := p.installation()
	if err != nil {
		return "", err
	}
	unlock, err := artifactlock.AcquireContext(ctx, p.Name)
	if err != nil {
		return "", err
	}
	defer unlock()
	return p.ProvisionAt(ctx, root, home, target, binaryfetch.FetchDir)
}

// ProvisionAt shares staging and publication with injected transport for tests.
// The caller holds the per-tool lock when operating a real installation.
func (p Profile) ProvisionAt(ctx context.Context, root, home string, target binaryfetch.Target, fetch FetchFunc) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("nil %s acquisition context", p.Name)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	expected, err := p.Target(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}
	if target != expected || fetch == nil {
		return "", fmt.Errorf("%s acquisition target differs from pinned owner closure", p.Name)
	}
	if _, err := os.Lstat(root); err == nil {
		path, _, err := p.SelectAt(ctx, root, home)
		return path, err
	} else if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(root)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	candidate, err := os.MkdirTemp(parent, "."+p.Name+"-release-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(candidate)
	tree := filepath.Join(candidate, "tree")
	if _, err := fetch(ctx, target, tree, nil); err != nil {
		return "", err
	}
	if err := p.VerifyLicense(tree); err != nil {
		return "", err
	}
	var size int64
	if err := filepath.WalkDir(tree, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.Type().IsRegular() {
			st, err := d.Info()
			if err != nil {
				return err
			}
			size += st.Size()
		}
		if size > p.MaxTreeBytes {
			return fmt.Errorf("%s extracted tree exceeds reviewed %d-byte ceiling", p.Name, p.MaxTreeBytes)
		}
		return nil
	}); err != nil {
		return "", err
	}
	digest, err := binaryfetch.TreeDigestContext(ctx, tree)
	if err != nil {
		return "", err
	}
	r := Receipt{Version: p.Version, ArchiveSHA256: target.SHA256, TreeSHA256: digest, TreeBytes: size, License: p.License}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(candidate, "release.json"), append(b, '\n'), 0o600); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.Rename(candidate, root); err != nil {
		return "", err
	}
	if err := cliinstall.RecordToolArtifacts(home, cliinstall.InstallEntry{Scope: cliinstall.ScopeRuntime, Kind: cliinstall.EntryDirectory, Path: root, Prefix: parent}); err != nil {
		return "", errors.Join(err, os.RemoveAll(root))
	}
	path, _, err := p.SelectAt(ctx, root, home)
	return path, err
}
