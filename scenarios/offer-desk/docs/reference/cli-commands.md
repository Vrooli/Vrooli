# CLI commands

The `offers` group covers `catalog-list`, `catalog-create`, `catalog-transition`, `catalog-edge`, `gates-trigger`, `gates-fact`, `gates-evaluate`, `gates-promote`, and `board-show`. Every command is bound to a generated Connect RPC and supports report-shaped human output plus `--json`.

`catalog-transition` returns the refusal rule and legal next states. `gates-evaluate` exposes unknown versus unsatisfied. `gates-promote` requires an operator role to change state; an agent invocation creates a proposal only. `board-show` preserves source availability and posture caveats.
