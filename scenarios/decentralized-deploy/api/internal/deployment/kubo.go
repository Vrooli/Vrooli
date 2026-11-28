package deployment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/vrooli/api-core/storage"
)

type KuboRunner interface {
	Run(context.Context, string, string, ...string) ([]byte, error)
}

type execKuboRunner struct{}

func (execKuboRunner) Run(ctx context.Context, binary, repo string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	env := make([]string, 0, len(os.Environ())+1)
	for _, pair := range os.Environ() {
		if !strings.HasPrefix(pair, "IPFS_PATH=") {
			env = append(env, pair)
		}
	}
	cmd.Env = append(env, "IPFS_PATH="+repo)
	return cmd.CombinedOutput()
}

// KuboAdapter owns one offline task repository. It invokes the exact managed
// Kubo executable, disables bootstrap/swarm networking at repository creation,
// pins a recursively added directory root, and removes the owned repository at
// teardown. It never reads or mutates the operator's default IPFS repository.
type KuboAdapter struct {
	Binary string
	Repo   string
	Entry  string
	Gate   Gate
	Runner KuboRunner
}

const kuboOwnershipMarker = ".decentralized-deploy-owned"
const kuboOwnershipValue = "decentralized-deploy/offline-kubo/v1\n"

func (k KuboAdapter) Name() string { return "kubo" }

func (k KuboAdapter) runner() KuboRunner {
	if k.Runner != nil {
		return k.Runner
	}
	return execKuboRunner{}
}

func (k KuboAdapter) validate() error {
	if !filepath.IsAbs(k.Binary) {
		return fmt.Errorf("Kubo binary must be an absolute path to the managed installation")
	}
	if !filepath.IsAbs(k.Repo) || k.Repo == string(filepath.Separator) {
		return fmt.Errorf("Kubo repository must be an absolute task-owned directory")
	}
	if (filepath.IsAbs(k.Entry) || strings.HasPrefix(filepath.Clean(k.Entry), "..")) && k.Entry != "" {
		return fmt.Errorf("Kubo entry path must be relative to the artifact root")
	}
	return nil
}

