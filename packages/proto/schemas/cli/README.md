# `vrooli.cli` proto contracts

Platform-level (non-scenario) wire contracts for the **root `vrooli` CLI's
`--json` output**. These are the single source of truth for the shape of
`vrooli <command> --json`, consumed across languages via generated code (Go for
the CLI itself and the ecosystem-manager API, TypeScript for the operator UI,
Python and others on demand).

This package is modeled on the fleet-shared packages (`common/`, `measures/`,
`architecture/`), **not** on a scenario package. It is message-only: it defines
output shapes, not a Connect-RPC service. The root CLI is a host-level command
dispatcher, not a networked service, so there is no transport here — only the
serialization contract.

## Why this exists

Consumers used to re-derive each command's JSON shape by hand (e.g. parsing into
`map[string]any` with stringly-typed key lookups). When the CLI's output drifted,
those consumers failed silently — an empty list rather than an error. Defining
the shape once here makes drift a **compile/build error** on every consumer at
once.

## Conventions

- Package `vrooli.cli.v1`; one `.proto` file per CLI command contract, named
  after the command (`resource_list.proto` ⇄ `vrooli resource list --json`).
- Producer marshals with `protojson` using `UseProtoNames: true` (snake_case
  wire names) and `EmitUnpopulated: true` (preserve `false`/`""` zero values),
  so the wire output is byte-compatible with the pre-proto hand-assembled JSON.
- Field names match the existing wire output 1:1 — adopting proto is a
  type-safety change, not a wire change.

## Validation

This is a fleet-shared package, so `proto-health` (which validates per-scenario)
does not cover it — the same as `common/` and `measures/`. Coverage comes from
`buf lint` (DEFAULT rules, run over all of `schemas/`) and
`make verify-committed-gen`.
