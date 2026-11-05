package derivation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"

	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

type VersionStore interface {
	Append(context.Context, Result) error
	NextVersion(context.Context, string) (int, error)
}

type Service struct {
	Registry Registry
	Runner   Runner
	Store    VersionStore
	mu       sync.Mutex
}

func NewService(registry Registry, runner Runner, store VersionStore) *Service {
	return &Service{Registry: registry, Runner: runner, Store: store}
}

func (s *Service) Derive(ctx context.Context, input Input) (Result, error) {
	if input.TierCeiling == documentpb.Tier_TIER_UNSPECIFIED {
		input.TierCeiling = documentpb.Tier_TIER_TWO
	}
	version, err := s.Store.NextVersion(ctx, input.DocumentHash)
	if err != nil {
		return Result{}, err
	}
	result := Result{DocumentHash: input.DocumentHash, Version: version}
	handler, matchErr := s.Registry.Match(input.Mime, input.TierCeiling)
	if matchErr != nil {
		if errors.Is(matchErr, ErrBlockedByPolicy) {
			result.State = documentpb.TerminalState_TERMINAL_STATE_BLOCKED_BY_POLICY
			result.Remedy = "reclassify the document or install a permitted local handler"
		} else {
			result.State = documentpb.TerminalState_TERMINAL_STATE_NO_HANDLER_FOR_FORMAT
			result.Remedy = "install a handler that declares this MIME type"
		}
		result.Reason = matchErr.Error()
		return result, s.Store.Append(ctx, result)
	}
	result.Chain = []string{handler.ID}
	result.Handlers = []string{handler.ID + "@" + handler.Version}
	if handler.Tier < 0 || handler.Tier > int(documentpb.Tier_TIER_THREE) {
		return Result{}, fmt.Errorf("handler %s declares invalid tier %d", handler.ID, handler.Tier)
	}
	result.Tier = documentpb.Tier(handler.Tier) // #nosec G115 -- tier is range-checked against the proto enum above.
	model, runErr := s.Runner.Run(ctx, handler, input)
	if runErr != nil {
		var handlerErr *HandlerError
		if errors.As(runErr, &handlerErr) && handlerErr.Unavailable {
			result.State = documentpb.TerminalState_TERMINAL_STATE_HANDLER_UNAVAILABLE
			result.Remedy = "start or install the declared handler resource"
		} else if errors.As(runErr, &handlerErr) && handlerErr.Variant {
			result.State = documentpb.TerminalState_TERMINAL_STATE_UNSUPPORTED_VARIANT
			result.Remedy = "remove the protection or provide a supported document variant"
		} else {
			result.State = documentpb.TerminalState_TERMINAL_STATE_HANDLER_FAILED
			result.Remedy = "inspect the handler error and retry with another chain"
		}
		result.Reason = runErr.Error()
		return result, s.Store.Append(ctx, result)
	}
	result.Model = model
	result.State = documentpb.TerminalState_TERMINAL_STATE_PARSED
	if err := s.Store.Append(ctx, result); err != nil {
		return Result{}, err
	}
	return result, nil
}

// NativeRunner keeps the tier-1 markup path in-process. Resource handlers are
// invoked through their public CLI, so the scenario never imports a parser.
type NativeRunner struct{}

func (NativeRunner) Run(ctx context.Context, handler Handler, input Input) (Model, error) {
	if handler.Runtime == "in-process" {
		return nativeModel(input.Bytes, handler)
	}
	allowed := map[string]bool{"doc-parse": true, "doc-ocr": true, "unstructured-io": true}
	if !allowed[handler.ID] {
		return Model{}, &HandlerError{Err: fmt.Errorf("handler %q is not an allowlisted resource", handler.ID)}
	}
	cmd := exec.CommandContext(ctx, "resource-"+handler.ID, "parse", "-", "--capabilities", strings.Join(handler.Capabilities, ",")) // #nosec G204 -- handler IDs are restricted to the resource allowlist above.
	cmd.Stdin = strings.NewReader(string(input.Bytes))
	output, err := cmd.Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return Model{}, &HandlerError{Unavailable: true, Err: ctx.Err()}
		}
		if exitErr, ok := err.(*exec.ExitError); ok && strings.Contains(string(exitErr.Stderr), "artifact not found") {
			return Model{}, &HandlerError{Unavailable: true, Err: err}
		}
		return Model{}, &HandlerError{Err: err}
	}
	var model Model
	if err := json.Unmarshal(output, &model); err != nil {
		return Model{}, &HandlerError{Err: fmt.Errorf("decode %s output: %w", handler.ID, err)}
	}
	return model, nil
}

func nativeModel(data []byte, handler Handler) (Model, error) {
	text := string(data)
	if strings.TrimSpace(text) == "" {
		return Model{Units: []Unit{{Index: 0, Kind: documentpb.AnchorKind_ANCHOR_KIND_LOGICAL, Confidence: 1}}}, nil
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	units := make([]Unit, 0, len(lines))
	for index, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		kind := documentpb.AnchorKind_ANCHOR_KIND_LOGICAL
		if contains(handler.Capabilities, "tables") && strings.Contains(line, ",") {
			kind = documentpb.AnchorKind_ANCHOR_KIND_TABULAR
		}
		units = append(units, Unit{Index: index, Text: line, Kind: kind, Confidence: 1})
	}
	return Model{Units: units}, nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

// HTTPStatusForState gives handlers a stable mapping without exposing storage
// errors or collapsing the terminal states into one response.
func HTTPStatusForState(state documentpb.TerminalState) int {
	if state == documentpb.TerminalState_TERMINAL_STATE_PARSED {
		return http.StatusOK
	}
	return http.StatusAccepted
}
