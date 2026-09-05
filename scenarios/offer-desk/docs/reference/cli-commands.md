# Offer Desk CLI reference

Every command supports `--json` and report-shaped human output. The CLI is the
same generated Connect contract used by the UI.

## Catalog commands

| Command | Purpose | Important flags |
|---|---|---|
| `catalog-list` | List typed graph nodes. | — |
| `catalog-import` | Rehearse or apply a declared catalog source. | `--source-path`, `--source-mode`, `--apply` |
| `catalog-merge` | Dry-run or apply an audited same-kind duplicate merge; the duplicate is retired after references move. Dry-run is the default. | `--surviving-id`, `--duplicate-id`, `--actor`, `--reason`, `--dry-run=false` to apply |
| `catalog-rename` | Rename a node while preserving graph references and rejecting same-kind name collisions. | `--node-id`, `--name`, `--actor`, `--reason` |
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

## Catalog

| Command | Purpose |
|---|---|
| `offers catalog-create --name <n> --kind <k> --status <s> --actor <a> --reason <r>` | Create a node. `--kind` is one of `offer`, `variant`, `channel`, `revenue-line`, `deliverable`; an unknown kind is refused rather than defaulted. Actor/reason are recorded for audited canon re-declarations. |
| `offers catalog-set-class --node-id <id> --class marketed\|enabling --finish-bar <bar>` | Classify a deliverable. Enabling deliverables cannot receive a release rank. |
| `offers meters` | Show the generated meter inventory and graph conformance gaps. |
| `offers release-ladder` | Show marketed schedule rows, enabling urgency, cumulative ramps, goal impacts, and unscheduled marketed deliverables. Retired nodes are excluded unless requested. |
| `offers release-prerequisites --stream-node-id <id> --max-depth <n> --include-shipped` | Walk all transitive prerequisites for a stream. |
| `offers catalog-map-account --node-id <id> --account-id <acct>` | Attach the Money Ledger account whose postings are this node's actuals. Omit `--account-id` to clear. Audited. |
| `offers catalog-merge --surviving-id <a> --duplicate-id <b>` | Audited duplicate-identity collapse. Dry-runs by default. |
| `offers catalog-verify --source-path <dir> --source-mode operator-supplied` | Reconcile the declared source tree against the live graph. Non-zero exit on drift. |

## Gates

| Command | Purpose |
|---|---|
| `offers gates-trigger --node-id <id> --fact-name <f> --operator <op> --threshold <n>` | Declare a single-clause trigger. |
| `… --clauses '[{"fact_name":…,"operator":…,"threshold":…}]' --composition all\|any` | Declare a multi-clause trigger. `all` requires every clause; a missing fact is `unknown`, which blocks rather than fires. |
