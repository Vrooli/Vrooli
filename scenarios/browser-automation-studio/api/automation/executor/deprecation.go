package executor

import (
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// DeprecationTracker tracks usage of deprecated Type/Params fields.
// This helps identify code paths that need migration to the typed Action field.
//
// Enable deprecation warnings by setting BAS_DEPRECATION_WARNINGS=1
var DeprecationTracker = &deprecationTracker{
	enabled: os.Getenv("BAS_DEPRECATION_WARNINGS") == "1" ||
		os.Getenv("BAS_DEPRECATION_WARNINGS") == "true",
}

type deprecationTracker struct {
	mu       sync.RWMutex
	enabled  bool
	logger   *logrus.Logger
	warnings map[string]int64 // caller location -> count
	totalHit int64
}

// Enable enables deprecation tracking with the given logger.
// If logger is nil, a default logger is created.
func (t *deprecationTracker) Enable(logger *logrus.Logger) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = true
	if logger != nil {
		t.logger = logger
	} else {
		t.logger = logrus.New()
		t.logger.SetLevel(logrus.WarnLevel)
	}
	if t.warnings == nil {
		t.warnings = make(map[string]int64)
	}
}

// Disable disables deprecation tracking.
func (t *deprecationTracker) Disable() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = false
}

// IsEnabled returns true if deprecation tracking is enabled.
func (t *deprecationTracker) IsEnabled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.enabled
}

// RecordTypeUsage records usage of the deprecated Type field.
// This should be called when InstructionStepType or PlanStepType
// falls back to using the legacy Type field.
func (t *deprecationTracker) RecordTypeUsage(context string) {
	t.recordUsage("Type", context)
}

// RecordParamsUsage records usage of the deprecated Params field.
// This should be called when InstructionParams or PlanStepParams
// falls back to using the legacy Params field.
func (t *deprecationTracker) RecordParamsUsage(context string) {
	t.recordUsage("Params", context)
}

func (t *deprecationTracker) recordUsage(field, context string) {
	if !t.IsEnabled() {
		return
	}

	atomic.AddInt64(&t.totalHit, 1)

	// Get caller location for deduplication
	_, file, line, ok := runtime.Caller(2)
	caller := "unknown"
	if ok {
		// Get just the filename, not full path
		for i := len(file) - 1; i >= 0; i-- {
			if file[i] == '/' {
				file = file[i+1:]
				break
			}
		}
		caller = file + ":" + itoa(line)
	}

	key := field + ":" + caller

	t.mu.Lock()
	if t.warnings == nil {
		t.warnings = make(map[string]int64)
	}
	count := t.warnings[key]
	t.warnings[key] = count + 1
	shouldLog := count == 0 || count == 10 || count == 100 || count == 1000
	t.mu.Unlock()

	// Log only at certain intervals to avoid spam
	if shouldLog && t.logger != nil {
		t.logger.WithFields(logrus.Fields{
			"field":    field,
			"caller":   caller,
			"context":  context,
			"count":    count + 1,
			"deprecat": "Type/Params",
		}).Warn("deprecated field access - migrate to Action")
	}
}

// Stats returns a summary of deprecation warnings.
func (t *deprecationTracker) Stats() DeprecationStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := DeprecationStats{
		TotalHits:     atomic.LoadInt64(&t.totalHit),
		UniqueCallers: len(t.warnings),
		ByField:       make(map[string]int64),
		TopCallers:    make(map[string]int64),
	}

	for key, count := range t.warnings {
		// Extract field name
		for i := 0; i < len(key); i++ {
			if key[i] == ':' {
				field := key[:i]
				stats.ByField[field] += count
				break
			}
		}
		// Track top callers
		if count >= 10 {
			stats.TopCallers[key] = count
		}
	}

	return stats
}

// Reset clears all tracked deprecation warnings.
func (t *deprecationTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.warnings = make(map[string]int64)
	atomic.StoreInt64(&t.totalHit, 0)
}

// DeprecationStats holds summary statistics about deprecation usage.
type DeprecationStats struct {
	TotalHits     int64
	UniqueCallers int
	ByField       map[string]int64
	TopCallers    map[string]int64
}

// itoa converts int to string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	buf := make([]byte, 20)
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
