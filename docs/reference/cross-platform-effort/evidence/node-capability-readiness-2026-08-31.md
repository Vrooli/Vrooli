# Node capability readiness — observed state, 2026-08-31

This record is the evidence base for the node-capability-readiness plan. Every
figure below was read from the live control plane, the live `minimouse` node,
and the working tree on branch `agi` on 2026-08-31. Nothing was installed,
applied, or changed on any machine while gathering it.

The narrative companion is
[`../node-capability-readiness-2026-08-31.html`](../node-capability-readiness-2026-08-31.html).

## Observed symptom

Web Console created a remote session on `minimouse` with a coding agent that is
not installed there. Every readiness light was green. The launcher offered the
agent because its list is a static array, not a projection of the target.

## Command transcripts

### Registry and onboarding history

```text
### nodes list
{
  "nodes": [
    {
      "id": "25c7e426-c76c-421a-8351-aaf964589802",
      "name": "minimouse",
      "os": "darwin",
      "arch": "amd64",
      "scopes": [
        "vrooli-bridge:read",
        "vrooli-bridge:write",
        "*:read",
        "*:write"
      ],
      "status": "NODE_STATUS_ONLINE",
      "online": true,
      "created_at": "2026-08-17T15:39:06.606107059Z",
      "updated_at": "2026-08-31T00:17:17.380180133Z",
      "last_seen_at": "2026-08-31T04:08:32.714884Z",
      "registry_record_present": true,
      "heartbeat_fresh": true,
      "heartbeat_age_seconds": "11",
      "channel_held": true,
      "protocol_compatible": true,
      "dispatchable": true,
      "kind": "NODE_KIND_AGENT",
      "machine_arch": "amd64",
      "binary_arch": "amd64"
    },
    {
      "id": "697b6224-6283-4a31-90e2-73724e424c05",
      "name": "swarminator",
      "os": "linux",
      "arch": "amd64",
      "status": "NODE_STATUS_OFFLINE",
      "created_at": "2026-08-10T15:55:49.348005106Z",
      "updated_at": "2026-08-25T01:46:52.819226024Z",
      "last_seen_at": "2026-08-19T18:42:21.589036906Z",
      "registry_record_present": true,
      "heartbeat_age_seconds": "984382",
      "kind": "NODE_KIND_CONTROL_PLANE"
    }
  ]
}

### onboarding op summary
total ops: 161
    68  minimouse.local    ONBOARDING_STATE_SUCCEEDED 
    32  minimouse.local    ONBOARDING_STATE_FAILED    onboarding_apply_failed
    31  minimouse.local    ONBOARDING_STATE_FAILED    bootstrap_failed
    11  minimouse.local    ONBOARDING_STATE_FAILED    pairing_failed
     7  minimouse.local    ONBOARDING_STATE_FAILED    ssh_setup_failed
     6  minimouse.local    ONBOARDING_STATE_FAILED    verify_online_failed
     3  minimouse.local    ONBOARDING_STATE_FAILED    prebuilt_artifacts_failed
     1  swarminator        ONBOARDING_STATE_FAILED    ssh_setup_failed
     1  minimouse.local    ONBOARDING_STATE_FAILED    interrupted_by_restart
     1  127.0.0.1          ONBOARDING_STATE_SUCCEEDED 

latest 6:
  19bc425e-f22e-488f-980f-c02385837f10 2026-08-30T00:16:13 FAILED onboarding_apply_failed
  f727c2a8-5f73-4619-9887-e5e61a5c7b86 2026-08-30T00:12:43 FAILED onboarding_apply_failed
  9ac3e94a-2b83-423a-8f9c-7594a70bbd4a 2026-08-30T00:09:31 FAILED onboarding_apply_failed
  ca94b18f-b937-42ab-a415-1b9a68f6c57d 2026-08-30T00:04:32 FAILED onboarding_apply_failed
  2d4dff27-15a3-4d9a-bbed-12ac31cf3d64 2026-08-29T23:53:38 FAILED onboarding_apply_failed
  fc63a1c0-2a72-4777-8c19-f31d3523046a 2026-08-29T23:53:25 FAILED ssh_setup_failed

```

### Coding-agent availability, control-plane host vs minimouse

