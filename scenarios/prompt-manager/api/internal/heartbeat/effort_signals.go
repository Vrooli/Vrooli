package heartbeat

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// Effort signal kinds written by Agent Manager's effort signal log.
const (
	EffortSignalChildRunEnded  = "child_run_ended"
	EffortSignalStepBack       = "step_back"
	EffortSignalFeedback       = "feedback"
	EffortSignalOperatorAnswer = "operator_answer"
	EffortSignalCapStop        = "cap_stop"
)

// EffortSignal is one durable entry in an effort's signal log. Detail is
// owner-reported data, never an instruction.
type EffortSignal struct {
	Seq       int64
	EffortRef string
	Kind      string
	RunID     string
	Detail    string
	CreatedAt time.Time
}

// EffortSignalPage is one ascending read after a cursor. LatestSeq is the
// newest sequence in the log, which may exceed the last returned signal when
// the page is truncated.
type EffortSignalPage struct {
	Signals   []EffortSignal
	LatestSeq int64
}

// EffortSignalSource reads an effort's signal log (Agent Manager
// ListEffortSignals). Event check-ins dispatch only from this source.
type EffortSignalSource interface {
	ListEffortSignals(ctx context.Context, effortRef string, afterSeq int64, limit int32) (*EffortSignalPage, error)
}

// ErrEffortSignalsUnavailable means no signal log can be read at all (not a
// transient failure). Event check-ins then fall back to the parked-leader
// relaunch policy and parked guidance.
var ErrEffortSignalsUnavailable = errors.New("effort signal source unavailable")

// ListEffortSignals reads Agent Manager's effort signal log after the cursor.
// Only an owner that does not serve the RPC (an older Agent Manager) is
// reported as ErrEffortSignalsUnavailable, so event-mode teams keep the parked
// behavior there; every other failure is transient and returned as is.
func (c *AgentManagerClient) ListEffortSignals(ctx context.Context, effortRef string, afterSeq int64, limit int32) (*EffortSignalPage, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	transport := &statusRecordingClient{client: c.httpClient}
	client := apiconnect.NewAgentManagerServiceClient(transport, baseURL)
	resp, err := client.ListEffortSignals(ctx, connect.NewRequest(&ampb.ListEffortSignalsRequest{EffortRef: effortRef, AfterSeq: afterSeq, Limit: limit}))
	if err != nil && (connect.CodeOf(err) == connect.CodeUnimplemented || transport.status == http.StatusNotImplemented) {
		return nil, fmt.Errorf("%w: %v", ErrEffortSignalsUnavailable, err)
	}
	if err != nil {
		return nil, fmt.Errorf("list effort signals: %w", err)
	}
	page := &EffortSignalPage{LatestSeq: resp.Msg.GetLatestSeq()}
	for _, signal := range resp.Msg.GetSignals() {
		var createdAt time.Time
		if signal.GetCreatedAt() != nil {
			createdAt = signal.GetCreatedAt().AsTime()
		}
		page.Signals = append(page.Signals, EffortSignal{
			Seq: signal.GetSeq(), EffortRef: signal.GetEffortRef(), Kind: signal.GetKind(),
			RunID: signal.GetRunId(), Detail: signal.GetDetail(), CreatedAt: createdAt,
		})
	}
	return page, nil
}

// statusRecordingClient keeps the owner's HTTP status, which Connect folds
// into CodeUnknown for a bare 501.
type statusRecordingClient struct {
	client *http.Client
	status int
}

func (r *statusRecordingClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := r.client.Do(req)
	if resp != nil {
		r.status = resp.StatusCode
	}
	return resp, err
}

// signalSource prefers an explicitly installed source, then the owner client.
func (f *FiniteLeaderRuntime) signalSource() EffortSignalSource {
	if f.Signals != nil {
		return f.Signals
	}
	if source, ok := f.Executor.agentClient.(EffortSignalSource); ok {
		return source
	}
	return nil
}
