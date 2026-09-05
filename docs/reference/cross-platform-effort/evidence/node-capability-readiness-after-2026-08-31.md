# Node capability readiness — after-state evidence

Date: 2026-08-31  
Plan: `node-capability-readiness-probe-what-a-machine-can-run.md`

This record separates verified local/runtime behavior from remote proof. The
reference node was online during the final check, and its connected agent sent
a five-item capability inventory; `agy` was observed ready after installation,
while the other four were explicitly observed as missing.

## Web Console runtime

`vrooli scenario status web-console --json` returned `status=running`,
`health=healthy`, and `start_operation=succeeded`.

## Live target catalog

| Target / capability | State | Version |
| --- | --- | --- |
| local / agy | ready | 1.1.15 |
| local / claude | ready | 2.1.251 (Claude Code) |
| local / codex | ready | codex-cli 0.149.1 |
| local / grok | ready | grok 0.2.72 (5884696a1a) [stable] |
| local / opencode | ready | 1.17.9 |

`minimouse` (`25c7e426-c76c-421a-8351-aaf964589802`) reported registry,
heartbeat, channel, protocol, dispatch, scope, and session readiness as ready.
At `2026-08-31T08:16:09.426291Z`, its agent reported `agy` as
`CAPABILITY_OBSERVATION_STATE_READY`, version `1.1.22`, path
`/Users/matthalloran8/.local/bin/agy`; `claude`, `codex`, `grok`, and `opencode`
were `CAPABILITY_OBSERVATION_STATE_MISSING`, detail `command is not on PATH`.

## Governed install attempt

The Web Console action requested `antigravity`/`agy` on `minimouse` with target id
`bridge-node:25c7e426-c76c-421a-8351-aaf964589802`. The typed relay result was
`RELAY_CALL_OUTCOME_COMPLETED`, and the subsequent probe recorded `agy` ready
with version `1.1.22`. A separate `opencode` request was correctly rejected as
unsupported on `darwin/amd64`.

```text
install opencode: exit status 1
Error: unsupported installer platform darwin/amd64
```

The remote session path was then exercised through Web Console: session
`70e0f746-be87-42e8-802b-2292fa6641d0` was created on the node with launch
command `agy --help`, and the response identified the remote target and its
ready `agy` observation. A missing-agent request for `codex --help` was refused
before session creation with `failed_precondition`, naming `codex` and the
recovery action to install it on the selected machine.

An API list-before/list-after check held the active-session count at 16 across
that refused request, confirming no session row was created.

The verification command `agent-manager --node no-such-node runner list --json`
now exits 1 with an explicit refusal because Agent Manager has no governed
remote runner route; it no longer returns local runner results for an unknown
node.

The Bridge machine projection for machine
`451ea636-a80f-4080-82b7-fa65d0e3289a` exposes the desired `managed-connection`
profile (`v1`) and reports typed drift: the profile has not been applied and
the required `ssh.management` capability is not reported by the current node.

The remote working-tree onboarding path completed its sync and agent update,
then failed configuration with exactly three blockers: the two unavailable
credential addresses `vrooli/openrouter:api-key` and `vrooli/postgres:password`,
plus missing host safeguard `autoheal_watchdog`. macOS-only applicability now
correctly excludes `tpm_credential_access` and `workspace_sandbox_userns`.
The onboarding operation `a4038ea8-6c65-4a4c-91dc-17c0ff202143` reached the
three-blocker path but exposed the unconfigured-registry write failure. The
hardened retry `fa2ff4ba-66b3-4080-93f9-83d8ca2bb582` reached the same path after
recovering the remote native CLI. Its node projection now records
`configuration_state=failed`, configuration op ID
`6407616b-c618-45a7-b9d1-afa02815e324`, and timestamp
`2026-08-31T07:45:20.954391802Z`; the unmet projection is
`onboarding_apply_failed`. The production recorder now reuses the configured
registry authority and falls back through the machine's current lineage.
Remote apply items also use `outcome` on this node, so Bridge now normalizes
that field into the stable per-item disposition vocabulary.

After the hardened bootstrap restored the remote CLI, a governed Codex install
request reached `CapabilitiesService/RunAction` and the Bridge relay, but the
relay timed out after three minutes with HTTP 504. The node heartbeat after the
attempt still reported Codex missing, so no ready/version transition or install
success is claimed.

## Validation disposition

Verified: local capability probing and version projection; remote five-item
capability inventory and timestamps; remote install-to-ready transition for
`agy`; target-scoped launcher and machines actions; authenticated relay wiring;
typed unsupported-platform path; focused API/UI tests; protobuf generation
consistency; three-blocker macOS onboarding comparison; remote ready-agent
session creation and missing-agent preflight refusal; fail-closed Agent Manager
node selection; live typed machine drift projection.

Repository-level `make check` and root `make cross-compile` were also run. They
remain non-green on unrelated pre-existing findings elsewhere in the repository
(the changed resource files pass targeted lint, and the Bridge agent's own
`make check` passes its six-platform matrix); these are not represented as
plan-scoped passes.

Not verified: a successful remote install for the remaining four missing agents
or a long-running interactive transcript beyond the `agy --help` launch proof.
