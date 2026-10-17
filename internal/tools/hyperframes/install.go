// Package hyperframes owns the pinned local HyperFrames tool and bounded renders.
package hyperframes

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const Version = "0.8.112"

//go:embed package.json package-lock.json
var closure embed.FS

func hashBytes(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }

func fileHash(path string) (string, error) { return fileHashContext(context.Background(), path) }

func fileHashContext(ctx context.Context, path string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	f, err := openRegularContext(ctx, path, 2<<30)
	if err != nil {
		return "", err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !before.Mode().IsRegular() || before.Size() > 2<<30 {
		return "", fmt.Errorf("hash input is not a bounded regular file: %s", path)
	}
	h := sha256.New()
	n, err := io.Copy(h, contextReader{ctx: ctx, reader: io.LimitReader(f, 2<<30+1)})
	if err != nil {
		return "", err
	}
	after, err := f.Stat()
	if err != nil {
		return "", err
	}
	if n > 2<<30 || n != before.Size() || after.Size() != before.Size() || after.ModTime() != before.ModTime() {
		return "", fmt.Errorf("file changed while hashing: %s", path)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ClosureID identifies the committed dependency inputs, including all integrity pins.
func ClosureID() string {
	a, _ := closure.ReadFile("package.json")
	b, _ := closure.ReadFile("package-lock.json")
	return hashBytes(append(append(a, '\n'), b...))
}

func InstallRoot() (string, error) {
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return "", err
	}
	cache, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyCache)
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, "host-tools", "hyperframes"), nil
}

