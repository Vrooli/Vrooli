# Node setup seam — measured baseline, 2026-08-31

Companion evidence for
[`../node-setup-seam-2026-08-31.html`](../node-setup-seam-2026-08-31.html) and for the
`node-setup-seam` implementation plan. Every number below was measured against the
working tree on branch `agi` on 2026-08-31. Nothing was modified to produce them.
Re-run the command in the right column to reproduce or to detect drift.

## 1. Discovery port-lookup ladder — captured before-state

`cliutil.LookupScenarioPort` is a three-rung ladder: peer file, runtime registry,
then a `vrooli scenario port` fork. Rungs 1 and 2 miss on every production call
because callers pass the env-var spelling while both stores are keyed by claim name.

| Measurement | Value | Command |
| --- | --- | --- |
| Peer records present | 83 | `ls ~/.vrooli/peers/ \| wc -l` |
| Peer record port keys | `api`, `ui` | `jq -r '.ports \| keys \| join(", ")' ~/.vrooli/peers/web-console.json` |
| `runtime_port_claims` rows for `API_PORT` | **0** | `sqlite3 -readonly -noheader ~/.vrooli/state/runtime.db "SELECT count(*) FROM runtime_port_claims WHERE port_name='API_PORT';"` |
| `runtime_port_claims` bound rows for `api` | **82** | `sqlite3 -readonly -noheader ~/.vrooli/state/runtime.db "SELECT count(*) FROM runtime_port_claims WHERE port_name='api' AND status='bound';"` |
| Production callers passing `API_PORT`/`UI_PORT` | all of them | `grep -rn "DetectPortFromVrooli(" --include=*.go . \| grep -v vendor \| grep -v _test` |
| Copies of `port_detector.go` | 2, byte-identical | `find . -name port_detector.go -not -path "*/vendor/*"` |
| Production code setting `VrooliPath` (ladder bypass) | none | `grep -rn "VrooliPath:" --include=*.go . \| grep -v vendor \| grep -v _test` |

Normalization exists only in `internal/scenario.runtimePortCandidates`, inside the CLI
command that rung 3 forks. `lookupPeerRecord` does a raw `record.Ports[portVar]`;
`lookupRuntimeRegistry` queries `port_name = 'API_PORT'` literally.

The ladder tests replace `peerRecordLookupFn` and `runtimeRegistryLookupFn` with stubs,
so they assert ordering and never exercise the real key vocabulary.

## 2. `--environment` carries one bit

Its only mechanism anywhere is filtering host tool/safeguard declarations by their
`environments` list in `internal/hostreq/resolve.go`. No resource manifest declares
`environments`. Nothing reads it after setup.

| Declared set | Count | Effect |
| --- | --- | --- |
| `development` | 17 | build toolchain only — go, node, pnpm, python, java, quint, buf, protoc + plugins |
| `development,production,minimal` | 8 | filters nothing |

Reproduce:

```bash
{ jq -r '[.hostTools[]?, .hostSafeguards[]?] | .[] | select(.environments) | (.environments|join(","))' .vrooli/service.json
  for f in scenarios/*/.vrooli/service.json; do
    jq -r '[.hostTools[]?, .hostSafeguards[]?] | .[] | select(.environments) | (.environments|join(","))' "$f" 2>/dev/null
  done
} | sort | uniq -c | sort -rn
```

`production` and `minimal` therefore select identical requirement sets today.

## 3. Configuration representations of "what should this machine be"

| # | Representation | Location | Fields |
| --- | --- | --- | --- |
| 1 | operator-state document | `.vrooli/operator-state.json` (gitignored) | 20 top-level keys |
| 2 | operator-input queue | `~/.vrooli/state/operator-input.json` | 8 typed kinds; 2 unanswered on this host |
| 3 | `Selection` handoff | `scenarios/vrooli-bridge/api/internal/onboarding/client.go` | 5 fields; optional, nil-defaulted |
| 4 | built-in profiles | `scenarios/vrooli-bridge/api/internal/machines/policy.go` | 6 values, all with empty `Scenarios` |
| 5 | setup strings | `StartOnboardingRequest` fields 15-18 | 4 free-text values |

Manifest-versus-selection pairing works as documented: the project manifest declares
26 host safeguards; operator-state records 9 opted in with typed config.

Live operator-input queue on this host:

```
credential-escrow:sink                  [path]    required=true   Escrow destination
credential-escrow:recovery_passphrase   [secret]  required=true   Recovery-bundle passphrase
```

## 4. Scope catalog readiness for RPC-level admission

| Measurement | Value | Command |
| --- | --- | --- |
| Manifest-declared commands | 2,392 | see below |
| Bound to a concrete `connect-rpc` service+method | **1,894 (79%)** | see below |
| Scenarios shipping `cli/manifest.json` | 80 of 123 | `ls scenarios/*/cli/manifest.json \| wc -l` |
| `vrooli-onboarding` manifest | **absent** | `ls scenarios/vrooli-onboarding/cli/manifest.json` |

