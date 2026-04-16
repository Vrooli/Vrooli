package network

import (
	"fmt"
	"os"
	"strings"

	"network-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `network` subcommand group covering the scenario's
// network diagnostic endpoints under /api/v1/network. Each command is a thin
// wrapper: parse flags, call the API, render a standard report.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "network",
		Description: "Run network diagnostics and manage monitored targets",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "http", Description: "Perform an HTTP request", Run: func(args []string) error { return runHTTP(core, args) }},
			{Name: "dns", Description: "Resolve DNS records", Run: func(args []string) error { return runDNS(core, args) }},
			{Name: "ping", Description: "Test ICMP connectivity to a target", Run: func(args []string) error { return runConnectivity(core, args, "ping") }},
			{Name: "trace", Aliases: []string{"traceroute"}, Description: "Trace the route to a target", Run: func(args []string) error { return runConnectivity(core, args, "traceroute") }},
			{Name: "scan", Description: "Scan TCP ports on a target", Run: func(args []string) error { return runScan(core, args) }},
			{Name: "ssl", Description: "Validate a TLS/SSL certificate", Run: func(args []string) error { return runSSL(core, args) }},
			{Name: "targets", Description: "List monitored network targets", Run: func(args []string) error { return runTargets(core, args) }},
			{Name: "target-create", Description: "Register a new monitored target", Run: func(args []string) error { return runTargetCreate(core, args) }},
			{Name: "alerts", Description: "List active alerts", Run: func(args []string) error { return runAlerts(core, args) }},
		},
	}
}

// ---------- http ----------

func runHTTP(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("network http")
	method := fs.String("method", "GET", "HTTP method (GET, POST, PUT, DELETE, ...)")
	headers := fs.String("header", "", "Comma-separated header pairs, e.g. 'X-Foo:1,Authorization:Bearer x'")
	bodyStr := fs.String("body", "", "Raw request body string")
	bodyFile := fs.String("body-file", "", "Path to a JSON file to use as the full request payload (overrides other flags)")
	timeoutMs := fs.Int("timeout-ms", 0, "Request timeout in milliseconds (server default when 0)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: network http <url> [--method GET] [--header K:V,K:V] [--body STR] [--timeout-ms N] [--body-file PATH]")
		}
		req := map[string]interface{}{
			"url":    fs.Arg(0),
			"method": strings.ToUpper(strings.TrimSpace(*method)),
		}
		if hdrs := parseHeaderPairs(*headers); len(hdrs) > 0 {
			req["headers"] = hdrs
		}
		if strings.TrimSpace(*bodyStr) != "" {
			req["body"] = *bodyStr
		}
		if *timeoutMs > 0 {
			req["options"] = map[string]interface{}{"timeout_ms": *timeoutMs}
		}
		payload = req
	}

	body, err := core.Request("POST", "/network/http", nil, payload)
	if err != nil {
		return err
	}
	var resp support.HTTPResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Status: %d", resp.StatusCode),
		fmt.Sprintf("Response time: %dms", resp.ResponseTimeMs),
	}
	if resp.FinalURL != "" {
		results = append(results, "Final URL: "+resp.FinalURL)
	}
	if len(resp.RedirectChain) > 0 {
		results = append(results, "Redirects: "+strings.Join(resp.RedirectChain, " -> "))
	}
	if len(resp.Headers) > 0 {
		results = append(results, "--- Headers ---")
		results = append(results, support.MapRows(headersToAny(resp.Headers))...)
	}
	if strings.TrimSpace(resp.Body) != "" {
		results = append(results, "--- Body ---", resp.Body)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("HTTP request completed (status %d)", resp.StatusCode)},
		ResultsHeading: "Response",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s network http <url> --method POST --body-file request.json", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// ---------- dns ----------

func runDNS(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("network dns")
	recordType := fs.String("type", "A", "DNS record type (A, CNAME, MX, TXT)")
	dnsServer := fs.String("server", "", "Custom DNS server to query")
	bodyFile := fs.String("body-file", "", "Path to a JSON file to use as the full request payload (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: network dns <domain> [--type A|CNAME|MX|TXT] [--server IP] [--body-file PATH]")
		}
		payload = map[string]interface{}{
			"query":       fs.Arg(0),
			"record_type": strings.ToUpper(strings.TrimSpace(*recordType)),
			"dns_server":  strings.TrimSpace(*dnsServer),
		}
	}

	body, err := core.Request("POST", "/network/dns", nil, payload)
	if err != nil {
		return err
	}
	var resp support.DNSResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Answers))
	if len(resp.Answers) == 0 {
		results = append(results, "(no answers)")
	}
	for _, a := range resp.Answers {
		results = append(results, fmt.Sprintf("%s %s TTL=%d -> %s", a.Name, a.Type, a.TTL, a.Data))
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("DNS query: %s (%s)", resp.Query, resp.RecordType),
			fmt.Sprintf("Answers: %d in %dms", len(resp.Answers), resp.ResponseTimeMs),
		},
		ResultsHeading: "Records",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s network dns <domain> --type MX", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// ---------- connectivity (ping / traceroute) ----------

