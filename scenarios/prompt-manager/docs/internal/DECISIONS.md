# Architecture Decisions

Durable decisions for Prompt Manager. Newer rows supersede older rows only
when they name the superseded decision explicitly.

| Date | Decision | Rationale | Rejected alternative |
|---|---|---|---|
| 2026-08-19 | Proto-first Connect is the sole target transport. | Generated contracts make CLI, UI, other scenarios, and Program Runtime consume the same typed surface. | Keep REST as the primary API and add selective Connect facades. This preserves two sources of truth and leaves most agent operations unbindable. |
| 2026-08-19 | Use one proto package per business domain and no generic `common` package. | Ownership remains visible and shared messages evolve with the domain that defines them. | One monolithic prompt-manager proto or a grab-bag common package. Both hide boundaries and create broad regeneration/change coupling. |
| 2026-08-19 | Split transport adapters into `api/handlers/<domain>` and services/ports into `api/internal/<domain>`. | The filesystem screams capability while dependencies point inward. | Keep mixed handler/service packages at `api/<domain>` or merely populate empty mirror directories. Both retain ambiguous ownership. |
| 2026-08-19 | Retire REST operation-by-operation after consumer search and parity, with no permanent dual surface. | One supported route prevents semantic drift and makes failures attributable to a slice. | Leave REST and Connect indefinitely for compatibility. This doubles contract maintenance and makes consumers choose accidentally. |
| 2026-08-19 | Bind stable behavior; omit only CLI-local, interactive, or operator-destructive behavior with an owner and exit condition. | Omissions remain reviewable governance decisions instead of a denominator escape hatch. | Mark all commands run-eligible, or omit anything inconvenient. The first is unsafe; the second earns a meaningless maturity score. |
| 2026-08-19 | Treat runtime-visible commands, not the 84-entry legacy manifest, as the final denominator. | Team and graph expose undeclared leaves. A manifest-only audit would falsely certify an incomplete surface. | Preserve the current manifest denominator and ignore runtime-only leaves. |
| 2026-08-19 | Migrate heartbeat and memberflow in one slice while keeping separate domain/proto ownership. | Scheduling consumes member declarations and memberflow consumes prompt sections; their temporal contract must move coherently. | Merge them into one domain, or migrate them separately. Merging erases ownership; separate slices can strand live scheduling between contracts. |
| 2026-08-19 | Keep generated wire types outside the domain model. | Thin mapping prevents Connect concerns from spreading through persistence and orchestration. | Reuse generated messages as storage/domain structs. This couples internal evolution to wire compatibility and protobuf presence semantics. |
| 2026-08-19 | Include world scale/seats in the structural slice and experiments in the orchestration slice. | Small persisted UI configuration must not be stranded; experiment receipts/promotion align with runtime evidence. | Leave them REST-only or invent miscellaneous proto packages later. |
| 2026-08-19 | Require real measures for stateful domains and named waivers for genuine exceptions. | Missing or empty condition evidence cannot support reliable automation. | Declare placeholder measures or treat absence as healthy. |

The binding/omission rules and exact slice order are normative in
`docs/concepts/DOMAINS.md`; changing either requires a row here.
