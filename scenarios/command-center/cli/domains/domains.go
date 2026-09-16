package domains

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	walkv1 "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/walk"
	walkconnect "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/walk/walk_v1connect"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	integrationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/integrations/integrations_v1connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// CommandGroups aggregates flat command groups from domain packages.
//
// Keep app.go focused on CLI metadata and cli-core wiring. As the scenario
// grows, add domains like domains/tasks or domains/projects and append their
// registrations here. For greenfield scenarios, domain packages are the
// default architecture; do not treat flat command files as the long-term plan.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	integrations := integrationconnect.NewIntegrationsServiceClient(httpClient, baseURL)
	writeProtoJSON := func(msg proto.Message) error {
		body, err := protojson.Marshal(msg)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(append(body, '\n'))
		return err
	}
	read := func(path string) func([]string) error {
		return func(args []string) error {
			body, err := core.Get(path, nil)
			if err != nil {
				return err
			}
			if hasJSON(args) {
				_, err = os.Stdout.Write(body)
				if err == nil {
					_, err = os.Stdout.Write([]byte("\n"))
				}
				return err
			}
			fmt.Println(string(body))
			return nil
		}
	}
	room := func(args []string) error {
		if len(args) == 0 || strings.HasPrefix(args[0], "-") {
			return fmt.Errorf("room requires an id")
		}
		path := "/rooms/" + args[0]
		if len(args) > 1 && args[1] == "--samples" && len(args) > 2 {
			path += "?samples=" + args[2]
		}
		return read(path)(args)
	}
	integrationRefresh := func(args []string) error {
		resp, err := integrations.Refresh(context.Background(), connect.NewRequest(&commonv1.RefreshIntegrationsRequest{}))
		if err != nil {
			return cliapp.WrapAPIError("refresh integrations", err, nil)
		}
		if hasJSON(args) {
			return writeProtoJSON(resp.Msg)
		}
		fmt.Printf("Refreshed %d integration(s).\n", len(resp.Msg.GetIntegrations()))
		return nil
	}
	integrationList := func(args []string) error {
		resp, err := integrations.List(context.Background(), connect.NewRequest(&commonv1.ListIntegrationsRequest{}))
		if err != nil {
			return cliapp.WrapAPIError("list integrations", err, nil)
		}
		if hasJSON(args) {
			return writeProtoJSON(resp.Msg)
		}
		for _, integration := range resp.Msg.GetIntegrations() {
			status := "unknown"
			if integration.GetLifecycle() != nil {
				status = integration.GetLifecycle().GetStatus().String()
			}
			fmt.Printf("%s\t%s\t%s\n", integration.GetId(), status, integration.GetOrigin())
		}
		return nil
	}
	integrationAction := func(args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("integration-action requires an id and action")
		}
		confirmed := false
		for _, arg := range args[2:] {
			if arg == "--confirm" {
				confirmed = true
			}
		}
		if !confirmed {
			return fmt.Errorf("integration-action requires --confirm")
		}
		action, err := parseActionKind(args[1])
		if err != nil {
			return err
		}
		resp, err := integrations.RunAction(context.Background(), connect.NewRequest(&commonv1.RunIntegrationActionRequest{IntegrationId: args[0], Action: action, Confirmed: true}))
		if err != nil {
			return cliapp.WrapAPIError("run integration action", err, nil)
		}
		if hasJSON(args) {
			return writeProtoJSON(resp.Msg)
		}
		fmt.Printf("%s: %s\n", resp.Msg.GetStatus(), resp.Msg.GetMessage())
		return nil
	}
	promote := func(args []string) error {
		return runPromotion(core, args)
	}
	return []cliapp.CommandGroup{{Title: "Instrument reads", Commands: []cliapp.Command{{Name: "board", Description: "Show board shape", NeedsAPI: true, Run: read("/board")}, {Name: "room", Description: "Show one composed room", NeedsAPI: true, Run: room}, {Name: "focus", Description: "Show ranked findings", NeedsAPI: true, Run: read("/focus")}, {Name: "open-loop", Description: "Show dated open holes", NeedsAPI: true, Run: read("/open-loop")}, {Name: "describe", Description: "Describe the sensor space", NeedsAPI: true, Run: read("/capabilities/describe")}, {Name: "gaps", Description: "Show compatibility gaps", NeedsAPI: true, Run: read("/gaps")}, {Name: "integrations", Description: "Show lifecycle and feature state for declared integrations", NeedsAPI: true, Run: integrationList}, {Name: "integrations-refresh", Description: "Refresh declared integration state", NeedsAPI: true, Run: integrationRefresh}, {Name: "integration-action", Description: "Run a confirmed allowlisted integration action", NeedsAPI: true, Run: integrationAction}, {Name: "promote", Description: "Compare a Command Center reading with its producer and store promotion evidence", NeedsAPI: true, Run: promote}}}}
}

