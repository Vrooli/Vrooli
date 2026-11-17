package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"connectrpc.com/connect"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	cliv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1/cliv1connect"
)

type ollamaOptimizationServiceHandler struct {
	cliv1connect.UnimplementedOllamaOptimizationServiceHandler
	runResourceCLI func(string, []string, io.Writer, io.Writer) error
}

func (h *ollamaOptimizationServiceHandler) InspectRoleCandidates(_ context.Context, req *connect.Request[cliv1.InspectOllamaRoleCandidatesRequest]) (*connect.Response[cliv1.InspectOllamaRoleCandidatesResponse], error) {
	if h == nil || h.runResourceCLI == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("Ollama optimizer owner is unavailable"))
	}
	input := req.Msg
	request := struct {
		RequestID           string   `json:"request_id"`
		Target              string   `json:"target"`
		Role                string   `json:"role"`
		CandidateModels     []string `json:"candidate_models"`
		FixedSuiteID        string   `json:"fixed_suite_id"`
		FixedCaseIDs        []string `json:"fixed_case_ids"`
		EvaluationReceiptID string   `json:"evaluation_receipt_id,omitempty"`
		WinnerModel         string   `json:"winner_model,omitempty"`
	}{
		RequestID: input.GetRequestId(), Target: input.GetTarget(), Role: input.GetRole(),
		CandidateModels: input.GetCandidateModels(), FixedSuiteID: input.GetFixedSuiteId(),
		FixedCaseIDs: input.GetFixedCaseIds(), EvaluationReceiptID: input.GetEvaluationReceiptId(),
		WinnerModel: input.GetWinnerModel(),
	}
	encodedRequest, err := json.Marshal(request)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode Ollama role request: %w", err))
	}
	file, err := os.CreateTemp("", "vrooli-ollama-optimize-*.json")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create bounded Ollama request file: %w", err))
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("secure Ollama request file: %w", err))
	}
	if _, err := file.Write(encodedRequest); err != nil {
		_ = file.Close()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("write Ollama request file: %w", err))
	}
	if err := file.Close(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("close Ollama request file: %w", err))
	}
	var stdout, stderr bytes.Buffer
	err = h.runResourceCLI("ollama", []string{"optimize", "role-adapter", "--request", file.Name(), "--json"}, &stdout, &stderr)
	receipt := strings.TrimSpace(stdout.String())
	if !json.Valid([]byte(receipt)) {
		if err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("Ollama owner operation failed: %w: %s", err, strings.TrimSpace(stderr.String())))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("Ollama owner returned invalid receipt JSON"))
	}
	return connect.NewResponse(&cliv1.InspectOllamaRoleCandidatesResponse{ReceiptJson: receipt}), nil
}

var _ cliv1connect.OllamaOptimizationServiceHandler = (*ollamaOptimizationServiceHandler)(nil)
