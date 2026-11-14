package tuning

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/vrooli/envkit-go"
	"github.com/vrooli/vrooli/internal/logx"
)

// Count levers bound how many of something may run at once. They are
// resolved exactly like duration levers: once per process, with a malformed
// or non-positive override warning once and keeping the compiled fallback.

type countDefinition struct {
	Name            string
	CompiledDefault int
	Unit            string
}

var countDefinitions = []countDefinition{
	{Name: "BuildWidth", CompiledDefault: defaultBuildWidth(), Unit: "processes"},
}

type cachedCount struct {
	once     sync.Once
	override *int
}

var countCache sync.Map

// Count resolves one process-wide count lever. An absent, malformed or
// non-positive override preserves the compiled fallback; bad values warn once.
func Count(name string, fallback int) int {
	entry, _ := countCache.LoadOrStore(name, &cachedCount{})
	cached := entry.(*cachedCount)
	cached.once.Do(func() {
		key := EnvironmentVariable(name)
		raw, ok := os.LookupEnv(key)
		if !ok || strings.TrimSpace(raw) == "" {
			return
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || parsed <= 0 {
			logx.WithSubsystem(slog.Default(), "tuning").Warn(
				"Ignoring malformed count override",
				logx.AttrEnvVar, key,
				logx.AttrValue, raw,
			)
			return
		}
		cached.override = &parsed
	})
	if cached.override != nil {
		return *cached.override
	}
	return fallback
}

func countHasOverride(name string) bool {
	entry, ok := countCache.Load(name)
	if !ok {
		return false
	}
	return entry.(*cachedCount).override != nil
}

// BuildWidth bounds how many compile or link processes one build may run at
// once. Admission (how many builds start) stays with the lifecycle memory
// budget; this lever governs width only.
func BuildWidth() int {
	return Count("BuildWidth", defaultBuildWidth())
}

// defaultBuildWidth is envkit's derivation so the control plane and the
// shared packages that cannot import this lever agree on one number.
func defaultBuildWidth() int { return envkit.DefaultBuildWidth() }

func resetCountCacheForTest() { countCache = sync.Map{} }
