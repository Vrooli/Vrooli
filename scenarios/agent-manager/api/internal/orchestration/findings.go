// Findings responsibility: expose persisted investigation recurrence rows to
// transport handlers without allowing those handlers database access.
package orchestration

import (
	"context"
	"fmt"

	"agent-manager/internal/findings"
)

func (o *Orchestrator) ListFindings(ctx context.Context, filter findings.Filter) ([]findings.Finding, error) {
	if o.findings == nil {
		return nil, fmt.Errorf("findings repository is not configured")
	}
	return o.findings.List(ctx, filter)
}
