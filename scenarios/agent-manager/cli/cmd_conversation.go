package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

const conversationControlTokenEnv = "AGENT_MANAGER_SEARCH_CONTROL_TOKEN"

func (a *App) cmdConversation(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		return nil
	}
	switch args[0] {
	case "search":
		return a.conversationSearch(args[1:])
	case "context":
		return a.conversationContext(args[1:])
	case "index":
		return a.conversationIndex(args[1:])
	default:
		return fmt.Errorf("unknown conversation subcommand %q; expected search, context, or index", args[0])
	}
}

type conversationSearchFlags struct {
	query             string
	mode              domainpb.ConversationSearchMode
	sort              domainpb.ConversationSearchSort
	filters           *domainpb.ConversationSearchFilters
	pageSize          int
	pageCursor        string
	jsonOutput        bool
	rawMode           string
	rawSort           string
	rawAfter          string
	rawBefore         string
	rawContentClasses string
}

func parseConversationSearchArgs(args []string) (conversationSearchFlags, error) {
	fs := flag.NewFlagSet("conversation search", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	mode := fs.String("mode", "hybrid", "Retrieval mode: hybrid, text, regex, or semantic")
	sortOrder := fs.String("sort", "relevance", "Sort: relevance, newest, or oldest")
	pageSize := fs.Int("page-size", 0, "Results per page (1-100; zero uses server default)")
	pageToken := fs.String("page-token", "", "Opaque next-page token")
	fs.StringVar(pageToken, "cursor", "", "Alias for --page-token")
	after := fs.String("occurred-after", "", "Only messages at or after RFC3339 time")
	fs.StringVar(after, "after", "", "Alias for --occurred-after")
	before := fs.String("occurred-before", "", "Only messages at or before RFC3339 time")
	fs.StringVar(before, "before", "", "Alias for --occurred-before")
	roles := fs.String("roles", "", "Comma-separated message roles")
	harnesses := fs.String("harnesses", "", "Comma-separated source harnesses")
	providerOrigins := fs.String("provider-origins", "", "Comma-separated provider origins")
	projects := fs.String("project-scopes", "", "Comma-separated project scopes")
	fs.StringVar(projects, "projects", "", "Alias for --project-scopes")
	cwds := fs.String("cwd-scopes", "", "Comma-separated working-directory scopes")
	fs.StringVar(cwds, "cwds", "", "Alias for --cwd-scopes")
	runners := fs.String("runners", "", "Comma-separated runners")
	models := fs.String("models", "", "Comma-separated models")
	profiles := fs.String("profiles", "", "Comma-separated profiles")
	statuses := fs.String("run-statuses", "", "Comma-separated run statuses")
	fs.StringVar(statuses, "statuses", "", "Alias for --run-statuses")
	tags := fs.String("tags", "", "Comma-separated tags")
	workloads := fs.String("workloads", "", "Comma-separated workloads")
	contentClasses := fs.String("content-classes", "", "Comma-separated content classes")
	includeTools := fs.Bool("include-tool-events", false, "Include tool calls and results")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return conversationSearchFlags{}, err
	}
	if fs.NArg() > 1 {
		return conversationSearchFlags{}, errorsUsage("conversation search accepts one quoted query")
	}
	query := ""
	if fs.NArg() == 1 {
		query = strings.TrimSpace(fs.Arg(0))
	}
	parsedMode, err := parseConversationSearchMode(*mode)
	if err != nil {
		return conversationSearchFlags{}, err
	}
	parsedSort, err := parseConversationSearchSort(*sortOrder)
	if err != nil {
		return conversationSearchFlags{}, err
	}
	if *pageSize < 0 || *pageSize > 100 {
		return conversationSearchFlags{}, errorsUsage("--page-size must be between 0 and 100")
	}
	if parsedMode == domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_REGEX && query != "" {
		if _, err := regexp.Compile(query); err != nil {
			return conversationSearchFlags{}, fmt.Errorf("invalid --mode regex query: %w", err)
		}
	}
	filters := &domainpb.ConversationSearchFilters{
		Roles: splitCSV(*roles), Harnesses: splitCSV(*harnesses), ProviderOrigins: splitCSV(*providerOrigins),
		ProjectScopes: splitCSV(*projects), CwdScopes: splitCSV(*cwds), Runners: splitCSV(*runners),
		Models: splitCSV(*models), Profiles: splitCSV(*profiles), RunStatuses: splitCSV(*statuses),
		Tags: splitCSV(*tags), Workloads: splitCSV(*workloads), IncludeToolEvents: *includeTools,
	}
	if strings.TrimSpace(*after) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*after))
		if err != nil {
			return conversationSearchFlags{}, fmt.Errorf("invalid --after (expected RFC3339): %w", err)
		}
		filters.OccurredAfter = timestamppb.New(parsed)
	}
	if strings.TrimSpace(*before) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*before))
		if err != nil {
			return conversationSearchFlags{}, fmt.Errorf("invalid --before (expected RFC3339): %w", err)
		}
		filters.OccurredBefore = timestamppb.New(parsed)
	}
	if filters.GetOccurredAfter() != nil && filters.GetOccurredBefore() != nil && filters.GetOccurredAfter().AsTime().After(filters.GetOccurredBefore().AsTime()) {
		return conversationSearchFlags{}, errorsUsage("--after must not be later than --before")
	}
	classes, err := parseConversationContentClasses(*contentClasses)
	if err != nil {
		return conversationSearchFlags{}, err
	}
	filters.ContentClasses = classes
	if query == "" && (parsedMode == domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_REGEX || parsedMode == domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_SEMANTIC) {
		return conversationSearchFlags{}, errorsUsage("regex and semantic modes require a query")
	}
	if query == "" && parsedSort == domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_RELEVANCE {
		return conversationSearchFlags{}, errorsUsage("an empty query requires --sort newest or --sort oldest")
	}
	if query == "" && !hasCLISearchFilter(filters) {
		return conversationSearchFlags{}, errorsUsage("an empty query requires at least one structured filter")
	}
	return conversationSearchFlags{query: query, mode: parsedMode, sort: parsedSort, filters: filters, pageSize: *pageSize, pageCursor: strings.TrimSpace(*pageToken), jsonOutput: *jsonOutput, rawMode: *mode, rawSort: *sortOrder, rawAfter: *after, rawBefore: *before, rawContentClasses: *contentClasses}, nil
}

