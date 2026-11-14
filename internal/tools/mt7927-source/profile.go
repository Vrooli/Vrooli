// Package mt7927source stages reviewed wireless sources and vendor archives as data.
package mt7927source

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/binaryfetch"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

const (
	Name     = "mt7927-source"
	Version  = "5467fd1237391e8b76625c0fd6fad3285d7ff930"
	Manifest = "source-profile.json"

	NobleName            = "mt7927-noble-source"
	NobleVersion         = "35a5dcc71735eb42606be34e313f9eb8148b6a42"
	ASUSBluetoothName    = "asus-mt6639-local-source"
	ASUSBluetoothVersion = "fc088966f2357af65f16f48178e047fa553ed5eabad34bcf0c7ab4ce2033435a"
	ASUSBluetoothArchive = "DRV_Bluetooth_MTK_MT7925_27_TP_W11_64_V111470610_20260706R.zip"
)

//go:embed source-profile.json
var manifest []byte

//go:embed noble-source-profile.json
var nobleManifest []byte

//go:embed asus-bluetooth-source-profile.json
var asusBluetoothManifest []byte

// Profile is one of the exact inputs compiled into the acquisition owner.
type Profile struct {
	name, version, filename, description string
	manifest                             []byte
}

func Lookup(name string) (Profile, bool) {
	switch name {
	case Name:
		return Profile{Name, Version, Manifest, "MT7927 source and Wi-Fi firmware", manifest}, true
	case NobleName:
		return Profile{NobleName, NobleVersion, "noble-source-profile.json", "MT7927 Noble kernel source and PCIe patch series", nobleManifest}, true
	case ASUSBluetoothName:
		return Profile{ASUSBluetoothName, ASUSBluetoothVersion, "asus-bluetooth-source-profile.json", "ASUS MT6639 vendor archive for operator-local qualification", asusBluetoothManifest}, true
	default:
		return Profile{}, false
	}
}

func (p Profile) Version() string     { return p.version }
func (p Profile) Description() string { return p.description }

// DescriptorSHA256 identifies the exact descriptor published and read back
// with this immutable source closure. Native state contracts include it so
// prior binaries cannot silently reinterpret persisted acquisition inputs.
func (p Profile) DescriptorSHA256() string { return fmt.Sprintf("%x", sha256.Sum256(p.manifest)) }

func (p Profile) ManifestPath(root string) string {
	return filepath.Join(root, "internal", "tools", Name, p.filename)
}

// StagedPath is the immutable publication location for this exact profile.
func (p Profile) StagedPath() (string, error) {
	parent, _, err := p.StageBudget()
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, p.version), nil
}

// VerifyStaged verifies a published closure or its private scratch copy.
func (p Profile) VerifyStaged(ctx context.Context, path string) error {
	items, err := p.inputs()
	if err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("unexpected MT7927 artifact path %s", path)
	}
	return p.verifyStage(ctx, path, items)
}

type input struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

func (p Profile) inputs() ([]input, error) {
	var profile struct {
		Version string  `json:"version"`
		Inputs  []input `json:"inputs"`
	}
	if err := json.Unmarshal(p.manifest, &profile); err != nil {
		return nil, err
	}
	if profile.Version != p.version || len(profile.Inputs) == 0 {
		return nil, errors.New("MT7927 source profile identity mismatch")
	}
	names := map[string]bool{Manifest: true}
	for _, item := range profile.Inputs {
		if !filepath.IsLocal(item.Name) || filepath.Base(item.Name) != item.Name || item.Bytes <= 0 || names[item.Name] {
			return nil, fmt.Errorf("invalid MT7927 source input %q", item.Name)
		}
		names[item.Name] = true
	}
	return profile.Inputs, nil
}

// VerifyManifest prevents an older gateway from applying pins different from
// the source checkout that it describes in its dry run.
func (p Profile) VerifyManifest(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, p.manifest) {
		return errors.New("MT7927 source profile differs from the dependency gateway build")
	}
	return nil
}

// StageBudget covers every write under a single contract-owned directory.
func (p Profile) StageBudget() (string, int64, error) {
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return "", 0, err
	}
	root, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyArtifacts)
	if err != nil {
		return "", 0, err
	}
	items, err := p.inputs()
	if err != nil {
		return "", 0, err
	}
	// Allow for filesystem metadata, allocation rounding, receipt and lock.
	peak := int64(1 << 20)
	for _, item := range items {
		peak += item.Bytes
	}
	return filepath.Join(root, p.name), peak, nil
}

// Provision publishes the complete verified closure at an immutable path.
// Downloads, locking and publication use the same disk-admitted parent.
func (p Profile) Provision(ctx context.Context) (string, error) {
	parent, _, err := p.StageBudget()
	if err != nil {
		return "", err
	}
	items, err := p.inputs()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	unlock, err := artifactlock.AcquireWithPathContext(ctx, p.name, func(string) (string, error) {
		return filepath.Join(parent, ".stage.lock"), nil
	})
	if err != nil {
		return "", err
	}
	defer unlock()
	return p.stage(ctx, parent, items)
}

func (p Profile) stage(ctx context.Context, parent string, items []input) (string, error) {
	final := filepath.Join(parent, p.version)
	if info, err := os.Lstat(final); err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("unexpected MT7927 artifact path %s", final)
		}
		if err := p.verifyStage(ctx, final, items); err != nil {
			return "", fmt.Errorf("existing MT7927 artifact failed verification: %w", err)
		}
		return final, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	candidate, err := os.MkdirTemp(parent, ".stage-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(candidate)
	for _, item := range items {
		target := binaryfetch.Target{URL: item.URL, SHA256: item.SHA256}
		if err := binaryfetch.DownloadExact(ctx, target, filepath.Join(candidate, item.Name), item.Bytes); err != nil {
			return "", fmt.Errorf("stage %s: %w", item.Name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(candidate, Manifest), p.manifest, 0600); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.Rename(candidate, final); err != nil {
		return "", err
	}
	return final, nil
}

func (p Profile) verifyStage(ctx context.Context, root string, items []input) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	if len(entries) != len(items)+1 {
		return errors.New("MT7927 artifact contains an unexpected set of files")
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected MT7927 artifact entry %s", entry.Name())
		}
	}
	if err := p.VerifyManifest(filepath.Join(root, Manifest)); err != nil {
		return err
	}
	for _, item := range items {
		path := filepath.Join(root, item.Name)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() != item.Bytes {
			return fmt.Errorf("unexpected staged input size or type for %s", item.Name)
		}
		sum, err := binaryfetch.FileDigestContext(ctx, path)
		if err != nil {
			return err
		}
		if sum != item.SHA256 {
			return fmt.Errorf("staged input checksum mismatch for %s", item.Name)
		}
	}
	return nil
}
