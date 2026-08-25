# API Endpoints

The machine-readable declaration is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json).
It is generated from the router, and a test fails if the two disagree in either
direction — so this page and that file cannot drift from the code independently.

No endpoint accepts, returns, logs, or echoes a credential value.

## Health

| Method | Path | Returns |
|---|---|---|
| GET | `/health` | Service health for infrastructure probes |
| GET | `/api/v1/health` | The same payload for clients |

## Read models

All read models are derived from manifests plus committed operator state on
every request. None is cached, and none has independent storage.

| Method | Path | Returns |
|---|---|---|
| GET | `/api/v2/scenarios` | Every scenario with `system_required`, `runtime.kind`, the auto-restart recommendation, the effective operator choice, and its declared scenario and resource dependencies |
| GET | `/api/v2/closure` | The transitive closure of the current selection: every member, and why it entered the set |
| GET | `/api/v2/resources` | Required (derived and locked), optional (`optional_dependencies`), and standalone resources with their effective enablement |
| GET | `/api/v2/credentials` | Credential descriptors for the selected stack, complete with `label`, `description`, `obtain_url`, `required`, and configured status — never a value |
| GET | `/api/v2/host-requirements` | Host tools and safeguards from selected manifests with `risk`, `privilege`, `bundling`, `platforms`, and any declared config schema |
| GET | `/api/v2/readiness` | Composed readiness over credentials, host tools, host safeguards, and resource reachability, with a remediation on every non-ready item |
| GET | `/api/v2/union` | The union of scenarios, resources, tools, and safeguards a target must carry to run the current selection |
| POST | `/api/v2/handoff` | Resolve the effective capability-shaped selection for vrooli-bridge from node identity; returns no operator-state internals or credential values |

`/api/v2/union` is what bundle packaging, VPS provisioning, and vrooli-bridge
read to decide what to ship. It is the same computation the wizard's rollup
uses, exported rather than re-derived.

## Session

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v2/session` | The step pointer, which steps are satisfied, and the first unsatisfied step |
| POST | `/api/v2/session/step` | Move the pointer. Shared state, so another surface resumes at the same step |

## Operator state

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/operator-state` | The committed document, with effective values resolved |
| PATCH | `/api/v2/operator-state` | Apply an [RFC 7386](https://www.rfc-editor.org/rfc/rfc7386) JSON Merge Patch |

`PATCH` is the only write. The body names only the fields being changed; every
other field in the stored document is preserved, including fields this binary
does not model. Setting a key to `null` removes it. The merged document is
validated against `operator-state.schema.json` **before** the write; a rejection
names the failing path and leaves the stored document untouched.

`PUT /api/v1/operator-state` is retired. A whole-document write from a partial
writer silently truncates the document, and the state schema's
`additionalProperties: false` makes that loss unrecoverable without hand repair.

## Apply

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/v2/apply` | Install opted-in tools, apply opted-in safeguards, enable resources, start scenarios |
| GET | `/api/v2/apply/{run_id}` | The per-item report for a run |

Apply is idempotent: a second run against unchanged state performs no mutation
and reports every item as already satisfied. A failing item skips its dependants,
names the blocking dependency, and does not abandon independent work; the run is
then recorded as partially applied. Every action is performed by its owning
control-plane handler — onboarding orders and reports, and implements no host
remediation itself.

## Credentials

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/v2/credentials/provision` | Relay a value to the credential authority |
| GET | `/api/v2/credentials/doctor` | Diagnose the credential backend and name the fix |

`provision` accepts the value only in the request body and returns only
`logical_id` and `field`. `doctor` distinguishes an unset value, an unreachable
secure store, and a host with no backend, because each has a different fix.

## Resources and glossary

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/resources` | Resource inventory with status, category, and installed flag |
| GET | `/api/v1/resources/{name}` | One resource |
| GET | `/api/v1/resources/health` | Live availability for the health dashboard |
| GET | `/api/v1/glossary` | Plain-language term lookup, optionally filtered by `q` |

Categories and glossary entries are read from manifests. Neither is a table in
Go.

## Retired

| Path | Replacement |
|---|---|
| `PUT /api/v1/operator-state` | `PATCH /api/v2/operator-state` |
| `GET|PUT /api/v1/progress` | `GET|POST /api/v2/session` |
| `POST /api/v1/config/generate` | None. Onboarding never authors `service.json` |
| `POST /api/v1/config/validate` | `GET /api/v2/readiness` |
| `POST /api/v1/config/export` | None |
| `GET /api/v1/setup-order` | `GET /api/v2/closure` |

## Errors

Every error response carries a machine-readable code and an operator-facing
message that names the next action. A tier that cannot supply a manifest catalog
returns a typed degraded state, not a server error — "this tier does not carry
that catalog" and "this host is broken" have different operator responses, and
collapsing them into a 500 loses that.
