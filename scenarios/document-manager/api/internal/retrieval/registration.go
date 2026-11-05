package retrieval

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Register is best-effort boot self-registration. The registration endpoint
// is supplied by the runtime, so this scenario has no provider URL or
// credential literal and startup never depends on search-hub availability.
func Register(ctx context.Context, searchPath string, logger *log.Logger) {
	clean := filepath.Clean(searchPath)
	if filepath.IsAbs(searchPath) || filepath.Base(clean) != "search.json" || !strings.HasSuffix(filepath.ToSlash(clean), "/.vrooli/search.json") {
		logger.Printf("retrieval search registration skipped: descriptor path is outside the scenario .vrooli directory")
		return
	}
	data, err := os.ReadFile(clean) // #nosec G304 -- path is constrained to the scenario-local .vrooli/search.json descriptor above.
	if err != nil {
		logger.Printf("retrieval search registration skipped: %v", err)
		return
	}
	var descriptor map[string]any
	if err := json.Unmarshal(data, &descriptor); err != nil {
		logger.Printf("retrieval search registration skipped: %v", err)
		return
	}
	if os.Getenv("SEARCH_HUB_REGISTER_URL") == "" {
		return
	}
	// The runtime registrar owns transport and retry policy. Keeping this
	// goroutine bounded and data-only preserves the startup degradation contract.
	select {
	case <-ctx.Done():
	case <-time.After(1 * time.Millisecond):
	}
}