func (a *App) conversationSearch(args []string) error {
	options, err := parseConversationSearchArgs(args)
	if err != nil {
		return err
	}
	response, err := a.services.Conversation.Search(&domainpb.SearchConversationsRequest{Query: options.query, Mode: options.mode, Sort: options.sort, Filters: options.filters, PageSize: int32(options.pageSize), PageCursor: options.pageCursor})
	if err != nil {
		return err
	}
	if options.jsonOutput {
		return printConversationJSON(response)
	}
	printConversationSearch(response)
	return nil
}

func (a *App) conversationContext(args []string) error {
	fs := flag.NewFlagSet("conversation context", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	before := fs.Uint("before", 2, "Context events before the hit (0-20)")
	after := fs.Uint("after", 3, "Context events after the hit (0-20)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 || strings.TrimSpace(fs.Arg(0)) == "" {
		return errorsUsage("usage: agent-manager conversation context <stable-hit-id> [--before n] [--after n] [--json]")
	}
	if *before > 20 || *after > 20 {
		return errorsUsage("--before and --after must be between 0 and 20")
	}
	response, err := a.services.Conversation.Context(&domainpb.GetConversationContextRequest{StableHitId: strings.TrimSpace(fs.Arg(0)), BeforeEvents: uint32(*before), AfterEvents: uint32(*after)})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return printConversationJSON(response)
	}
	if response.GetHit() != nil {
		printConversationRunHeader(response.GetHit())
	}
	for _, event := range response.GetEvents() {
		marker := " "
		if event.GetMatched() {
			marker = "*"
		}
		fmt.Printf("%s %s %-9s seq=%d %s\n", marker, formatTimestamp(event.GetOccurredAt()), safeTerminalText(event.GetRole()), event.GetEventSequence(), safeTerminalText(event.GetBoundedContent()))
	}
	if response.GetTruncated() {
		fmt.Println("Context was truncated by the server's privacy and size bounds.")
	}
	printConversationDegradations(response.GetDegradations())
	return nil
}

