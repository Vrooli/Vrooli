package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// GovernedRootInputs are the host facts a governed storage root may name.
//
// A governed root (`storage.roots[].root` in the repo contract) is written in a
// portable form such as `$USER_HOME/.cache/go-build` or
// `$VROOLI_HOME/tmp/go-work`. Every consumer — storage-manager's cleanup
// providers, its recovery planner, and the agent-policy disposition snapshot —
// must turn that text into the same path, so the expansion lives here once.
type GovernedRootInputs struct {
	// Home is the operator's home directory.
	Home string
	// RuntimeHome is the Vrooli runtime home ($VROOLI_HOME).
	RuntimeHome string
	// RepoRoot is the source checkout ($REPO_ROOT); empty when unknown.
	RepoRoot string
	// TempDir is the platform temporary directory ($TMPDIR).
	TempDir string
	// CacheDir is the platform user cache directory ($XDG_CACHE_HOME).
	CacheDir string
	// GoCache and GoModCache are the Go toolchain caches, when configured.
	GoCache    string
	GoModCache string
}

// HostGovernedRootInputs resolves the inputs for this host.
//
// The runtime home honours an explicit VROOLI_HOME and otherwise comes from the
// repo contract's runtime_home authority, so the directory name is never
// restated here. repoRoot may be empty; the contract is then located from the
// environment or the working directory.
func HostGovernedRootInputs(repoRoot string) (GovernedRootInputs, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return GovernedRootInputs{}, err
	}
	in := GovernedRootInputs{Home: home, RepoRoot: strings.TrimSpace(repoRoot), TempDir: os.TempDir()}
	if cache, cacheErr := os.UserCacheDir(); cacheErr == nil {
		in.CacheDir = cache
	}
	if cache := strings.TrimSpace(os.Getenv("GOCACHE")); cache != "" && !strings.EqualFold(cache, "off") {
		in.GoCache = cache
	}
	in.GoModCache = strings.TrimSpace(os.Getenv("GOMODCACHE"))
	if override := strings.TrimSpace(os.Getenv("VROOLI_HOME")); override != "" {
		in.RuntimeHome = filepath.Clean(override)
		return in, nil
	}
	contract, root, err := loadGovernedContract(in.RepoRoot)
	if err != nil {
		return GovernedRootInputs{}, err
	}
	if in.RepoRoot == "" {
		in.RepoRoot = root
	}
	in.RuntimeHome, err = contract.RuntimeHome(home)
	if err != nil {
		return GovernedRootInputs{}, err
	}
	return in, nil
}

func loadGovernedContract(repoRoot string) (*repocontract.Contract, string, error) {
	if repoRoot != "" {
		contract, err := repocontract.LoadDefault(repoRoot)
		return contract, repoRoot, err
	}
	return repocontract.LoadDefaultFromEnvOrCWD()
}

// ResolveGovernedRoot expands the variables a governed root may use and cleans
// the result. It never evaluates shell syntax: an unknown variable stays
// literal, so a caller can detect it rather than receive a guessed path.
func ResolveGovernedRoot(raw string, in GovernedRootInputs) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "~/") || strings.HasPrefix(value, `~\`) {
		value = "$USER_HOME" + value[1:]
	}
	replacements := []struct{ token, with string }{
		{"${VROOLI_HOME}", in.RuntimeHome},
		{"$VROOLI_HOME", in.RuntimeHome},
		{"$USER_HOME", in.Home},
		{"${HOME}", in.Home},
		{"$HOME", in.Home},
		{"$REPO_ROOT", in.RepoRoot},
		{"$XDG_CACHE_HOME", in.CacheDir},
		{"$TMPDIR", in.TempDir},
		{"$GOMODCACHE", in.GoModCache},
		{"$GOCACHE", in.GoCache},
	}
	for _, replacement := range replacements {
		if replacement.with == "" {
			continue
		}
		value = strings.ReplaceAll(value, replacement.token, replacement.with)
	}
	return filepath.Clean(value)
}

// ErrGovernedRootUnresolved reports a governed root that still names a
// variable after expansion.
var ErrGovernedRootUnresolved = errors.New("governed root names a variable this host cannot resolve")

// ResolveGovernedRootStrict is ResolveGovernedRoot for callers that must act on
// the path: it refuses a result that is relative or still carries a variable.
func ResolveGovernedRootStrict(raw string, in GovernedRootInputs) (string, error) {
	value := ResolveGovernedRoot(raw, in)
	if value == "" || strings.Contains(value, "$") || !filepath.IsAbs(value) {
		return "", ErrGovernedRootUnresolved
	}
	return value, nil
}
