# Problems

## Open Follow-Ups

- Severity 3: The scenario still uses a lightweight static JavaScript UI rather than the preferred React + TypeScript + Vite stack. This should be handled as a dedicated UI modernization pass, not a polish pass.
- Severity 3: Requirement validations are now automated-test references, but many targets remain pending until implementation-specific tests are expanded and synced by the full scenario test suite.
- Severity 2: `scenario-auditor standards violations` currently returns cached findings for many scenarios, not only this one. Use scan summaries carefully and filter to `bookmark-intelligence-hub` paths during follow-up work.
- Severity 4: `vrooli scenario test bookmark-intelligence-hub all` fails before phase execution because the Go CLI module reports `go: updates to go.mod needed; to update it: go mod tidy`. Dependency metadata changes were left untouched because package/dependency updates require explicit permission.
- Severity 4: `scenario-auditor standards scan bookmark-intelligence-hub --wait` still reports 47 standards violations, including missing `@vrooli/api-base`, missing TypeScript/ESLint strict config, Go workspace-independence failures, and a reported required-layout Makefile issue even though `Makefile` exists.
- Severity 2: `vrooli scenario status bookmark-intelligence-hub` still reports unhealthy despite `/health` on both API and UI returning healthy JSON responses on the allocated ports. The lifecycle health cache or registry status path needs follow-up.