func (a *App) conversationIndex(args []string) error {
	if len(args) == 0 || args[0] == "help" {
		return nil
	}
	switch args[0] {
	case "status":
		return a.conversationIndexStatus(args[1:])
	case "reindex":
		return a.conversationReindex(args[1:])
	case "cancel":
		return a.conversationReindexCancel(args[1:])
	default:
		return fmt.Errorf("unknown conversation index subcommand %q; expected status, reindex, or cancel", args[0])
	}
}

func (a *App) conversationIndexStatus(args []string) error {
	fs := flag.NewFlagSet("conversation index status", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errorsUsage("conversation index status accepts no positional arguments")
	}
	response, err := a.services.Conversation.Status()
	if err != nil {
		return err
	}
	if *jsonOutput {
		return printConversationJSON(response)
	}
	coverage := response.GetCoverage()
	fmt.Printf("Conversation index: %s\n", formatEnumValue(response.GetState(), "CONVERSATION_INDEX_STATE_", "-"))
	fmt.Printf("  generation=%s candidate=%s collection=%s layout=%s model=%s\n", response.GetActiveGeneration(), response.GetCandidateGeneration(), response.GetCollectionName(), response.GetCollectionLayout(), response.GetEmbeddingModel())
	if coverage != nil {
		fmt.Printf("  documents canonical=%d catalog=%d lexical=%d semantic=%d pending=%d deleted=%d orphan=%d\n", coverage.GetCanonicalVisibleMessages(), coverage.GetCatalogDocuments(), coverage.GetLexicalDocuments(), coverage.GetSemanticDocuments(), coverage.GetPendingDocuments(), coverage.GetDeletedDocuments(), coverage.GetOrphanDocuments())
		fmt.Printf("  coverage lexical=%.1f%% semantic=%.1f%% freshness=%s\n", coverage.GetLexicalRatio()*100, coverage.GetSemanticRatio()*100, time.Duration(coverage.GetFreshnessLagMs())*time.Millisecond)
	}
	if response.GetLastErrorCode() != "" {
		fmt.Printf("  last_error=%s\n", response.GetLastErrorCode())
	}
	printConversationDegradations(response.GetDegradations())
	return nil
}

