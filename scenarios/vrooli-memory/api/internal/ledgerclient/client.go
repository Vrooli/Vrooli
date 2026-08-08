// Package ledgerclient is the only source-ledger dependency of vrooli-memory.
// The memory scenario owns the harness projection/status state; the durable
// journal and all derived memory engines live behind this client.
package ledgerclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	sourcefacetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/facets/facets_v1connect"
	sourceforest "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/forest"
	sourceforestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/forest/forest_v1connect"
	sourcejournalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
	sourcerecallconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall/recall_v1connect"
	sourcerulesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/rules/rules_v1connect"
	sourcescopesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes/scopesv1connect"
)

const DefaultScope = "agent-memory"

// Client holds Connect clients created once during startup. Keeping discovery
// out of request paths makes dependency resolution deterministic and avoids a
// partially discovered client graph during an outage.
type Client struct {
	BaseURL string
	Journal sourcejournalconnect.JournalServiceClient
	Facets  sourcefacetsconnect.FacetsServiceClient
	Forest  sourceforestconnect.ForestServiceClient
	Recall  sourcerecallconnect.RecallServiceClient
	Rules   sourcerulesconnect.ClassificationRulesServiceClient
	Scopes  sourcescopesconnect.ScopesServiceClient
}

func New(ctx context.Context) (*Client, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "source-ledger")
	if err != nil {
		return nil, fmt.Errorf("resolve source-ledger endpoint: %w", err)
	}
	return NewAt(baseURL), nil
}

func NewAt(baseURL string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	httpClient := http.DefaultClient
	return &Client{
		BaseURL: baseURL,
		Journal: sourcejournalconnect.NewJournalServiceClient(httpClient, baseURL),
		Facets:  sourcefacetsconnect.NewFacetsServiceClient(httpClient, baseURL),
		Forest:  sourceforestconnect.NewForestServiceClient(httpClient, baseURL),
		Recall:  sourcerecallconnect.NewRecallServiceClient(httpClient, baseURL),
		Rules:   sourcerulesconnect.NewClassificationRulesServiceClient(httpClient, baseURL),
		Scopes:  sourcescopesconnect.NewScopesServiceClient(httpClient, baseURL),
	}
}

// NewWithClients is intended for focused tests of outage and projection
// behavior. Production code should use New so discovery is resolved once.
func NewWithClients(journal sourcejournalconnect.JournalServiceClient, facets sourcefacetsconnect.FacetsServiceClient, forest sourceforestconnect.ForestServiceClient, recall sourcerecallconnect.RecallServiceClient, rules sourcerulesconnect.ClassificationRulesServiceClient, scopes sourcescopesconnect.ScopesServiceClient) *Client {
	return &Client{Journal: journal, Facets: facets, Forest: forest, Recall: recall, Rules: rules, Scopes: scopes}
}

// UnavailableError is used below the transport boundary so projection and
// maintenance can distinguish a source-ledger outage from a bad request.
type UnavailableError struct {
	Operation string
	Err       error
}

func (e *UnavailableError) Error() string {
	return fmt.Sprintf("source-ledger %s unavailable: %v", e.Operation, e.Err)
}
func (e *UnavailableError) Unwrap() error { return e.Err }

func IsUnavailable(err error) bool {
	var unavailable *UnavailableError
	return errors.As(err, &unavailable) || connect.CodeOf(err) == connect.CodeUnavailable
}

func NormalizeError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if connect.CodeOf(err) == connect.CodeUnavailable {
		return &UnavailableError{Operation: operation, Err: err}
	}
	return err
}

// Translate copies the wire-compatible source-ledger/memory protobuf shapes
// without coupling the public memory package to source-ledger package types.
// DiscardUnknown is deliberate: source-ledger may add response fields while
// preserving the older vrooli-memory public contract.
func Translate(in, out proto.Message) error {
	b, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(in)
	if err != nil {
		return err
	}
	return (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, out)
}

func TranslateWithScope(in, out proto.Message, scope string) error {
	if err := Translate(in, out); err != nil {
		return err
	}
	scope = NormalizeScope(scope)
	fields := out.ProtoReflect().Descriptor().Fields()
	field := fields.ByName(protoreflect.Name("scope"))
	if field != nil && field.Kind() == protoreflect.StringKind {
		out.ProtoReflect().Set(field, protoreflect.ValueOfString(scope))
	}
	return nil
}

func NormalizeScope(scope string) string {
	if strings.TrimSpace(scope) == "" {
		return DefaultScope
	}
	return strings.TrimSpace(scope)
}

func ForwardHeaders(from, to http.Header) {
	for key, values := range from {
		for _, value := range values {
			to.Add(key, value)
		}
	}
}

func RPCError(operation string, err error) error {
	if err == nil {
		return nil
	}
	err = NormalizeError(operation, err)
	if IsUnavailable(err) {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	if code := connect.CodeOf(err); code != connect.CodeUnknown {
		return err
	}
	return connect.NewError(connect.CodeInternal, err)
}

type CompactionResult struct{ CompactedCount, EligibleFrontierBefore, EligibleFrontierAfter, Target int }

func (c *Client) RunBounded(ctx context.Context, _ int) (CompactionResult, error) {
	resp, err := c.Forest.RunCompactionPass(ctx, connect.NewRequest(&sourceforest.RunCompactionPassRequest{Scope: DefaultScope}))
	if err != nil {
		return CompactionResult{}, NormalizeError("compaction", err)
	}
	return CompactionResult{CompactedCount: int(resp.Msg.GetCompactedCount())}, nil
}
