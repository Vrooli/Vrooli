package budgets

import (
	"database/sql"
	"log"

	internalbudgets "performance-health/internal/budgets"
	"performance-health/internal/module"
	"performance-health/internal/trend"

	"github.com/gorilla/mux"
	budgetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/budgets"
	budgetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/budgets/budgets_v1connect"
)

// ProtoFile is the FileDescriptor backing the Connect-mounted BudgetService.
var ProtoFile = budgetsv1.File_performance_health_v1_budgets_budgets_proto

// Module mounts the BudgetService backed by the declarative config store
// (.vrooli/perf-budgets.json under each scenario) and the trend store as the
// measurement source CheckBudget evaluates against — the baseline-diff gate.
func Module(logger *log.Logger, repoRoot string, db *sql.DB) module.Module {
	store := internalbudgets.NewConfigStore(repoRoot, nil)
	opts := []internalbudgets.Option{}
	if db != nil {
		opts = append(opts, internalbudgets.WithMeasurementSource(internalbudgets.NewTrendMeasurementSource(trend.NewStore(db))))
	}
	svc := internalbudgets.NewService(store, opts...)
	handler := NewHandler(svc, logger)
	path, connectHandler := budgetsconnect.NewBudgetServiceHandler(handler)
	return module.Module{
		Name: "budgets",
		Mount: func(r *mux.Router) {
			r.PathPrefix(path).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the empty schema: budgets are declarative in scenario config.
func Schema() string { return "" }

// Endpoints is the static endpoint metadata for codegen and the parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "budget_get_budget",
		Path:        budgetsconnect.BudgetServiceGetBudgetProcedure,
		Method:      "POST",
		Summary:     "Read a scenario's declared performance budget",
		Description: "Returns the declared performance budget for a scenario (defaults when none is declared).",
		Category:    "budgets",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"budget": "Budget", "declared": "bool"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario is missing"}},
	},
	{
		ID:          "budget_set_budget",
		Path:        budgetsconnect.BudgetServiceSetBudgetProcedure,
		Method:      "POST",
		Summary:     "Write a scenario's performance budget",
		Description: "Writes/updates a scenario's declared performance budget; honors X-Dry-Run for a no-write validation.",
		Category:    "budgets",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"budget": "Budget"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"budget": "Budget", "dry_run": "bool"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario is missing"}},
	},
	{
		ID:          "budget_check_budget",
		Path:        budgetsconnect.BudgetServiceCheckBudgetProcedure,
		Method:      "POST",
		Summary:     "Check a scenario's measurements against its budget",
		Description: "Evaluates a scenario's latest measurements against its budget and reports violations — the baseline-diff regression gate.",
		Category:    "budgets",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "passed": "bool", "violations": "array<BudgetViolation>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario is missing"}},
	},
}
