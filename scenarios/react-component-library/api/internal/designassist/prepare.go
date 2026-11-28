package designassist

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"react-component-library/internal/designinference"
)

// maxPromptRunes bounds the person's request. The gateway's instruction budget
// is 8000 characters; the fixed framing is ~1500, so 2000 leaves room without
// letting a paste become the whole context.
const maxPromptRunes = 2000

// Prepare builds the model request and the ids a proposal may target.
//
// The gateway role is the existing `extract.structured`: it is the only role
// the durable inference contract accepts today, its 4096-token output fits a
// small patch comfortably, and it already carries the confidential privacy
// class required for a private design.
func Prepare(doc Document, scope Scope, prompt string) (designinference.Request, map[string]struct{}, error) {
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" || utf8.RuneCountInString(trimmed) > maxPromptRunes {
		return designinference.Request{}, nil, fmt.Errorf("%w: prompt must be 1 to %d characters", ErrInvalid, maxPromptRunes)
	}
	for _, r := range trimmed {
		if r == '\n' || r == '\t' {
			continue
		}
		if r < 0x20 || r == 0x7f {
			return designinference.Request{}, nil, fmt.Errorf("%w: prompt contains a control character", ErrInvalid)
		}
	}
	if err := validateScope(scope); err != nil {
		return designinference.Request{}, nil, err
	}
	contextJSON, allowed, err := BuildContext(doc, scope)
	if err != nil {
		return designinference.Request{}, nil, err
	}
	schema, err := Schema()
	if err != nil {
		return designinference.Request{}, nil, err
	}
	return designinference.Request{
		Role:            "extract.structured",
		MaxOutputTokens: 4096,
		Source:          string(contextJSON),
		Schema:          schema,
		Instruction:     Instruction(trimmed),
		Context:         contextJSON,
	}, allowed, nil
}

func validateScope(scope Scope) error {
	switch scope.Kind {
	case ScopeDesign:
		if scope.FrameID != "" || scope.NodeID != "" {
			return fmt.Errorf("%w: design scope does not take a frame or layer id", ErrInvalid)
		}
	case ScopeFrame:
		if strings.TrimSpace(scope.FrameID) == "" || scope.NodeID != "" {
			return fmt.Errorf("%w: frame scope requires a frame id", ErrInvalid)
		}
	case ScopeLayer:
		if strings.TrimSpace(scope.FrameID) == "" || strings.TrimSpace(scope.NodeID) == "" {
			return fmt.Errorf("%w: layer scope requires a frame and a layer id", ErrInvalid)
		}
	default:
		return fmt.Errorf("%w: scope kind must be design, frame or layer", ErrInvalid)
	}
	return nil
}