type promotionEvidence struct {
	MetricID       string          `json:"metric_id"`
	Window         string          `json:"window"`
	CapturedAt     time.Time       `json:"captured_at"`
	Reading        json.RawMessage `json:"reading"`
	Producer       json.RawMessage `json:"producer"`
	Verdict        string          `json:"verdict"`
	Differences    []string        `json:"differences,omitempty"`
	ProducerOrigin string          `json:"producer_origin"`
}

type promotionReading struct {
	ID         string          `json:"id"`
	Value      any             `json:"value"`
	Unit       string          `json:"unit"`
	ObservedAt string          `json:"observedAt"`
	Trust      string          `json:"trust"`
	Source     promotionSource `json:"source"`
}

type promotionSource struct {
	Read            string `json:"read"`
	Select          string `json:"select"`
	ExpectedUnit    string `json:"expectedUnit"`
	ContractVersion string `json:"contractVersion"`
}

func runPromotion(core *cliapp.ScenarioApp, args []string) error {
	if core == nil {
		return fmt.Errorf("promotion requires a CLI application")
	}
	metricID, window, err := promotionArgs(args)
	if err != nil {
		return err
	}
	readingBody, reading, err := readPromotionRoom(core, metricID, "ledger")
	if err != nil {
		// Promotion is a metric operation, not a Ledger-only operation. The
		// credit economy moved to Ledger, but LPBS still owns readings in
		// Broadcast; search that room when the requested metric is not in
		// Ledger so the same fail-closed command covers both room surfaces.
		readingBody, reading, err = readPromotionRoom(core, metricID, "broadcast")
	}
	if err != nil {
		return err
	}
	producerOrigin := strings.TrimRight(strings.TrimSpace(os.Getenv("COMMAND_CENTER_LPBS_PRODUCTION_URL")), "/")
	if producerOrigin == "" {
		producerOrigin = "https://vrooli.com"
	}
	token := strings.TrimSpace(os.Getenv("COMMAND_CENTER_LPBS_READER_TOKEN"))
	if token == "" {
		return writeUnavailablePromotionEvidence(metricID, window, readingBody, producerOrigin, fmt.Errorf("COMMAND_CENTER_LPBS_READER_TOKEN is required for production promotion"))
	}
	producerBody, err := readProducer(producerOrigin, reading.Source.Read, token, window)
	if err != nil {
		return writeUnavailablePromotionEvidence(metricID, window, readingBody, producerOrigin, err)
	}
	differences := comparePromotion(reading, producerBody)
	evidence := promotionEvidence{MetricID: metricID, Window: window, CapturedAt: time.Now().UTC(), Reading: readingBody, Producer: producerBody, ProducerOrigin: producerOrigin}
	if len(differences) == 0 && reading.Trust == "VALID" {
		evidence.Verdict = "match"
	} else {
		evidence.Verdict = "mismatch"
		if reading.Trust != "VALID" {
			differences = append(differences, "Command Center trust is "+reading.Trust+", want VALID")
		}
		evidence.Differences = differences
	}
	if err := writePromotionEvidence(evidence); err != nil {
		return err
	}
	encoded, _ := json.MarshalIndent(evidence, "", "  ")
	fmt.Println(string(encoded))
	if evidence.Verdict != "match" {
		return fmt.Errorf("promotion mismatch for %s: %s", metricID, strings.Join(differences, "; "))
	}
	return nil
}

func writeUnavailablePromotionEvidence(metricID, window string, readingBody []byte, producerOrigin string, cause error) error {
	reason := cause.Error()
	evidence := promotionEvidence{
		MetricID:       metricID,
		Window:         window,
		CapturedAt:     time.Now().UTC(),
		Reading:        readingBody,
		Producer:       json.RawMessage("null"),
		Verdict:        "unavailable",
		Differences:    []string{reason},
		ProducerOrigin: producerOrigin,
	}
	if err := writePromotionEvidence(evidence); err != nil {
		return err
	}
	return fmt.Errorf("promotion unavailable for %s: %s", metricID, reason)
}

