package resources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// ManagedUpdateRuntime freezes non-secret owner inputs before native dispatch.
// Credentials remain logical references in the contract, resolved in the role's
// private state directory. No caller environment is retained or inherited.
type ManagedUpdateRuntime struct {
	Root           string                 `json:"root"`
	Home           string                 `json:"home"`
	StateHome      string                 `json:"state_home,omitempty"`
	StateDir       string                 `json:"state_dir"`
	Launcher       string                 `json:"launcher"`
	LauncherSHA256 string                 `json:"launcher_sha256"`
	Port           int                    `json:"port"`
	Limits         platform.ServiceLimits `json:"limits"`
}

func (r ManagedUpdateRuntime) validate() error {
	for _, p := range []string{r.Root, r.Home, r.StateDir, r.Launcher} {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p || strings.ContainsRune(p, 0) {
			return fmt.Errorf("update runtime requires canonical absolute owner paths")
		}
	}
	if r.StateHome != "" && (!filepath.IsAbs(r.StateHome) || filepath.Clean(r.StateHome) != r.StateHome || strings.ContainsRune(r.StateHome, 0)) {
		return fmt.Errorf("invalid update state home")
	}
	if r.Port < 1 || r.Port > 65535 || !updateDigest(r.LauncherSHA256) || r.Limits.MemoryMaxBytes <= 0 || r.Limits.TasksMax <= 0 || r.Limits.CPUWeight < 1 || r.Limits.CPUWeight > 10000 {
		return fmt.Errorf("update runtime requires a port, launcher checksum and service ceilings")
	}
	return nil
}

func (r ManagedUpdateRuntime) roleRoot(operation, instance string) (string, error) {
	if err := r.validate(); err != nil {
		return "", err
	}
	if !managedUpdateKey.MatchString(operation) || !managedUpdateKey.MatchString(instance) {
		return "", fmt.Errorf("invalid update runtime namespace")
	}
	return filepath.Join(r.StateDir, "managed-update-exec", operation, instance, "runtime"), nil
}

type managedUpdateInvocation struct {
	Path, Dir                 string
	Args, Env                 []string
	ConfigPath, ConfigContent string
	StateRoot, RuntimeRoot    string
}