```text
### resource status: local
  antigravity    enabled=True cli_installed=True installed=True running=True healthy=True code=ok :: available
  claude-code    enabled=True cli_installed=True installed=True running=True healthy=True code=ok :: available
  codex          enabled=True cli_installed=True installed=True running=True healthy=True code=ok :: available
  grok           enabled=True cli_installed=True installed=True running=True healthy=True code=ok :: available
  opencode       enabled=True cli_installed=True installed=True running=True healthy=True code=ok :: available

### resource status: minimouse (via bridge relay)
  antigravity    enabled=True cli_installed=True installed=False running=False healthy=None code=unavailable :: agy is unavailable
  claude-code    enabled=True cli_installed=True installed=True running=True healthy=True code=ok :: available
  codex          enabled=True cli_installed=True installed=True running=True healthy=True code=ok :: available
  grok           enabled=True cli_installed=True installed=False running=False healthy=None code=unavailable :: grok is unavailable
  opencode       enabled=True cli_installed=True installed=False running=False healthy=None code=unavailable :: opencode is unavailable
```

Every agent reports `enabled=True` and `cli_installed=True` on both hosts.
The Vrooli wrapper CLI is installed for all five on `minimouse`; the upstream
binary the wrapper runs is absent for three.

### Step traces — the failed and the succeeded op run the same steps

