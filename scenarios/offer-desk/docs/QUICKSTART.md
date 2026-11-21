# Quickstart — Offer Desk

This is the first useful path for a new operator: start Offer Desk with Money
Ledger, create one offer node, declare a trigger/fact, inspect the board, and
exercise the agent/operator promotion boundary.

## Start

From the scenario directory:

```bash
make setup
make start
make status
```

The lifecycle starts the declared Money Ledger dependency and owns ports and
health checks. Open the reported UI port, or use `make open`.

## Create the first offer through the console

1. Open **Offers** and create an offer node with a name and lifecycle status.
2. Open **Triggers** and declare a trigger for that node.
3. Record a fact with its freshness window, then run the dry-run evaluation.
4. Open the **Dashboard**. Fired, blocked, and earning-nothing regions remain
   separate; the posture card names Money Ledger source, age, and gaps.
5. Try **Promote** from an agent role. The node stays non-active and a
   proposal appears under **Proposals**. An operator can accept it, or decline
   it with a durable reason.

## Equivalent CLI path

```bash
offer-desk offers catalog-create --name "First offer" --json
offer-desk offers gates-trigger --node-id <node-id> --fact-name revenue \
  --operator ">=" --threshold 100 --json
offer-desk offers gates-fact --name revenue --value 125 --stale-after-days 30 --json
offer-desk offers gates-evaluate --json
offer-desk offers gates-promote --node-id <node-id> --actor agent --role agent --json
offer-desk offers gates-proposals --node-id <node-id> --json
offer-desk offers board-show --json
```

Use `catalog-import --source-path <copy> --source-mode <mode>` for a rehearsal
copy. Dry-run first; malformed status or reference drift blocks apply and does
not write the catalog. The board never renders unavailable actuals as zero.

## Lifecycle and tests

```bash
make logs
make test
make stop
```

For the comprehensive server-owned suite, use `vrooli scenario test
offer-desk` and wait once on the returned run ID. See
[`reference/cli-commands.md`](reference/cli-commands.md) and
[`concepts/UI-ARCHITECTURE.md`](concepts/UI-ARCHITECTURE.md) for the complete
surface contract.