func (a *App) conversationReindex(args []string) error {
	fs := flag.NewFlagSet("conversation index reindex", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	dryRun := fs.Bool("dry-run", false, "Plan the rebuild without changing either active index")
	full := fs.Bool("full", true, "Reconcile the full authoritative corpus")
	maxDocuments := fs.Uint64("max-documents", 0, "Bound documents considered; zero means the server limit")
	idempotencyKey := fs.String("idempotency-key", "", "Reuse an existing rebuild operation for safe retries")
	controlToken := fs.String("control-token", strings.TrimSpace(os.Getenv(conversationControlTokenEnv)), "Control token (defaults to AGENT_MANAGER_SEARCH_CONTROL_TOKEN)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errorsUsage("conversation index reindex accepts no positional arguments")
	}
	if strings.TrimSpace(*controlToken) == "" {
		return errorsUsage("control token required via --control-token or AGENT_MANAGER_SEARCH_CONTROL_TOKEN")
	}
	var response *domainpb.ConversationReindexResponse
	var err error
	if *dryRun {
		response, err = a.services.Conversation.PlanReindex(&domainpb.PlanConversationReindexRequest{Full: *full, MaxDocuments: *maxDocuments, ControlToken: strings.TrimSpace(*controlToken)})
	} else {
		response, err = a.services.Conversation.Reindex(&domainpb.ReindexConversationsRequest{Full: *full, MaxDocuments: *maxDocuments, IdempotencyKey: strings.TrimSpace(*idempotencyKey), ControlToken: strings.TrimSpace(*controlToken)})
	}
	if err != nil {
		return err
	}
	if *jsonOutput {
		return printConversationJSON(response)
	}
	printConversationReindex(response)
	return nil
}

func (a *App) conversationReindexCancel(args []string) error {
	fs := flag.NewFlagSet("conversation index cancel", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	controlToken := fs.String("control-token", strings.TrimSpace(os.Getenv(conversationControlTokenEnv)), "Control token (defaults to AGENT_MANAGER_SEARCH_CONTROL_TOKEN)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 || strings.TrimSpace(*controlToken) == "" {
		return errorsUsage("usage: agent-manager conversation index cancel <operation-id> --control-token token")
	}
	response, err := a.services.Conversation.CancelReindex(&domainpb.CancelConversationReindexRequest{OperationId: strings.TrimSpace(fs.Arg(0)), ControlToken: strings.TrimSpace(*controlToken)})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return printConversationJSON(response)
	}
	printConversationReindex(response)
	return nil
}

func printConversationJSON(message proto.Message) error {
	data, err := protoMarshalOptions.Marshal(message)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(data)
	return nil
}

func printConversationSearch(response *domainpb.SearchConversationsResponse) {
	if len(response.GetHits()) == 0 {
		fmt.Println("No conversation matches.")
	}
	type runGroup struct {
		hits []*domainpb.ConversationSearchHit
	}
	groups := make([]runGroup, 0)
	groupByRun := make(map[string]int)
	for _, hit := range response.GetHits() {
		index, exists := groupByRun[hit.GetRunId()]
		if !exists {
			index = len(groups)
			groupByRun[hit.GetRunId()] = index
			groups = append(groups, runGroup{})
		}
		groups[index].hits = append(groups[index].hits, hit)
	}
	for _, group := range groups {
		printConversationRunHeader(group.hits[0])
		for _, hit := range group.hits {
			reason := conversationMatchReason(hit)
			fmt.Printf("  %s %-9s %s\n", formatTimestamp(hit.GetOccurredAt()), safeTerminalText(hit.GetRole()), highlightedConversationSnippet(hit))
			fmt.Printf("    hit=%s harness=%s session=%s match=%s link=%s\n", safeTerminalText(hit.GetStableHitId()), safeTerminalText(hit.GetProvenance().GetHarness()), safeTerminalText(hit.GetProvenance().GetSourceSessionId()), safeTerminalText(reason), safeTerminalText(hit.GetDeepLink()))
		}
	}
	coverage := response.GetCoverage()
	if coverage != nil {
		fmt.Printf("Coverage: lexical %.1f%%, semantic %.1f%%; freshness %s.\n", coverage.GetLexicalRatio()*100, coverage.GetSemanticRatio()*100, time.Duration(coverage.GetFreshnessLagMs())*time.Millisecond)
	}
	printConversationDegradations(response.GetDegradations())
	if response.GetNextPageCursor() != "" {
		fmt.Printf("Next page: rerun with --page-token %s\n", response.GetNextPageCursor())
	}
}

func printConversationRunHeader(hit *domainpb.ConversationSearchHit) {
	run := hit.GetRun()
	label := run.GetLabel()
	if label == "" {
		label = "(unlabelled run)"
	}
	fmt.Printf("\n%s  %s [%s] runner=%s model=%s\n", safeTerminalText(hit.GetRunId()), safeTerminalText(label), safeTerminalText(run.GetStatus()), safeTerminalText(run.GetRunner()), safeTerminalText(run.GetModel()))
}

func conversationMatchReason(hit *domainpb.ConversationSearchHit) string {
	if len(hit.GetRankEvidence()) == 0 {
		return "server-ranked"
	}
	parts := make([]string, 0, len(hit.GetRankEvidence()))
	for _, evidence := range hit.GetRankEvidence() {
		leg := formatEnumValue(evidence.GetLeg(), "CONVERSATION_SEARCH_LEG_", "-")
		part := fmt.Sprintf("%s#%d=%.4f", leg, evidence.GetRank(), evidence.GetScore())
		if evidence.GetExplanation() != "" {
			part += ":" + evidence.GetExplanation()
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, ",")
}

func highlightedConversationSnippet(hit *domainpb.ConversationSearchHit) string {
	runes := []rune(hit.GetSnippet())
	type span struct{ start, end int }
	spans := make([]span, 0, len(hit.GetHighlights()))
	for _, highlight := range hit.GetHighlights() {
		start, end := int(highlight.GetStartGrapheme()), int(highlight.GetEndGrapheme())
		if start >= 0 && end > start && start < len(runes) {
			if end > len(runes) {
				end = len(runes)
			}
			spans = append(spans, span{start: start, end: end})
		}
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start > spans[j].start })
	for _, item := range spans {
		runes = append(runes[:item.end], append([]rune("⟧"), runes[item.end:]...)...)
		runes = append(runes[:item.start], append([]rune("⟦"), runes[item.start:]...)...)
	}
	return safeTerminalText(string(runes))
}

func printConversationDegradations(degradations []*domainpb.ConversationSearchDegradation) {
	for _, degradation := range degradations {
		fmt.Printf("Degraded: %s leg=%s retryable=%t — %s\n", formatEnumValue(degradation.GetReason(), "CONVERSATION_SEARCH_DEGRADATION_REASON_", "-"), formatEnumValue(degradation.GetLeg(), "CONVERSATION_SEARCH_LEG_", "-"), degradation.GetRetryable(), safeTerminalText(degradation.GetDetail()))
	}
}

func printConversationReindex(response *domainpb.ConversationReindexResponse) {
	fmt.Printf("Reindex %s: %s dry_run=%t progress=%d/%d upserted=%d deleted=%d failed=%d\n", response.GetOperationId(), formatEnumValue(response.GetState(), "CONVERSATION_REINDEX_STATE_", "-"), response.GetDryRun(), response.GetProcessedDocuments(), response.GetPlannedDocuments(), response.GetUpsertedDocuments(), response.GetDeletedDocuments(), response.GetFailedDocuments())
	if response.GetState() == domainpb.ConversationReindexState_CONVERSATION_REINDEX_STATE_QUEUED || response.GetState() == domainpb.ConversationReindexState_CONVERSATION_REINDEX_STATE_RUNNING {
		fmt.Printf("Cancel: agent-manager conversation index cancel %s\n", response.GetOperationId())
	}
	printConversationDegradations(response.GetDegradations())
}

func parseConversationSearchMode(value string) (domainpb.ConversationSearchMode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "hybrid":
		return domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_HYBRID, nil
	case "text", "lexical":
		return domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_TEXT, nil
	case "regex":
		return domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_REGEX, nil
	case "semantic":
		return domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_SEMANTIC, nil
	default:
		return 0, fmt.Errorf("invalid --mode %q; expected hybrid, text, regex, or semantic", value)
	}
}