type Installation struct {
	Version string `json:"version"`
	Closure string `json:"closure_sha256"`
	Tree    string `json:"tree_sha256"`
	Entry   string `json:"entry"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

// TreeHash includes file paths, regular bytes and symlink targets without following
// links. It rejects absolute/escaping links, devices and excessive artifact trees.
func TreeHash(root string) (string, error) { return TreeHashContext(context.Background(), root) }

func TreeHashContext(ctx context.Context, root string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	h := sha256.New()
	var total int64
	count := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." || rel == "installation.json" {
			return nil
		}
		count++
		if count > 100000 {
			return errors.New("tool tree exceeds file budget")
		}
		fmt.Fprintf(h, "%s\x00%d\x00", filepath.ToSlash(rel), d.Type())
		if d.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			joined := filepath.Clean(filepath.Join(filepath.Dir(rel), target))
			if filepath.IsAbs(target) || joined == ".." || strings.HasPrefix(joined, ".."+string(filepath.Separator)) {
				return fmt.Errorf("escaping tool symlink %s", rel)
			}
			fmt.Fprintln(h, target)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("unsupported tool file %s", rel)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		if total > 2<<30 {
			return errors.New("tool tree exceeds 2 GiB")
		}
		digest, err := fileHashContext(ctx, path)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%o\x00%s\x00", info.Mode().Perm(), digest)
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyInstallation never trusts PATH presence as managed provenance.
func VerifyInstallation(dir string) (Installation, error) {
	return VerifyInstallationContext(context.Background(), dir)
}

func VerifyInstallationContext(ctx context.Context, dir string) (Installation, error) {
	if err := ctx.Err(); err != nil {
		return Installation{}, err
	}
	var result Installation
	b, err := readFileContext(ctx, filepath.Join(dir, "installation.json"), 1<<20)
	if err != nil {
		return result, err
	}
	if err = json.Unmarshal(b, &result); err != nil {
		return result, err
	}
	if result.Version != Version || result.Closure != ClosureID() || result.Entry != "node_modules/hyperframes/bin/hyperframes.mjs" || result.OS != runtime.GOOS || result.Arch != runtime.GOARCH {
		return result, errors.New("tool installation identity mismatch")
	}
	tree, err := TreeHashContext(ctx, dir)
	if err != nil {
		return result, err
	}
	if tree != result.Tree {
		return result, errors.New("tool installation bytes changed")
	}
	return result, nil
}

func SelectedInstallation() (string, Installation, error) {
	return SelectedInstallationContext(context.Background())
}

func SelectedInstallationContext(ctx context.Context) (string, Installation, error) {
	if err := ctx.Err(); err != nil {
		return "", Installation{}, err
	}
	root, err := InstallRoot()
	if err != nil {
		return "", Installation{}, err
	}
	dir := filepath.Join(root, ClosureID()+"-"+runtime.GOOS+"-"+runtime.GOARCH)
	receipt, err := VerifyInstallationContext(ctx, dir)
	return dir, receipt, err
}

// ValidateSource requires the committed closure; a caller cannot substitute a
// different lock while claiming the registered tool's exact version.
func ValidateSource(source string) error {
	return ValidateSourceContext(context.Background(), source)
}

func ValidateSourceContext(ctx context.Context, source string) error {
	for _, name := range []string{"package.json", "package-lock.json"} {
		want, _ := closure.ReadFile(name)
		got, err := readFileContext(ctx, filepath.Join(source, name), 8<<20)
		if err != nil {
			return err
		}
		if hashBytes(got) != hashBytes(want) {
			return fmt.Errorf("unrecognized HyperFrames dependency input %s", name)
		}
	}
	// A newly compiled binary may have embedded a crash-partial pair or source
	// refresh whose owner version metadata has not yet been reviewed/aligned.
	manifest, _ := closure.ReadFile("package.json")
	lock, _ := closure.ReadFile("package-lock.json")
	var m struct{ Dependencies map[string]string }
	var l struct {
		Packages map[string]struct{ Version string }
	}
	if json.Unmarshal(manifest, &m) != nil || json.Unmarshal(lock, &l) != nil || m.Dependencies["hyperframes"] != Version || l.Packages["node_modules/hyperframes"].Version != Version {
		return errors.New("dependency input pair and owner version are inconsistent; recover or review both source inputs and version metadata before provisioning")
	}
	return nil
}

// Provision is called only after SDA governance accepts the request. npm itself
// remains behind SDA's installer seam; this function owns file publication.
func Provision(ctx context.Context, source string, install func(context.Context, string) (string, error)) (string, error) {
	if err := ValidateSourceContext(ctx, source); err != nil {
		return "", err
	}
	root, err := InstallRoot()
	if err != nil {
		return "", err
	}
	release, err := artifactlock.Acquire("hyperframes-" + ClosureID())
	if err != nil {
		return "", err
	}
	defer release()
	dest := filepath.Join(root, ClosureID()+"-"+runtime.GOOS+"-"+runtime.GOARCH)
	if _, err := VerifyInstallationContext(ctx, dest); err == nil {
		return "Verified existing frozen HyperFrames installation", nil
	}
	if _, err := os.Lstat(dest); err == nil {
		return "", fmt.Errorf("existing installation is invalid; preserve and inspect %s", dest)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if _, err = config.EnsureOwnedDir(root); err != nil {
		return "", err
	}
	candidate, err := os.MkdirTemp(root, ".candidate-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(candidate)
	for _, name := range []string{"package.json", "package-lock.json"} {
		b, _ := closure.ReadFile(name)
		if err = os.WriteFile(filepath.Join(candidate, name), b, 0o600); err != nil {
			return "", err
		}
	}
	output, err := install(ctx, candidate)
	if err != nil {
		return output, err
	}
	if err = ctx.Err(); err != nil {
		return output, err
	}
	// npm ci may update ancillary metadata but cannot rewrite the frozen inputs.
	if err = ValidateSource(candidate); err != nil {
		return output, err
	}
	var pkg struct {
		Version string `json:"version"`
	}
	b, err := os.ReadFile(filepath.Join(candidate, "node_modules", "hyperframes", "package.json"))
	if err != nil {
		return output, err
	}
	if err = json.Unmarshal(b, &pkg); err != nil {
		return output, err
	}
	if pkg.Version != Version {
		return output, errors.New("installed HyperFrames version mismatch")
	}
	entry := "node_modules/hyperframes/bin/hyperframes.mjs"
	if _, err = os.Stat(filepath.Join(candidate, entry)); err != nil {
		return output, err
	}
	tree, err := TreeHashContext(ctx, candidate)
	if err != nil {
		return output, err
	}
	receipt, _ := json.MarshalIndent(Installation{Version: Version, Closure: ClosureID(), Tree: tree, Entry: entry, OS: runtime.GOOS, Arch: runtime.GOARCH}, "", "  ")
	if err = os.WriteFile(filepath.Join(candidate, "installation.json"), append(receipt, '\n'), 0o600); err != nil {
		return output, err
	}
	if err = os.Rename(candidate, dest); err != nil {
		return output, err
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return output, err
	}
	if err = cliinstall.RecordToolArtifacts(home, cliinstall.InstallEntry{Scope: cliinstall.ScopeRuntime, Kind: cliinstall.EntryDirectory, Path: dest, Prefix: root}); err != nil {
		return output, fmt.Errorf("record managed tool inventory: %w", err)
	}
	return output, nil
}

func NewHandler(manifest hostreqkit.ToolManifest) hostreqkit.Handler {
	return hostreqkit.InstallerHandler{
		Manifest: manifest, KindValue: hostreqspec.KindTool,
		InspectFunc: func(host hostreqkit.Host, r hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
			s := hostreqkit.BaseStatus(r)
			s.SupportClass = hostreqkit.SupportSupported
			s.InstallSupported = true
			s.PackageName = "hyperframes@" + Version
			if host.OS != "linux" {
				s.SupportClass = hostreqkit.SupportUnsupported
				s.ExecutionState = hostreqkit.ExecutionUnsupported
				return s
			}
			dir, _, err := SelectedInstallation()
			if err == nil {
				s.Installed = true
				s.Command = filepath.Join(dir, "node_modules/hyperframes/bin/hyperframes.mjs")
				s.Version = Version
				s.ExecutionState = hostreqkit.ExecutionAlreadyPresent
			} else {
				s.ExecutionState = hostreqkit.ExecutionPending
				s.Notes = append(s.Notes, "SDA frozen provisioning required: "+err.Error())
			}
			return s
		}, ApplyFunc: func(s hostreqkit.ItemStatus, o hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
			if s, done := hostreqkit.ApplyStatus(s); done {
				return s, nil
			}
			args := []string{"deps", "install", "npm/hyperframes@" + Version, "--scenario", "hyperframes", "--surface", "host-tool", "--exact", "--json"}
			if o.DryRun {
				s.ExecutionState = hostreqkit.ExecutionWouldInstall
				s.Notes = append(s.Notes, "scenario-dependency-analyzer "+strings.Join(args, " "))
				return s, nil
			}
			args = append(args, "--apply")
			if err := hostreqkit.RunAsInvokingUser("scenario-dependency-analyzer", args, o); err != nil {
				return s, err
			}
			dir, _, err := SelectedInstallation()
			if err != nil {
				return s, err
			}
			s.Installed = true
			s.Command = filepath.Join(dir, "node_modules/hyperframes/bin/hyperframes.mjs")
			s.Version = Version
			s.ExecutionState = hostreqkit.ExecutionInstalled
			return s, nil
		},
	}
}

// contextReader deliberately hides WriterTo so io.Copy checks cancellation on
// every bounded read instead of taking os.File's uninterruptible fast path.
type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	if e := r.ctx.Err(); e != nil {
		return n, e
	}
	return n, err
}

func readFileContext(ctx context.Context, path string, max int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := openRegularContext(ctx, path, max)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(contextReader{ctx: ctx, reader: io.LimitReader(f, max+1)})
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, errors.New("file exceeds byte budget")
	}
	return b, nil
}
