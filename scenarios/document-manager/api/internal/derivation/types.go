package derivation

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

type Format string

type Handler struct {
	ID           string   `json:"id"`
	Version      string   `json:"version"`
	Formats      []string `json:"formats"`
	Capabilities []string `json:"capabilities"`
	Tier         int      `json:"tier"`
	Runtime      string   `json:"runtime"`
}

type Registry struct {
	Handlers []Handler `json:"handlers"`
}

type Unit struct {
	Index      int                   `json:"index"`
	Text       string                `json:"text"`
	Kind       documentpb.AnchorKind `json:"anchor_kind"`
	Confidence float32               `json:"confidence"`
	Metadata   map[string]string     `json:"metadata,omitempty"`
}

type Model struct {
	Units []Unit `json:"units"`
}

type Result struct {
	DocumentHash string                   `json:"document_hash"`
	Version      int                      `json:"version"`
	Tier         documentpb.Tier          `json:"tier"`
	Chain        []string                 `json:"chain"`
	Handlers     []string                 `json:"handlers"`
	Model        Model                    `json:"model"`
	State        documentpb.TerminalState `json:"state"`
	Reason       string                   `json:"reason"`
	Remedy       string                   `json:"remedy"`
	Skipped      []string                 `json:"skipped_capabilities,omitempty"`
}

type Input struct {
	DocumentHash string
	Mime         string
	Bytes        []byte
	PrivacyClass documentpb.PrivacyClass
	TierCeiling  documentpb.Tier
}

type HandlerError struct {
	Unavailable bool
	Variant     bool
	Err         error
}

func (e *HandlerError) Error() string { return e.Err.Error() }
func (e *HandlerError) Unwrap() error { return e.Err }

var (
	ErrNoHandler       = errors.New("no handler for format")
	ErrBlockedByPolicy = errors.New("handler blocked by policy")
)

//go:embed registry.json
var registryBytes embed.FS

func LoadRegistry() (Registry, error) {
	data, err := registryBytes.ReadFile("registry.json")
	if err != nil {
		return Registry{}, err
	}
	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return Registry{}, err
	}
	return registry, nil
}

type Runner interface {
	Run(context.Context, Handler, Input) (Model, error)
}

func (r Registry) Match(mime string, ceiling documentpb.Tier) (Handler, error) {
	matched := false
	for _, handler := range r.Handlers {
		for _, format := range handler.Formats {
			if strings.EqualFold(format, mime) {
				matched = true
				if handler.Tier > int(ceiling) {
					continue
				}
				return handler, nil
			}
		}
	}
	if matched {
		return Handler{}, fmt.Errorf("%w: %s", ErrBlockedByPolicy, mime)
	}
	return Handler{}, fmt.Errorf("%w: %s", ErrNoHandler, mime)
}
