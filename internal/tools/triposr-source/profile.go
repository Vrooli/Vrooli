// Package triposrsource owns the pinned, data-only TripoSR upstream source tree.
// It is acquired as a managed host artifact and never resolved from ambient paths.
package triposrsource

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/binaryfetch"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	ToolName            = "triposr-source"
	Commit              = "107cefdc244c39106fa830359024f6a2f1c78871"
	ArchiveURL          = "https://codeload.github.com/VAST-AI-Research/TripoSR/tar.gz/107cefdc244c39106fa830359024f6a2f1c78871"
	ArchiveSHA256       = "bcb414550dcfcb9f5ea6a7b9c12f2bbff889f5b4a564a493178393360d9034ec"
	ArchiveBytes  int64 = 31241354
	ArchiveRoot         = "TripoSR-107cefdc244c39106fa830359024f6a2f1c78871"
)

type Receipt struct {
	Schema             int    `json:"schema_version"`
	Commit             string `json:"upstream_commit"`
	ArchiveSHA256      string `json:"archive_sha256"`
	ArchiveBytes       int64  `json:"archive_bytes"`
	TreeSHA256         string `json:"tree_sha256"`
	License            string `json:"license"`
	Adapter            string `json:"adapter_version,omitempty"`
	AdapterTreeSHA256  string `json:"adapter_tree_sha256,omitempty"`
	UpstreamTreeSHA256 string `json:"upstream_tree_sha256,omitempty"`
}

func InstallRoot() (string, error) {
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return "", err
	}
	root, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "opt", ToolName), nil
}

// SelectedInstallation resolves only the ledger-owned pinned source tree and
// rehashes it under the acquisition lock before returning it to the provider.
func SelectedInstallation(ctx context.Context) (string, Receipt, error) {
	if ctx == nil {
		return "", Receipt{}, errors.New("nil context")
	}
	home, err := config.HomeDir()
	if err != nil {
		return "", Receipt{}, err
	}
	root, err := InstallRoot()
	if err != nil {
		return "", Receipt{}, err
	}
	unlock, err := artifactlock.AcquireContext(ctx, ToolName)
	if err != nil {
		return "", Receipt{}, err
	}
	defer unlock()
	ledger, err := cliinstall.LoadInstallRecord(home)
	if err != nil {
		return "", Receipt{}, err
	}
	owned := false
	for _, e := range ledger.Entries {
		if e.Scope == cliinstall.ScopeRuntime && e.Kind == cliinstall.EntryDirectory && filepath.Clean(e.Path) == filepath.Clean(root) {
			owned = true
			break
		}
	}
	if !owned {
		return "", Receipt{}, errors.New("TripoSR source tree is not recorded as runtime-owned")
	}
	var receipt Receipt
	b, err := os.ReadFile(filepath.Join(root, "source-profile.json"))
	if err != nil {
		return "", Receipt{}, err
	}
	if err := json.Unmarshal(b, &receipt); err != nil {
		return "", Receipt{}, err
	}
	if receipt.Schema != 2 || receipt.Commit != Commit || receipt.ArchiveSHA256 != ArchiveSHA256 || receipt.ArchiveBytes != ArchiveBytes || receipt.License != "MIT" || receipt.Adapter != AdapterVersion {
		return "", Receipt{}, errors.New("TripoSR source profile adapter is missing or stale; run `vrooli host install triposr-source`")
	}
	tree := filepath.Join(root, "adapter", "source")
	digest, err := binaryfetch.TreeDigestContext(ctx, tree)
	if err != nil {
		return "", Receipt{}, err
	}
	if digest != receipt.TreeSHA256 || digest != receipt.AdapterTreeSHA256 {
		return "", Receipt{}, errors.New("TripoSR source adapter tree hash differs from managed receipt; reinstall triposr-source")
	}
	upstreamDigest, err := binaryfetch.TreeDigestContext(ctx, filepath.Join(root, "source"))
	if err != nil || upstreamDigest != receipt.UpstreamTreeSHA256 {
		return "", Receipt{}, errors.New("TripoSR upstream source tree hash differs from managed receipt")
	}
	return tree, receipt, nil
}

