package projectstate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/vrooli/vrooli/internal/config"
)

const (
	stateProjectsDir = "state/projects"
	setupDirName     = "setup"
	projectKeyHash   = 12
)

type Locator struct {
	home string
	root string
	key  string
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
	return Locator{
		home: filepath.Clean(resolvedHome),
		root: resolvedRoot,
		key:  projectKey(resolvedRoot),
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
	return filepath.Join(config.VrooliDir(l.home), filepath.FromSlash(stateProjectsDir), l.key, setupDirName)
}

func (l Locator) SetupCompletePath() string {
	return filepath.Join(l.SetupStateDir(), ".setup-complete")
}

func (l Locator) ResourcesPopulatedPath() string {
	return filepath.Join(l.SetupStateDir(), ".resources-populated")
}

func (l Locator) ResourcePopulatedPath(resource string) string {
	return filepath.Join(l.SetupStateDir(), "."+safeMarkerName(resource)+"-populated")
}

func (l Locator) HasSetupComplete() bool {
	return fileExists(l.SetupCompletePath())
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
