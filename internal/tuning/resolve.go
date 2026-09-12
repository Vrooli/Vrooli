package tuning

import (
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/vrooli/vrooli/internal/logx"
)

const environmentPrefix = "VROOLI_TUNING_"

type cachedDuration struct {
	once     sync.Once
	override *time.Duration
}

var durationCache sync.Map

// Duration resolves one process-wide timing lever. An absent or malformed
// override preserves the compiled fallback; malformed values warn once.
func Duration(name string, fallback time.Duration) time.Duration {
	entry, _ := durationCache.LoadOrStore(name, &cachedDuration{})
	cached := entry.(*cachedDuration)
	cached.once.Do(func() {
		key := EnvironmentVariable(name)
		raw, ok := os.LookupEnv(key)
		if !ok || strings.TrimSpace(raw) == "" {
			return
		}
		parsed, err := time.ParseDuration(strings.TrimSpace(raw))
		if err != nil {
			logx.WithSubsystem(slog.Default(), "tuning").Warn(
				"Ignoring malformed timing override",
				logx.AttrEnvVar, key,
				logx.AttrValue, raw,
				logx.ErrorAttr(err),
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

func durationHasOverride(name string) bool {
	entry, ok := durationCache.Load(name)
	if !ok {
		return false
	}
	return entry.(*cachedDuration).override != nil
}

func upperSnake(name string) string {
	runes := []rune(name)
	var result strings.Builder
	for index, current := range runes {
		if current == '-' || current == ' ' {
			if result.Len() > 0 && !strings.HasSuffix(result.String(), "_") {
				result.WriteByte('_')
			}
			continue
		}
		if unicode.IsUpper(current) && index > 0 &&
			(unicode.IsLower(runes[index-1]) || unicode.IsDigit(runes[index-1]) ||
				(index+1 < len(runes) && unicode.IsUpper(runes[index-1]) && unicode.IsLower(runes[index+1]))) {
			result.WriteByte('_')
		}
		result.WriteRune(unicode.ToUpper(current))
	}
	return result.String()
}

func resetDurationCacheForTest() { durationCache = sync.Map{} }
