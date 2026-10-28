# Scenario Dependency Analyzer Architecture

## Pinned Akash CLI provisioning

The closed `source/akt@1.0.4` request on owner `akt`, surface `host-tool`,
delegates to `internal/tools/akt`. The owner uses native checksum-verified
binaryfetch, a private candidate, the shared artifact lock and native install
ledger. Resolution is revalidated against the exact source manifest, version,
command, paths and execution policy before acquisition. It executes no install
script, creates no wallet or context, and does not accept arbitrary tools or
paths. The selected executable remains outside ambient PATH adoption; callers
use the owner's integrity-checked selection. Only Linux amd64 is qualified by
this release profile; other platforms return explicit unsupported results.

## Frozen control-plane tool provisioning

The closed `source/kubo@0.43.1` request on owner `kubo`, surface `host-tool`,
uses the same pinned-release dispatch and delegates acquisition and selection to
`internal/tools/kubo`. The runtime supplies its existing generic manifest
inspection; the owner requires a verified native selection and delegates writes
to SDA instead of adopting an ambient `ipfs` executable or using the generic
direct downloader. The owner retains the checksum-verified release tree,
MIT/Apache notices, receipt and install-ledger attribution. Only Linux amd64 is
qualified. Provisioning starts no IPFS node, creates no repository or peer key,
and makes no network publication. Offline import/pin/GC acceptance belongs to
the invoking task, with separate isolation and cleanup evidence.

The `host-tool` surface addresses declared tool owners in `internal/tools/<name>`;
the existing `tools/<package>` spelling continues to address shared packages.
Frozen tool requests require the declared exact npm package/version and committed
integrity lock. Governance is evaluated before provisioning. Provisioning uses
`npm ci --ignore-scripts --no-audit --no-fund` in a private candidate and records
the frozen closure before atomic selection. It never modifies a scenario UI,
downloads a browser, or replaces the current tool after a failed candidate.
The control-plane tool owner owns candidate validation and selection; SDA owns
dependency approval and the only package-manager execution seam.

The existing Codex, Claude Code, OpenCode and Grok resource owners also use this
closed surface. Their reviewed `resource.json` acquisition declarations own
package identity; SDA approves the exact requested version for that owner.
Harness updates install with scripts disabled into a private candidate, then run
only `--version` at the final artifact path through the bounded native-operation
owner, with IP networking isolated and a private HOME. Absolute host files and
AF_UNIX sockets remain visible in the read-only host mount; this is not hostile-code
credential isolation. A missing
postinstall artifact, wrong version, source change, failed probe or unknown cleanup
preserves the previous selection. Candidates are hashed and published to separate resource/version directories;
only successful final-path qualification selects the link atomically. Reviewed `bin_path` may select an exact optional package's
native file; the owner verifies main/native identities, exact optional reference,
version and containment before probing. This avoids invoking vendor postinstall
scripts or copying their binaries. Current native-path qualification is Linux
amd64; compressed Grok preparation and legacy regular-file adoption remain unsupported. This is exact
main-package acquisition, not a committed frozen transitive closure. Existing
prefixes and running harness processes remain available for rollback. The artifact
lock serializes cooperating owners; local hashes detect consistency drift, not
independent provenance or arbitrary external-writer races. Updaters
accept only a matching typed approved runtime receipt, and reject trailing flags;
use SDA's default preview before `--apply` for a real dependency preview.

An explicit `development` install option saves JavaScript dependencies as dev
dependencies; its default preserves existing classification. Non-JavaScript,
override and frozen host-tool operations reject that option before effects.
An explicit `refresh_lock` host-tool operation generates only the exact declared
package's source manifest/lock in a private validated candidate. It is governed
against the target version, keeps lifecycle scripts disabled and does not replace
an installed runtime. Source publication is hash-fenced with rollback for ordinary
errors; a crash between the two file replacements fails the exact-closure check
and requires recovery. `source_updated` and `installed` are distinct outcomes.
Root reviews the source diff and owner version metadata before separately
admitting frozen provisioning of the new closure.

SDA is evolving from a declared-dependency reporter into Vrooli's actual cross-scenario interface graph authority.

## Role

- **Ecosystem role**: Meta / interface-enabler.
- **Primary consumers**: Operators through the UI/CLI; `test-genie`, `tech-tree-designer`, and future deployment planners through the programmatic surface.
- **Stable seam**: `DescribeInterfaceGraph`, a Connect RPC that returns evidence-tagged scenario edges.

## Boundary Of Responsibility

