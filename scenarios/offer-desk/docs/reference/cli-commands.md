# Offer Desk CLI reference

Every command supports `--json` and report-shaped human output. The CLI is the
same generated Connect contract used by the UI.

## Catalog commands

| Command | Purpose | Important flags |
|---|---|---|
| `catalog-list` | List typed graph nodes. | — |
| `catalog-import` | Rehearse or apply a declared catalog source. | `--source-path`, `--source-mode`, `--apply` |
| `catalog-merge` | Dry-run or apply an audited same-kind duplicate merge. Dry-run is the default. | `--surviving-id`, `--duplicate-id`, `--dry-run=false` to apply |
| `catalog-create` | Create a graph node. | `--name`, `--actual-account-id` |
| `catalog-transition` | Apply a legal transition or show refusal/remedy. | `--node-id`, `--status`, `--actor` |
| `catalog-edge` / `catalog-edges` | Create or list typed relationships. | `--from-id`, `--to-id`, `--kind`, `--currency` |

## Gate and proposal commands

| Command | Purpose | Important flags |
|---|---|---|
| `gates-trigger` | Declare a machine-evaluable trigger. | `--node-id`, `--fact-name`, `--operator`, `--threshold` |
| `gates-fact` | Record an observed fact. | `--name`, `--value`, `--stale-after-days` |
| `gates-evaluate` | Evaluate triggers as satisfied, unsatisfied, or UNKNOWN. | — |
| `gates-promote` | Promote as operator or create an agent proposal. | `--node-id`, `--actor`, `--role` |
| `gates-proposals` | List proposals and decline history. | `--node-id`, `--status` |

## Board and projection

| Command | Purpose |
|---|---|
| `board-show` | Show fired, blocked, and earning-nothing ranking, posture, evaluation condition, and named source gaps. |
| `space` | Show monetization obligation/projection cells. |

Agent promotion never silently activates a node: it creates a proposal and
leaves the node non-active. The board never turns unavailable Money Ledger
actuals into zero.

Use `make start`, `make test`, `make logs`, and `make stop` for lifecycle. The
comprehensive suite is `vrooli scenario test offer-desk`.
