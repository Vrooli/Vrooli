package runtimeapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/vrooli/vrooli/internal/maintenance"
)

// ExecutorScope is a local read; it does not open the registry, contact a
// scenario, exchange an owner credential, or mutate lifecycle/business state.
func (app *Service) ExecutorScope(ctx context.Context, in io.Reader, out io.Writer) error {
	ctx, cancel := context.WithTimeout(ctx, maintenance.ExecutorScopeTimeout)
	defer cancel()
	request, err := decodeExecutorScopeRequest(in)
	if err != nil {
		return err
	}
	report, err := (&maintenance.Controller{}).ExecutorScope(ctx, request)
	if err != nil {
		return err
	}
	return writeExecutorScopeReport(ctx, out, report)
}

func decodeExecutorScopeRequest(in io.Reader) (maintenance.ExecutorScopeRequest, error) {
	const maxInput = maintenance.ExecutorScopeMaxBytes
	var request maintenance.ExecutorScopeRequest
	body, err := io.ReadAll(io.LimitReader(in, maxInput+1))
	if err != nil {
		return request, err
	}
	if len(body) > maxInput {
		return request, fmt.Errorf("executor-scope input exceeds 8 MiB")
	}
	if err := json.Unmarshal(body, &request); err != nil {
		return request, fmt.Errorf("invalid executor-scope JSON input")
	}
	return request, nil
}

func writeExecutorScopeReport(ctx context.Context, out io.Writer, report maintenance.ExecutorScopeReport) error {
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	if len(data)+1 > maintenance.ExecutorScopeMaxBytes {
		return fmt.Errorf("executor-scope output exceeds 8 MiB")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err = out.Write(append(data, '\n'))
	return err
}
