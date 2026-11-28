package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"sync"

	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

// Batch bounds. A batch carries one question set and many states, so its body
// may exceed one decision's 8 MiB ceiling; each state still reaches the daemon
// as its own request under that ceiling.
const (
	maxBatchStates       = 256
	maxBatchRequestBytes = 64 << 20
	defaultBatchWorkers  = 4
)

// BatchRequest is one question set asked about many states.
type BatchRequest struct {
	Questions json.RawMessage   `json:"questions"`
	States    []json.RawMessage `json:"states"`
}

// BatchItem is one state's outcome: a normalized result or a typed failure.
type BatchItem struct {
	Result *upstream.Result `json:"result,omitempty"`
	Error  *BatchError      `json:"error,omitempty"`
}

// BatchError is a per-state failure carried in the batch output instead of the
// stderr marker, so one state's refusal (for example a state that overflows the
// model's context) never fails the other states.
type BatchError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BatchResponse holds one item per state, in request order.
type BatchResponse struct {
	Results []BatchItem `json:"results"`
}

// DecideBatch answers one question set about many states on the role's
// upstream. Role resolution, rights and manifest proof, and placement
// admission run once for the batch, so a batch costs one process and one
// admission instead of one per state. The first state loads the model inside
// the load slot; the slot is released before the rest run concurrently, so a
// batch never holds other callers' loads behind its warm decisions.
//
// A failure that concerns the batch (an unknown role, refused rights, a model
// that cannot be placed) fails the command with the usual marker line. A
// failure that concerns one state is that state's item error.
func (h *Handlers) DecideBatch(args []string) error {
	flags, err := h.parseDecisionFlags("gateway decide-batch", args, func(fs *flag.FlagSet, out *decisionFlags) {
		fs.IntVar(&out.concurrency, "concurrency", defaultBatchWorkers, "Decisions in flight at once after the model is loaded")
	})
	if err != nil {
		return err
	}
	if flags.concurrency < 1 {
		return errors.New("gateway decide-batch --concurrency must be at least 1")
	}
	raw, err := readDecisionInput(h.Stdin, maxBatchRequestBytes)
	if err != nil {
		return err
	}
	var request BatchRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		return fmt.Errorf("decode decision batch: %w", err)
	}
	if len(bytes.TrimSpace(request.Questions)) == 0 || len(request.States) == 0 {
		return errors.New("decision batch requires questions and at least one state")
	}
	if len(request.States) > maxBatchStates {
		return fmt.Errorf("decision batch has %d states; the maximum is %d", len(request.States), maxBatchStates)
	}
	prepareCtx, cancelPrepare := context.WithTimeout(context.Background(), flags.timeout)
	resolution, release, err := h.prepareDecision(prepareCtx, flags)
	cancelPrepare()
	if err != nil {
		return h.fail(err)
	}
	client := h.client()
	results := make([]BatchItem, len(request.States))
	decide := func(index int) {
		ctx, cancel := context.WithTimeout(context.Background(), flags.timeout)
		defer cancel()
		result, err := client.Decide(ctx, resolution, upstream.Request{State: request.States[index], Questions: request.Questions})
		if err != nil {
			results[index] = BatchItem{Error: batchError(err)}
			return
		}
		results[index] = BatchItem{Result: &result}
	}
	decide(0)
	release()
	slots := make(chan struct{}, flags.concurrency)
	var wait sync.WaitGroup
	for index := 1; index < len(request.States); index++ {
		wait.Add(1)
		slots <- struct{}{}
		go func(index int) {
			defer wait.Done()
			defer func() { <-slots }()
			decide(index)
		}(index)
	}
	wait.Wait()
	response := BatchResponse{Results: results}
	if flags.json {
		return writeJSON(h.Stdout, response)
	}
	for index, item := range results {
		if item.Error != nil {
			fmt.Fprintf(h.Stdout, "[%d] error %s: %s\n", index, item.Error.Code, item.Error.Message)
			continue
		}
		fmt.Fprintf(h.Stdout, "[%d] upstream=%s model=%s latency_ms=%d %s\n", index, item.Result.Upstream, item.Result.Model, item.Result.LatencyMS, string(item.Result.Answers))
	}
	return nil
}

func batchError(err error) *BatchError {
	var providerErr *upstream.ProviderError
	if errors.As(err, &providerErr) {
		return &BatchError{Code: providerErr.Code, Message: providerErr.Error()}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &BatchError{Code: "timeout", Message: err.Error()}
	}
	return &BatchError{Code: upstream.CodeProviderFailed, Message: err.Error()}
}
