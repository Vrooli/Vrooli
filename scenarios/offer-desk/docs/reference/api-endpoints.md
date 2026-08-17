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
| `MergeNodes` | Dry-run or apply an explicit same-kind duplicate merge; moves references, reports collapsed edges, audits both ids, and deletes only on apply. |

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
