# Offer Desk API endpoints

The machine-readable source of truth is [`../../.vrooli/endpoints.json`](../../.vrooli/endpoints.json).
Wire messages and services live in `packages/proto/schemas/offer-desk/v1`;
the generated Connect clients are used by the UI and CLI.

## Operational endpoints

| Method | Path | Purpose |
|---|---|---|
| GET | `/health` | Lifecycle/readiness health response. |
| GET | `/api/v1/capabilities/describe` | Typed capability metadata. |

## Catalog

All RPCs use `POST /vrooli.offer_desk.v1.offers.CatalogService/<Method>`.

| Method | Purpose |
|---|---|
| `CreateNode` | Create an offer, variant, channel, revenue-line, or deliverable node. |
| `ListNodes` | List typed catalog nodes and lifecycle status. |
| `Transition` | Apply a legal lifecycle transition or return a refusal/remedy. |
| `CreateEdge` | Add a typed relationship, including optional intended price/currency. |
| `ListEdges` | List graph relationships. |
| `ImportCatalog` | Dry-run or apply a declared catalog source; malformed status and reference drift block apply. |
| `MergeNodes` | Dry-run or apply an explicit same-kind duplicate merge; moves references, retires the duplicate, reports collapsed edges, and audits the change. |
| `RenameNode` | Rename a node with same-kind/name uniqueness validation; edges and references remain attached and the prior name is returned. |
| `SetReleaseRank` | Set a unique operator-owned rank for a marketed deliverable; enabling deliverables are refused. The reason is audited. |
| `SetDeliverableClass` | Classify a deliverable as marketed or enabling and set its orthogonal finish bar; the reason is audited. |
| `GetMeterInventory` | Read the generated meter vocabulary and report undeclared streams or deliverable-meter gaps. |

`BoardService.GetBoard` derives `rank_reason` from status and actuals availability:

| Status / evidence | `rank_reason` |
|---|---|
| unspecified | `status not set` |
| idea | `captured, not planned against` |
| candidate | `blocked: trigger not met` |
| trigger met | `trigger fired` |
| proposed | `awaiting operator decision` |
| active/shipped with unavailable actuals | `<status>; earnings unknown — <source> unavailable` |
| active/shipped with reachable zero actuals | `<status> and earning nothing` |
| active/shipped with reachable nonzero actuals | `<status> and earning` |
| retired | `retired` |

An unavailable actuals read never produces an earnings claim; the row instead
keeps `actuals_available=false` and names the source in `availability`.

## Gates and proposals

All RPCs use `POST /vrooli.offer_desk.v1.offers.GatesService/<Method>`.

| Method | Purpose |
|---|---|
| `DeclareTrigger` | Attach a machine-evaluable trigger to a node. |
| `AddFact` | Record an observed fact with freshness metadata. |
| `Evaluate` | Return satisfied, unsatisfied, or UNKNOWN with explanation and evidence age. |
| `Promote` | Operator promotion changes state; an agent role creates a proposal and leaves the node non-active. |
| `ListProposals` | List proposer, requested status, reason, creation/evidence metadata, and decline history. |

## Board and projection

| Service / method | Path | Purpose |
|---|---|---|
| `BoardService/GetBoard` | `POST /vrooli.offer_desk.v1.offers.BoardService/GetBoard` | Returns ranked fired, blocked, and earning-nothing regions plus Money Ledger posture and named availability. |
| `SpaceService/GetProjection` | `POST /vrooli.offer_desk.v1.offers.SpaceService/GetProjection` | Returns obligation/projection cells without taking ownership of ledger balances. |

The board never turns an unavailable actual into zero. Coverage, trigger
condition, and empirical actuals are returned as separate evidence fields.
Connect errors use the canonical envelope; UI errors are announced as text.

## Change procedure

Change the proto first, regenerate through the scenario lifecycle, update the
thin handler and CLI binding, and refresh endpoint metadata through the
scenario tooling. Do not hand-edit `.vrooli/endpoints.json`. Validate against
[`cli-commands.md`](cli-commands.md) and the API/contract suites.

## CatalogService.MapAccount

`POST /vrooli.offer_desk.v1.offers.CatalogService/MapAccount`

Attaches an existing node to the Money Ledger account holding its actuals, or
clears the mapping when `actual_account_id` is empty. Requires `actor`. Returns
the updated node and `prior_account_id` so a caller can see what it replaced.

This exists because `actual_account_id` was previously settable only at
`CreateNode`, while the operator importer writes an empty value for every record
it materializes — leaving the entire imported catalog permanently unjoinable to
actuals.

Every call writes a `catalog_audit` row. Corrections are new entries, never
edits, consistent with `OT-P0-007`.

## Release ladder

`ReleaseLadderService/GetReleaseLadder` returns marketed deliverables in rank
order, reports unranked marketed deliverables as `unscheduled`, filters retired
nodes by default, and exposes enabling deliverables separately. Enabling
urgency is derived through `enables` closure and the earliest scheduled
marketed opener of a ramp or stream; zero means no scheduled opener. `GetPrerequisites` walks
incoming `unlocks` and `enables` edges transitively, with depth, path, finish
bar, live status, and derived urgency. `include_shipped` controls the
unshipped list; the tree remains complete for the requested depth.