```text
=== op 19bc425e-f22e-488f-980f-c02385837f10 ===
state= ONBOARDING_STATE_FAILED failure= onboarding_apply_failed exit= 2
  1 ssh-setup              STARTED  establishing passwordless SSH
  2 ssh-setup              OK       passwordless SSH established; sudo: already-passwordless
  3 candidate-admission    STARTED  probing Bridge endpoint from candidate node
  4 candidate-admission    OK       endpoint http://192.168.1.173:18767; candidate source 192.168.1.176; curl-exit-0
  5 push-script            STARTED  copying bootstrap script to node
  6 push-script            OK       bootstrap script staged on node
  7 sync-tree              STARTED  shipping control-plane working tree to node
  8 sync-tree              OK       shipped 57038 file(s), 491.1 MiB, digest c0c04f586a72 → /Users/matthalloran8/vrooli
  9 prebuilt-artifacts     STARTED  cross-building prebuilt binaries for node
 10 prebuilt-artifacts     OK       received prebuilt binaries for darwin/amd64 (fingerprint 7576d160a1d7)
 11 run                    STARTED  vrooli-bridge node bootstrap
 12 detect-os              STARTED  identify platform
 13 detect-os              OK       os=darwin arch=amd64
 14 prebuilt-artifacts     STARTED  verify transferred prebuilt binaries
 15 prebuilt-artifacts     OK       received prebuilt binaries for darwin/amd64 (fingerprint 7576d160a1d7)
 16 prereqs                STARTED  ensure git/curl (clone prerequisites)
 17 prereqs                SKIPPED  pre-synced tree + prebuilt binaries require no clone prerequisites
 18 clone                  STARTED  clone/converge https://github.com/Vrooli/Vrooli.git
 19 clone                  OK       using pre-synced working tree at /Users/matthalloran8/vrooli (5e19c2469d7560611b42c07843b4b4be0f4d7b37+dirty, digest c0c04f586a7228950396c0dc2e1eab703
 20 setup                  STARTED  vrooli setup
 21 setup                  SKIPPED  --skip-setup (node cannot run jobs until setup is run later)
 22 toolchain              STARTED  verify build toolchains resolve
 23 toolchain              OK       build toolchains resolve — recovered off-PATH and amended PATH: go (/Users/matthalloran8/.vrooli/bin)
 24 native-vrooli          STARTED  build the host-native Vrooli CLI
 25 native-vrooli          OK       native CGO-enabled Vrooli CLI installed at /Users/matthalloran8/.vrooli/bin/vrooli
 26 setup-finalize         STARTED  complete setup with the host-native CLI
 27 setup-finalize         SKIPPED  --skip-setup (native setup deferred)
 28 build-agent            STARTED  prepare node-agent
 29 build-agent            SKIPPED  received prebuilt /Users/matthalloran8/.local/lib/vrooli-bridge/bootstrap/artifacts-aceec84e49d7c963/vrooli-bridge-agent; no node-side build
 30 stable-agent           STARTED  install stable typed-helper launcher
 31 stable-agent           OK       typed helper launcher installed at /Users/matthalloran8/.local/bin/vrooli-bridge-agent
 32 build-cli              STARTED  prepare vrooli-bridge CLI
 33 build-cli              SKIPPED  received prebuilt /Users/matthalloran8/.local/lib/vrooli-bridge/bootstrap/artifacts-aceec84e49d7c963/vrooli-bridge; no node-side build
 34 node-key               STARTED  generate/load node keypair
 35 node-key               OK       node public key ready (fingerprint 8156c3e953344ae5)
 36 pair-redeem            STARTED  redeem pairing code + pin control-plane key
 37 pair-redeem            OK       paired as 25c7e426-c76c-421a-8351-aaf964589802
 38 pin-verify             STARTED  verify pinned control-plane key
 39 pin-verify             OK       pinned key present, node 25c7e426-c76c-421a-8351-aaf964589802
 40 provisioner-install    STARTED  install privileged provisioning helper
 41 provisioner-install    SKIPPED  BRIDGE_PROVISION_SERVICE_USER is unset; provisioning remains unavailable until a separate principal is configured
 42 service-install        STARTED  install + start node-agent service
 43 service-install        OK       service installed and running (presence-only=false)
 44 autostart              STARTED  enable headless auto-start
 45 autostart              OK       system LaunchDaemon KeepAlive handles restart; no GUI login or auto-login is required
 46 install-record         STARTED  record bootstrap-owned artifacts
 47 install-record         OK       bootstrap-owned paths recorded at /Users/matthalloran8/.vrooli/state/install-record.json
 48 verify-online          STARTED  confirm dial-out channel is live
 49 verify-online          OK       service running (journal unavailable: could not confirm channel log directly)
 50 run                    OK       node 25c7e426-c76c-421a-8351-aaf964589802 paired and online
 51 verify-online-confirm  STARTED  verifying Bridge key after bootstrap and pairing
 52 verify-online-confirm  STARTED  confirming node is online in the fleet
 53 verify-online-confirm  OK       node is online with control-plane key pinned and final SSH trust verified
 54 apply-selection        STARTED  applying the committed onboarding selection
 55 apply-selection        FAILED   remote onboarding readiness exited 2: ✅ installed CLI to /Users/matthalloran8/.vrooli/bin/vrooli-onboarding | ℹ️  If you are replacing an existing CLI c
 56 run                    FAILED   remote onboarding readiness exited 2: ✅ installed CLI to /Users/matthalloran8/.vrooli/bin/vrooli-onboarding | ℹ️  If you are replacing an existing CLI c
=== op 7c92408a-01cf-4929-89b2-882f3757ad51 ===
state= ONBOARDING_STATE_SUCCEEDED failure=  exit= None
  1 ssh-setup              STARTED  establishing passwordless SSH
  2 ssh-setup              OK       passwordless SSH established; sudo: already-passwordless
  3 candidate-admission    STARTED  probing Bridge endpoint from candidate node
  4 candidate-admission    OK       endpoint http://192.168.1.173:18767; candidate source 192.168.1.176; curl-exit-0
  5 push-script            STARTED  copying bootstrap script to node
  6 push-script            OK       bootstrap script staged on node
  7 sync-tree              STARTED  shipping control-plane working tree to node
  8 sync-tree              OK       shipped 55076 file(s), 476.3 MiB, digest 14aa4083b6d4 → /Users/matthalloran8/vrooli
  9 prebuilt-artifacts     STARTED  cross-building prebuilt binaries for node
 10 prebuilt-artifacts     OK       received prebuilt binaries for darwin/amd64 (fingerprint db577f1d0412)
 11 run                    STARTED  vrooli-bridge node bootstrap
 12 detect-os              STARTED  identify platform
 13 detect-os              OK       os=darwin arch=amd64
 14 prebuilt-artifacts     STARTED  verify transferred prebuilt binaries
 15 prebuilt-artifacts     OK       received prebuilt binaries for darwin/amd64 (fingerprint db577f1d0412)
 16 prereqs                STARTED  ensure git/curl (clone prerequisites)
 17 prereqs                SKIPPED  pre-synced tree + prebuilt binaries require no clone prerequisites
 18 clone                  STARTED  clone/converge https://github.com/Vrooli/Vrooli.git
 19 clone                  OK       using pre-synced working tree at /Users/matthalloran8/vrooli (36d84ab6bb049e13c5794c6adfe7df5eaf38388e+dirty, digest 14aa4083b6d48a243d2724a18c9b2972e
 20 setup                  STARTED  vrooli setup
 21 setup                  SKIPPED  --skip-setup (node cannot run jobs until setup is run later)
 22 toolchain              STARTED  verify build toolchains resolve
 23 toolchain              OK       build toolchains resolve — recovered off-PATH and amended PATH: go (/Users/matthalloran8/.vrooli/bin)
 24 native-vrooli          STARTED  build the host-native Vrooli CLI
 25 native-vrooli          OK       native CGO-enabled Vrooli CLI installed at /Users/matthalloran8/.vrooli/bin/vrooli
 26 setup-finalize         STARTED  complete setup with the host-native CLI
 27 setup-finalize         SKIPPED  --skip-setup (native setup deferred)
 28 build-agent            STARTED  prepare node-agent
 29 build-agent            SKIPPED  received prebuilt /Users/matthalloran8/.local/lib/vrooli-bridge/bootstrap/artifacts-b1879534c90dddd0/vrooli-bridge-agent; no node-side build
 30 stable-agent           STARTED  install stable typed-helper launcher
 31 stable-agent           OK       typed helper launcher installed at /Users/matthalloran8/.local/bin/vrooli-bridge-agent
 32 build-cli              STARTED  prepare vrooli-bridge CLI
 33 build-cli              SKIPPED  received prebuilt /Users/matthalloran8/.local/lib/vrooli-bridge/bootstrap/artifacts-b1879534c90dddd0/vrooli-bridge; no node-side build
 34 node-key               STARTED  generate/load node keypair
 35 node-key               OK       node public key ready (fingerprint 8156c3e953344ae5)
 36 pair-redeem            STARTED  redeem pairing code + pin control-plane key
 37 pair-redeem            OK       paired as 25c7e426-c76c-421a-8351-aaf964589802
 38 pin-verify             STARTED  verify pinned control-plane key
 39 pin-verify             OK       pinned key present, node 25c7e426-c76c-421a-8351-aaf964589802
 40 provisioner-install    STARTED  install privileged provisioning helper
 41 provisioner-install    OK       helper running as root (uid 0); runner uid 501; socket /Users/matthalloran8/.local/state/vrooli-bridge-agent/provision.sock
 42 service-install        STARTED  install + start node-agent service
 43 service-install        OK       service installed and running (presence-only=false)
 44 autostart              STARTED  enable headless auto-start
 45 autostart              OK       system LaunchDaemon KeepAlive handles restart; no GUI login or auto-login is required
 46 install-record         STARTED  record bootstrap-owned artifacts
 47 install-record         OK       bootstrap-owned paths recorded at /Users/matthalloran8/.vrooli/state/install-record.json
 48 verify-online          STARTED  confirm dial-out channel is live
 49 verify-online          OK       service running (journal unavailable: could not confirm channel log directly)
 50 run                    OK       node 25c7e426-c76c-421a-8351-aaf964589802 paired and online
 51 verify-online-confirm  STARTED  verifying Bridge key after bootstrap and pairing
 52 verify-online-confirm  STARTED  confirming node is online in the fleet
 53 verify-online-confirm  OK       node is online with control-plane key pinned and final SSH trust verified
 54 break-glass-provision  SKIPPED  operator declined the break-glass passphrase; target-bound break-glass remains missing
```

