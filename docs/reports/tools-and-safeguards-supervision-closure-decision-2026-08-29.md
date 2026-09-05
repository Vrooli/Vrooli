# Tools and safeguards in the supervision closure

Date: 2026-08-29  
Status: accepted for a successor plan; excluded from the current supervision-authority plan

## Decision

Vrooli should eventually represent host-tool and host-safeguard requirements as typed edges in the supervision closure. That work must be a successor plan. The current plan must not add either dependency kind.

This boundary is intentional. The current plan consolidates existing scenario and resource declarations into one supervision authority. Adding two new edge kinds at the same time would change manifest contracts, host-consent behavior, dependency analysis, closure semantics, validation, and every downstream supervision-set consumer. The successor work must preserve the existing host-requirement authority instead of creating a competing registry.

## Current state

Scenario dependency objects currently accept exactly two kinds: `resources` and `scenarios`. A repository-wide read of all 120 scenario manifests found no `dependencies.tools` or `dependencies.safeguards` declaration. The schema's `dependencies` object also permits only those two properties.

Tools and safeguards are already declarations, but they are host requirements rather than dependency-graph edges:

- The root `.vrooli/service.json` declares 19 `hostTools` entries and 26 `hostSafeguards` entries.
- The 120 scenario manifests contain 52 tool declarations across 15 scenarios and 2 safeguard declarations across 2 scenarios.
- The 29 resource manifests contain 6 tool declarations across 6 resources and 6 safeguard declarations across 6 resources.
- Tool definitions live under `internal/tools/<name>/tool.json`. The filesystem registry currently contains 69 definitions.
- Safeguard definitions live under `internal/safeguards/<name>/safeguard.json`. The filesystem registry currently contains 29 definitions.
- Operator choices and typed parameters live in `.vrooli/operator-state.json` under `host_tools` and `host_safeguards`.
- `internal/hostreq` already resolves root, scenario, resource, platform, environment, lifecycle-stage, and operator-state declarations. This remains the authority for whether a host requirement is applicable and allowed.

The word `tools` appears in two scenario lifecycle `needs` arrays. Those values name lifecycle capabilities; they are not dependency kinds and must not be interpreted as supervision edges.

## Proposed edge shapes

The successor plan should extend the existing dependency object without moving or duplicating host-requirement definitions. A graph edge should refer to the same canonical name used by `hostTools` or `hostSafeguards`.

```json
{
  "dependencies": {
    "tools": {
      "ffmpeg": {
        "required": false,
        "startup_policy": "try_start"
      }
    },
    "safeguards": {
      "workspace_sandbox_userns": {
        "required": true,
        "startup_policy": "must_start"
      }
    }
  }
}
```

The edge shape should reuse the dependency intent contract introduced for scenario and resource edges:

- `required`: compatibility input for the existing declaration model.
- `startup_policy`: explicit `must_start`, `try_start`, or `ignore` value.
- `supervision_precedence`: required only when `required` and `startup_policy` disagree, under the same validation table as other dependency kinds.

The successor plan should consider replacing the suffix `_start` with a kind-neutral vocabulary at the API boundary. It must not introduce a second intent axis merely because a tool is installed and a safeguard is applied rather than started.

## Meaning by kind

For a tool edge:

- `must_start` means the applicable tool must resolve and be available before the declaring member is ready. The action is "ensure present," not "run a daemon."
- `try_start` means inspect and attempt installation when the platform and environment are eligible. Absence is visible degradation and does not remove the declaring member from the set.
- `ignore` means retain the declaration for documentation or another lifecycle stage, but exclude it from supervision closure and repair.

For a safeguard edge:

- `must_start` means the applicable invariant must be inspected and satisfied before the declaring member is ready. Existing risk, privilege, maintenance-window, and reboot-verification gates still apply. The graph must never bypass host-requirement consent or operation-risk policy.
- `try_start` means inspect and, when existing consent and eligibility permit, attempt application. Refusal, not-applicable, reboot-required, and manual-action states remain typed outcomes rather than generic failures.
- `ignore` means retain the declaration but exclude it from supervision closure and repair.

For both kinds, closure inclusion does not itself authorize mutation. `internal/hostreq` remains responsible for resolution and the tool/safeguard runtimes remain responsible for execution policy.

## Consumers that must change

The successor plan must change all of these consumers together:

1. Manifest schema and canonical Go models for scenario and resource dependency maps.
2. Scenario validation, resource validation, precedence validation, fixtures, and schema synchronization.
3. Scenario Dependency Analyzer parsing, graph nodes and edges, closure traversal, attribution chains, reports, API, CLI, and tests.
4. The shared core-set/supervision-set response model and any generated protocol types.
5. Control-plane supervision-set rendering, filtering, schema validation, and documentation.
6. `internal/hostreq` resolution so closure attribution and existing declarations cannot disagree or double-apply.
7. Setup, lifecycle, readiness, and resource-action enforcement, including platform and environment filtering.
8. Operator-state validation and onboarding presentation for opt-in, parameters, risk, maintenance windows, and reboot-required outcomes.
9. Autoheal target construction, status reporting, interlocks, action persistence, retry policy, and incident reporting for the two new member kinds.
10. Tool and safeguard registry conformance tests and cross-platform tests for Linux, macOS, and Windows.

## Blast-radius estimate

The minimum declaration blast radius is 150 manifests: 120 scenario manifests, 29 resource manifests, and the root manifest. Those files currently hold 45 root host requirements, 54 scenario host-requirement edges, and 12 resource host-requirement edges. Migration may leave many manifests unchanged, but every manifest must be read and validated because an incomplete migration would silently omit a required host capability.

The current host-requirement implementation alone spans at least 20 directly matching Go source and test files across canonical models, resolution, setup, lifecycle, operator state, resource enforcement, and fixtures. The graph and public-surface work adds Scenario Dependency Analyzer, shared core-set/protocol packages, control-plane handlers, onboarding, and autoheal. A realistic successor-plan boundary is therefore dozens of implementation and test files plus repository-wide validation of all 150 manifests. This estimate excludes the 98 tool and safeguard definition manifests because the proposed edge shape references them without rewriting them.

## Why this work is not in the current plan

The current incident was caused by competing authorities for scenarios and resources. Phases 7 through 12 can remove that ambiguity using dependency kinds that already exist. Tools and safeguards have distinct authority and safety contracts, especially consent and privileged host mutation. Folding them in without designing those contracts would widen a supervision repair into a host-policy migration and could make closure membership look like authorization.

The successor plan should begin only after the scenario/resource supervision set is operational and measured. Its first acceptance gate should prove that adding tool and safeguard nodes cannot bypass applicability, operator consent, maintenance windows, or reboot verification.

## Independent-reader check

A reader can answer the two required questions from this document alone:

1. What would change? Two typed dependency edge maps would reference existing tool and safeguard declarations; schemas, graph/closure, validation, control-plane surfaces, host-requirement resolution, onboarding, and autoheal would consume them.
2. Why is it deferred? It touches at least 150 declaration manifests and dozens of implementation files, and safeguard membership must not be confused with authority to mutate a host. The present plan is limited to consolidating the already-existing scenario and resource edge kinds.
