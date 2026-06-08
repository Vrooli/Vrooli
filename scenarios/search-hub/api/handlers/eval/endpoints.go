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
	{
		ID:          "evals_sweep",
		Path:        evalconnect.EvalServiceSweepProcedure,
		Method:      "POST",
		Summary:     "Sweep a suite's tuning (overfit-safe) and optionally write back the winner",
		Description: "Runs the two-tier parameter sweep over the suite's provider — query-time factors full-factorial via per-request overrides, index-time factors by coordinate-ascent via config-push+reindex — stores one tagged run per arm, and promotes a winner only when it is statistically significant (bootstrap CI), held-out-validated, constraint-satisfying, and past the complexity tie-break. --apply gates the write-back; default previews the ranked table + recommendation.",
		Category:    "eval",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"suite_id": "string (required)", "query_time_only": "bool (skip the reindex-per-arm index-time tier)", "apply": "bool (persist the winner via config-write; default preview-only)", "limit": "int (per-case fetch depth)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"result": "SweepResult (ranked arms + winner_tag + promoted + recommendation + stats)"}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No such suite"},
			{Status: 400, Code: "failed_precondition", Description: "Provider unregistered / declares no control plane / suite has no positive cases"},
		},
		CLIMapping: &module.CLIMapping{Command: "search-hub evals sweep", Args: []string{"<suite_id>", "--apply"}},
	},
	{
		ID:          "evals_generate",
		Path:        evalconnect.EvalServiceGenerateProcedure,
		Method:      "POST",
		Summary:     "Generate golden cases for a suite by sampling + inverting the provider's index",
		Description: "Samples the provider's live index (stratified), inverts each sampled item to a natural-language query that should retrieve it (+ optional hard negatives), de-dupes against the existing corpus, and proposes the cases — each marked tags:[\"generated\"] so the sweep holds them out of tuning. Preview-only by default; --apply appends the proposals to the suite. Returns the resulting corpus's warn-level adequacy.",
		Category:    "eval",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"suite_id": "string (required)", "count": "int (target positive cases; 0 = default)", "negatives": "bool (also propose hard negatives)", "apply": "bool (append proposals to the suite; default preview-only)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"proposed": "array<GeneratedCase>", "suite": "EvalSuite (when applied)", "applied": "bool", "adequacy": "array<AdequacyWarning>", "summary": "string"}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No such suite"},
			{Status: 400, Code: "failed_precondition", Description: "Provider unregistered / index could not be sampled / proposals failed validation"},
		},
		CLIMapping: &module.CLIMapping{Command: "search-hub evals generate", Args: []string{"<suite_id>", "--count", "<n>", "--negatives", "--apply"}},
	},
}