// installAdapterLocked is called only by the SDA-owned Provision path while the
// source-profile artifact lock is held. It retains upstream bytes and atomically
// selects a separately-built adapter tree.
func installAdapterLocked(ctx context.Context, root string, receipt Receipt) (Receipt, error) {
	original := filepath.Join(root, "source")
	upstreamDigest, err := binaryfetch.TreeDigestContext(ctx, original)
	if err != nil {
		return Receipt{}, err
	}
	if receipt.Commit != Commit || receipt.ArchiveSHA256 != ArchiveSHA256 || receipt.License != "MIT" {
		return Receipt{}, errors.New("upstream source identity differs from pinned adapter")
	}
	if receipt.Schema == 2 {
		adapted := filepath.Join(root, "adapter", "source")
		digest, e := binaryfetch.TreeDigestContext(ctx, adapted)
		if e == nil && digest == receipt.AdapterTreeSHA256 && upstreamDigest == receipt.UpstreamTreeSHA256 && receipt.Adapter == AdapterVersion {
			return receipt, nil
		}
		if upstreamDigest != receipt.UpstreamTreeSHA256 {
			return Receipt{}, errors.New("upstream TripoSR source tree changed; refusing adapter repair")
		}
		// Preserve the damaged selected tree for diagnosis, then stage a clean
		// adapter from immutable upstream bytes. The old receipt remains until the
		// candidate is complete and atomically renamed into place.
		quarantine := filepath.Join(root, "adapter-quarantine-"+receipt.AdapterTreeSHA256[:12])
		if _, statErr := os.Lstat(quarantine); statErr == nil {
			actual := "unknown"
			if e == nil {
				actual = digest[:12]
			}
			quarantine += "-" + actual
			if _, secondErr := os.Lstat(quarantine); secondErr == nil {
				return Receipt{}, errors.New("adapter quarantine destination already exists; refusing replacement")
			} else if !os.IsNotExist(secondErr) {
				return Receipt{}, secondErr
			}
		} else if !os.IsNotExist(statErr) {
			return Receipt{}, statErr
		}
		if err = os.Rename(filepath.Join(root, "adapter"), quarantine); err != nil {
			return Receipt{}, err
		}
		receipt.Schema = 1
		receipt.Adapter = ""
		receipt.AdapterTreeSHA256 = ""
		receipt.TreeSHA256 = upstreamDigest
	}
	if receipt.Schema != 1 || upstreamDigest != receipt.TreeSHA256 {
		return Receipt{}, errors.New("unrecognized or modified upstream TripoSR source profile")
	}
	stage := filepath.Join(root, ".adapter-stage")
	if _, err = os.Lstat(stage); err == nil {
		return Receipt{}, errors.New("unrecognized existing TripoSR adapter staging directory")
	} else if !os.IsNotExist(err) {
		return Receipt{}, err
	}
	candidate := filepath.Join(stage, "source")
	if err = os.MkdirAll(stage, 0700); err != nil {
		return Receipt{}, err
	}
	digest, err := applySourceAdapter(original, candidate)
	if err != nil {
		_ = os.RemoveAll(stage)
		return Receipt{}, err
	}
	if err = os.Rename(stage, filepath.Join(root, "adapter")); err != nil {
		_ = os.RemoveAll(stage)
		return Receipt{}, err
	}
	receipt.Schema = 2
	receipt.Adapter = AdapterVersion
	receipt.UpstreamTreeSHA256 = upstreamDigest
	receipt.AdapterTreeSHA256 = digest
	receipt.TreeSHA256 = digest
	b, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return Receipt{}, err
	}
	if err = writeAtomic(filepath.Join(root, "source-profile.json"), b, 0600); err != nil {
		return Receipt{}, err
	}
	return receipt, nil
}