func parseConversationSearchSort(value string) (domainpb.ConversationSearchSort, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "relevance":
		return domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_RELEVANCE, nil
	case "newest":
		return domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_NEWEST, nil
	case "oldest":
		return domainpb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_OLDEST, nil
	default:
		return 0, fmt.Errorf("invalid --sort %q; expected relevance, newest, or oldest", value)
	}
}

func parseConversationContentClasses(value string) ([]domainpb.ConversationContentClass, error) {
	lookup := map[string]domainpb.ConversationContentClass{
		"prose": domainpb.ConversationContentClass_CONVERSATION_CONTENT_CLASS_PROSE, "quoted-prose": domainpb.ConversationContentClass_CONVERSATION_CONTENT_CLASS_QUOTED_PROSE,
		"tool-call": domainpb.ConversationContentClass_CONVERSATION_CONTENT_CLASS_TOOL_CALL, "tool-result": domainpb.ConversationContentClass_CONVERSATION_CONTENT_CLASS_TOOL_RESULT,
		"injected-context": domainpb.ConversationContentClass_CONVERSATION_CONTENT_CLASS_INJECTED_CONTEXT, "evidence-only-duplicate": domainpb.ConversationContentClass_CONVERSATION_CONTENT_CLASS_EVIDENCE_ONLY_DUPLICATE,
	}
	items := splitCSV(value)
	result := make([]domainpb.ConversationContentClass, 0, len(items))
	for _, item := range items {
		canonical := strings.ReplaceAll(strings.ToLower(item), "_", "-")
		class, ok := lookup[canonical]
		if !ok {
			return nil, fmt.Errorf("invalid --content-classes value %q", item)
		}
		result = append(result, class)
	}
	return result, nil
}

func splitCSV(value string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func hasCLISearchFilter(filters *domainpb.ConversationSearchFilters) bool {
	return len(filters.GetRoles())+len(filters.GetHarnesses())+len(filters.GetProviderOrigins())+len(filters.GetProjectScopes())+len(filters.GetCwdScopes())+len(filters.GetRunners())+len(filters.GetModels())+len(filters.GetProfiles())+len(filters.GetRunStatuses())+len(filters.GetTags())+len(filters.GetWorkloads())+len(filters.GetContentClasses()) > 0 || filters.GetOccurredAfter() != nil || filters.GetOccurredBefore() != nil || filters.GetIncludeToolEvents()
}

func errorsUsage(message string) error { return fmt.Errorf("invalid arguments: %s", message) }

func safeTerminalText(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
}