func runConnectivity(core *cliapp.ScenarioApp, args []string, testType string) error {
	fs := support.NewFlagSet("network " + testType)
	count := fs.Int("count", 0, "Packet count (ping only; 0 uses server default)")
	maxHops := fs.Int("max-hops", 0, "Maximum hops (traceroute only; 0 uses server default)")
	timeoutMs := fs.Int("timeout-ms", 0, "Per-packet timeout in milliseconds (0 uses server default)")
	bodyFile := fs.String("body-file", "", "Path to a JSON file to use as the full request payload (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: network %s <target> [--count N] [--max-hops N] [--timeout-ms N] [--body-file PATH]", testType)
		}
		options := map[string]interface{}{}
		if *count > 0 {
			options["count"] = *count
		}
		if *maxHops > 0 {
			options["max_hops"] = *maxHops
		}
		if *timeoutMs > 0 {
			options["timeout_ms"] = *timeoutMs
		}
		req := map[string]interface{}{
			"target":    fs.Arg(0),
			"test_type": testType,
		}
		if len(options) > 0 {
			req["options"] = options
		}
		payload = req
	}

	body, err := core.Request("POST", "/network/test/connectivity", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ConnectivityResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	stats := resp.Statistics
	results := []string{
		fmt.Sprintf("Target: %s (%s)", resp.Target, resp.TestType),
		fmt.Sprintf("Packets: sent=%d received=%d loss=%.1f%%", stats.PacketsSent, stats.PacketsReceived, stats.PacketLossPercent),
		fmt.Sprintf("RTT ms: min=%.2f avg=%.2f max=%.2f stddev=%.2f", stats.MinRTTMs, stats.AvgRTTMs, stats.MaxRTTMs, stats.StdDevRTTMs),
	}
	if len(resp.RouteHops) > 0 {
		results = append(results, "--- Route hops ---")
		for i, hop := range resp.RouteHops {
			results = append(results, fmt.Sprintf("%2d. %s", i+1, hop))
		}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s completed for %s", resp.TestType, resp.Target)},
		ResultsHeading: "Statistics",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s network ping <target> --count 10", support.CLIName),
			fmt.Sprintf("%s network trace <target> --max-hops 30", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// ---------- scan ----------

func runScan(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("network scan")
	ports := fs.String("ports", "", "Comma-separated port list (blank uses API defaults)")
	scanType := fs.String("type", "port", "Scan type (port, service, ...)")
	bodyFile := fs.String("body-file", "", "Path to a JSON file to use as the full request payload (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: network scan <target> [--ports 22,80,443] [--type port] [--body-file PATH]")
		}
		portList, err := support.ParsePorts(*ports)
		if err != nil {
			return err
		}
		req := map[string]interface{}{
			"target":    fs.Arg(0),
			"scan_type": strings.TrimSpace(*scanType),
		}
		if len(portList) > 0 {
			req["ports"] = portList
		}
		payload = req
	}

	body, err := core.Request("POST", "/network/scan", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ScanResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Results))
	if len(resp.Results) == 0 {
		results = append(results, "(no ports scanned)")
	}
	for _, p := range resp.Results {
		line := fmt.Sprintf("%d/%s: %s", p.Port, p.Protocol, p.State)
		if p.Service != "" {
			line += " (" + p.Service + ")"
		}
		results = append(results, line)
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scanned %s", resp.Target),
			fmt.Sprintf("Ports inspected: %d", len(resp.Results)),
		},
		ResultsHeading: "Ports",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s network scan <target> --ports 22,80,443", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// ---------- ssl ----------

func runSSL(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("network ssl")
	checkExpiry := fs.Bool("check-expiry", true, "Verify certificate validity window")
	checkChain := fs.Bool("check-chain", true, "Verify certificate chain")
	checkHostname := fs.Bool("check-hostname", true, "Verify hostname matches certificate")
	timeoutMs := fs.Int("timeout-ms", 0, "Connection timeout in milliseconds (0 uses server default)")
	bodyFile := fs.String("body-file", "", "Path to a JSON file to use as the full request payload (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: network ssl <url> [--check-expiry] [--check-chain] [--check-hostname] [--timeout-ms N] [--body-file PATH]")
		}
		options := map[string]interface{}{
			"check_expiry":   *checkExpiry,
			"check_chain":    *checkChain,
			"check_hostname": *checkHostname,
		}
		if *timeoutMs > 0 {
			options["timeout_ms"] = *timeoutMs
		}
		payload = map[string]interface{}{
			"url":     fs.Arg(0),
			"options": options,
		}
	}

	body, err := core.Request("POST", "/network/ssl/validate", nil, payload)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"SSL validation completed"},
		ResultsHeading: "Certificate report",
		Results:        support.MapRows(data),
		RetrievalHints: []string{
			fmt.Sprintf("%s network ssl https://example.com", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// ---------- targets ----------

func runTargets(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("network targets")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/network/targets", nil)
	if err != nil {
		return err
	}
	var targets []support.Target
	if err := support.Decode(body, &targets); err != nil {
		return err
	}

	results := make([]string, 0, len(targets))
	if len(targets) == 0 {
		results = append(results, "(no monitored targets)")
	}
	for _, t := range targets {
		line := fmt.Sprintf("%s (%s) | %s | %s", t.Name, support.ShortID(t.ID), t.TargetType, t.Address)
		if t.Port != nil {
			line += fmt.Sprintf(":%d", *t.Port)
		}
		if t.Protocol != "" {
			line += " [" + t.Protocol + "]"
		}
		results = append(results, line)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Monitored targets: %d", len(targets))},
		ResultsHeading: "Targets",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s network target-create --name NAME --address HOST --type host", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runTargetCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("network target-create")
	name := fs.String("name", "", "Target name (required)")
	address := fs.String("address", "", "Target address (required)")
	targetType := fs.String("type", "", "Target type (required, e.g. host|service|website)")
	port := fs.Int("port", 0, "Optional port")
	protocol := fs.String("protocol", "", "Optional protocol (tcp/udp/http/...)")
	tags := fs.String("tags", "", "Comma-separated tags")
	bodyFile := fs.String("body-file", "", "Path to a JSON file to use as the full request payload (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if strings.TrimSpace(*name) == "" || strings.TrimSpace(*address) == "" || strings.TrimSpace(*targetType) == "" {
			return fmt.Errorf("usage: network target-create --name NAME --address HOST --type TYPE [--port N] [--protocol tcp] [--tags a,b] [--body-file PATH]")
		}
		req := map[string]interface{}{
			"name":        strings.TrimSpace(*name),
			"address":     strings.TrimSpace(*address),
			"target_type": strings.TrimSpace(*targetType),
		}
		if *port > 0 {
			req["port"] = *port
		}
		if strings.TrimSpace(*protocol) != "" {
			req["protocol"] = strings.TrimSpace(*protocol)
		}
		if tagList := splitCSV(*tags); len(tagList) > 0 {
			req["tags"] = tagList
		}
		payload = req
	}

	body, err := core.Request("POST", "/network/targets", nil, payload)
	if err != nil {
		return err
	}
	var created map[string]interface{}
	if err := support.Decode(body, &created); err != nil {
		return err
	}
	id := ""
	if v, ok := created["id"].(string); ok {
		id = v
	}

	report := cliapp.MutationReport{
		Result:  []string{"Target created"},
		Changes: []string{fmt.Sprintf("New target ID: %s", id)},
		NextCommand: []string{
			fmt.Sprintf("%s network targets", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

// ---------- alerts ----------

func runAlerts(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("network alerts")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/network/alerts", nil)
	if err != nil {
		return err
	}
	var alerts []support.Alert
	if err := support.Decode(body, &alerts); err != nil {
		return err
	}

	results := make([]string, 0, len(alerts))
	if len(alerts) == 0 {
		results = append(results, "(no active alerts)")
	}
	for _, a := range alerts {
		ts := "unknown"
		if a.CreatedAt != nil {
			ts = support.FormatTimeValue(*a.CreatedAt)
		}
		line := fmt.Sprintf("[%s] %s | %s | target=%s | %s", strings.ToUpper(a.Severity), a.AlertType, a.Title, a.TargetName, ts)
		if a.Message != "" {
			line += " -> " + a.Message
		}
		results = append(results, line)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Active alerts: %d", len(alerts))},
		ResultsHeading: "Alerts",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s network targets", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// ---------- helpers ----------

func parseHeaderPairs(spec string) map[string]string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil
	}
	out := map[string]string{}
	for _, pair := range strings.Split(spec, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		idx := strings.Index(pair, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(pair[:idx])
		value := strings.TrimSpace(pair[idx+1:])
		if key == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func splitCSV(spec string) []string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil
	}
	parts := strings.Split(spec, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func headersToAny(headers map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(headers))
	for k, v := range headers {
		out[k] = v
	}
	return out
}