func readPromotionRoom(core *cliapp.ScenarioApp, metricID, room string) ([]byte, promotionReading, error) {
	body, err := core.APIClient.Get("/rooms/"+room, url.Values{"samples": []string{"hide"}})
	if err != nil {
		return nil, promotionReading{}, fmt.Errorf("read Command Center %s: %w", room, err)
	}
	reading, err := findPromotionReading(body, metricID)
	if err != nil {
		return nil, promotionReading{}, err
	}
	return body, reading, nil
}

func promotionArgs(args []string) (string, string, error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", "", fmt.Errorf("promote requires a metric id")
	}
	window := "30d"
	for i := 1; i < len(args); i++ {
		if args[i] == "--window" && i+1 < len(args) {
			window = strings.TrimSpace(args[i+1])
			i++
		}
	}
	if window == "" {
		return "", "", fmt.Errorf("--window cannot be empty")
	}
	return args[0], window, nil
}

func findPromotionReading(body []byte, metricID string) (promotionReading, error) {
	var envelope struct {
		Metrics  []promotionReading `json:"metrics"`
		Readings []promotionReading `json:"readings"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return promotionReading{}, fmt.Errorf("decode Command Center ledger: %w", err)
	}
	readings := envelope.Readings
	if len(readings) == 0 {
		readings = envelope.Metrics
	}
	for _, reading := range readings {
		if reading.ID == metricID {
			return reading, nil
		}
	}
	return promotionReading{}, fmt.Errorf("ledger reading %q not found", metricID)
}

func readProducer(origin, readPath, token, window string) ([]byte, error) {
	parsed, err := url.Parse(origin + "/" + strings.TrimLeft(readPath, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse producer URL: %w", err)
	}
	if days := strings.TrimSuffix(strings.TrimSpace(window), "d"); days != "" {
		if _, err := strconv.Atoi(days); err == nil {
			query := parsed.Query()
			query.Set("window_days", days)
			parsed.RawQuery = query.Encode()
		}
	}
	method := http.MethodGet
	requestURL := parsed.String()
	var requestBody io.Reader
	if readPath == "/api/v1/admin/dashboard/revenue" {
		requestURL = strings.TrimRight(origin, "/") + "/landing_page_business_suite.v1.AdminRevenueService/GetRevenueSummary"
		method = http.MethodPost
		requestBody = strings.NewReader(`{}`)
	} else if strings.HasPrefix(parsed.Path, "/landing_page_business_suite.v1.") {
		method = http.MethodPost
		body := map[string]any{}
		query := parsed.Query()
		if dimension := query.Get("dimension"); dimension != "" {
			body["dimension"] = dimension
		}
		if limit := query.Get("limit"); limit != "" {
			if parsedLimit, parseErr := strconv.Atoi(limit); parseErr == nil {
				body["limit"] = parsedLimit
			}
		}
		if days := query.Get("window_days"); days != "" {
			if parsedDays, parseErr := strconv.Atoi(days); parseErr == nil {
				body["window_days"] = parsedDays
			}
		}
		encoded, encodeErr := json.Marshal(body)
		if encodeErr != nil {
			return nil, fmt.Errorf("encode producer request: %w", encodeErr)
		}
		requestBody = bytes.NewReader(encoded)
		requestURL = parsed.Scheme + "://" + parsed.Host + parsed.Path
	}
	req, err := http.NewRequest(method, requestURL, requestBody)
	if err != nil {
		return nil, fmt.Errorf("create producer request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("read producer %s: %w", parsed.Path, err)
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("read producer response: %w", readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("producer %s returned HTTP %d: %s", parsed.Path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return normalizeRevenueProducer(body), nil
}

func normalizeRevenueProducer(body []byte) []byte {
	var root map[string]any
	if json.Unmarshal(body, &root) != nil {
		return body
	}
	if !hasJSONKey(root, "mrr_minor") {
		return body
	}
	number := func(selector string) float64 {
		value, _ := jsonField(root, selector)
		parsed, _ := promotionNumber(value)
		return parsed
	}
	lookup := func(selector string) any {
		value, _ := jsonField(root, selector)
		return value
	}
	normalized := map[string]any{
		"observed_at":      lookup("observed_at"),
		"contract_version": "revenue-summary.v1",
		"units": map[string]string{
			"revenue_mrr": "currency", "revenue_today": "currency", "revenue_rollup": "currency",
			"subscriber_counts": "count", "churn": "count", "credit_balances": "count", "usage_operations_30d": "count",
			"ai_cost_30d": "currency", "credit_margin_30d": "currency", "revenue_by_line": "currency",
		},
		"revenue": map[string]any{
			"mrr": number("mrr_minor") / 100, "today": number("revenue_today_minor") / 100, "month": number("revenue_window_minor") / 100,
		},
		"subscriptions": map[string]any{
			"active": number("active_subscriptions"), "churned_30d": number("subscriptions_churned_window"),
		},
		"credits":         map[string]any{"balance_total": number("credit_balance_total"), "burned_window": number("credit_burned_window")},
		"cost":            map[string]any{"micros": number("cost_micros"), "usd": number("cost_micros") / 1_000_000},
		"revenue_by_line": lookup("revenue_by_line"),
		"usage":           map[string]any{"records": number("usage_records_window")},
		"currency":        lookup("currency"),
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return body
	}
	return encoded
}

func hasJSONKey(object map[string]any, wanted string) bool {
	wanted = normalizeJSONKey(wanted)
	for key := range object {
		if normalizeJSONKey(key) == wanted {
			return true
		}
	}
	return false
}

func comparePromotion(reading promotionReading, producer []byte) []string {
	var payload any
	if err := json.Unmarshal(producer, &payload); err != nil {
		return []string{"producer response is not JSON"}
	}
	differences := []string{}
	producerValue, valueOK := promotionValue(payload, reading.Source.Select)
	if !valueOK || !sameJSONValue(reading.Value, producerValue) {
		differences = append(differences, fmt.Sprintf("value differs: command-center=%v producer=%v", reading.Value, producerValue))
	}
	if unit, ok := promotionUnit(payload, reading.Source.Select); ok && reading.Unit != "" && unit != reading.Unit {
		differences = append(differences, fmt.Sprintf("unit differs: command-center=%s producer=%s", reading.Unit, unit))
	}
	if contract, ok := stringField(payload, "contract_version"); ok && reading.Source.ContractVersion != "" && contract != reading.Source.ContractVersion {
		differences = append(differences, fmt.Sprintf("contract version differs: command-center=%s producer=%s", reading.Source.ContractVersion, contract))
	}
	if observed, ok := stringField(payload, "observed_at"); ok && reading.ObservedAt != "" && observed != reading.ObservedAt {
		differences = append(differences, fmt.Sprintf("observation time differs: command-center=%s producer=%s", reading.ObservedAt, observed))
	}
	return differences
}

func promotionUnit(payload any, selector string) (string, bool) {
	if unit, ok := stringField(payload, "unit"); ok {
		return unit, true
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return "", false
	}
	units, ok := root["units"].(map[string]any)
	if !ok {
		return "", false
	}
	unit, ok := units[selector].(string)
	return unit, ok && strings.TrimSpace(unit) != ""
}

func promotionValue(payload any, selector string) (any, bool) {
	if value, ok := jsonField(payload, selector); ok {
		if numeric, numericOK := promotionNumber(value); numericOK {
			return numeric, true
		}
		if aggregate, aggregateOK := aggregatePromotionValue(selector, payload, value); aggregateOK {
			return aggregate, true
		}
	}
	if aggregate, aggregateOK := aggregatePromotionValue(selector, payload, nil); aggregateOK {
		return aggregate, true
	}
	return nil, false
}

func aggregatePromotionValue(selector string, payload, value any) (float64, bool) {
	if selector == "credit_margin_30d" {
		lines, linesOK := jsonField(payload, "revenue_by_line")
		cost, costOK := jsonPath(payload, "cost.usd")
		if !linesOK || !costOK {
			return 0, false
		}
		items, ok := lines.([]any)
		if !ok {
			return 0, false
		}
		creditRevenue := 0.0
		for _, item := range items {
			line, lineOK := item.(map[string]any)
			key, keyOK := line["key"].(string)
			amount, amountOK := line["amount_minor"].(float64)
			if !lineOK || !keyOK || !amountOK {
				return 0, false
			}
			if key == "credit_top_up" || key == "credits_topup" || key == "credits-topup" || key == "credits" {
				creditRevenue += amount
			}
		}
		costUSD, ok := promotionNumber(cost)
		return creditRevenue/100 - costUSD, ok
	}
	if selector == "revenue_by_line" {
		amount, ok := sumPromotionArray(value, "amount_minor")
		return amount / 100, ok
	}
	if selector == "digest_credit_app" || selector == "digest_credit_model" {
		credits, ok := sumPromotionArray(value, "credits")
		return credits, ok
	}
	if selector == "digest_apps" {
		downloads, ok := sumPromotionArray(value, "downloads")
		return downloads, ok
	}
	if selector == "digest_experiment" {
		return sumNestedPromotionArray(value, "paid", "successes")
	}
	if selector == "digest_funnel" {
		return sumPromotionArray(value, "value")
	}
	if strings.HasPrefix(selector, "traffic_") {
		root, ok := payload.(map[string]any)
		if ok {
			if total, totalOK := root["total_visitors"].(float64); totalOK {
				return total, true
			}
			if total, totalOK := root["totalVisitors"].(float64); totalOK {
				return total, true
			}
		}
		return sumPromotionArray(value, "visitors")
	}
	return sumPromotionArray(value, "value")
}

func sumPromotionArray(value any, field string) (float64, bool) {
	items, ok := value.([]any)
	if !ok {
		return 0, false
	}
	total := 0.0
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			return 0, false
		}
		number, ok := promotionNumber(object[field])
		if !ok {
			return 0, false
		}
		total += number
	}
	return total, true
}

func sumNestedPromotionArray(value any, parent, field string) (float64, bool) {
	items, ok := value.([]any)
	if !ok {
		return 0, false
	}
	total := 0.0
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			return 0, false
		}
		nested, ok := object[parent].(map[string]any)
		if !ok {
			return 0, false
		}
		number, ok := promotionNumber(nested[field])
		if !ok {
			return 0, false
		}
		total += number
	}
	return total, true
}

func promotionNumber(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func jsonField(value any, field string) (any, bool) {
	for _, path := range promotionFieldPaths(field) {
		if found, ok := jsonPath(value, path); ok {
			return found, true
		}
	}
	wanted := normalizeJSONKey(field)
	if object, ok := value.(map[string]any); ok {
		for key, candidate := range object {
			normalizedKey := normalizeJSONKey(key)
			if normalizedKey == wanted || normalizedKey == strings.TrimSuffix(wanted, "_minor") {
				return candidate, true
			}
		}
		for _, candidate := range object {
			if found, ok := jsonField(candidate, field); ok {
				return found, true
			}
		}
	}
	return nil, false
}

// promotionFieldPaths bridges Command Center's stable metric selectors to the
// nested producer envelope used by the revenue summary. The room selector is
// intentionally stable even when the producer groups fields by domain.
func promotionFieldPaths(field string) []string {
	paths := map[string][]string{
		"revenue_mrr":          {"revenue.mrr", "mrr"},
		"revenue_today":        {"revenue.today", "today"},
		"revenue_rollup":       {"revenue.month", "month"},
		"subscriber_counts":    {"subscriptions.active", "active"},
		"churn":                {"subscriptions.churned_30d", "churned_30d"},
		"credit_balances":      {"credits.balance_total", "balance_total"},
		"ai_cost_30d":          {"cost.usd", "cost"},
		"usage_operations_30d": {"usage.records", "records"},
		"digest_paid_subs":     {"growth.new_paid_subscriptions", "new_paid_subscriptions"},
		"digest_signups":       {"growth.signups", "signups"},
		"digest_waitlist":      {"growth.waitlist_joins", "waitlist_joins"},
		"digest_credits":       {"credits.credits_burned", "credits_burned"},
		"digest_purchased":     {"credits.credits_purchased", "credits_purchased"},
		"digest_operations":    {"credits.operations", "operations"},
		"digest_credit_app":    {"credits.by_app", "by_app"},
		"digest_credit_model":  {"credits.by_model", "by_model"},
		"digest_apps":          {"apps"},
		"digest_experiment":    {"experiment.arms", "arms"},
		"digest_funnel":        {"funnel.steps", "steps"},
	}
	if aliases, ok := paths[field]; ok {
		return aliases
	}
	return []string{field}
}

func jsonPath(value any, path string) (any, bool) {
	current := value
	for _, segment := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		wanted := normalizeJSONKey(segment)
		var found bool
		for key, candidate := range object {
			if normalizeJSONKey(key) == wanted {
				current, found = candidate, true
				break
			}
		}
		if !found {
			return nil, false
		}
	}
	return current, true
}

func normalizeJSONKey(key string) string {
	var normalized strings.Builder
	for i, r := range key {
		if r == '-' {
			normalized.WriteByte('_')
			continue
		}
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				normalized.WriteByte('_')
			}
			normalized.WriteRune(r + ('a' - 'A'))
			continue
		}
		normalized.WriteRune(r)
	}
	return strings.ToLower(normalized.String())
}

func stringField(value any, field string) (string, bool) {
	found, ok := jsonField(value, field)
	result, stringOK := found.(string)
	return result, ok && stringOK
}

func sameJSONValue(left, right any) bool {
	leftNumber, leftOK := left.(float64)
	rightNumber, rightOK := right.(float64)
	if leftOK && rightOK {
		return leftNumber == rightNumber
	}
	return fmt.Sprint(left) == fmt.Sprint(right)
}

func writePromotionEvidence(evidence promotionEvidence) error {
	dir := strings.TrimSpace(os.Getenv("COMMAND_CENTER_PROMOTION_EVIDENCE_DIR"))
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve promotion evidence directory: %w", err)
		}
		dir = filepath.Join(home, ".vrooli", "plan-artifacts", "ledger-room-true-ai-cost-capture-revenue-by-line-and-the", "promotion-evidence")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create promotion evidence directory: %w", err)
	}
	body, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return fmt.Errorf("encode promotion evidence: %w", err)
	}
	path := filepath.Join(dir, evidence.MetricID+".json")
	if err := os.WriteFile(path, append(body, '\n'), 0o644); err != nil {
		return fmt.Errorf("write promotion evidence: %w", err)
	}
	return nil
}

func parseActionKind(raw string) (commonv1.ActionKind, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "owner_guidance":
		return commonv1.ActionKind_ACTION_KIND_OWNER_GUIDANCE, nil
	case "scenario_start":
		return commonv1.ActionKind_ACTION_KIND_SCENARIO_START, nil
	case "scenario_restart":
		return commonv1.ActionKind_ACTION_KIND_SCENARIO_RESTART, nil
	case "operator_command":
		return commonv1.ActionKind_ACTION_KIND_OPERATOR_COMMAND, nil
	default:
		return commonv1.ActionKind_ACTION_KIND_UNSPECIFIED, fmt.Errorf("unsupported integration action %q", raw)
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
//
// Prefer domain packages as the default growth path:
//
//	cli/domains/tasks/register.go
//	cli/domains/projects/register.go
//
// For API-backed commands:
//   - set NeedsAPI: true so stale-check + --auto-start preflight works
//   - call core.Get(...) / core.Request(...) for versioned /api/v1 routes
//   - use cliapp.RenderOperationalReport / RenderListReport /
//     RenderMutationReport for default human output contracts
//   - use cliapp.PrintReportJSON(...) when a --json mode should mirror the
//     same structured report
func SubcommandGroups(core *cliapp.ScenarioApp, manifest []byte) []cliapp.SubcommandGroup {
	httpClient, base := cliapp.NewConnectHTTPClient(core)
	client := walkconnect.NewWalkServiceClient(httpClient, base)
	call := func(ctx cliapp.OperationContext) (*walkv1.ReadResponse, error) {
		limit, err := strconv.Atoi(ctx.Flag("limit"))
		if err != nil || limit < 1 || limit > 100 {
			return nil, fmt.Errorf("limit must be 1-100")
		}
		resp, err := client.Read(context.Background(), connect.NewRequest(&walkv1.ReadRequest{Limit: int32(limit)}))
		if err != nil {
			return nil, cliapp.WrapAPIError("read walk evidence", err, nil)
		}
		return resp.Msg, nil
	}
	report := func(_ cliapp.OperationContext, msg *walkv1.ReadResponse) cliapp.ListReport {
		lines := []string{}
		for _, r := range msg.GetReadings() {
			lines = append(lines, fmt.Sprintf("%s: coverage=%s trust=%s empirical=%s observed=%s — %s", r.GetId(), r.GetCoverage(), r.GetTrust(), r.GetEmpirical(), r.GetObservedAt(), r.GetReason()))
		}
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d of %d readings; truncated=%t", len(lines), msg.GetTotal(), msg.GetTruncated())}, Results: lines}
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, "walk", walkHandlers(client, call, report))
	if err != nil {
		panic(err)
	}
	return []cliapp.SubcommandGroup{group}
}

func hasJSON(args []string) bool {
	for _, a := range args {
		if a == "--json" {
			return true
		}
	}
	return false
}
