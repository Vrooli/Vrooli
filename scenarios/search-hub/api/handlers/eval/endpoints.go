package eval

import (
	"search-hub/internal/module"

	evalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval/eval_v1connect"
)

// Endpoints is the machine-readable description of the eval module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so adding or renaming an RPC in eval.proto breaks this file at
// compile time. The global parity test (TestProtoConnectParity in
// api/internal/modules/registry_test.go) walks the proto FileDescriptor and
// asserts every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "evals_register_suite",
		Path:        evalconnect.EvalServiceRegisterSuiteProcedure,
		Method:      "POST",
		Summary:     "Register (upsert) an eval suite",
		Description: "Validates and persists a provider-owned golden suite (upsert keyed by suite_id). created=false when an existing suite was updated.",
		Category:    "eval",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"suite": "EvalSuite (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"suite": "EvalSuite", "created": "bool"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Suite failed validation"},
			{Status: 500, Code: "internal", Description: "Eval store write failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "search-hub evals register", Args: []string{"--suite", "<json>"}},
	},
	{
		ID:          "evals_list_suites",
		Path:        evalconnect.EvalServiceListSuitesProcedure,
		Method:      "POST",
		Summary:     "List eval suites",
		Description: "Returns registered eval suites, optionally filtered by provider_id.",
		Category:    "eval",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"provider_id": "string (optional filter)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"suites": "array<EvalSuite>"}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Eval store read failure"}},
		CLIMapping:  &module.CLIMapping{Command: "search-hub evals list"},
	},
	{
		ID:          "evals_get_suite",
		Path:        evalconnect.EvalServiceGetSuiteProcedure,
		Method:      "POST",
		Summary:     "Get an eval suite",
		Description: "Returns the suite with the given suite_id.",
		Category:    "eval",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"suite_id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"suite": "EvalSuite"}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No such suite"},
			{Status: 500, Code: "internal", Description: "Eval store read failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "search-hub evals show", Args: []string{"<suite_id>"}},
	},
	{
		ID:          "evals_run_suite",
		Path:        evalconnect.EvalServiceRunSuiteProcedure,
		Method:      "POST",
		Summary:     "Run an eval suite",
		Description: "Executes the suite against its provider's registered endpoint and stores an immutable, tagged run. Soft labels only — never a pass/fail gate.",
		Category:    "eval",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"suite_id": "string (required)", "tag": "string (experiment label)", "limit": "int (per-case fetch depth)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"run": "EvalRun"}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No such suite"},
			{Status: 400, Code: "failed_precondition", Description: "Suite's provider is not registered/reachable"},
			{Status: 500, Code: "internal", Description: "Eval store write failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "search-hub evals run", Args: []string{"<suite_id>", "--tag", "<tag>"}},
	},
	{
		ID:          "evals_list_runs",
		Path:        evalconnect.EvalServiceListRunsProcedure,
		Method:      "POST",
		Summary:     "List eval runs",
		Description: "Returns a suite's run history (newest first), optionally filtered by tag and capped by limit.",
		Category:    "eval",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"suite_id": "string (required)", "tag": "string (optional filter)", "limit": "int (optional cap)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"runs": "array<EvalRun>"}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Eval store read failure"}},
		CLIMapping:  &module.CLIMapping{Command: "search-hub evals runs", Args: []string{"<suite_id>"}},
	},
	{
		ID:          "evals_get_run",
		Path:        evalconnect.EvalServiceGetRunProcedure,
		Method:      "POST",
		Summary:     "Get an eval run",
		Description: "Returns the immutable run with the given run_id.",
		Category:    "eval",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"run_id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"run": "EvalRun"}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No such run"},
			{Status: 500, Code: "internal", Description: "Eval store read failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "search-hub evals show-run", Args: []string{"<run_id>"}},
	},
	{
		ID:          "evals_compare_runs",
		Path:        evalconnect.EvalServiceCompareRunsProcedure,
		Method:      "POST",
		Summary:     "Compare two eval runs",
		Description: "Returns both runs plus a per-case delta (outcome and score change) — the A/B experimentation surface.",
		Category:    "eval",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"run_a": "string (required)", "run_b": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"run_a": "EvalRun", "run_b": "EvalRun", "deltas": "array<CaseDelta>"}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No such run"},
			{Status: 500, Code: "internal", Description: "Eval store read failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "search-hub evals compare", Args: []string{"<run_a>", "<run_b>"}},
	},
}