func (k KuboAdapter) run(ctx context.Context, repo string, args ...string) ([]byte, error) {
	if err := k.validate(); err != nil {
		return nil, err
	}
	root, err := openOwnedRepo(repo)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	output, err := k.runner().Run(ctx, k.Binary, repo, args...)
	if err != nil {
		return output, fmt.Errorf("Kubo %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func (k KuboAdapter) ensureOfflineRepo(ctx context.Context, repo string) (bool, error) {
	created := false
	if _, err := os.Lstat(repo); os.IsNotExist(err) {
		if err := storage.EnsureDirectory(repo, 0o700); err != nil {
			return created, fmt.Errorf("create task Kubo repo: %w", err)
		}
		created = true
		root, err := os.OpenRoot(repo)
		if err != nil {
			return created, fmt.Errorf("open task Kubo repository: %w", err)
		}
		defer root.Close()
		if err := storage.WriteFileAtomicInRoot(root, kuboOwnershipMarker, []byte(kuboOwnershipValue), 0o600); err != nil {
			return created, fmt.Errorf("mark task Kubo repo: %w", err)
		}
	} else if err != nil {
		return created, fmt.Errorf("inspect task Kubo repo: %w", err)
	} else {
		root, err := openOwnedRepo(repo)
		if err != nil {
			return created, err
		}
		defer root.Close()
	}
	if _, err := os.Stat(filepath.Join(repo, "config")); errorsIsNotExist(err) {
		if _, err := k.run(ctx, repo, "init", "--empty-repo=true"); err != nil {
			return created, err
		}
		for _, args := range [][]string{
			{"config", "Bootstrap", "[]", "--json"},
			{"config", "Addresses.Swarm", "[]", "--json"},
			{"config", "Routing.Type", "none"},
		} {
			if _, err := k.run(ctx, repo, args...); err != nil {
				return created, fmt.Errorf("disable networking on task Kubo repo: %w", err)
			}
		}
	} else if err != nil {
		return created, fmt.Errorf("inspect task Kubo repo: %w", err)
	}
	return created, nil
}

func errorsIsNotExist(err error) bool { return err != nil && os.IsNotExist(err) }

func (k KuboAdapter) deploymentRepo(deploymentID string) (string, string, error) {
	if err := k.validate(); err != nil {
		return "", "", err
	}
	if deploymentID == "" {
		return k.Repo, "", nil
	}
	if _, err := os.Lstat(k.Repo); os.IsNotExist(err) {
		if err := storage.EnsureDirectory(k.Repo, 0o700); err != nil {
			return "", "", fmt.Errorf("create Kubo repo root: %w", err)
		}
	} else if err != nil {
		return "", "", fmt.Errorf("inspect Kubo repo root: %w", err)
	}
	info, err := os.Lstat(k.Repo)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", "", fmt.Errorf("Kubo repo root must be an owned directory, not a symlink")
	}
	sum := sha256.Sum256([]byte(deploymentID))
	repoID := hex.EncodeToString(sum[:16])
	return filepath.Join(k.Repo, repoID), repoID, nil
}

func (k KuboAdapter) contentRepo(repoID string) (string, error) {
	if repoID == "" {
		return k.Repo, nil
	}
	if len(repoID) != 32 {
		return "", fmt.Errorf("invalid Kubo repository reference")
	}
	if _, err := hex.DecodeString(repoID); err != nil {
		return "", fmt.Errorf("invalid Kubo repository reference")
	}
	return filepath.Join(k.Repo, repoID), nil
}

func openOwnedRepo(repo string) (*os.Root, error) {
	info, err := os.Lstat(repo)
	if err != nil {
		return nil, fmt.Errorf("inspect task Kubo repository %s: %w", repo, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("refusing non-directory or symlink Kubo repository %s", repo)
	}
	root, err := os.OpenRoot(repo)
	if err != nil {
		return nil, fmt.Errorf("open task Kubo repository: %w", err)
	}
	marker, err := root.ReadFile(kuboOwnershipMarker)
	if err != nil || string(marker) != kuboOwnershipValue {
		_ = root.Close()
		return nil, fmt.Errorf("refusing to access unowned Kubo repository %s", repo)
	}
	return root, nil
}

func removeOwnedRepo(repo string) error {
	root, err := openOwnedRepo(repo)
	if err != nil {
		return err
	}
	closeErr := root.Close()
	if closeErr != nil {
		return closeErr
	}
	parent, err := os.OpenRoot(filepath.Dir(repo))
	if err != nil {
		return err
	}
	defer parent.Close()
	return parent.RemoveAll(filepath.Base(repo))
}

func removeEmptyRepoRoot(repo string) error {
	info, err := os.Lstat(repo)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("refusing to remove non-directory Kubo repo root %s", repo)
	}
	entries, err := os.ReadDir(repo)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return nil
	}
	parent, err := os.OpenRoot(filepath.Dir(repo))
	if err != nil {
		return err
	}
	defer parent.Close()
	if err := parent.Remove(filepath.Base(repo)); err != nil && !errors.Is(err, syscall.ENOTEMPTY) && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (k KuboAdapter) Deploy(ctx context.Context, artifact Artifact) (StoredContent, error) {
	if err := k.Gate.Check(ctx, EffectDevnet, k.Name()); err != nil {
		return StoredContent{}, err
	}
	if err := k.validate(); err != nil {
		return StoredContent{}, err
	}
	info, err := os.Lstat(artifact.Path)
	if err != nil {
		return StoredContent{}, fmt.Errorf("inspect artifact: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return StoredContent{}, fmt.Errorf("Kubo artifact must not be a symlink")
	}
	carImport := !info.IsDir()
	if carImport && (!info.Mode().IsRegular() || !strings.EqualFold(filepath.Ext(artifact.Path), ".car")) {
		return StoredContent{}, fmt.Errorf("Kubo artifact must be a directory or a regular .car archive")
	}
	if artifact.Name == "" {
		return StoredContent{}, fmt.Errorf("Kubo artifact name is required")
	}
	if !carImport && k.Entry == "" {
		return StoredContent{}, fmt.Errorf("directory artifacts require a relative entry path")
	}
	repo, repoID, err := k.deploymentRepo(artifact.DeploymentID)
	if err != nil {
		return StoredContent{}, err
	}
	if !carImport {
		if err := filepath.WalkDir(artifact.Path, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Type()&fs.ModeSymlink != 0 {
				return fmt.Errorf("artifact contains a symlink: %s", path)
			}
			return nil
		}); err != nil {
			return StoredContent{}, fmt.Errorf("inspect artifact tree: %w", err)
		}
	}
	created, err := k.ensureOfflineRepo(ctx, repo)
	if err != nil {
		if created {
			_ = removeOwnedRepo(repo)
		}
		return StoredContent{}, err
	}
	defer func() {
		if created {
			_ = removeOwnedRepo(repo)
		}
	}()
	var cid string
	if carImport {
		output, err := k.run(ctx, repo, "--enc=json", "dag", "import", "--pin-roots=true", artifact.Path)
		if err != nil {
			return StoredContent{}, err
		}
		cid, err = importedRootCID(output)
		if err != nil {
			return StoredContent{}, fmt.Errorf("import CAR: %w", err)
		}
	} else {
		added, err := k.run(ctx, repo, "add", "--recursive", "--quieter", "--cid-version=1", artifact.Path)
		if err != nil {
			return StoredContent{}, err
		}
		cid = strings.TrimSpace(string(added))
		if cid == "" || strings.ContainsAny(cid, "\r\n ") {
			return StoredContent{}, fmt.Errorf("Kubo returned an invalid root CID %q", cid)
		}
		if _, err := k.run(ctx, repo, "pin", "add", "--recursive", cid); err != nil {
			return StoredContent{}, fmt.Errorf("pin directory root: %w", err)
		}
	}
	created = false
	return StoredContent{Provider: k.Name(), CID: cid, URI: "ipfs://" + cid, EntryPath: filepath.ToSlash(k.Entry), RepoID: repoID}, nil
}

type dagImportEvent struct {
	Root struct {
		CID         map[string]string `json:"Cid"`
		PinErrorMsg string            `json:"PinErrorMsg"`
	} `json:"Root"`
}

func importedRootCID(output []byte) (string, error) {
	var events []dagImportEvent
	if err := json.Unmarshal(output, &events); err != nil {
		var event dagImportEvent
		if singleErr := json.Unmarshal(output, &event); singleErr != nil {
			return "", fmt.Errorf("decode Kubo dag import result: %w (output %q)", err, strings.TrimSpace(string(output)))
		}
		events = []dagImportEvent{event}
	}
	if len(events) != 1 {
		return "", fmt.Errorf("expected one CAR root, got %d", len(events))
	}
	event := events[0]
	if strings.TrimSpace(event.Root.PinErrorMsg) != "" {
		return "", fmt.Errorf("Kubo did not pin CAR root: %s", event.Root.PinErrorMsg)
	}
	cid := event.Root.CID["/"]
	if cid == "" || strings.ContainsAny(cid, "\r\n ") {
		return "", fmt.Errorf("Kubo returned an invalid CAR root CID %q", cid)
	}
	return cid, nil
}

func (k KuboAdapter) Renew(ctx context.Context, content StoredContent) error {
	if err := k.Gate.Check(ctx, EffectDevnet, k.Name()); err != nil {
		return err
	}
	if content.Provider != k.Name() || content.CID == "" {
		return fmt.Errorf("invalid Kubo content reference")
	}
	repo, err := k.contentRepo(content.RepoID)
	if err != nil {
		return err
	}
	_, err = k.run(ctx, repo, "pin", "add", "--recursive", content.CID)
	return err
}

func (k KuboAdapter) Teardown(ctx context.Context, content StoredContent) error {
	if err := k.Gate.Check(ctx, EffectDevnet, k.Name()); err != nil {
		return err
	}
	if content.Provider != k.Name() || content.CID == "" {
		return fmt.Errorf("invalid Kubo content reference")
	}
	repo, err := k.contentRepo(content.RepoID)
	if err != nil {
		return err
	}
	if _, err := k.run(ctx, repo, "pin", "rm", "--recursive", content.CID); err != nil {
		return err
	}
	if _, err := k.run(ctx, repo, "repo", "gc"); err != nil {
		return fmt.Errorf("collect unpinned local blocks: %w", err)
	}
	if err := removeOwnedRepo(repo); err != nil {
		return err
	}
	if repo != k.Repo {
		return removeEmptyRepoRoot(k.Repo)
	}
	return nil
}

func (k KuboAdapter) Resolve(ctx context.Context, content StoredContent) ([]byte, error) {
	if err := k.Gate.Check(ctx, EffectDevnet, k.Name()); err != nil {
		return nil, err
	}
	if content.Provider != k.Name() || content.CID == "" {
		return nil, fmt.Errorf("invalid Kubo content reference")
	}
	path := content.CID
	if content.EntryPath != "" {
		path += "/" + filepath.ToSlash(content.EntryPath)
	}
	repo, err := k.contentRepo(content.RepoID)
	if err != nil {
		return nil, err
	}
	data, err := k.run(ctx, repo, "cat", path)
	if err != nil {
		return nil, fmt.Errorf("retrieve by CID: %w", err)
	}
	return data, nil
}
