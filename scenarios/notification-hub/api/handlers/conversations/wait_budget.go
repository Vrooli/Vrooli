package conversations

import (
	"fmt"
	"net/http"
	"time"

	apiserver "github.com/vrooli/api-core/server"

	"notification-hub/internal/hub"
)

// waitBudget is the work budget of one Wait call: the longest wait the Hub lets
// a call last, plus a margin for the answer to be read and written.
const waitBudget = hub.MaxWaitSlice + 30*time.Second

// budgetWait gives Wait an operation budget. The server's ordinary 30 s write
// deadline would otherwise cut every wait that outlasts it, and the caller saw a
// bare EOF instead of an answer. The budget follows the request, so a caller
// that leaves still ends the wait.
func budgetWait(wait http.Handler) http.Handler {
	budgeted, err := apiserver.WithOperationBudget(wait, waitBudget)
	if err != nil {
		// Only a nil handler or a non-positive budget fails, and the budget is a
		// constant here, so this is a programming error.
		panic(fmt.Sprintf("conversations: mount wait budget: %v", err))
	}
	return budgeted
}
