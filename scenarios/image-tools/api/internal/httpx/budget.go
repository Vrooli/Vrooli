package httpx

import (
	"fmt"
	"net/http"
	"time"

	apiserver "github.com/vrooli/api-core/server"
)

// SyncOperationBudget bounds a REST edge whose response waits for real work: a
// durable job behind capacity admission, a model load, an external tool. api-core
// gives every request a 30 s response write deadline, which cuts such a connection
// with a bare EOF at the client although the work finishes and the server logs a
// 200 (PaddleOCR on a busy CPU takes 15-55 s). The budget replaces that deadline
// for the edge and follows the request, so a caller that leaves still cancels the
// work. An edge that can run longer than 30 s is mounted through SyncBudget; one
// that only submits and returns 202 needs nothing.
const SyncOperationBudget = 10 * time.Minute

// SyncBudget gives h the SyncOperationBudget.
func SyncBudget(h http.Handler) http.Handler {
	bounded, err := apiserver.WithOperationBudget(h, SyncOperationBudget)
	if err != nil {
		// Only a nil handler or a non-positive budget can fail; both are fixed at
		// compile time here, so this is a programming error.
		panic(fmt.Sprintf("httpx: mount sync budget: %v", err))
	}
	return bounded
}

// SyncBudgetFunc is SyncBudget for a handler function.
func SyncBudgetFunc(h http.HandlerFunc) http.Handler { return SyncBudget(h) }
