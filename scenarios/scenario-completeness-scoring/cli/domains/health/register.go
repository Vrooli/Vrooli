package health

import (
	"net/http"
	"os"

	"scenario-completeness-scoring/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

const cliName = "scenario-completeness-scoring"

type collectorsResponse struct {
	Status     string                 `json:"status"`
	Collectors map[string]interface{} `json:"collectors"`
	Summary    map[string]int         `json:"summary"`
}

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Collector Health",
		Commands: []cliapp.Command{
			{Name: "collectors", NeedsAPI: true, Description: "Show collector health status", Run: func(args []string) error { return runCollectors(core, args) }},
			{Name: "circuit-breaker", NeedsAPI: true, Description: "View or reset the circuit breaker status", Run: func(args []string) error { return runCircuitBreaker(core, args) }},
		},
	}
}

func runCollectors(core *cliapp.ScenarioApp, args []string) error {
	fs, jsonOut, err := support.ParseFlags("collectors", args)
	if err != nil {
		return err
	}
	_ = fs

	body, err := core.Get("/health/collectors", nil)
	if err != nil {
		return err
	}

	var response collectorsResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			"Collector health: " + response.Status,
			"Total collectors: " + support.StringValue(response.Summary["total"]),
			"Healthy: " + support.StringValue(response.Summary["healthy"]),
			"Degraded: " + support.StringValue(response.Summary["degraded"]),
			"Failed: " + support.StringValue(response.Summary["failed"]),
		},
		ResultsHeading: "Collector Details",
		Results:        support.JSONLines(body),
		RetrievalHints: []string{
			cliName + " circuit-breaker",
			cliName + " status",
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCircuitBreaker(core *cliapp.ScenarioApp, args []string) error {
	fs, jsonOut, err := support.ParseFlags("circuit-breaker", args)
	if err != nil {
		return err
	}

	method := http.MethodGet
	path := "/health/circuit-breaker"
	summary := []string{"Circuit breaker status loaded"}
	if fs.NArg() > 0 && fs.Arg(0) == "reset" {
		method = http.MethodPost
		path = "/health/circuit-breaker/reset"
		summary = []string{"Circuit breakers reset"}
	}

	body, err := core.Request(method, path, nil, map[string]interface{}{})
	if method == http.MethodGet {
		body, err = core.Get(path, nil)
	}
	if err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: summary,
		Triage: []cliapp.TriageGroup{
			{
				Heading: "Breakers",
				Items:   support.JSONLines(body),
			},
		},
		NextSteps: []string{
			cliName + " collectors",
			cliName + " status",
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}
