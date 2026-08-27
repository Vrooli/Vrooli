# Canonical Seam Rules

Use `.vrooli/canonical-seams.json` to declare one reviewed implementation for a job and the Go syntax that bypasses it. Tidiness Manager parses Go files with `go/ast`; it does not search source text. Comments and documentation examples do not produce findings.

Each seam defines:

- `id`, `canonical`, `why`, and `remediation` for stable operator evidence;
- a `bypass.kind` of `call` or `literal` and a Go regular-expression `pattern`;
- `scope.include` and `scope.exclude` path globs, relative to the scanned tree;
- a severity and a non-negative budget.

The scanner parses each applicable Go file once. It emits `BYPASSED_SEAM` for matches above the declared budget and reports the total above-budget count as `summary.bypassed_seams`.

The control-plane provider scans the bounded `internal/` validation target but resolves repository-level seam rules from its parent. Other scenario scans remain rooted at the requested scenario and do not inherit repository rules.

A literal pattern matches an unquoted string value or the source spelling of a numeric literal. To restrict a literal to a direct call argument, match `<resolved-call-name>:<value>`. For a duration multiplication, match `<resolved-selector-name>:<numeric-value>`. For example, `^t\.Setenv:(HOME|USER)$` distinguishes identity-fixture setup from ordinary test data that mentions `HOME`, while `^time\.Second:[0-9]+$` selects unnamed second literals.

Validate rule changes with:

```bash
go test ./... -run 'Test(ScanSeams|LoadSeams|RepositoryCanonical)' -count=1
```

Run this command from `scenarios/tidiness-manager/api`.