```bash
tot=0; rpc=0
for m in scenarios/*/cli/manifest.json; do
  t=$(jq '[.. | objects | select(has("binding"))] | length' "$m" 2>/dev/null || echo 0)
  r=$(jq '[.. | objects | select(.binding?.kind=="connect-rpc")] | length' "$m" 2>/dev/null || echo 0)
  tot=$((tot+t)); rpc=$((rpc+r))
done; echo "commands=$tot connect-rpc-bound=$rpc"
```

`scopecatalog.Scope` already carries `{Scenario, Service, Method, Effect, Permissions}`
and `Catalog.Reconcile([]RPCMethod)` reports per-method coverage.

## 5. Module shape for the reach-client relocation

`packages/api-core/go.mod` requires the generated Bridge protocol and the reach
client is now part of the `github.com/vrooli/api-core` module at
`packages/api-core/nodereach`.

The relocation added exactly **one** new module requirement:
`github.com/gorilla/websocket`.

## 6. Credential delivery is complete on both sides

| Stage | Location |
| --- | --- |
| Grant created | `handlers/credentialgrant/module.go:deliverGrant` |
| Value resolved | `api/main.go` binds `ResolveValue` to `credentialauthority.Default()` |
| Node key | `pairingSvc.SealingPublicKey` — X25519, registered at pairing |
| Sealed | `internal/credentialgrant/transport.go:SealPush`, AAD binds node+address+field+generation |
| Delivered | `ServerFrame_CredentialPush`, control-plane signed |
| Opened | `agent/internal/channel/channel.go:631` → `credentialpush.Apply` → `sealing.Open` |
| Stored | durable sink or ephemeral store; plaintext zeroed |

Bridge exposes the grant lifecycle through its `credentials` CLI group and the Web
Console machine detail surface. Secret values remain metadata-only at both surfaces.

Live locked-store probe on `minimouse` (node
`25c7e426-c76c-421a-8351-aaf964589802`) at 2026-09-01: `credentials store status
--format json` returned `initialized=false, unlocked=false`. A durable grant
push for `vrooli/openrouter:api-key` produced receipt for grant
`50aaf503-eb39-46ac-9611-0b825dcc0e81`, but the previously deployed agent
reported only `credential push: store durable value: node credential authority
provision: exit status 1`. The working-tree agent now probes this metadata first
and returns typed `uninitialized` recovery (`vrooli credentials store init`),
or typed `locked` recovery (`vrooli credentials store unlock`); its parser and
agent suite pass. The live receipt remains evidence of the pre-refresh binary,
not of the new typed wording, until the node agent is refreshed.

## 7. RCL asset inventory

| Measurement | Value |
| --- | --- |
| Catalog assets with an implementation | 234 of 467 |
| Forms domain built | 21 of 35 |
| Adapters built | **17 of 17** |
| Consumers of `GeneratedForm` | **2** |
| RCL imports — web-console | configuration form uses shared assets |
| RCL imports — vrooli-bridge / vrooli-onboarding | 15 / 9 |

The plan-required `FormWizard`, `ValidationAdapter`, `PasswordInput`, and `PinInput`
assets now have implementations and are consumed by the two configuration surfaces.
Other catalog assets such as `GeneratedSettingsForm`, `MultiSelect`, and
`AutosaveIndicator` remain outside this plan.

`GeneratedForm`'s field spec is function-valued (`when`, `compute`, `format`,
`renderItem`, `objectChildren`), so it is not driven by a serialized schema as-is.
Its catalog entry already declares `expects: validation-adapter`.

## 8. Live node state — captured before-state

Node `25c7e426-c76c-421a-8351-aaf964589802` (`minimouse`, darwin/amd64), machine
`451ea636-a80f-4080-82b7-fa65d0e3289a`. All seven readiness rungs ready. Remote session
`70e0f746-be87-42e8-802b-2292fa6641d0` created from Web Console with `agy --help`.

Onboarding op `fa2ff4ba-66b3-4080-93f9-83d8ca2bb582` reached configuration and failed
with three blockers: `vrooli/openrouter:api-key`, `vrooli/postgres:password`, and host
safeguard `autoheal_watchdog`. Node projection records `configuration_state=failed`,
unmet `onboarding_apply_failed`.

## 9. Delivered after-state remeasurement — 2026-09-01

The original measurements above remain the captured before-state. The
remeasurement harness is now present at
`remeasure-node-setup-seam.sh`; the current values are reproduced by that
script.