| Capability | Owner | SDA dependency |
|---|---|---|
| Protobuf surface extraction | `proto-health` | Batch proto surface RPC |
| Language-level import extraction | `code-facts` | Batch imports fact RPC |
| Import/proto path attribution to scenario slugs | SDA | Fleet naming conventions |
| Declared-vs-actual drift interpretation | SDA | `service.json` + actual edge set |
| Dependency surface freshness | SDA | Code Facts surfaces + package-manager checks |
| Root hygiene dependency presentation | root `vrooli hygiene` | SDA-owned provider output |
| Runtime URL and CLI shell-out facts | Future AST facts | Follow-up plan |

SDA must not scan source files for actual graph evidence. Its responsibility is
interpretation, not extraction. For dependency freshness, SDA is also the domain
owner: root `vrooli hygiene` is only an aggregate registry and presentation
surface. It must not embed package-topology rules, shared-package trigger lists,
or Go module drift logic that belongs in SDA.

## Actual Graph Model

The actual graph is assembled from upstream facts and served from a derived cache when fresh:

1. Fetch batch proto surfaces from `proto-health`.
2. Fetch batch import facts from `code-facts`.
3. Attribute import paths to scenario slugs.
4. Merge evidence into source-to-target scenario edges.
5. Persist the derived graph by request signature for fast repeat reads.
6. Compare actual edges with declared scenario dependencies.

Edges carry:

- `evidence[]`: `proto_import`, `go_import`, or future AST fact kinds.
- `transport_world`: forwarded from proto surface facts where available.
- `stability`: forwarded from proto surface facts where available.

The derived graph cache is persisted in SQLite for repeat-read performance. It is rebuildable from upstream facts and must not become the source of truth for proto surfaces or language imports.

## Drift Semantics

| Drift kind | Meaning | Severity |
|---|---|---|
| `undeclared_but_used` | Actual import evidence targets a scenario not declared in `service.json` | `WARNING` |
| `declared_without_import_evidence` | `service.json` declares a scenario but no import evidence exists | `INFO` |

The second case is intentionally advisory because runtime discovery calls and CLI shell-outs are deferred to AST fact providers.

## Dependency Freshness Semantics

Dependency freshness is evaluated per discovered dependency surface. Code Facts
is the preferred inventory source because it can report every package root; SDA
falls back to conventional `api`, `cli`, and `ui` roots only as degraded
behavior. Go readiness checks cover `go.mod` presence, local replace targets,
`go mod tidy -diff`, and workspace-disabled `go build ./...`. Node readiness checks cover
`package.json`, exactly one supported lockfile, and local install state.

Fleet/touched freshness is an aggregation mode over the same findings, not a
separate root-owned model. Touched files are mapped to in-repo package/module
roots and then to impacted scenario surfaces. Root `go.mod` and `go.sum`
changes fan out to applicable Go surfaces. Findings report the scenario,
surface, package root, rule ID, observed value, expected value, remediation, and
fix class so `vrooli hygiene` can render next actions without interpreting
dependency logic.

Freshness maturity maps Go and Node package coherence to
`package_readiness`. Surface discovery, unsupported language, and package-root
classification failures map to `surface_inventory`. Deterministic Go fixes may
be previewed/applied by SDA for a specific surface; all ambiguous or
ecosystem-specific repairs remain advisory until SDA owns a safe typed fixer.

## Storage

SDA uses SQLite for local, history-bearing data. Schemas are domain-owned (`api/internal/<domain>/schema.sql` + `schema.go`) and registered at boot through the shared database schema helper. The actual interface graph is recomputed from upstream facts instead of stored.

## API Shape

Existing Gin REST endpoints remain during the hybrid migration for UI/operator compatibility. New scenario-to-scenario consumers use the Connect surface.

The CLI must expose machine-readable access to the same graph and drift data, with `graph actual --json` and `drift --json` as the intended operator surface.

## Package-wide npm override replacement

The governed install gateway's `override:` operation writes one package-wide
resolver choice. It removes superseded simple version-qualified selectors for
that exact package, including scoped package names, so those older policies cannot
keep selecting a vulnerable version. Parent-qualified constraints and unrelated
package overrides remain unchanged. Parent-selector recognition follows pnpm's
delimiter grammar, preserving comparison ranges such as `>=4` as ordinary version
qualifiers. Manifest fields unrelated to overrides are preserved.

Install admission uses the same scenario-exception predicate as dependency
validation. A scenario outside a nonempty allowed list, or explicitly denied,
cannot mutate the manifest or invoke a package manager, including override
requests. An empty allowed list retains the existing global grant semantics.