// renderManagedUpdateInvocation is pure. All mutable resource paths, including
// configuration and credentials, belong to this exact role instance.
func renderManagedUpdateInvocation(c ManagedUpdateContract, r ManagedUpdateRuntime, operation, instance, token string) (managedUpdateInvocation, error) {
	if err := c.validate(); err != nil {
		return managedUpdateInvocation{}, err
	}
	m := c.Manifest
	if m.ManagedService.UpdatePolicy == nil {
		return managedUpdateInvocation{}, fmt.Errorf("update policy is required")
	}
	if err := m.ManagedService.ValidateUpdatePolicy(); err != nil {
		return managedUpdateInvocation{}, err
	}
	if err := manifestpkg.ValidateManagedServiceUpdate(m); err != nil {
		return managedUpdateInvocation{}, err
	}
	root, err := r.roleRoot(operation, instance)
	if err != nil {
		return managedUpdateInvocation{}, err
	}
	if token == "" || strings.ContainsAny(token, "\x00\r\n") {
		return managedUpdateInvocation{}, fmt.Errorf("invalid ownership token")
	}
	artifactRoot := filepath.Dir(c.Artifact.Path)
	if m.ManagedService.Artifact.Layout == "dir" {
		artifactRoot = c.Artifact.Path
	}
	values := map[string]string{
		"HOME": filepath.Join(root, "home"), "ROOT": r.Root, "PATH": "/usr/local/bin:/usr/bin:/bin",
		"RESOURCE_ROOT":                   filepath.Join(r.Root, "resources", m.Name),
		"RESOURCE_CONFIG_DIR":             filepath.Join(root, "config"),
		"RESOURCE_CACHE_DIR":              filepath.Join(root, "cache"),
		"RESOURCE_LOGS_DIR":               filepath.Join(root, "logs"),
		"RESOURCE_STATE_DIR":              filepath.Join(root, "state"),
		"RESOURCE_ARTIFACT_DIR":           artifactRoot,
		"VROOLI_MANAGED_SERVICE_ARTIFACT": c.Artifact.Path,
		managedServiceOwnershipTokenEnv:   token,
		resourcedeployment.ServicePortEnvironmentName(m.ManagedService.UpdatePolicy.CandidatePort): strconv.Itoa(r.Port),
	}
	expand, err := managedUpdateTemplateResolver(values, m.ManagedService.Environment)
	if err != nil {
		return managedUpdateInvocation{}, err
	}
	args := append([]string(nil), m.ManagedService.ArgumentsFor(runtime.GOOS, runtime.GOARCH)...)
	for i, arg := range args {
		args[i], err = expand(arg)
		if err != nil {
			return managedUpdateInvocation{}, err
		}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	env := make([]string, 0, len(keys))
	for _, key := range keys {
		env = append(env, key+"="+values[key])
	}
	invocation := managedUpdateInvocation{Path: c.Artifact.Path, Dir: artifactRoot, Args: args, Env: env, StateRoot: r.StateDir, RuntimeRoot: root}
	if declaration := m.ManagedService.Config; declaration != nil {
		if err := declaration.Validate(); err != nil {
			return managedUpdateInvocation{}, err
		}
		invocation.ConfigPath = filepath.Join(values["RESOURCE_CONFIG_DIR"], filepath.FromSlash(declaration.Path))
		invocation.ConfigContent, err = expand(declaration.Content)
		if err != nil {
			return managedUpdateInvocation{}, err
		}
	}
	return invocation, nil
}

func managedUpdateImageDigest(ctx context.Context, file *os.File) (string, error) {
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return "", fmt.Errorf("update executable is not a nonempty regular file")
	}
	h := sha256.New()
	if _, err := io.Copy(h, &managedUpdateContextReader{ctx: ctx, reader: file}); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func materializeManagedUpdateInvocation(u ManagedServiceUpdate, c ManagedUpdateContract, invocation managedUpdateInvocation) error {
	values := managedServiceEnvValues(invocation.Env)
	if err := ensureManagedUpdatePrivateDirectory(invocation.StateRoot, invocation.RuntimeRoot); err != nil {
		return err
	}
	for _, key := range []string{"HOME", "RESOURCE_CONFIG_DIR", "RESOURCE_CACHE_DIR", "RESOURCE_LOGS_DIR", "RESOURCE_STATE_DIR"} {
		if err := ensureManagedUpdatePrivateDirectory(invocation.RuntimeRoot, values[key]); err != nil {
			return err
		}
	}
	if invocation.ConfigPath != "" {
		if err := ensureManagedUpdatePrivateDirectory(invocation.RuntimeRoot, filepath.Dir(invocation.ConfigPath)); err != nil {
			return err
		}
		if err := config.WriteOwnedFileDurable(invocation.ConfigPath, []byte(invocation.ConfigContent), 0o600); err != nil {
			return err
		}
	}
	for _, credential := range c.Manifest.ManagedService.CredentialFiles {
		if err := ensureManagedUpdatePrivateDirectory(invocation.RuntimeRoot, filepath.Dir(filepath.Join(values["RESOURCE_STATE_DIR"], filepath.FromSlash(credential.Path)))); err != nil {
			return err
		}
	}
	return materializeManagedUpdateCredentials(u, c, invocation)
}

// Only the established resource state root may have an owner-configured alias.
// Role descendants must be actual private directories, so a stale alias cannot
// redirect candidate materialization into another serving instance's paths.
func ensureManagedUpdatePrivateDirectory(base, destination string) error {
	if !filepath.IsAbs(base) || filepath.Clean(base) != base {
		return fmt.Errorf("invalid private runtime base")
	}
	rel, err := filepath.Rel(base, destination)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("private runtime path escapes owner base")
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return err
	}
	path := base
	if rel == "." {
		return nil
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			if err = os.Mkdir(path, 0o700); err != nil && !os.IsExist(err) {
				return err
			}
			info, err = os.Lstat(path)
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("private runtime path contains a directory alias or non-directory")
		}
	}
	return nil
}

// managedUpdateTemplateResolver expands only controlled and declared inputs.
func managedUpdateTemplateResolver(values, declared map[string]string) (func(string) (string, error), error) {
	if len(declared) > 256 {
		return nil, fmt.Errorf("update environment exceeds its declaration bound")
	}
	for key := range declared {
		if !validManagedServiceEnvironmentKey(key) {
			return nil, fmt.Errorf("invalid declared environment name")
		}
		if _, owner := values[key]; owner {
			return nil, fmt.Errorf("declared environment overrides owner input %s", key)
		}
	}
	visiting := map[string]bool{}
	var resolve func(string) (string, error)
	var expand func(string) (string, error)
	expand = func(s string) (string, error) {
		var failure error
		v := os.Expand(s, func(key string) string {
			if failure != nil {
				return ""
			}
			v, err := resolve(key)
			if err != nil {
				failure = err
			}
			return v
		})
		if failure != nil {
			return "", failure
		}
		if strings.ContainsRune(v, 0) {
			return "", fmt.Errorf("rendered update input contains NUL")
		}
		return v, nil
	}
	resolve = func(key string) (string, error) {
		if v, ok := values[key]; ok {
			return v, nil
		}
		source, ok := declared[key]
		if !ok {
			return "", fmt.Errorf("undeclared update environment reference %s", key)
		}
		if visiting[key] {
			return "", fmt.Errorf("cyclic update environment reference %s", key)
		}
		visiting[key] = true
		v, err := expand(source)
		delete(visiting, key)
		if err == nil {
			values[key] = v
		}
		return v, err
	}
	for key := range declared {
		if _, err := resolve(key); err != nil {
			return nil, err
		}
	}
	return expand, nil
}
