package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	coreRetention "github.com/vrooli/api-core/retention"
	"storage-manager/internal/cleanup"
)

// RootSpec is the declarative contract for a filesystem root. The contract is
// intentionally data-shaped: adding a cache must not require a new provider
// implementation or a new recovery branch.
type RootSpec struct {
	ID                    string             `json:"id"`
	Root                  string             `json:"root"`
	Class                 string             `json:"class"`
	Tier                  cleanup.SafetyTier `json:"tier"`
	MaxAge                string             `json:"max_age"`
	MaxBytes              string             `json:"max_bytes"`
	Protect               []string           `json:"protect"`
	LeaseCheck            string             `json:"lease_check"`
	Platforms             []string           `json:"platforms"`
	ToolPruneCommand      []string           `json:"tool_prune_command"`
	Proof                 RootSpecProof      `json:"proof"`
	HotWriterBytesPerHour int64              `json:"hot_writer_bytes_per_hour"`
	Rationale             string             `json:"rationale"`
}

type RootSpecProof struct {
	Derived       bool `json:"derived"`
	ToolRecreates bool `json:"tool_recreates"`
	ExactRoot     bool `json:"exact_root"`
	NoLease       bool `json:"no_lease"`
}

// ValidateRootSpec enforces the safety properties that cannot be expressed by
// the repository JSON schema alone.
func ValidateRootSpec(spec RootSpec) error {
	if strings.TrimSpace(spec.ID) == "" || strings.TrimSpace(spec.Root) == "" {
		return fmt.Errorf("root spec %q: id and root are required", spec.ID)
	}
	switch spec.Tier {
	case cleanup.SafetyTierSafe, cleanup.SafetyTierRegenerable, cleanup.SafetyTierSafeWithOwner, cleanup.SafetyTierConditional:
	default:
		return fmt.Errorf("root spec %q: unsupported tier %q", spec.ID, spec.Tier)
	}
	if len(spec.Platforms) == 0 {
		return fmt.Errorf("root spec %q: platforms are required", spec.ID)
	}
	if strings.TrimSpace(spec.Rationale) == "" {
		return fmt.Errorf("root spec %q: rationale is required", spec.ID)
	}
	if spec.Tier == cleanup.SafetyTierRegenerable && !(spec.Proof.Derived && spec.Proof.ToolRecreates && spec.Proof.ExactRoot && spec.Proof.NoLease) {
		return fmt.Errorf("root spec %q: regenerable roots require complete proof", spec.ID)
	}
	return nil
}

// Applicable reports whether a root is supported by the current operating
// system. Unknown platform names are deliberately not treated as applicable.
func (s RootSpec) Applicable() bool {
	want := runtime.GOOS
	if want == "darwin" {
		want = "macos"
	}
	for _, platform := range s.Platforms {
		if platform == want {
			return true
		}
	}
	return false
}

// ResolveRoot expands only explicitly supported path variables. It does not
// expand arbitrary shell syntax and never executes ToolPruneCommand.
func ResolveRoot(raw, home string) string {
	value := strings.TrimSpace(raw)
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	vrooliHome := filepath.Join(home, ".vrooli")
	value = strings.ReplaceAll(value, "$USER_HOME", home)
	value = strings.ReplaceAll(value, "$HOME", home)
	value = strings.ReplaceAll(value, "${HOME}", home)
	value = strings.ReplaceAll(value, "$VROOLI_HOME", vrooliHome)
	value = strings.ReplaceAll(value, "${VROOLI_HOME}", vrooliHome)
	if cache, err := os.UserCacheDir(); err == nil {
		value = strings.ReplaceAll(value, "$XDG_CACHE_HOME", cache)
	}
	if tmp := os.TempDir(); tmp != "" {
		value = strings.ReplaceAll(value, "$TMPDIR", tmp)
	}
	if cache := strings.TrimSpace(os.Getenv("GOCACHE")); cache != "" && !strings.EqualFold(cache, "off") {
		value = strings.ReplaceAll(value, "$GOCACHE", cache)
	}
	if mod := strings.TrimSpace(os.Getenv("GOMODCACHE")); mod != "" {
		value = strings.ReplaceAll(value, "$GOMODCACHE", mod)
	}
	if strings.HasPrefix(value, "~/") {
		value = filepath.Join(home, strings.TrimPrefix(value, "~/"))
	}
	return filepath.Clean(value)
}

// NewSpecProvider adapts a validated declarative root to the existing bounded
// file provider. Keeping this adapter small makes the root contract the only
// place that names a physical cache while preserving the provider's re-stat,
// containment, ownership and batch semantics.
func NewSpecProvider(files cleanup.FileSystem, clock cleanup.Clock, spec RootSpec, cfg FileProviderConfig) (*FileProvider, error) {
	if err := ValidateRootSpec(spec); err != nil {
		return nil, err
	}
	if !spec.Applicable() {
		cfg.Roots = nil
	} else {
		cfg.Roots = resolveSpecRoots(ResolveRoot(spec.Root, ""))
	}
	cfg.ID = spec.ID
	cfg.Name = "Governed " + spec.ID
	cfg.Tier = spec.Tier
	cfg.RetentionMaxAge = parseSpecAge(spec.MaxAge)
	cfg.RetentionMaxBytes = parseSpecBytes(spec.MaxBytes)
	cfg.ProtectedGlobs = append([]string(nil), spec.Protect...)
	cfg.LeaseCheck = strings.TrimSpace(spec.LeaseCheck)
	return NewCacheProvider(files, clock, cfg), nil
}

func resolveSpecRoots(root string) []string {
	if !strings.ContainsAny(filepath.Base(root), "*?[") {
		return []string{root}
	}
	matches, err := filepath.Glob(root)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if filepath.Base(root) == "go-build*" {
			suffix := strings.TrimPrefix(filepath.Base(match), "go-build")
			if suffix == "" || strings.Trim(suffix, "0123456789") != "" {
				continue
			}
		}
		if info, statErr := os.Stat(match); statErr == nil && info.IsDir() {
			out = append(out, filepath.Clean(match))
		}
	}
	return out
}

func parseSpecAge(raw string) time.Duration {
	if strings.TrimSpace(raw) == "" {
		return 0
	}
	d, err := coreRetention.ParseAge(raw)
	if err != nil {
		return 0
	}
	return d
}

func parseSpecBytes(raw string) int64 {
	value, err := coreRetention.ParseBytes(raw)
	if err != nil {
		return 0
	}
	return value
}
