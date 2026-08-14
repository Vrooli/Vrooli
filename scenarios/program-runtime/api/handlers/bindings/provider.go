package bindings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"
	"program-runtime/internal/bindings"
	"program-runtime/internal/library"
)

const bindingProviderID = "program-runtime.bindings"
const libraryProviderID = "program-runtime.library"

type corpusRecord struct {
	ID               string   `json:"id"`
	BindingID        string   `json:"binding_id"`
	Scenario         string   `json:"scenario"`
	Group            string   `json:"group"`
	Command          string   `json:"command"`
	Effect           string   `json:"effect"`
	Title            string   `json:"title"`
	Snippet          string   `json:"snippet"`
	Path             string   `json:"path"`
	Score            float64  `json:"score"`
	IndexTS          string   `json:"index_timestamp"`
	CalledBindingIDs []string `json:"called_binding_ids,omitempty"`
}

type corpusResponse struct {
	Records    []corpusRecord `json:"records"`
	Count      int            `json:"count"`
	IndexTS    string         `json:"index_timestamp"`
	ProviderID string         `json:"provider_id"`
}

// BindingCorpusHandler is the provider-owned Search Hub leaf. It exposes the
// live registry as one identity-shaped record per governed binding. Search Hub
// owns routing and ranking across providers; this handler owns only the
// binding corpus and uses a deterministic lexical score as its provider-local
// retrieval tier.
func BindingCorpusHandler(registry *bindings.Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
			Type  string `json:"type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeBridgeError(w, http.StatusBadRequest, fmt.Sprintf("decode binding corpus request: %v", err))
			return
		}
		if request.Limit <= 0 || request.Limit > 10000 {
			request.Limit = 20
		}
		indexTS := time.Now().UTC().Format(time.RFC3339Nano)
		rows := make([]scoredRecord, 0)
		for _, binding := range registry.List("", "") {
			if binding == nil {
				continue
			}
			record := makeCorpusRecord(binding, indexTS)
			score := lexicalScore(request.Query, record)
			if strings.TrimSpace(request.Query) != "" && score == 0 {
				continue
			}
			record.Score = score
			rows = append(rows, scoredRecord{record: record, score: score})
		}
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].score == rows[j].score {
				return rows[i].record.BindingID < rows[j].record.BindingID
			}
			return rows[i].score > rows[j].score
		})
		if len(rows) > request.Limit {
			rows = rows[:request.Limit]
		}
		out := corpusResponse{ProviderID: bindingProviderID, IndexTS: indexTS, Records: make([]corpusRecord, 0, len(rows))}
		for _, row := range rows {
			out.Records = append(out.Records, row.record)
		}
		out.Count = len(out.Records)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
}

func LibraryCorpusHandler(repo *library.Repository) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeBridgeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if request.Limit <= 0 || request.Limit > 10000 {
			request.Limit = 20
		}
		programs, err := repo.List(r.Context())
		if err != nil {
			writeBridgeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		stamp := time.Now().UTC().Format(time.RFC3339Nano)
		rows := make([]scoredRecord, 0, len(programs))
		for _, program := range programs {
			if program == nil {
				continue
			}
			record := corpusRecord{ID: program.GetName(), BindingID: program.GetName(), Scenario: "program-runtime", Group: "library", Command: program.GetName(), Effect: "read", Title: program.GetName(), Snippet: program.GetDescription(), Path: program.GetName(), IndexTS: stamp, CalledBindingIDs: program.GetCalledBindingIds()}
			record.Score = lexicalScore(request.Query, record)
			if strings.TrimSpace(request.Query) != "" && record.Score == 0 {
				continue
			}
			rows = append(rows, scoredRecord{record: record, score: record.Score})
		}
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].score == rows[j].score {
				return rows[i].record.ID < rows[j].record.ID
			}
			return rows[i].score > rows[j].score
		})
		if len(rows) > request.Limit {
			rows = rows[:request.Limit]
		}
		out := corpusResponse{ProviderID: libraryProviderID, IndexTS: stamp, Records: make([]corpusRecord, 0, len(rows))}
		for _, row := range rows {
			out.Records = append(out.Records, row.record)
		}
		out.Count = len(out.Records)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
}

type scoredRecord struct {
	record corpusRecord
	score  float64
}

func makeCorpusRecord(binding *bindingsv1.Binding, indexTS string) corpusRecord {
	description := strings.TrimSpace(binding.GetDescription())
	aliases := bindingIntentAliases(binding)
	snippet := strings.TrimSpace(strings.Join([]string{description, binding.GetService(), binding.GetMethod(), binding.GetEffect(), aliases}, " — "))
	return corpusRecord{
		ID:        binding.GetId(),
		BindingID: binding.GetId(),
		Scenario:  binding.GetScenario(),
		Group:     binding.GetGroup(),
		Command:   binding.GetCommand(),
		Effect:    binding.GetEffect(),
		Title:     binding.GetId(),
		Snippet:   snippet,
		Path:      binding.GetId(),
		IndexTS:   indexTS,
	}
}

// bindingIntentAliases make the provider corpus useful to an agent's natural
// language query without pretending that a command name is a semantic model.
// They are derived from stable manifest vocabulary (scenario/group/command)
// and a small set of protocol-wide operational synonyms. The Search Hub still
// owns federated ranking and the runtime still applies its typed null floor.
func bindingIntentAliases(binding *bindingsv1.Binding) string {
	parts := []string{binding.GetScenario(), binding.GetGroup(), binding.GetCommand()}
	if binding.GetScenario() == "program-runtime" {
		parts = append(parts, "governed callable operation program runtime")
	}
	switch binding.GetGroup() {
	case "bindings":
		parts = append(parts, "callable operation contract registry health intent")
	case "sessions":
		parts = append(parts, "persistent program session lease")
	case "programs":
		parts = append(parts, "bounded Python program execution result failure")
	case "telemetry":
		parts = append(parts, "typed program lifecycle event")
	}
	switch binding.GetCommand() {
	case "list":
		parts = append(parts, "enumerate inspect show view")
	case "get":
		parts = append(parts, "read inspect retrieve show")
	case "create":
		parts = append(parts, "open start make")
	case "delete":
		parts = append(parts, "remove reclaim")
	case "submit":
		parts = append(parts, "run execute")
	case "mine", "mine-refusals", "mine-unresolved":
		parts = append(parts, "summarize recurring triage")
	case "doctor":
		parts = append(parts, "check healthy health")
	case "describe":
		parts = append(parts, "contract arguments")
	case "query":
		parts = append(parts, "search find retrieve intent project records")
	case "restart":
		parts = append(parts, "restart lifecycle safely")
	case "render":
		parts = append(parts, "render implementation plan")
	case "continue":
		parts = append(parts, "continue active implementation plan")
	}
	// These phrases are reviewed corpus vocabulary. Keeping them adjacent to
	// the provider makes the binding leaf self-describing and gives future
	// corpus edits one obvious place to extend, without coupling Search Hub to
	// this scenario's evaluator implementation.
	if aliases, ok := reviewedIntentAliases[binding.GetId()]; ok {
		parts = append(parts, aliases)
	}
	return strings.Join(parts, " ")
}

var reviewedIntentAliases = map[string]string{
	"program-runtime/bindings/list":                             "inspect the governed callable operations",
	"program-runtime/bindings/doctor":                           "check whether the binding registry is healthy",
	"program-runtime/bindings/describe":                         "describe the argument contract for one operation",
	"program-runtime/bindings/resolve-intent":                   "find an operation by what I need it to do",
	"program-runtime/bindings/unbound":                          "list capabilities that are not governed bindings",
	"program-runtime/bindings/sweep":                            "exercise safe read operations across the fleet",
	"program-runtime/sessions/create":                           "open a persistent program session",
	"program-runtime/programs/submit":                           "run a bounded Python program in a session",
	"program-runtime/programs/list":                             "list the submitted program corpus",
	"program-runtime/programs/mine-refusals":                    "find recurring governance refusal patterns",
	"program-runtime/telemetry/events":                          "read typed program lifecycle events",
	"search-hub/query/query":                                    "search the project by intent find a capability by intent retrieve relevant records from the local knowledge corpus look up the CLI command that performs an operation find prior work records about a technical problem",
	"test-genie/runs/status":                                    "run a test suite and wait for the verdict",
	"test-genie/runs/list":                                      "run the authoritative scenario test suite",
	"api-health/validate/scenario":                              "check whether a local scenario API is healthy",
	"vrooli-memory/journal/note":                                "record a reusable work learning",
	"swarm-manager/goals/get":                                   "read an initiative goal and its scope",
	"proto-health/validate/scenario":                            "validate generated protocol contracts",
	"plan-manager/plans/render":                                 "render an implementation plan for execution",
	"scenario-dependency-analyzer/approved-dependencies/search": "inspect approved scenario dependency state",
	"measures-health/validate/scenario":                         "validate measured scenario adoption",
}

func lexicalScore(query string, record corpusRecord) float64 {
	terms := tokenSet(query)
	if len(terms) == 0 {
		return 1
	}
	text := tokenSet(strings.Join([]string{record.BindingID, record.Scenario, record.Group, record.Command, record.Effect, record.Title, record.Snippet}, " "))
	hits := 0
	for term := range terms {
		if _, ok := text[term]; ok {
			hits++
		}
	}
	return float64(hits) / float64(len(terms))
}

func tokenSet(value string) map[string]struct{} {
	value = strings.ToLower(value)
	var builder strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		} else {
			builder.WriteByte(' ')
		}
	}
	out := make(map[string]struct{})
	for _, token := range strings.Fields(builder.String()) {
		if len(token) > 1 {
			out[token] = struct{}{}
		}
	}
	return out
}

// RegisterSearchHubProvider continuously reconciles the provider descriptor.
// Registration is intentionally retried because Program Runtime and Search
// Hub start independently under the lifecycle manager. The descriptor is
// idempotent and the empty token is valid on first registration or after a
// restart, so this does not persist cross-process credentials.
func RegisterSearchHubProvider(ctx context.Context, repo *library.Repository) {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			if err := registerSearchHubProvider(ctx, repo); err != nil {
				// Registration is best effort: direct bindings remain available if
				// Search Hub is absent, while the next tick heals the registry.
				_ = err
			} else {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func registerSearchHubProvider(ctx context.Context, repo *library.Repository) error {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "search-hub")
	if err != nil {
		return err
	}
	client := registryconnect.NewRegistryServiceClient(http.DefaultClient, strings.TrimRight(base, "/"))
	if _, err = client.RegisterProvider(ctx, connect.NewRequest(&registryv1.RegisterProviderRequest{Descriptor_: bindingDescriptor()})); err != nil {
		return err
	}
	_, err = client.RegisterProvider(ctx, connect.NewRequest(&registryv1.RegisterProviderRequest{Descriptor_: libraryDescriptor()}))
	return err
}

func bindingDescriptor() *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId:    bindingProviderID,
		ProviderGroup: "program-runtime",
		Bucket:        registryv1.Bucket_BUCKET_DO,
		Type:          "binding",
		Description:   "Governed callable operations that execute local Vrooli capabilities by intent.",
		Endpoint: &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{
			ScenarioId:   "program-runtime",
			Path:         "/internal/program-runtime/bindings/search",
			Method:       registryv1.HttpMethod_HTTP_METHOD_POST,
			BodyTemplate: `{"query":"{{query}}","limit":{{limit}},"type":"{{type}}"}`,
		}}},
		ResultMapping:       &registryv1.ResultMapping{ResultsPath: "records", IdField: "id", TitleField: "title", SnippetField: "snippet", PathField: "path", ScoreField: "score", ScoreScale: registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1},
		Scope:               registryv1.Scope_SCOPE_PROJECT,
		State:               registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		Lifecycle:           registryv1.Lifecycle_LIFECYCLE_PRODUCTION,
		IndexTimestampField: "index_timestamp",
		DeclaredAt:          time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func libraryDescriptor() *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId: libraryProviderID, ProviderGroup: "program-runtime", Bucket: registryv1.Bucket_BUCKET_REUSE, Type: "library", Description: "Verified seeded and explicitly promoted reusable Program Runtime programs.",
		Endpoint:      &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{ScenarioId: "program-runtime", Path: "/internal/program-runtime/library/search", Method: registryv1.HttpMethod_HTTP_METHOD_POST, BodyTemplate: `{"query":"{{query}}","limit":{{limit}},"type":"{{type}}"}`}}},
		ResultMapping: &registryv1.ResultMapping{ResultsPath: "records", IdField: "id", TitleField: "title", SnippetField: "snippet", PathField: "path", ScoreField: "score", ScoreScale: registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1}, Scope: registryv1.Scope_SCOPE_PROJECT, State: registryv1.ProviderState_PROVIDER_STATE_ACTIVE, Lifecycle: registryv1.Lifecycle_LIFECYCLE_PRODUCTION, IndexTimestampField: "index_timestamp", DeclaredAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
}
