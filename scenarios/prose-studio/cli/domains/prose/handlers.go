package prose

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/prose-studio/v1/prose"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prose-studio/v1/prose/prose_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type handlers struct {
	client connectv1.ProseStudioServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: connectv1.NewProseStudioServiceClient(httpClient, baseURL)}
}

func requestFromFlag[T proto.Message](ctx cliapp.OperationContext, message T) (T, error) {
	payload := ctx.Flag("request-json")
	if payload == "" {
		payload = `{}`
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal([]byte(payload), message); err != nil {
		return message, fmt.Errorf("decode --json: %w", err)
	}
	return message, nil
}

func (h *handlers) registryCall(ctx cliapp.OperationContext) (*v1.RegistryResponse, error) {
	request, err := requestFromFlag(ctx, &v1.RegistryRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.Registry(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose Registry", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) createStyleCall(ctx cliapp.OperationContext) (*v1.CreateStyleResponse, error) {
	request, err := requestFromFlag(ctx, &v1.CreateStyleRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.CreateStyle(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose CreateStyle", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) resolveProfileCall(ctx cliapp.OperationContext) (*v1.ResolveProfileResponse, error) {
	request, err := requestFromFlag(ctx, &v1.ResolveProfileRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.ResolveProfile(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose ResolveProfile", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) generateCall(ctx cliapp.OperationContext) (*v1.GenerateResponse, error) {
	request, err := requestFromFlag(ctx, &v1.GenerateRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.Generate(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose Generate", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) rerollCall(ctx cliapp.OperationContext) (*v1.RerollResponse, error) {
	request, err := requestFromFlag(ctx, &v1.RerollRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.Reroll(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose Reroll", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) sessionActionCall(ctx cliapp.OperationContext) (*v1.SessionActionResponse, error) {
	request, err := requestFromFlag(ctx, &v1.SessionActionRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.SessionAction(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose SessionAction", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) reindexCall(ctx cliapp.OperationContext) (*v1.ReindexDeclarationsResponse, error) {
	request, err := requestFromFlag(ctx, &v1.ReindexDeclarationsRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.ReindexDeclarations(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose ReindexDeclarations", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) validateCall(ctx cliapp.OperationContext) (*v1.ValidateDeclarationsResponse, error) {
	request, err := requestFromFlag(ctx, &v1.ValidateDeclarationsRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.ValidateDeclarations(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose ValidateDeclarations", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) createDocumentCall(ctx cliapp.OperationContext) (*v1.CreateDocumentResponse, error) {
	request, err := requestFromFlag(ctx, &v1.CreateDocumentRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.CreateDocument(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose CreateDocument", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) assembleDocumentCall(ctx cliapp.OperationContext) (*v1.AssembleDocumentResponse, error) {
	request, err := requestFromFlag(ctx, &v1.AssembleDocumentRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.AssembleDocument(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose AssembleDocument", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) resumeDocumentCall(ctx cliapp.OperationContext) (*v1.ResumeDocumentResponse, error) {
	request, err := requestFromFlag(ctx, &v1.ResumeDocumentRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.ResumeDocument(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose ResumeDocument", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) conformanceCall(ctx cliapp.OperationContext) (*v1.ConformanceResponse, error) {
	request, err := requestFromFlag(ctx, &v1.ConformanceRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.Conformance(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose Conformance", err, nil)
	}
	return responseMessage(response)
}

func responseMessage[T any](response *connect.Response[T]) (*T, error) {
	var zero *T
	if response == nil || response.Msg == nil {
		return zero, fmt.Errorf("server returned no prose response")
	}
	return response.Msg, nil
}

func protoReport[T proto.Message](label string) func(cliapp.OperationContext, T) cliapp.ListReport {
	return func(_ cliapp.OperationContext, message T) cliapp.ListReport {
		raw, _ := (protojson.MarshalOptions{UseProtoNames: true, Multiline: true, Indent: "  "}).Marshal(message)
		var pretty any
		if json.Unmarshal(raw, &pretty) == nil {
			raw, _ = json.MarshalIndent(pretty, "", "  ")
		}
		return cliapp.ListReport{Summary: []string{label}, ResultsHeading: "Response", Results: []string{string(raw)}}
	}
}

// listDocumentsCall answers "what has this scenario written?". The flags are
// decoded into the typed request here rather than through a generic dispatcher,
// which is why the manifest declares them local_only with a waiver.
func (h *handlers) listDocumentsCall(ctx cliapp.OperationContext) (*v1.ListDocumentsResponse, error) {
	request := &v1.ListDocumentsRequest{Status: strings.TrimSpace(ctx.Flag("status"))}
	if raw := strings.TrimSpace(ctx.Flag("limit")); raw != "" {
		// Parsed at 32 bits and range-checked against the server's own ceiling, so
		// the value can never overflow the typed request field it lands in.
		limit, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || limit < 0 || limit > maxListLimit {
			return nil, fmt.Errorf("--limit must be a whole number between 0 and %d, got %q", maxListLimit, raw)
		}
		request.Limit = int32(limit)
	}
	response, err := h.client.ListDocuments(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose ListDocuments", err, nil)
	}
	return responseMessage(response)
}

func (h *handlers) getDocumentCall(ctx cliapp.OperationContext) (*v1.GetDocumentResponse, error) {
	id := strings.TrimSpace(ctx.Flag("id"))
	if id == "" {
		return nil, fmt.Errorf("--id is required; run 'prose-studio prose document-list' to find one")
	}
	response, err := h.client.GetDocument(context.Background(), connect.NewRequest(&v1.GetDocumentRequest{Id: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose GetDocument", err, nil)
	}
	return responseMessage(response)
}

// maxListLimit mirrors the server's listing ceiling so the CLI rejects an
// out-of-range value with a clear message instead of sending one the server
// will silently clamp.
const maxListLimit = 200

func usd(micros int64) string { return fmt.Sprintf("$%.4f", float64(micros)/1_000_000) }

// shortStamp renders a creation time, or a dash when there is not one. Go's
// zero time serialises as a valid RFC3339 string rather than as an absent
// field, so a document written before creation times were recorded parses
// cleanly and would otherwise be displayed as a date in the year zero.
func shortStamp(rfc3339 string) string {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(rfc3339))
	if err != nil || parsed.IsZero() || parsed.Year() < 2000 {
		return "-"
	}
	return parsed.Local().Format("2006-01-02 15:04")
}

func truncate(text string, width int) string {
	runes := []rune(strings.TrimSpace(strings.ReplaceAll(text, "\n", " ")))
	if len(runes) <= width {
		return string(runes)
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

// documentListReport renders a table a reader can scan, not a protojson dump.
// The identifier is shown in full because it is the argument to document-show,
// and an identifier the reader has to reassemble from a truncated column is not
// usable.
func documentListReport(_ cliapp.OperationContext, message *v1.ListDocumentsResponse) cliapp.ListReport {
	documents := message.GetDocuments()
	if len(documents) == 0 {
		return cliapp.ListReport{
			Summary:        []string{"No documents yet."},
			RetrievalHints: []string{"Generate one with 'prose-studio prose document-create'."},
		}
	}
	rows := []string{fmt.Sprintf("%-36s  %-10s  %6s  %8s  %8s  %9s  %-16s  %s", "ID", "STATUS", "WORDS", "SECTIONS", "COHERENT", "COST", "CREATED", "TITLE")}
	for _, doc := range documents {
		coherent := "no"
		if doc.GetCoherent() {
			coherent = "yes"
		}
		rows = append(rows, fmt.Sprintf("%-36s  %-10s  %6d  %8d  %8s  %9s  %-16s  %s",
			doc.GetId(), doc.GetStatus(), doc.GetWordCount(), doc.GetSectionCount(),
			coherent, usd(doc.GetTotalCostMicros()), shortStamp(doc.GetCreatedAt()), truncate(doc.GetTitle(), 60)))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d document(s), newest first", len(documents))},
		ResultsHeading: "Documents",
		Results:        rows,
		RetrievalHints: []string{"Read one with 'prose-studio prose document-show --id <ID>'."},
	}
}

// documentShowReport renders a document the way a person reads one: what it is,
// how it was planned, how it measured, and then the prose. --text-only prints
// the prose alone so the output can be piped into a file or an editor.
func documentShowReport(ctx cliapp.OperationContext, message *v1.GetDocumentResponse) cliapp.ListReport {
	doc := message.GetDocument()
	if doc == nil {
		return cliapp.ListReport{Summary: []string{"No document returned."}}
	}
	body := strings.TrimSpace(doc.GetAssembledText())
	if body == "" {
		body = "(no assembled text; the document is " + doc.GetStatus() + ")"
	}
	if ctx.BoolFlag("text-only") {
		// Still name the document. The renderer always emits its section
		// headings, so an empty summary reads as "(no summary available)" rather
		// than as bare prose, which is worse than one identifying line.
		return cliapp.ListReport{Summary: []string{truncate(doc.GetTitle(), 100)}, Results: []string{body}}
	}

	provenance := doc.GetDocumentProvenance()
	summary := []string{
		truncate(doc.GetTitle(), 100),
		fmt.Sprintf("status %s · profile %s · created %s", doc.GetStatus(), doc.GetProfileKey(), shortStamp(doc.GetCreatedAt())),
	}
	if provenance != nil {
		summary = append(summary, fmt.Sprintf("%d words across %d sections · %s · %s",
			provenance.GetWordCount(), provenance.GetSectionCount(), usd(provenance.GetTotalCostMicros()),
			strings.Join(append(provenance.GetProviders(), provenance.GetModels()...), " "))) //nolint:gocritic // provider and model read as one provenance line
	}

	var results []string
	if outline := doc.GetOutline(); len(outline) > 0 {
		results = append(results, "Outline")
		for i, section := range outline {
			results = append(results, fmt.Sprintf("  %d. %s  [%dw]", i+1, section.GetIntent(), section.GetTargetWords()))
		}
		results = append(results, "")
	}
	if coherence := doc.GetCoherence(); coherence != nil {
		results = append(results, "Coherence")
		for _, line := range coherenceLines(coherence.AsMap()) {
			results = append(results, "  "+line)
		}
		results = append(results, "")
	}
	results = append(results, "Article", "")
	results = append(results, body)
	return cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Document",
		Results:        results,
		RetrievalHints: []string{"Print the prose alone with --text-only."},
	}
}

// coherenceLines states each measure against its threshold, and says plainly
// when a measure was not taken. An unmeasured axis reported as a pass is the
// failure this surface exists to prevent.
func coherenceLines(coherence map[string]any) []string {
	thresholds, _ := coherence["thresholds"].(map[string]any)
	number := func(container map[string]any, key string) (float64, bool) {
		if container == nil {
			return 0, false
		}
		value, ok := container[key].(float64)
		return value, ok
	}
	line := func(label, valueKey, thresholdKey string) string {
		value, ok := number(coherence, valueKey)
		if !ok {
			return fmt.Sprintf("%-28s not measured", label)
		}
		limit, hasLimit := number(thresholds, thresholdKey)
		if !hasLimit {
			return fmt.Sprintf("%-28s %.4f", label, value)
		}
		verdict := "PASS"
		if value > limit {
			verdict = "FAIL"
		}
		return fmt.Sprintf("%-28s %.4f  (max %.2f)  %s", label, value, limit, verdict)
	}
	lines := []string{
		line("lexical repetition", "cross_section_repetition", "max_cross_section_repetition"),
		line("semantic repetition", "semantic_section_repetition", "max_semantic_section_repetition"),
		line("style drift", "style_drift", "max_style_drift"),
	}
	if reason, ok := coherence["semantic_unavailable"].(string); ok && reason != "" {
		lines = append(lines, "semantic measurement unavailable: "+truncate(reason, 90))
	}
	if verdict, ok := coherence["verdict"].(map[string]any); ok {
		if coherent, ok := verdict["coherent"].(bool); ok {
			state := "FAIL"
			if coherent {
				state = "PASS"
			}
			lines = append(lines, fmt.Sprintf("%-28s %s", "overall", state))
		}
	}
	return lines
}
