package projectstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

const (
	setupDirName   = "setup"
	projectKeyHash = 12
)

type Locator struct {
	home          string
	root          string
	key           string
	setupStateDir string
}

func NewLocator(home, root string) (Locator, error) {
	resolvedHome := strings.TrimSpace(home)
	if resolvedHome == "" {
		var err error
		resolvedHome, err = config.HomeDir()
		if err != nil {
			return Locator{}, fmt.Errorf("resolve home: %w", err)
		}
	}
	resolvedRoot := strings.TrimSpace(root)
	if resolvedRoot == "" {
		return Locator{}, fmt.Errorf("project root is required")
	}
	absRoot, err := filepath.Abs(resolvedRoot)
	if err == nil {
		resolvedRoot = absRoot
	}
	resolvedRoot = filepath.Clean(resolvedRoot)
	cleanHome := filepath.Clean(resolvedHome)
	key := projectKey(resolvedRoot)
	// Resolve the project-state directory from the runtime_home authority. The
	// contract is required; a load failure surfaces here (no fallback).
	projectStateDir, err := repocontract.RuntimeHomeScopedPath(cleanHome, repocontract.ScopedProjectState, map[string]string{"project_key": key})
	if err != nil {
		return Locator{}, fmt.Errorf("resolve project state dir: %w", err)
	}
	return Locator{
		home:          cleanHome,
		root:          resolvedRoot,
		key:           key,
		setupStateDir: filepath.Join(projectStateDir, setupDirName),
	}, nil
}

func MustLocator(home, root string) Locator {
	locator, err := NewLocator(home, root)
	if err != nil {
		panic(err)
	}
	return locator
}

func (l Locator) Home() string {
	return l.home
}

func (l Locator) Root() string {
	return l.root
}

func (l Locator) ProjectKey() string {
	return l.key
}

func (l Locator) SetupStateDir() string {
	return l.setupStateDir
}

// BootstrapCompletePath is written by the elevated, non-interactive setup
// phase. It is intentionally distinct from configuration completion, which is
// owned by onboarding.
func (l Locator) BootstrapCompletePath() string {
	return filepath.Join(l.SetupStateDir(), ".bootstrap-complete")
}

func (l Locator) ConfigurationCompletePath() string {
	return filepath.Join(l.SetupStateDir(), ".configuration-complete")
}

// SetupCompletePath remains as a source-compatible alias for callers that
// only need to know whether bootstrap ran. New code should name the phase.
func (l Locator) SetupCompletePath() string { return l.BootstrapCompletePath() }

func (l Locator) ResourcesPopulatedPath() string {
	return filepath.Join(l.SetupStateDir(), ".resources-populated")
}

func (l Locator) ResourcePopulatedPath(resource string) string {
	return filepath.Join(l.SetupStateDir(), "."+safeMarkerName(resource)+"-populated")
}

func (l Locator) HasSetupComplete() bool {
	return l.HasBootstrapComplete()
}

func (l Locator) HasBootstrapComplete() bool {
	return fileExists(l.BootstrapCompletePath())
}

func (l Locator) HasConfigurationComplete() bool {
	return fileExists(l.ConfigurationCompletePath())
}

func (l Locator) HasResourcesPopulated() bool {
	return fileExists(l.ResourcesPopulatedPath())
}

func (l Locator) HasResourcePopulated(resource string) bool {
	return fileExists(l.ResourcePopulatedPath(resource))
}

func projectKey(root string) string {
	base := safeProjectName(filepath.Base(root))
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "project"
	}
	sum := sha256.Sum256([]byte(filepath.Clean(root)))
	return base + "-" + hex.EncodeToString(sum[:])[:projectKeyHash]
}

func safeProjectName(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		ok := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
		if !ok {
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
			continue
		}
		b.WriteRune(r)
		lastDash = r == '-'
	}
	return strings.Trim(b.String(), "-")
}

func safeMarkerName(value string) string {
	name := safeProjectName(value)
	if name == "" {
		sum := sha256.Sum256([]byte(value))
		return "resource-" + hex.EncodeToString(sum[:])[:projectKeyHash]
	}
	return name
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// MarkConfigurationComplete records the second half of first run after
// onboarding has applied and validated the operator's choices. It never
// overwrites the bootstrap marker.
func MarkConfigurationComplete(home, root, selectionDigest string) error {
	locator, err := NewLocator(home, root)
	if err != nil {
		return err
	}
	if !locator.HasBootstrapComplete() {
		return fmt.Errorf("cannot mark configuration complete before bootstrap completion")
	}
	if _, err := config.EnsureOwnedDir(locator.SetupStateDir()); err != nil {
		return err
	}
	payload := map[string]any{
		"completed_at":     time.Now().UTC().Format(time.RFC3339),
		"phase":            "configuration_complete",
		"project_key":      locator.ProjectKey(),
		"selection_digest": strings.TrimSpace(selectionDigest),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return config.WriteOwnedFile(locator.ConfigurationCompletePath(), append(data, '\n'), 0o644)
}
