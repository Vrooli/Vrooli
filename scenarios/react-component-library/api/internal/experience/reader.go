// Package experience projects declarative component contracts and their latest
// Experience Manager evidence for the RCL API. It is deliberately server-side:
// the browser never calls Experience Manager directly.
package experience

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
	contractconnect "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract/contract_v1connect"
)

type Component struct {
	ID        string
	LibraryID string
	Slug      string
	Version   string
}

type (
	State struct{ ID, ExampleName, Description string }
	Claim struct {
		ID, Type, Statement, Tier string
		States                    []string
		Elements                  []string
		Params                    map[string]any
	}
)

type Evidence struct {
	ClaimID, ClaimType, Verdict, StateID, ExampleName, CaptureRef, CheckedAt, Message, Viewport string
	ViewportWidth, ViewportHeight                                                               int
	Measurement                                                                                 *contractv1.ClaimMeasurement
}
type Snapshot struct {
	ComponentID, LibraryID, Version, ContractID, Title, Purpose string
	States                                                      []State
	Claims                                                      []Claim
	Evidence                                                    []Evidence
	EvidenceStatus, EvidenceMessage                             string
}

type Reader interface {
	Get(context.Context, Component) (Snapshot, error)
}

type reader struct {
	repoRoot     string
	resolver     *discovery.Resolver
	client       *http.Client
	listEvidence func(context.Context, string) ([]Evidence, error)
}

func NewReader(repoRoot string) Reader {
	return &reader{repoRoot: repoRoot, resolver: discovery.NewResolver(discovery.ResolverConfig{}), client: &http.Client{Timeout: 10 * time.Second}}
}

func (r *reader) Get(ctx context.Context, component Component) (Snapshot, error) {
	contractID := kebabID(component.Slug)
	if contractID == "" {
		contractID = kebabID(strings.TrimPrefix(component.LibraryID, "react-component-library:"))
	}
	out := Snapshot{ComponentID: component.ID, LibraryID: component.LibraryID, Version: component.Version, ContractID: contractID, EvidenceStatus: "unavailable"}
	data, err := r.readVersionContract(component)
	if err != nil {
		if os.IsNotExist(err) {
			out.EvidenceStatus = "not-configured"
			out.EvidenceMessage = "This exact library version does not yet have a canonical experience contract."
			return out, nil
		}
		return Snapshot{}, fmt.Errorf("read component experience contract: %w", err)
	}
	var doc struct {
		Component struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Purpose string `json:"purpose"`
		} `json:"component"`
		States []struct {
			ID          string `json:"id"`
			Example     string `json:"example"`
			Description string `json:"description"`
		} `json:"states"`
		Claims []struct {
			ID        string         `json:"id"`
			Type      string         `json:"type"`
			Statement string         `json:"statement"`
			Tier      string         `json:"tier"`
			States    []string       `json:"states"`
			Elements  []string       `json:"elements"`
			Params    map[string]any `json:"params"`
		} `json:"claims"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return Snapshot{}, fmt.Errorf("decode component experience contract: %w", err)
	}
	out.ContractID, out.Title, out.Purpose = doc.Component.ID, doc.Component.Title, doc.Component.Purpose
	for _, state := range doc.States {
		out.States = append(out.States, State{ID: state.ID, ExampleName: state.Example, Description: state.Description})
	}
	for _, claim := range doc.Claims {
		out.Claims = append(out.Claims, Claim{ID: claim.ID, Type: claim.Type, Statement: claim.Statement, Tier: claim.Tier, States: append([]string(nil), claim.States...), Elements: append([]string(nil), claim.Elements...), Params: claim.Params})
	}
	evidence, err := r.evidence(ctx, out.ContractID)
	if err != nil {
		out.EvidenceMessage = "Experience Manager could not return evidence; declared behavior is shown without a live verdict."
		return out, nil
	}
	out.Evidence = evidence
	out.EvidenceStatus = "available"
	if len(out.Evidence) == 0 {
		out.EvidenceMessage = "No reconciliation capture has been recorded yet."
	}
	return out, nil
}

// readVersionContract resolves only an exact component version. Scenario-level
// contracts are intentionally not consulted: that would make a mutable slug
// the authority for an immutable library version.
func (r *reader) readVersionContract(component Component) ([]byte, error) {
	name := strings.TrimSpace(component.Slug)
	if name == "" {
		name = strings.TrimPrefix(strings.TrimSpace(component.LibraryID), "react-component-library:")
	}
	version := strings.TrimSpace(component.Version)
	if name == "" || version == "" {
		return nil, os.ErrNotExist
	}
	root := filepath.Join(r.repoRoot, "scenarios", "react-component-library", "library")
	// Experience contracts are part of every publishable library tier. The
	// reader used to accept only components and hooks, which silently turned
	// primitive/foundation captures into "not-configured" evidence and left
	// those assets below verified even when Experience Manager had passed them.
	for _, kind := range []string{"foundations", "primitives", "components", "hooks", "services"} {
		path := filepath.Join(root, kind, filepath.Clean(name), "versions", filepath.Clean(version), "experience-contract.json")
		data, err := os.ReadFile(path)
		if err == nil || !os.IsNotExist(err) {
			return data, err
		}
	}
	return nil, os.ErrNotExist
}

func (r *reader) evidence(ctx context.Context, componentID string) ([]Evidence, error) {
	if r.listEvidence != nil {
		return r.listEvidence(ctx, componentID)
	}
	base, err := r.resolver.ResolveScenarioURLDefault(ctx, "experience-manager")
	if err != nil {
		return nil, err
	}
	client := contractconnect.NewStudioSessionServiceClient(r.client, base)
	response, err := client.ListEvidence(ctx, connect.NewRequest(&contractv1.ListEvidenceRequest{Scenario: "react-component-library", Component: componentID, Limit: 100}))
	if err != nil {
		return nil, err
	}
	out := make([]Evidence, 0, len(response.Msg.GetEvidence()))
	for _, item := range response.Msg.GetEvidence() {
		out = append(out, Evidence{ClaimID: item.GetClaim(), ClaimType: item.GetClaimType(), Verdict: item.GetVerdict(), StateID: item.GetState(), ExampleName: item.GetExampleName(), CaptureRef: item.GetCaptureRef(), CheckedAt: item.GetCheckedAt(), Message: item.GetMessage(), Viewport: item.GetViewport(), ViewportWidth: int(item.GetViewportWidth()), ViewportHeight: int(item.GetViewportHeight()), Measurement: item.GetMeasurement()})
	}
	return out, nil
}

func kebabID(in string) string {
	var out []rune
	for i, r := range strings.TrimSpace(in) {
		if unicode.IsUpper(r) && i > 0 {
			out = append(out, '-')
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, unicode.ToLower(r))
		}
	}
	return strings.Trim(string(out), "-")
}