### Upstream artifact routes

```text
# upstream artifact URL probe (HTTP HEAD, follow redirects)
404  https://github.com/sst/opencode/releases/download/v1.17.9/opencode-darwin-amd64.tar.gz
404  https://github.com/sst/opencode/releases/download/v1.17.9/opencode-linux-amd64.tar.gz
404  https://x.ai/cli/grok-0.2.72-darwin-amd64
404  https://x.ai/cli/grok-0.2.72-linux-amd64
404  https://antigravity-cli-auto-updater-974169037036.us-central1.run.app/artifacts/darwin_amd64.tar.gz
404  https://antigravity-cli-auto-updater-974169037036.us-central1.run.app/artifacts/linux_amd64.tar.gz

# real sst/opencode release assets (latest)
latest tag: v1.18.25
   opencode-darwin-arm64.zip
   opencode-darwin-x64-baseline.zip
   opencode-darwin-x64.zip
   opencode-linux-arm64-musl.tar.gz
   opencode-linux-arm64.tar.gz
   opencode-linux-x64-baseline-musl.tar.gz
   opencode-linux-x64-baseline.tar.gz
   opencode-linux-x64-musl.tar.gz
   opencode-linux-x64.tar.gz
   opencode-windows-arm64.zip
   opencode-windows-x64-baseline.zip
   opencode-windows-x64.zip
```

## Code-level findings index

