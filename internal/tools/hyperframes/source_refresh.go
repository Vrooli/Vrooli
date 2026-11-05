package hyperframes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/vrooli/vrooli/internal/artifactlock"
)

var dependencyNames = []string{"package-lock.json", "package.json"}

type refreshOperations struct {
	acquire func(context.Context, string) (func(), error)
	rename  func(string, string) error
}

// RefreshSource is called only by SDA after approval. It refreshes dependency
// inputs, never installed runtime bytes. SDA owns the only resolver invocation.
func RefreshSource(ctx context.Context, source, version string, resolve func(context.Context, string) (string, error)) (string, error) {
	return refreshSource(ctx, source, version, resolve, refreshOperations{artifactlock.AcquireContext, os.Rename})
}

func readDependencyPair(ctx context.Context, dir string) (map[string][]byte, error) {
	out := map[string][]byte{}
	for _, name := range dependencyNames {
		limit := int64(8 << 20)
		if name == "package.json" {
			limit = 1 << 20
		}
		data, err := readFileContext(ctx, filepath.Join(dir, name), limit)
		if err != nil {
			return nil, err
		}
		out[name] = data
	}
	return out, nil
}

func refreshSource(ctx context.Context, source, version string, resolve func(context.Context, string) (string, error), ops refreshOperations) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := ValidateSourceContext(ctx, source); err != nil {
		return "", fmt.Errorf("source closure is not the committed pair; recover or review both dependency inputs and owner pin before retrying: %w", err)
	}
	absolute, err := filepath.Abs(source)
	if err != nil {
		return "", err
	}
	release, err := ops.acquire(ctx, "hyperframes-source-"+hashBytes([]byte(absolute)))
	if err != nil {
		return "", err
	}
	defer release()
	before, err := readDependencyPair(ctx, source)
	if err != nil {
		return "", err
	}
	if err := ValidateSourceContext(ctx, source); err != nil {
		return "", err
	}
	candidate, err := os.MkdirTemp("", "hyperframes-source-refresh-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(candidate)
	for _, name := range dependencyNames {
		if err := os.WriteFile(filepath.Join(candidate, name), before[name], 0o600); err != nil {
			return "", err
		}
	}
	output, err := resolve(ctx, candidate)
	if err != nil {
		return output, err
	}
	after, err := readDependencyPair(ctx, candidate)
	if err != nil {
		return output, err
	}
	if err := validateRefreshedInputs(before["package.json"], after, version); err != nil {
		return output, err
	}
	entries, err := os.ReadDir(candidate)
	if err != nil {
		return output, err
	}
	if len(entries) != 2 {
		return output, errors.New("lock-only resolver produced unexpected files or node_modules")
	}
	if err := publishDependencyPair(ctx, source, before, after, ops.rename); err != nil {
		return output, err
	}
	return output + "\nSource dependency pair updated; review owner version metadata before frozen provisioning. Previous immutable runtimes were not modified.", nil
}

func validateRefreshedInputs(original []byte, after map[string][]byte, version string) error {
	var expected, actual map[string]any
	if json.Unmarshal(original, &expected) != nil || json.Unmarshal(after["package.json"], &actual) != nil {
		return errors.New("invalid source package manifest")
	}
	deps, ok := expected["dependencies"].(map[string]any)
	if !ok || len(deps) != 1 {
		return errors.New("host-tool manifest must declare only hyperframes")
	}
	deps["hyperframes"] = version
	if !reflect.DeepEqual(expected, actual) {
		return errors.New("lock refresh changed manifest fields beyond the exact HyperFrames dependency pin")
	}
	var lock struct {
		Name, Version   string
		LockfileVersion int
		Packages        map[string]struct {
			Name, Version, Resolved, Integrity    string
			Link                                  bool
			Dependencies                          map[string]string
			DevDependencies, OptionalDependencies map[string]string
		}
	}
	if err := json.Unmarshal(after["package-lock.json"], &lock); err != nil {
		return err
	}
	root, exists := lock.Packages[""]
	if lock.LockfileVersion != 3 || !exists || lock.Name != actual["name"] || lock.Version != actual["version"] || root.Name != lock.Name || root.Version != lock.Version || len(root.DevDependencies) != 0 || len(root.OptionalDependencies) != 0 || len(root.Dependencies) != 1 || root.Dependencies["hyperframes"] != version || len(lock.Packages) > 10000 {
		return errors.New("refreshed lock root must match the exact source manifest")
	}
	if pkg, ok := lock.Packages["node_modules/hyperframes"]; !ok || pkg.Version != version {
		return errors.New("refreshed lock does not pin the requested HyperFrames version")
	}
	for path, pkg := range lock.Packages {
		if path == "" {
			continue
		}
		u, err := url.Parse(pkg.Resolved)
		integrity, decodeErr := base64.StdEncoding.DecodeString(strings.TrimPrefix(pkg.Integrity, "sha512-"))
		if !safeRelative(path, false) || !strings.HasPrefix(path, "node_modules/") || pkg.Link || pkg.Version == "" || err != nil || u.Scheme != "https" || u.Host != "registry.npmjs.org" || u.User != nil || !strings.HasPrefix(pkg.Integrity, "sha512-") || decodeErr != nil || len(integrity) != 64 {
			return fmt.Errorf("refreshed lock entry is not an integrity-pinned public npm package: %s", path)
		}
	}
	return nil
}

// publishDependencyPair uses a narrow two-file rollback. Each rename is atomic;
// the pair is not crash-atomic. Exact-closure validation must reject a partial
// pair. Refuse to overwrite independently edited source during rollback.
func publishDependencyPair(ctx context.Context, source string, before, after map[string][]byte, rename func(string, string) error) error {
	expected := map[string][]byte{}
	for name, data := range before {
		expected[name] = data
	}
	var published []string
	rollback := func(cause error) error {
		for i := len(published) - 1; i >= 0; i-- {
			name := published[i]
			current, err := readFileContext(context.Background(), filepath.Join(source, name), 8<<20)
			if err != nil || hashBytes(current) != hashBytes(after[name]) {
				return errors.Join(cause, errors.New("source changed during rollback; preserve evidence and recover the dependency pair explicitly"))
			}
			if err := replaceDependencyFile(source, name, before[name], os.Rename); err != nil {
				return errors.Join(cause, fmt.Errorf("source rollback failed; recover the dependency pair explicitly: %w", err))
			}
		}
		return cause
	}
	for _, name := range dependencyNames {
		current, err := readDependencyPair(ctx, source)
		if err != nil {
			return rollback(err)
		}
		for _, check := range dependencyNames {
			if hashBytes(current[check]) != hashBytes(expected[check]) {
				return rollback(errors.New("source dependency inputs changed; refresh publication refused"))
			}
		}
		if err := ctx.Err(); err != nil {
			return rollback(err)
		}
		if err := replaceDependencyFile(source, name, after[name], rename); err != nil {
			return rollback(err)
		}
		published = append(published, name)
		expected[name] = after[name]
	}
	return nil
}

func replaceDependencyFile(source, name string, data []byte, rename func(string, string) error) error {
	info, err := os.Lstat(filepath.Join(source, name))
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("dependency destination must remain a regular file: %s", name)
	}
	file, err := os.CreateTemp(source, ".dependency-input-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err = file.Chmod(info.Mode().Perm()); err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	err = errors.Join(err, file.Close())
	if err != nil {
		return err
	}
	return rename(file.Name(), filepath.Join(source, name))
}