| Measurement | After value | Command |
| --- | ---: | --- |
| Peer records present | 49 | `find ~/.vrooli/peers -maxdepth 1 -type f \| wc -l` |
| Manifest-declared commands | 2,435 | `for m in scenarios/*/cli/manifest.json; do jq '[.. \| objects \| select(has("binding"))] \| length' "$m"; done \| awk '{s+=$1} END {print s}'` |
| Connect-RPC-bound commands | 1,910 | `for m in scenarios/*/cli/manifest.json; do jq '[.. \| objects \| select(.binding?.kind=="connect-rpc")] \| length' "$m"; done \| awk '{s+=$1} END {print s}'` |
| Onboarding CLI manifest | present | `test -f scenarios/vrooli-onboarding/cli/manifest.json` |
| Bridge RCL imports | 15 | `rg -l '@vrooli/react-component-library' scenarios/vrooli-bridge/{ui,api} --glob '*.{ts,tsx,go}' \| wc -l` |
| Onboarding RCL imports | 9 | `rg -l '@vrooli/react-component-library' scenarios/vrooli-onboarding --glob '*.{ts,tsx,go}' \| wc -l` |

The setup wire audit returns no `setup_environment`, `SetupEnvironment`, or
legacy resource/scenario setup fields in Bridge API, CLI, or proto sources. The
only environment expansion is the final bootstrap host seam.

## 10. Current after-state refresh — 2026-09-01

The configuration RPC aliases and generated-selection persistence added after
the first after-state measurement changed the manifest counts. The current
reproducible values are:

| Measurement | Current value |
| --- | ---: |
| Peer records present | 49 |
| Manifest-declared commands | 2,435 |
| Connect-RPC-bound commands | 1,910 |
| Bridge RCL imports | 15 |
| Onboarding RCL imports | 9 |
| Catalog adapter assets | 17 |
| Port-ladder aggregate snapshot | evaluations=210, peer_hits=204, registry_hits=0, cli_hits=0 |

The peer-record count is host state and is not a product regression signal; it
changed from the earlier 13-record observation as lifecycle state changed. The
component-scoped wire and manifest audits remain clean.

The aggregate counter snapshot is cumulative for the shared local counter log;
the controlled `make start` before/after delta below is the cycle-specific
measurement.

The refreshed live Bridge path reached `minimouse` on 2026-09-01 through the
qualified scenario-owned relay verb `vrooli-onboarding wizard status`.
Correlation `18054a5f-0eee-4535-bc82-bbad09a7a725` returned
`RELAY_CALL_OUTCOME_COMPLETED` with `completion=false`, `first_unsatisfied_step=0`,
and `step=0`. Bridge doctor simultaneously reported the node online, channel-held,
protocol-compatible, and dispatchable. This proves remote onboarding command
admission and execution, but not configuration completion: a live credential-store
status probe still returned `initialized=false`, `unlocked=false`, and zero entries.

The same target axis also returned a live system-monitor snapshot with
`system-monitor --node minimouse metrics current --json` at
`2026-09-01T07:56:03.258058220Z`: CPU `7.126%`, memory `33.754%`, and `520`
TCP connections, with cycle id `cycle-1788249363258058220-260`. This is a
typed target-aware read, not a relay/stdout fallback.

The local and remote commands returned the same typed `metrics` object shape;
the sorted field sets were identical:
`connections,cpu,cpu_usage,cycle_id,disk,fragmentation_index,gpu,gpu_usage,major_faults,memory,memory_usage,swap_traffic,tcp_connections,timestamp`.

The lifecycle-managed local runtime also accepted a full generated selection
handoff at `POST http://127.0.0.1:19798/api/v2/handoff` on 2026-09-01. The
response preserved `schema_version=v1`, `target=local`, scenarios, host tools,
credential addresses, trust posture, update control, session mode, operating
mode, and `apply=true`. This proves the local contract path; it is not a
substitute for the pending `minimouse` end-to-end proof.

The port-ladder counter is exposed by `cliutil.PortLookupStats()` through the
existing control-plane `/metrics/processes` surface. Lookup events are also
written to the shared local append-only counter log, so the surface observes
lookup work performed by lifecycle and agent processes rather than only the API
process that serves the diagnostic request.

The refreshed control-plane `/metrics/processes` surface returned aggregate
events after the representative starts. In the controlled `make -C
scenarios/system-monitor start` cycle, the counter moved from
`{evaluations:20,peer_hits:18,registry_hits:0,cli_hits:0}` to
`{evaluations:21,peer_hits:19,registry_hits:0,cli_hits:0}`. The cycle therefore
performed one uncached lookup and answered it from rung 1 (100%); its scenario
reported unhealthy because its already-running dependency state was degraded,
but the scenario itself published API_PORT=16914 and UI_PORT=21232. A broader
best-effort `vrooli scenario start system-monitor` cycle also produced 6
evaluations with 4 peer hits (66.7%), which remains a majority.

A representative `web-console` start was attempted through `vrooli scenario
start`. It progressed past the fixed source-only `@vrooli/flow-runtime`
provisioning defect but timed out while optional audio dependencies were being
admitted; no claim about that heavier scenario is made here.