| ID | Finding | Primary location |
|---|---|---|
| RC-01 | `runInstallCommand` returns on the source-build branch before reading `install.platforms`, so `install-direct` never runs on a source checkout | `internal/resources/drivers_cli.go:468` |
| RC-02 | opencode, grok, and antigravity artifact URL templates resolve to HTTP 404 on every platform | `resources/{opencode,grok,antigravity}/cli/main.go`, `internal/resources/agentinstall/install.go:186` |
| RC-03 | All five `ai-cli` tool manifests are `manual: true`, so `vrooli setup` never installs a coding agent | `internal/tools/{claude,codex,opencode,grok,agy}/tool.json`, `internal/hostreqkit/installers.go:40` |
| RC-04 | web-console declares no `ai-cli` host tool or coding-agent resource dependency | `scenarios/web-console/.vrooli/service.json` |
| RC-05 | The shared readiness vocabulary has seven identities, all transport and trust | `packages/api-core/targetmodel/model.go:59` |
| RC-06 | The local target hardcodes one always-true readiness fact | `scenarios/web-console/api/target_catalog.go:85` |
| RC-07 | The registry `Node` message has no inventory, applied-config, or setup-outcome field | `packages/proto/schemas/vrooli-bridge/v1/registry/registry.proto:100` |
| RC-08 | `HealthSnapshot` already carries LookPath probes; `presence.Hub.Health()` has no callers | `packages/proto/schemas/vrooli-bridge/v1/shared/shared.proto:27`, `scenarios/vrooli-bridge/api/internal/presence/hub.go:356` |
| RC-09 | The applied selection is derived from the control-plane host's own operator state | `scenarios/vrooli-onboarding/api/v2_handoff.go:99` |
| RC-10 | Machine profiles are six hardcoded built-ins carrying no scenarios, resources, or capabilities | `scenarios/vrooli-bridge/api/internal/machines/policy.go:30` |
| RC-11 | No `applied_profile_version`; the successful-apply readiness report is discarded | `packages/proto/schemas/vrooli-bridge/v1/machines/machines.proto:41`, `scenarios/vrooli-bridge/api/internal/onboard/orchestrator.go:352` |
| RC-12 | `FromSetupProfile` drops `all`/`none`/`enabled`, so a whole-fleet profile applies nothing and the op reports success | `scenarios/vrooli-bridge/api/internal/onboarding/client.go:118` |
| RC-13 | `unsupported` and `missing` share a bucket, so a platform-inapplicable required item blocks forever | `scenarios/vrooli-onboarding/api/completion.go:67` |
| RC-14 | Two safeguards' macOS `platform_status.evidence` strings describe Windows | `internal/safeguards/{workspace-sandbox-userns,tpm-credential-access}/safeguard.json` |
| RC-15 | A credential-backend outage is reported as a missing credential with a "type it in" remediation | `scenarios/vrooli-onboarding/api/v2_readiness.go:270` |
| RC-16 | Onboarding failure lives on the op and is never projected onto the node | `scenarios/vrooli-bridge/api/internal/onboard/types.go:188` |
| RC-17 | A remote launch pastes the command into a PTY; a missing binary is scrollback text, not an error | `scenarios/web-console/api/handlers/sessions/adapter.go:188`, `internal/cli/vroolicli/agent.go:66` |
| RC-18 | The launcher agent grid is a static array of nine shell strings | `scenarios/web-console/ui/src/consts/shortcuts.ts:21` |
| RC-19 | `cliapp.GlobalOptions.Node` is parsed and advertised in every scenario CLI and read by nothing | `packages/cli-core/cliapp/app.go:157` |
| RC-20 | `hostreq` merges manifest `platforms` for safeguards only, so a tool manifest's platform gate is ignored | `internal/hostreq/resolve.go:362` |
| RC-21 | Bridge relay truncates human-format output; only `--json` inner commands are stable | `scenarios/vrooli-bridge/api/internal/relay` |

## Mechanisms that already exist and are unused

| Mechanism | Location | State |
|---|---|---|
| Per-agent availability with install hint | `scenarios/agent-manager/api/internal/adapters/runner/codecs/base.go:184`, `RunnerStatus` in `packages/proto/schemas/agent-manager/v1/domain/run.proto:550` | Works; local only; web-console does not consume it |
| Node LookPath probe channel | `scenarios/vrooli-bridge/agent/internal/health/health.go` | Ships every heartbeat; stored in `Hub.health`; read by nothing |
| Readiness fact contract with a rendering UI | `packages/api-core/targetmodel/model.go`, `scenarios/web-console/ui/src/components/launcher/MachinePicker.tsx:74` | Renders the first failing fact; no capability facts exist to render |
| Capability registry with status and operator action | `scenarios/web-console/api/internal/capabilities/registry.go:41` | Models ollama, openrouter, tmux, Bridge; no coding agents |
| Platform gate producing not-applicable | `internal/runtime/runtime.go:534`, `internal/hostreqkit/helpers.go:163` | Complete; the two blocking safeguards declare no `platforms` |
| Typed unsupported-platform install error | `internal/resources/agentinstall/install.go:38` | Returned before any download; onboarding does not distinguish it |
| Governed remote read path | `vrooli-bridge relay call --scenario vrooli --command "resource status"` | Allowlisted; produced the tables above |
