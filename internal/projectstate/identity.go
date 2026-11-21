package projectstate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

// The project key and marker paths are recomputed by later readers. Their
// source is retained in the persistent-state compatibility contract.
const (
	setupDirName        = "setup"
	projectKeyHash      = 12
	migrationLedgerName = "migrations.json"
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

// MigrationLedgerPath stores named compatibility migrations beside the other
// project-scoped setup state. It is deliberately not part of bootstrap marker
// payloads, whose setup_version meaning is unrelated.
func (l Locator) MigrationLedgerPath() string {
	return filepath.Join(l.SetupStateDir(), migrationLedgerName)
}

// ActiveSetupPath is the best-effort, versioned in-progress setup record.
// It is separate from terminal result files so interrupted setup can be
// diagnosed without changing the existing result contract.
func (l Locator) ActiveSetupPath() string {
	return filepath.Join(l.SetupStateDir(), "active-setup.json")
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

func (l Locator) ResourcesPopulatedPath() string {
	return filepath.Join(l.SetupStateDir(), ".resources-populated")
}

func (l Locator) ResourcePopulatedPath(resource string) string {
	return filepath.Join(l.SetupStateDir(), "."+safeMarkerName(resource)+"-populated")
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
