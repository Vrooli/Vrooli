// Responsibility: retain cmd runs declarations within their original package.
package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"sort"
	"strings"
)

func (a *App) runInvestigate(args []string) error {
	fs := flag.NewFlagSet("run investigate", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	runIDs := fs.String("run-ids", "", "Comma-separated run IDs to investigate (required)")
	filterJSON := fs.String("filter-json", "", "Retired: use --run-ids or investigation start")
	goalID := fs.String("goal-id", "", "Retired: use --run-ids or investigation start")
	customContext := fs.String("context", "", "Custom context for investigation")
	depth := fs.String("depth", "standard", "Investigation depth: quick, standard, deep")
	projectRoot := fs.String("project-root", "", "Retired: typed investigations are read-only")
	scopePaths := fs.String("scope-paths", "", "Retired: typed investigations are read-only")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if strings.TrimSpace(*runIDs) == "" {
		return fmt.Errorf("--run-ids is required; cohort and goal selectors are retired from this command")
	}
	if strings.TrimSpace(*filterJSON) != "" || strings.TrimSpace(*goalID) != "" {
		return fmt.Errorf("--filter-json and --goal-id are retired; resolve the bounded run set first and pass --run-ids")
	}
	if strings.TrimSpace(*projectRoot) != "" || strings.TrimSpace(*scopePaths) != "" {
		return fmt.Errorf("--project-root and --scope-paths are retired; typed investigations are bounded and read-only")
	}

	ids := make([]string, 0)
	seen := make(map[string]struct{})
	for _, rawID := range strings.Split(*runIDs, ",") {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return fmt.Errorf("--run-ids must contain at least one run ID")
	}
	sort.Strings(ids)

	maxTurns := int32(8)
	switch strings.ToLower(strings.TrimSpace(*depth)) {
	case "quick":
		maxTurns = 3
	case "standard", "":
		maxTurns = 8
	case "deep":
		maxTurns = 16
	default:
		return fmt.Errorf("--depth must be quick, standard, or deep")
	}
	question := strings.TrimSpace(*customContext)
	if question == "" {
		question = "Provide a bounded diagnosis of the selected agent runs, including supported findings and any unproven predicates."
	}
	requestKey, err := cliInvestigationRequestKey()
	if err != nil {
		return err
	}
	request := &apipb.InvestigationRequest{
		SchemaVersion:   "investigation-request/v1",
		RequestKey:      requestKey,
		CallerAuthority: "service",
		Subject: &apipb.InvestigationSubject{
			Owner:    "agent-manager",
			Kind:     "run-set",
			Ref:      "run-set:" + strings.Join(ids, ","),
			Revision: "current",
			RunIds:   ids,
		},
		Question: question,
		EvidencePolicy: &apipb.InvestigationEvidencePolicy{
			Mode:               "bounded_current",
			RequiredPlanes:     []string{"run_state", "events", "invocations"},
			OptionalPlanes:     []string{"receipts", "diff"},
			MaxEvents:          512,
			MaxEvidenceBytes:   262144,
			MaxReconciliations: 1,
		},
		Budget: &apipb.InvestigationBudget{
			MaxDelegatedRuns:  1,
			MaxTurns:          maxTurns,
			WallSeconds:       600,
			MaxChargeMicroUsd: 1000000,
		},
		RecommendationPolicy: &apipb.InvestigationRecommendationPolicy{
			AllowedKinds:         []string{"observe", "recommend_action"},
			AllowSubjectMutation: false,
		},
		Provenance: &apipb.InvestigationProvenance{Kind: "agent-manager-cli"},
	}

	body, response, err := a.services.Investigations.Start(&apipb.StartInvestigationRequest{Request: request})
	if err != nil {
		return err
	}

	if *jsonOutput || response.GetInvestigation() == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	item := response.GetInvestigation()
	fmt.Printf("Started typed investigation: %s status=%s reused=%t workflow=%s\n", item.GetInvestigationId(), item.GetOperationStatus(), response.GetReused(), item.GetWorkflowRef())
	return nil
}

func cliInvestigationRequestKey() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate investigation request key: %w", err)
	}
	return fmt.Sprintf("agent-manager-cli/%x", bytes), nil
}
