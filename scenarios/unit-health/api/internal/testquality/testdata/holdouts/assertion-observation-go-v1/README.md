# Assertion-observation Go holdout v1

This holdout contains `num[sot]:30` Go tests from the selected scenarios:
app-monitor, prompt-injection-arena, and test-genie.

Labels were written before observations on 2026-09-09 by a single reviewer (`execution-agent-single-reviewer`) from the source bodies only. The single-reviewer limitation is retained explicitly; this partition is evidence for calibration, not an approval to change enforcement.

Selection used the first `num[sot]:10` stable `Test` declarations from each
chosen scenario inventory, with source identities retained in both JSON files.
`num[sot]:5` structurally permissive tests are labelled `weak_oracle`; the
remaining rows are labelled `behavioral` or `valid_exception` according to the
source review.

The observations were recorded after labels were frozen, using rule version 1 and the Go source profile. The promotion decision remains `requested`, and the catalog keeps the default enforcement advisory.
