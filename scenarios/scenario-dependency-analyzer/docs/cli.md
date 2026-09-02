# Scenario Dependency Analyzer - CLI Reference

Complete command-line interface reference for analyzing and visualizing scenario dependencies within Vrooli.

## Table of Contents
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
  - [analyze](#analyze)
  - [scan](#scan)
  - [dag](#dag)
  - [graph](#graph)
  - [cycles](#cycles)
  - [impact](#impact)
  - [propose](#propose)
  - [optimize](#optimize)
  - [list](#list)
  - [deployment](#deployment)
  - [health](#health)
  - [deps approved](#deps-approved)
  - [status](#status)
- [Examples](#examples)
- [Output Formats](#output-formats)
- [Integration](#integration)

---

## Installation

The CLI is automatically installed when you set up the scenario:

```bash
cd scenarios/scenario-dependency-analyzer
vrooli scenario setup scenario-dependency-analyzer
```

The `scenario-dependency-analyzer` command will be available at:
`~/.vrooli/bin/scenario-dependency-analyzer`

Verify installation:
```bash
scenario-dependency-analyzer --version
```

---

## Quick Start

```bash
# Check service status
scenario-dependency-analyzer status

# Analyze a scenario's dependencies
scenario-dependency-analyzer analyze swarm-manager

# Scan and detect dependencies from code
scenario-dependency-analyzer scan swarm-manager

# Export recursive dependency DAG
scenario-dependency-analyzer dag export swarm-manager --recursive

# Generate dependency graph
scenario-dependency-analyzer graph combined --format json

# Generate the actual import-evidence interface graph
scenario-dependency-analyzer graph actual --json

# Report declared-vs-actual drift
scenario-dependency-analyzer drift swarm-manager --json

# Validate dependency health through the Test Genie producer contract
scenario-dependency-analyzer health swarm-manager --json

# Search approved dependency governance memory
scenario-dependency-analyzer deps approved search "React graph library" --json

# Check deployment readiness
scenario-dependency-analyzer deployment swarm-manager

# Compatibility view of the control-plane supervision set
scenario-dependency-analyzer core-set --json
```

The `core-set` verb is retained for Scenario Dependency Analyzer consumers. It
uses the same database-free service as `vrooli supervision-set`; it does not
own a separate seed or dependency list. New platform consumers should prefer
`vrooli supervision-set --json` or the in-process supervision service.

---

## Commands

### analyze

Analyze dependencies for a specific scenario or all scenarios.

**Usage:**
```bash
scenario-dependency-analyzer analyze <scenario> [OPTIONS]
```

**Arguments:**
- `scenario` - Scenario name (or "all" for system-wide analysis)

**Options:**
- `--transitive` - Include transitive dependencies
- `--json` - Output in JSON format
- `--output <format>` - Alias for format selection (supports: json)
- `--verbose` - Show detailed analysis

**Examples:**
```bash
# Analyze a single scenario
scenario-dependency-analyzer analyze swarm-manager

# Analyze with transitive dependencies
scenario-dependency-analyzer analyze api-manager --transitive

# Analyze all scenarios with JSON output
scenario-dependency-analyzer analyze all --json

# Verbose analysis
scenario-dependency-analyzer analyze chart-generator --verbose
```

**Output:**
```
🔍 Analyzing dependencies for: swarm-manager
✅ Analysis complete

📊 Dependency Summary:
   Resources: 5
   Scenarios: 3
   Shared Workflows: 0
```

---

### scan

Run dependency scan and optionally apply detected dependencies to service.json.

**Usage:**
```bash
scenario-dependency-analyzer scan <scenario> [OPTIONS]
```

**Arguments:**
- `scenario` - Scenario name to scan

**Options:**
- `--apply` - Apply inferred resources and scenarios
- `--apply-resources` - Apply only inferred resources
- `--apply-scenarios` - Apply only inferred scenarios
- `--json` - Raw JSON output

**Examples:**
```bash
# Scan without applying changes
scenario-dependency-analyzer scan swarm-manager

# Scan and apply all detected dependencies
scenario-dependency-analyzer scan api-tools --apply

# Scan and apply only resources
scenario-dependency-analyzer scan chart-gen --apply-resources

# Get JSON output
scenario-dependency-analyzer scan my-scenario --json
```

**Output:**
```
🛰️  Scanning scenario: swarm-manager
✅ Scan complete
  Applied changes: true
  Resources added: 2
  Scenarios added: 1
  Tip: run 'git diff scenarios/swarm-manager/.vrooli/service.json' to review
```

---

### dag

Export recursive dependency DAG (Directed Acyclic Graph) for deployment orchestration.

**Usage:**
```bash
scenario-dependency-analyzer dag export <scenario> [OPTIONS]
```

**Subcommands:**
- `export` - Export recursive dependency DAG

**Arguments:**
- `scenario` - Scenario name

**Options:**
- `--recursive` - Include full recursive tree (default: true)
- `--no-recursive` - Only top-level dependencies
- `--json` - Output raw JSON
- `--output <file>` - Save to file

**Examples:**
```bash
# Export full recursive DAG
scenario-dependency-analyzer dag export swarm-manager

# Export only top-level dependencies
scenario-dependency-analyzer dag export api-manager --no-recursive

# Save to file
scenario-dependency-analyzer dag export my-app --output deps.json

# Verbose JSON output
scenario-dependency-analyzer dag export chart-gen --json
```

**Output:**
```
📊 Exporting DAG for: swarm-manager (recursive=true)
✅ DAG exported

Dependency Tree:
  - resource: postgres (2 children)
  - resource: ollama
  - scenario: data-tools (3 children)
  - scenario: browser-automation-studio

⚠️  Metadata Gaps Detected:
  • Add deployment.tiers metadata to data-tools
  • Add deployment.dependencies metadata to postgres
```

---

### graph

Generate dependency graph visualization in various formats.

**Usage:**
```bash
scenario-dependency-analyzer graph [type] [OPTIONS]
```

**Arguments:**
- `type` - Graph type: resource, scenario, or combined (default: combined)

**Options:**
- `--type <type>` - Alternative way to specify graph type
- `--format <format>` - Output format: json, dot, mermaid (default: json)
- `--output <file>` - Save output to file
- `--json` - Force JSON output
- `--actual` - Legacy flag for the Connect-backed actual interface graph; prefer `graph actual --json` for new automation

**Examples:**
```bash
# Generate combined graph (default)
scenario-dependency-analyzer graph

# Generate resource-only graph
scenario-dependency-analyzer graph resource

# Export to DOT format for Graphviz
scenario-dependency-analyzer graph combined --format dot --output deps.dot

# Generate Mermaid diagram
scenario-dependency-analyzer graph scenario --format mermaid

# Save JSON to file
scenario-dependency-analyzer graph --type combined --output graph.json

# Rank scenario centrality for swarm-manager scheduling inputs
scenario-dependency-analyzer graph centrality

# Read one scenario's centrality as JSON
scenario-dependency-analyzer graph centrality --scenario test-genie --json
```

**Output (DOT format):**
```dot
digraph Dependencies {
  rankdir=LR;
  node [shape=box];
  postgres [label="postgres"];
  swarm-manager [label="swarm-manager"];
  swarm-manager -> postgres [label="requires"];
}
```

**Visualize DOT:**
```bash
# Generate PNG from DOT file
dot -Tpng deps.dot -o graph.png

# View in browser (macOS)
dot -Tsvg deps.dot | open -f -a Safari
```

**Centrality output:**
`graph centrality` reports each scenario's direct and transitive reverse
dependency counts, required-edge weighted score, and distance to the nearest
core seed. Swarm Manager consumes this as one input to derived scenario
importance.

**Actual graph output:**
`graph actual --json` reports nodes, evidence-tagged scenario edges, transport world, and stability metadata. It is backed by `DescribeInterfaceGraph` and is the preferred machine interface for future planners.

---

### drift

Report declared-vs-actual scenario dependency drift.

**Usage:**
```bash
scenario-dependency-analyzer drift [scenario] [OPTIONS]
```

**Arguments:**
- `scenario` - Optional scenario filter; omit for fleet-wide drift

**Options:**
- `--json` - Emit machine-readable findings

**Semantics:**
- `undeclared_but_used` is a warning because import evidence proves an actual scenario edge.
- `declared_without_import_evidence` is informational because runtime URL discovery and CLI shell-out usage are not yet represented in upstream AST facts.

---

### cycles

Detect circular dependencies in the dependency graph.

**Usage:**
```bash
scenario-dependency-analyzer cycles [OPTIONS]
```

**Options:**
- `--type <type>` - Graph type: resource, scenario, combined (default: combined)
- `--json` - Output in JSON format

**Examples:**
```bash
# Check for cycles in combined graph
scenario-dependency-analyzer cycles

# Check scenario dependencies only
scenario-dependency-analyzer cycles --type scenario

# Get JSON report
scenario-dependency-analyzer cycles --json
```

**Output (no cycles):**
```
🔍 Detecting circular dependencies in combined graph...
✅ No circular dependencies detected in graph
```

**Output (cycles found):**
```
🔍 Detecting circular dependencies in combined graph...
🔴 2 circular dependencies detected in dependency graph

Detected Cycles:
  • Circular dependency: scenario-a → scenario-b → scenario-a
    Type: scenario_cycle | Length: 2 hops
  • Circular dependency: resource-x → resource-y → resource-x
    Type: resource_cycle | Length: 2 hops | ⚠️  ALL REQUIRED

Affected Dependencies:
  - scenario-a
  - scenario-b
  - resource-x
  - resource-y
```

---

### impact

Analyze the impact of removing a dependency from the system.

**Usage:**
```bash
scenario-dependency-analyzer impact <dependency> [OPTIONS]
```

**Arguments:**
- `dependency` - Dependency name (resource or scenario)

**Options:**
- `--json` - Raw JSON output

**Examples:**
```bash
# Analyze impact of removing postgres
scenario-dependency-analyzer impact postgres

# Analyze scenario dependency impact
scenario-dependency-analyzer impact swarm-manager --json
```

**Output:**
```
🔍 Analyzing impact of removing: postgres
🔴 CRITICAL IMPACT

Removing postgres would break 12 scenarios and affect 25 indirect dependents.

Direct Dependents:
  - swarm-manager (REQUIRED)
    Purpose: Store dependency metadata and analysis results
  - api-manager (REQUIRED)
    Purpose: Main data storage for API configurations

Indirect Dependents (15):
  - chart-generator
  - data-tools
  - browser-automation-studio
  ...

Recommendations:
  • Consider migration plan to alternative database
  • Update 12 scenarios to use alternative storage
  • Test all affected scenarios before removal
```

---

### propose

Analyze dependencies for a proposed scenario before building it.

**Usage:**
```bash
scenario-dependency-analyzer propose [OPTIONS]
```

**Options:**
- `--name <name>` - Proposed scenario name (required)
- `--description <desc>` - Scenario description
- `--requirements <list>` - Comma-separated requirements (required)
- `--similar <scenarios>` - Similar existing scenarios
- `--json` - Output in JSON format

**Examples:**
```bash
# Propose a new AI chatbot scenario
scenario-dependency-analyzer propose \
  --name "ai-chatbot" \
  --requirements "nlp,database,api" \
  --description "AI-powered chat assistant"

# Propose with similar scenarios
scenario-dependency-analyzer propose \
  --name "task-scheduler" \
  --requirements "cron,database" \
  --similar "swarm-manager,system-monitor"

# Get JSON output
scenario-dependency-analyzer propose \
  --name "test-scenario" \
  --requirements "postgres" \
  --json
```

**Output:**
```
🔮 Analyzing proposed scenario: ai-chatbot
✅ Analysis complete

📋 Recommended Dependencies:
Resources:
  - ollama
  - postgres
  - redis

Related Scenarios:
  - swarm-manager
  - api-manager
```

---

### optimize

Get optimization recommendations for reducing dependencies or improving deployment.

**Usage:**
```bash
scenario-dependency-analyzer optimize [scenario] [OPTIONS]
```

**Arguments:**
- `scenario` - Scenario to optimize (or "all" for system-wide)

**Options:**
- `--type <type>` - Optimization type: resource, deployment, cost, all
- `--apply` - Apply safe optimizations automatically
- `--json` - Output in JSON format

**Examples:**
```bash
# Get optimization recommendations
scenario-dependency-analyzer optimize swarm-manager

# Optimize all scenarios
scenario-dependency-analyzer optimize all --type resource

# Apply safe optimizations
scenario-dependency-analyzer optimize --apply

# Get JSON report
scenario-dependency-analyzer optimize chart-gen --json
```

**Output:**
```
🔧 Getting optimization recommendations for: swarm-manager

Scenario: swarm-manager
  Recommendations: 3
  High priority: 1
    - [resource_swap] Consider lightweight AI alternative
      Description: Replace ollama with openrouter for lower resource usage
      Confidence: 0.85
      Suggested action: review
    - [dependency_reduction] Remove unused resource
      Description: redis is declared but never used in code
      Confidence: 0.92
      Suggested action: remove
```

---

### list

List dependencies for a specific scenario.

**Usage:**
```bash
scenario-dependency-analyzer list <scenario> [OPTIONS]
```

**Arguments:**
- `scenario` - Scenario name

**Options:**
- `--type <type>` - Filter by type: resources, scenarios, workflows
- `--json` - Output in JSON format

**Examples:**
```bash
# List all dependencies
scenario-dependency-analyzer list swarm-manager

# List only resources
scenario-dependency-analyzer list api-manager --type resources

# Get JSON output
scenario-dependency-analyzer list chart-gen --json
```

**Output:**
```
📋 Fetching dependencies for: swarm-manager
✅ Dependencies for swarm-manager:

resource:
  - postgres (required)
  - ollama (required)
  - qdrant
  - redis

scenario:
  - data-tools (required)
  - browser-automation-studio
```

---

### deployment

Show deployment readiness, tier fitness, and bundle metadata for a scenario.

**Usage:**
```bash
scenario-dependency-analyzer deployment <scenario> [OPTIONS]
```

**Arguments:**
- `scenario` - Scenario name

**Options:**
- `--json` - Raw JSON output

**Examples:**
```bash
# Check deployment readiness
scenario-dependency-analyzer deployment swarm-manager

# Get JSON report
scenario-dependency-analyzer deployment api-tools --json
```

**Output:**
```
🛰️  Loading deployment report for: swarm-manager
Scenario: swarm-manager
Generated: 2025-11-22T17:30:00Z

Tier readiness:
  - desktop: fitness 85%, dependencies: 5, blockers: none
  - server: fitness 95%, dependencies: 5, blockers: none
  - mobile: fitness 45%, dependencies: 5, blockers: ollama, postgres
  - saas: fitness 60%, dependencies: 5, blockers: local-file-access

Bundle dependencies:
  - resource :: postgres (tiers: desktop, server, saas)
  - resource :: ollama (tiers: desktop, server)
  - resource :: qdrant (tiers: desktop, server, saas)
  - scenario :: data-tools (tiers: desktop, server, saas, mobile)

Bundle files:
  - binary: api/swarm-manager-api (present)
  - config: .vrooli/service.json (present)
  - schema: api/internal/<domain>/schema.sql (present)
```

---

### health

Validate dependency health for one scenario through SDA's producer contract.
This is the public surface Test Genie will consume for its dependencies phase.
The current response includes Code Facts-backed surfaces, dependency readiness,
runtime dependency status for required resources/scenarios, approved dependency
governance, pnpm release-age policy validation, graph drift, and a stable
security-health dependency-index availability section.

**Usage:**
```bash
scenario-dependency-analyzer health <scenario> [OPTIONS]
```

**Arguments:**
- `scenario` - Scenario name

**Options:**
- `--json` - Output raw JSON
- `--use-cache` - Allow cached upstream facts (default: true)

**Examples:**
```bash
# Human-readable dependency health summary
scenario-dependency-analyzer health swarm-manager

# Get machine-readable producer output
scenario-dependency-analyzer health swarm-manager --json
```

**Output:**
```
Scenario: swarm-manager
Passed: false
Findings: 0
Degraded integrations: 0

Dependency Health Sections
pass: Code Facts surfaces - 3 Code Facts surface(s) discovered.
pass: Dependency readiness - Host commands, runtimes, modules, and packages passed readiness checks (6 command probe(s)).
pass: Runtime dependencies - 2 required resource(s), 1 required scenario dependency(ies) checked.
not_configured: Approved dependency governance - Approved dependency registry is present but has no records yet; observed dependencies are reported as needs-review guidance, not allowlist failures.
pass: Package release-age policy - 1 pnpm-managed dependency surface(s) checked for minimumReleaseAge >= 10080 minutes.
pass: Security Health dependency index - Security Health dependency index available=true ready=true indexed=47867 vulnerable=1041.
pass: Dependency graph drift - Declared scenario dependencies match import evidence.
```

Release-age policy uses pnpm's `minimumReleaseAge` setting. Vrooli's default is
10080 minutes. `minimumReleaseAgeExclude` entries are allowed only when the
exception is recorded in `.vrooli/dependencies/approved-dependencies.json` with
rationale and review expiry.

The `security-index` section calls `security-health deps status --json` to
report dependency-index availability and freshness. It may include aggregate
index counts, but it does not emit vulnerable-package findings into dependency
health. Vulnerability scanning and security-phase gating remain owned by
Security Health; SDA governance commands consume Security Health vulnerability
evidence only for approval, denial, and remediation decisions.

Dependency freshness belongs to SDA. `vrooli hygiene` may aggregate and render
dependency freshness, but root hygiene must call an SDA-owned provider instead
of reimplementing package topology, touched-file selection, or Go module drift
checks.

Per-scenario freshness uses Code Facts-discovered dependency surfaces and falls
back to conventional `api`, `cli`, and `ui` roots only when Code Facts is
unavailable. Fleet/touched freshness should use the same health findings after
mapping git-touched files to in-repo package/module roots and impacted scenario
surfaces. Root `go.mod` and `go.sum` changes affect all applicable Go surfaces.

Readiness findings stay on stable maturity rules: Go module presence, local
replace resolution, tidy drift, optional build freshness, Node package metadata,
lockfiles, and install state map to `package_readiness`; discovery and
classification failures map to `surface_inventory`. Any new emitted rule ID
must be registered in `.vrooli/maturity.json` before it is used.

Fixes use preview/apply semantics. SDA may safely apply deterministic Go fixes
for a specific surface, such as local replace reconciliation or `go mod tidy`.
Ambiguous mappings, unsupported ecosystems, optional build failures, and
third-party dependency governance changes are advisory until a typed fixer owns
them.

---

### freshness

Report fleet/touched package freshness for Go scenario surfaces. This is the
SDA-owned provider surface that root `vrooli hygiene` aggregates for dependency
freshness; root hygiene must not maintain its own shared-package trigger list.

**Usage:**
```bash
scenario-dependency-analyzer freshness [--touched|--all] [OPTIONS]
```

**Options:**
- `--touched` - Check only surfaces impacted by changed in-repo modules (default)
- `--all` - Check every discovered Go scenario surface
- `--apply` - Run `go mod tidy` on stale surfaces
- `--build` - Run `go build ./...` after tidy checks
- `--concurrency <n>` - Maximum package surfaces to check concurrently (default: `8`)
- `--repo-root <path>` - Repository root (defaults to `git rev-parse --show-toplevel`)
- `--json` - Output raw JSON

**Examples:**
```bash
# Machine-readable touched-surface report for hygiene aggregation
scenario-dependency-analyzer freshness --touched --json

# Repair deterministic tidy drift on impacted Go surfaces
scenario-dependency-analyzer freshness --touched --apply
```

The report includes the selected mode, git-touched files, checked scenario
surfaces, stale/error counts, diff paths, fixability labels, and next actions.
Stale tidy drift is automatic and points to `freshness --apply`; errored
surfaces caused by missing in-repo replaces are guided and point to
`deps reconcile --all --json` for preview before apply. Touched mode maps
changed files to in-repo Go module roots and then to scenario surfaces requiring
those modules; root `go.mod` and `go.sum` changes fan out to all Go surfaces.

---

### deps approved

Inspect approved third-party dependency governance records. Approved records are
review memory, not an exhaustive allowlist. If a better dependency is
appropriate, suggest it with purpose, version/range, alternatives considered,
and security/license notes so it can be reviewed and recorded.

**Usage:**
```bash
scenario-dependency-analyzer deps approved list [OPTIONS]
scenario-dependency-analyzer deps approved search <query> [OPTIONS]
scenario-dependency-analyzer deps approved explain <ecosystem>/<package> [OPTIONS]
scenario-dependency-analyzer deps approved validate <scenario> [OPTIONS]
scenario-dependency-analyzer deps approved validate --all [OPTIONS]
scenario-dependency-analyzer deps approved triage [OPTIONS]
scenario-dependency-analyzer deps approved findings [OPTIONS]
scenario-dependency-analyzer deps approved usage <ecosystem>/<package> [OPTIONS]
scenario-dependency-analyzer deps approved upsert --file <record.json> [OPTIONS]
scenario-dependency-analyzer deps approved propose-records [OPTIONS]
scenario-dependency-analyzer deps approved upsert-batch --file <proposals.json> [OPTIONS]
scenario-dependency-analyzer deps approved security-gaps [OPTIONS]
scenario-dependency-analyzer deps approved approve-observed <ecosystem>/<package> [OPTIONS]
scenario-dependency-analyzer deps approved widen-range <ecosystem>/<package> --to-major-line [OPTIONS]
scenario-dependency-analyzer deps approved approve <ecosystem>/<package> --rationale <text> [OPTIONS]
scenario-dependency-analyzer deps approved deny <ecosystem>/<package> --reason <text> [OPTIONS]
scenario-dependency-analyzer deps approved remediate <ecosystem>/<package> --vulnerability <id> [OPTIONS]
scenario-dependency-analyzer deps approved deny-vulnerable <ecosystem>/<package> --vulnerability <id> [OPTIONS]
```

**Options:**
- `--json` - Output raw JSON
- `--ecosystem` - Filter list/search by ecosystem such as `npm` or `go`
- `--state` - Filter list by governance state
- `--all` - Validate every discovered scenario
- `--policy-mode` - Override registry policy mode: `advisory`, `strict`, or `review_gate`
- `--section` - Filter `triage` output to `security`, `seeding`, `ranges`, `hotspots`, or `expired`
- `--limit` - Maximum triage groups per section, or grouped findings, in human output
- `--scenario`, `--package`, `--severity`, `--class` - Filter `findings` output; human mode groups by dependency and class
- `--minimum-severity` - Filter `security-gaps` output by normalized vulnerability severity such as `high`
- `--file` - Read an `ApprovedDependencyRecord` JSON document for `upsert`
- `--top-unrecorded` - Maximum unrecorded dependency groups to propose as draft records
- `--minimum-scenario-count` - Only propose records for dependencies observed in at least this many scenarios
- `--from-findings` - Build an `approve-observed` decision from fleet findings and observed usage
- `--to-major-line` - Widen an existing record to the single observed major line
- `--range-strategy` - Proposal or approve-observed range strategy: `observed`, `exact`, `major_line`, `minimum`, or `wildcard`
- `--range` - Version or version range for `approve` and `deny`
- `--range-policy` - How SDA evaluates `--range`: `exact`, `major_line`, `minimum`, `dev_tooling`, or `security_denied`
- `--rationale` / `--reason` - Required review rationale for approval or denial
- `--replacement` - Replacement or remediation guidance for denied dependencies
- `--vulnerability` - Security Health vulnerability id for remediation and security-derived denial
- `--affected-range` - Affected range to deny; defaults to Security Health evidence
- `--fixed-range` - Fixed range guidance; defaults to Security Health evidence
- `--dry-run` / `--apply` - Preview or apply a governance mutation; mutation commands dry-run by default

**Examples:**
```bash
# List all recorded approvals
scenario-dependency-analyzer deps approved list --json

# Search for a known-good graph package
scenario-dependency-analyzer deps approved search "React graph library" --json

# Explain one package record
scenario-dependency-analyzer deps approved explain npm/reactflow --json

# Validate one scenario's package declarations
scenario-dependency-analyzer deps approved validate graph-studio --json

# Validate dependency governance across the fleet
scenario-dependency-analyzer deps approved validate --all --json

# Show grouped governance decisions to make next
scenario-dependency-analyzer deps approved triage

# List raw fleet governance findings for automation
scenario-dependency-analyzer deps approved findings --severity WARNING --json

# Show every scenario and surface using one dependency
scenario-dependency-analyzer deps approved usage npm/react --json

# Propose draft records for the highest-frequency unrecorded dependencies
scenario-dependency-analyzer deps approved propose-records --top-unrecorded 25 --json

# Preview applying a proposal response without writing the registry
scenario-dependency-analyzer deps approved upsert-batch --file ./proposals.json --dry-run --json

# Show vulnerable dependency exposures that are not represented by denied governance records
scenario-dependency-analyzer deps approved security-gaps --minimum-severity high

# Preview approving a dependency directly from observed fleet usage
scenario-dependency-analyzer deps approved approve-observed npm/react --from-findings --json

# Preview widening an existing approval to the observed major line
scenario-dependency-analyzer deps approved widen-range npm/react --to-major-line --json

# Preview an approval without writing the registry
scenario-dependency-analyzer deps approved approve npm/react --range "18.2.0" --range-policy major_line --rationale "Approved UI runtime framework." --json

# Apply a denied dependency decision
scenario-dependency-analyzer deps approved deny npm/left-pad --range "*" --reason "Use native string padding." --replacement "String.prototype.padStart/padEnd" --apply --json

# Preview Security Health-derived remediation without writing the registry
scenario-dependency-analyzer deps approved remediate npm/vite --vulnerability GHSA-example --json

# Preview a vulnerability-derived denied range; add --apply to write the registry
scenario-dependency-analyzer deps approved deny-vulnerable npm/vite --vulnerability GHSA-example --json

# Apply a full record from JSON
scenario-dependency-analyzer deps approved upsert --file ./record.json --apply --json

# Apply a reviewed batch proposal after editing rationale/security/license notes
scenario-dependency-analyzer deps approved upsert-batch --file ./proposals.json --apply --json
```

Validation behavior:
- recorded dependency within range: pass/info
- recorded dependency outside range: warning by default
- recorded dependency outside an explicit scenario exception: warning or error depending on the exception
- unrecorded direct dependency: warning in advisory mode, error in strict mode
- unrecorded Go indirect dependency: observed-only by default unless a denied/deprecated record exists
- Security Health vulnerability evidence: exposed through `security-gaps` and `remediate`; `deny-vulnerable` records a denied affected range through the governance API and dry-runs by default
- denied or blocked dependency: error
- deprecated dependency: warning with replacement guidance
- expired governance review: warning

Signal categories:
- `direct_runtime` - direct dependency declarations used at runtime.
- `direct_dev` - direct development/tooling declarations.
- `indirect` - Go `require_indirect` dependencies, aggregated without ordinary approval seeding noise.
- `lockfile_transitive` - lockfile-only vulnerable evidence; handled through `security-gaps`, not normal approval seeding.
- `security_vulnerable` - Security Health evidence that should be upgraded, denied, or explicitly reviewed.

Range policy behavior:
- `exact` accepts only the exact recorded version unless the recorded range itself is parseable.
- `major_line` accepts versions on the same major line at or above the recorded baseline.
- `minimum` accepts versions at or above the recorded lower bound and honors explicit upper bounds such as `<20.0.0`.
- `dev_tooling` behaves like major-line policy for broad tooling approvals while Security Health-denied ranges still fail separately.
- `security_denied` is for vulnerability-derived denied records; matching versions fail, while versions outside the affected range are not blocked by that denied record.

---

### status

Show operational status and system overview.

**Usage:**
```bash
scenario-dependency-analyzer status
```

**Examples:**
```bash
scenario-dependency-analyzer status
```

**Output:**
```
📊 Scenario Dependency Analyzer Status

✅ Service: Running
   API: http://localhost:20400

Analysis System:
  Scenarios: 45
  Resources: 12
  Database: connected
  Last Analysis: 2025-11-22T17:15:00Z
```

---

## Examples

### Complete Dependency Analysis Workflow

```bash
# 1. Check service status
scenario-dependency-analyzer status

# 2. Scan a scenario to detect dependencies
scenario-dependency-analyzer scan my-scenario

# 3. Review detected dependencies
scenario-dependency-analyzer list my-scenario --type resources

# 4. Apply detected dependencies
scenario-dependency-analyzer scan my-scenario --apply

# 5. Check for circular dependencies
scenario-dependency-analyzer cycles

# 6. Check deployment readiness
scenario-dependency-analyzer deployment my-scenario

# 7. Export dependency DAG
scenario-dependency-analyzer dag export my-scenario --output my-dag.json

# 8. Generate visualization
scenario-dependency-analyzer graph combined --format dot --output graph.dot
dot -Tpng graph.dot -o graph.png
```

### Preparing for Deployment

```bash
# 1. Scan scenario and apply dependencies
scenario-dependency-analyzer scan production-app --apply

# 2. Check tier fitness
scenario-dependency-analyzer deployment production-app

# 3. Identify blockers
scenario-dependency-analyzer deployment production-app --json | jq '.aggregates.saas.blocking_dependencies'

# 4. Export recursive DAG for deployment-manager
scenario-dependency-analyzer dag export production-app --recursive --output prod-dag.json

# 5. Verify no circular dependencies
scenario-dependency-analyzer cycles --type scenario
```

### System-Wide Analysis

```bash
# Analyze all scenarios
scenario-dependency-analyzer analyze all --verbose

# Generate combined dependency graph
scenario-dependency-analyzer graph combined --format mermaid > system-graph.mmd

# Check for optimization opportunities
scenario-dependency-analyzer optimize all --type resource

# Detect all circular dependencies
scenario-dependency-analyzer cycles
```

---

## Output Formats

### JSON
Standard JSON output for programmatic consumption:
```bash
scenario-dependency-analyzer analyze my-scenario --json | jq .
```

### DOT (Graphviz)
Graph visualization in DOT format:
```bash
scenario-dependency-analyzer graph combined --format dot
```

Render with Graphviz:
```bash
scenario-dependency-analyzer graph combined --format dot | dot -Tpng > graph.png
```

### Mermaid
Markdown-compatible diagrams:
```bash
scenario-dependency-analyzer graph combined --format mermaid
```

Use in Markdown:
````markdown
```mermaid
graph TD
  swarm-manager[swarm-manager] --> postgres[postgres]
  swarm-manager[swarm-manager] --> ollama[ollama]
```
````

---

## Integration

### CI/CD Pipeline

```yaml
# .github/workflows/dependency-check.yml
name: Dependency Analysis

on: [push, pull_request]

jobs:
  analyze:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Setup Vrooli
        run: vrooli setup --yes yes
      - name: Start analyzer
        run: vrooli scenario run scenario-dependency-analyzer
      - name: Scan dependencies
        run: scenario-dependency-analyzer scan my-scenario --json > deps.json
      - name: Check for cycles
        run: scenario-dependency-analyzer cycles || exit 1
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: dependency-analysis
          path: deps.json
```

### Deployment Manager Integration

```bash
# Export DAG for deployment
scenario-dependency-analyzer dag export my-app --recursive > /tmp/my-app-dag.json

# Use in deployment-manager
deployment-manager deploy my-app \
  --dag /tmp/my-app-dag.json \
  --tier saas
```

### Shell Script Integration

```bash
#!/bin/bash
# check-dependencies.sh

SCENARIO=$1

# Scan and get JSON
RESULT=$(scenario-dependency-analyzer scan "$SCENARIO" --json)

# Parse results
RESOURCES_ADDED=$(echo "$RESULT" | jq -r '.apply_summary.resources_added | length')
SCENARIOS_ADDED=$(echo "$RESULT" | jq -r '.apply_summary.scenarios_added | length')

echo "Scan complete:"
echo "  Resources added: $RESOURCES_ADDED"
echo "  Scenarios added: $SCENARIOS_ADDED"

# Check for critical issues
if scenario-dependency-analyzer cycles --json | jq -e '.severity == "critical"'; then
  echo "ERROR: Critical circular dependencies detected!"
  exit 1
fi
```

---

## Troubleshooting

### Service Not Running

```bash
$ scenario-dependency-analyzer status
❌ Error: scenario-dependency-analyzer is not running
   Start it with: vrooli scenario run scenario-dependency-analyzer
```

**Solution:**
```bash
vrooli scenario run scenario-dependency-analyzer
```

### Port Conflicts

If the scenario fails to start due to port conflicts, check allocated ports:

```bash
vrooli scenario port scenario-dependency-analyzer API_PORT
```

### Permission Issues

Ensure the CLI is executable:

```bash
chmod +x ~/.vrooli/bin/scenario-dependency-analyzer
```

---

## See Also

- [API Reference](./api.md) - HTTP API documentation
- [Integration Guide](./integration.md) - Integration patterns and examples
- [README](../README.md) - Scenario overview
- [PRD](../PRD.md) - Product requirements