func writeAtomic(path string, contents []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".source-profile-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err = tmp.Write(contents); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func Provision(ctx context.Context) (string, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return "", errors.New("TripoSR source profile is qualified only for Linux amd64")
	}
	root, err := InstallRoot()
	if err != nil {
		return "", err
	}
	unlock, err := artifactlock.AcquireContext(ctx, ToolName)
	if err != nil {
		return "", err
	}
	defer unlock()
	if _, err := os.Lstat(root); err == nil {
		home, homeErr := config.HomeDir()
		if homeErr != nil {
			return "", homeErr
		}
		ledger, ledgerErr := cliinstall.LoadInstallRecord(home)
		if ledgerErr != nil {
			return "", ledgerErr
		}
		owned := false
		for _, entry := range ledger.Entries {
			if entry.Scope == cliinstall.ScopeRuntime && entry.Kind == cliinstall.EntryDirectory && filepath.Clean(entry.Path) == filepath.Clean(root) {
				owned = true
				break
			}
		}
		if !owned {
			return "", errors.New("existing TripoSR source profile is not recorded as runtime-owned")
		}
		var receipt Receipt
		b, readErr := os.ReadFile(filepath.Join(root, "source-profile.json"))
		if readErr != nil {
			return "", readErr
		}
		if readErr = json.Unmarshal(b, &receipt); readErr != nil {
			return "", readErr
		}
		receipt, err = installAdapterLocked(ctx, root, receipt)
		if err != nil {
			return "", err
		}
		return root, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(root)
	if _, err := config.EnsureOwnedDir(parent); err != nil {
		return "", err
	}
	candidate, err := os.MkdirTemp(parent, ".triposr-source-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(candidate)
	archive := binaryfetch.Target{Name: ToolName, URL: ArchiveURL, SHA256: ArchiveSHA256, Archive: "tar.gz", Layout: "dir"}
	if err := binaryfetch.FetchTree(ctx, archive, filepath.Join(candidate, "archive"), nil); err != nil {
		return "", err
	}
	source := filepath.Join(candidate, "archive", ArchiveRoot)
	digest, err := binaryfetch.TreeDigestContext(ctx, source)
	if err != nil {
		return "", err
	}
	r := Receipt{Schema: 1, Commit: Commit, ArchiveSHA256: ArchiveSHA256, ArchiveBytes: ArchiveBytes, TreeSHA256: digest, License: "MIT"}
	if _, err = os.Stat(filepath.Join(source, "LICENSE")); err != nil {
		return "", errors.New("pinned TripoSR source archive is missing its MIT license file")
	}
	if err = os.Rename(source, filepath.Join(candidate, "source")); err != nil {
		return "", err
	}
	if err = os.RemoveAll(filepath.Join(candidate, "archive")); err != nil {
		return "", err
	}
	b, _ := json.MarshalIndent(r, "", "  ")
	if err = os.WriteFile(filepath.Join(candidate, "source-profile.json"), b, 0600); err != nil {
		return "", err
	}
	if r, err = installAdapterLocked(ctx, candidate, r); err != nil {
		return "", err
	}
	if err = os.Rename(candidate, root); err != nil {
		return "", err
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return "", err
	}
	if err = cliinstall.RecordToolArtifacts(home, cliinstall.InstallEntry{Scope: cliinstall.ScopeRuntime, Kind: cliinstall.EntryDirectory, Path: root, Prefix: parent}); err != nil {
		return "", err
	}
	return root, nil
}

func TreeDigest(root string) (string, error) { return binaryfetch.TreeDigest(root) }
func ArchiveIdentity() string {
	sum := sha256.Sum256([]byte(Commit + ":" + ArchiveSHA256))
	return hex.EncodeToString(sum[:])
}

// NewHandler exposes this data-only acquisition through the native host tool
// lifecycle; SDA approves and dispatches the immutable source closure.
func NewHandler(manifest hostreqkit.ToolManifest) hostreqkit.Handler {
	return hostreqkit.InstallerHandler{Manifest: manifest, KindValue: hostreqspec.KindTool,
		InspectFunc: func(host hostreqkit.Host, r hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
			s := hostreqkit.BaseStatus(r)
			s.SupportClass = hostreqkit.SupportSupported
			s.InstallSupported = true
			s.PackageName = ToolName + "@" + Commit
			if host.OS != "linux" {
				s.SupportClass = hostreqkit.SupportUnsupported
				s.ExecutionState = hostreqkit.ExecutionUnsupported
				return s
			}
			if source, _, err := SelectedInstallation(context.Background()); err == nil {
				s.Installed = true
				s.Command = source
				s.ExecutionState = hostreqkit.ExecutionAlreadyPresent
			} else {
				s.ExecutionState = hostreqkit.ExecutionPending
				s.Notes = append(s.Notes, "SDA pinned source acquisition required: "+err.Error())
			}
			return s
		}, ApplyFunc: func(s hostreqkit.ItemStatus, o hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
			if s, done := hostreqkit.ApplyStatus(s); done {
				return s, nil
			}
			args := []string{"deps", "install", "source/triposr-source@" + Commit, "--scenario", ToolName, "--surface", "host-tool", "--exact", "--json"}
			if o.DryRun {
				s.ExecutionState = hostreqkit.ExecutionWouldInstall
				s.Notes = append(s.Notes, "scenario-dependency-analyzer "+strings.Join(args, " "))
				return s, nil
			}
			args = append(args, "--apply")
			if err := hostreqkit.RunAsInvokingUser("scenario-dependency-analyzer", args, o); err != nil {
				return s, err
			}
			source, _, err := SelectedInstallation(context.Background())
			if err != nil {
				return s, err
			}
			s.Command = source
			s.Installed = true
			s.Version = Commit
			s.ExecutionState = hostreqkit.ExecutionInstalled
			return s, nil
		}}
}
