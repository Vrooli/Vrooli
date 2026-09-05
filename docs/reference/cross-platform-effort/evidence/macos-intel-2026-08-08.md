# macOS Intel hardware evidence — 2026-08-08

## Host

- Host: `minimouse` / Macmini8,1
- macOS: 15.7.2, Darwin 24.6.0
- Architecture: Intel x86_64, 6 cores, 8 GiB RAM
- Free home-volume space at discovery: approximately 201 GiB
- Docker: absent by choice; Docker-dependent resources are not applicable to
  this targeted development profile.
- Apple Silicon: no Apple Silicon hardware was available; those rows remain
  unqualified.

## Installation and convergence

The host began with an interrupted partial installation: 98 files under
`~/.vrooli/bin`, a Bridge LaunchAgent, runtime state, but no `vrooli` binary or
source checkout. Bridge working-tree operations
`89100f52-8d75-4f02-b89f-a24fc584ee49`,
`fa7e5857-e28c-474a-a789-88885c80f305`, and
`9119401a-c1a3-4f49-ade9-e9e5cf792c42` converged it without deleting the old
state. The final operation used the local working tree and completed with
durable `SUCCEEDED` status.

The setup profiles were exercised as:

```text
vrooli --no-stale-check setup --environment minimal --resources none --scenarios none --result-file <result>
vrooli --no-stale-check setup --environment development --resources none --scenarios none --result-file <result>
```

For the headless host, the credential-store passphrase was supplied through
the dedicated stdin channel. It was never placed in argv, an environment
variable, a log, or this document.

## Development validation

After a native project build, the managed lifecycle command
`vrooli --no-stale-check develop --environment development --resources none
--scenarios none` reported:

```text
Vrooli API healthy on port 8092
Vrooli orchestrator skipped (--scenarios none)
```

The health endpoint then reported `healthy`, zero orphan processes, and zero
zombies. Darwin port diagnostics reported one expected API listener and zero
orphans. The Mac CLI reported a Mach-O x86_64 Vrooli CLI, and the final Bridge
projection was `minimouse`, Darwin `amd64`, `NODE_STATUS_ONLINE`.

## Supervisor and security observations

The final agent supervisor was:

```text
/Library/LaunchDaemons/com.vrooli.bridge.vrooli-bridge-agent.plist
state = running
```

Its stdout/stderr destinations were under
`/Users/matthalloran8/.local/state/vrooli-bridge-agent/`. The encrypted
credential store was initialized. Headless launchd required a system-daemon
fallback because `gui/<uid>` is unavailable in an SSH-only session; that
fallback is explicitly recorded as a non-setup privileged finding.

## Limits and tier

This document records targeted setup, convergence, native build, managed
development, process diagnostics, supervisor, and Bridge online evidence. It
does not claim Docker resource support, Apple Silicon support, a physical
reboot, or a repository-wide baseline pass. The broad authored baseline was
intentionally abandoned at the operator's direction and is not a release gate.

The temporary effort credentials were torn down after validation. The Mac
effort key is refused, the local effort store and sudo drop-in are absent, and
Bridge-only verification still reports the Mac node ONLINE.

## Phase 7 probe — 2026-08-19

The current Bridge machine identity is
`451ea636-a80f-4080-82b7-fa65d0e3289a`, and the current node identity is
`25c7e426-c76c-421a-8351-aaf964589802`. Working-tree onboarding operations
`479d41b1-514d-4fc8-ab88-a192bee7fca3` and
`ed244981-924f-4384-95de-9c149740b64a` both completed successfully; the latter
redeployed the node-agent session fixes from this plan. The Web Console remote
session used for the typed probe was
`remote:19f9ff71-7c05-4dfb-a811-8fd57e0df822`.
The follow-up working-tree onboarding operation
`648a4b95-19d2-4620-8c55-fed202326be6` also completed successfully after the
probe-derived host-inventory fact fix.

The probe returned the following non-secret host identity:

```text
model: Macmini8,1
architecture: x86_64 / darwin amd64
macOS: 15.7.2 (24G325)
Darwin kernel: 24.6.0
memory: 8589934592 bytes
```

`xcrun` was present, but both required developer tools were unusable:
`xcrun xcodebuild -version` exited 72 with “unable to find utility
`xcodebuild`”, and `xcrun simctl list runtimes` exited 72 with “unable to find
utility `simctl`”. No simulator was booted, no app or screenshot was produced,
and no iOS capability row was promoted. This is a typed unsupported result,
not an inference from the Darwin declaration.

The same result was captured through the typed Bridge host-inventory operation
`61bd41b7-f600-4125-8013-90f2f889281c`: `xcodebuild` was recorded
`present=false` with its unusable `/usr/bin/xcodebuild` shim, while `xcrun` was
present and no `simctl` runtime was available. The live
`scenario-to-ios` target inventory then classified minimouse as
`available=false`, missing `xcodebuild`, with degraded health and no promoted
iOS capability. The target response was observed at
`2026-08-19T08:36:31.021326251Z`.

After the later working-tree re-onboarding, the fresh typed host-inventory run
`b44c8516-ef1d-4c54-88eb-8191ad3f7015` again passed the inventory operation and
recorded `darwin/amd64`, 6 cores, 8192 MB, Intel UHD Graphics 630, and the
warning `xcodebuild -version: exit status 1`. This re-measurement preserves the
typed unsupported classification after the current CLI deployment.

The live Bridge agent is installed at
`/Users/matthalloran8/.local/bin/vrooli-bridge-agent`; its system LaunchDaemon
is `/Library/LaunchDaemons/com.vrooli.bridge.vrooli-bridge-agent.plist` and
the fresh remote-session probe observed `state = running`. The same probe
reported owner `matthalloran8`, group `staff`, mode `-rwx------` for the agent,
and owner `matthalloran8`, group `staff`, mode `-rwxr-xr-x` for
`/Users/matthalloran8/.vrooli/bin/vrooli`. A fresh interactive remote PTY
(not a post-reboot login-shell observation) reported PATH
`/Users/matthalloran8/.vrooli/bin:/usr/bin:/bin:/usr/sbin:/sbin`, and the
governed `vrooli agent launch` command therefore resolved through the native
user-local CLI path. No physical reboot was performed.

The fresh Bridge-backed Web Console session used for this evidence was
`remote:af2b4bbe-e5b6-4edf-842c-726a05bd9fe3`. It executed the identity,
LaunchDaemon, PATH, Docker, sandbox, and Workspace Sandbox lifecycle probes.
The Bridge registry was temporarily reporting `heartbeat fresh=false` and
`channel held=false` after the control-plane restart even though the existing
session executed successfully; this stale-presence condition is recorded as a
limitation rather than promoted to an online qualification claim.

The runtime-supervisor checks were explicit negative results:
`launchctl print gui/501/com.vrooli.runtime-supervisor` returned “Domain does
not support specified action”, and the system-domain lookup reported that the
service was not found. `/usr/bin/sandbox-exec` was present, and the remote
`workspace-sandbox` lifecycle reported `Status: running` and `Health: healthy`,
but the required Seatbelt write-denial, network-denial, approval, and
provenance shakeout was not executed in this probe. Docker was unavailable
(`DOCKER_UNAVAILABLE`), so Docker-backed PostgreSQL/Redis provider,
bootstrap, readiness, and durable-data evidence is not claimed. These rows
remain unqualified.

With the Bridge node healthy again, a fresh Web Console session
`remote:9fc2c8d9-ab7b-4228-9868-0fb5f289ec29` launched the governed command
`vrooli agent launch --runner claude --arg=--dangerously-skip-permissions` and
streamed Claude Code v2.0.57's interactive welcome screen. Claude reported
`Missing API key · Run /login`; no third-party credential was available, so no
model-generated hello was claimed. The same command also passed through the
typed Bridge dispatch surface as run
`a5dd42e0-9fb0-4b46-8d26-519709142a01`, with accepted audit record
`bb4c3736-77a5-4c48-a648-5bc07f07e3d7`. A reduced-scope dispatch was refused by
audit record `1923f4ad-7669-4136-99c5-b9a4d7fe47ca` with the typed reason
`missing "vrooli:write"`. These records prove command admission and refusal;
they do not substitute for the pending interactive sign-in.

The runner-specific prompt correction was then deployed through working-tree
onboarding operation `8ca165e6-877c-4ce2-aeb9-8e1844100c59`, which completed
with dirty revision `872271a917a1117d17dd1897037ef21030e86548+dirty` and a
native Darwin CLI. A governed Web Console/Codex execution
(`remote:4e06591c-a70e-40c6-b3a3-bd4f2896b2a5`) reached OpenAI Codex v0.64.0
with model `gpt-5.1-codex-max` and provider `openai`, then failed before a
model response because the existing refresh token had already been used.
This is additional command-path evidence, not proof of the required
one-time operator sign-in or real response.

The relevant Test Genie evidence includes the earlier Web Console run
`20260819-073817-4f5cd9ab` and the fresh authoritative run
`20260819-093106-a2229671`; both terminated broadly red (10/23 phases passed)
outside this remote-session path. The targeted delivery-ramp host-capability
classifier tests, manifest-derived CLI invocation conformance, and focused
Bridge/Web Console checks pass. The fresh Git Control Tower Bridge boundary
diff `20260819-111345-349c2db0` also passed all 20 of 20 phases after temporary
probe helpers were removed. The root `vrooli capability ledger --json`
also remains healthy, but it is a generic manifest/policy ledger and does not
claim node-specific iOS capability; that classification comes from the live
typed `scenario-to-ios` probe above. The pre-plan matrix therefore remains
unchanged for every unproven row.

After the baseline publication guard was added, the selector-free recapture
`trustworthy-remote-invocation-one-invocation-path-per-baseline-recapture-20260819`
completed with terminal `passed` and complete coverage for all eight required
members. Its child runs were
`20260819-122629-9219f72e` (browser-automation-studio),
`20260819-122630-e051eae8` (git-control-tower),
`20260819-122630-c533fcba` (infrastructure-manager),
`20260819-122631-b55d2bcf` (scenario-completeness-scoring),
`20260819-122632-3fc1f687` (scenario-dependency-analyzer),
`20260819-122633-c303c1b1` (structure-health),
`20260819-122950-6152d398` (vrooli-bridge), and
`20260819-123035-4532dc07` (web-console), all at revision
`872271a917a1117d17dd1897037ef21030e86548`. The phase-6 validation operation
`ce7a8908-00ba-4171-9a26-30fd15f1c908` then passed at fresh scope generation 2:
`vrooli-bridge` was `clean` and `web-console` was `preexisting`.

## Current operator handoff and runner availability — 2026-08-19

The remaining interactive acceptance is deliberately left as an operator
handoff. A managed Web Console restart invalidated the earlier in-memory
handoff session and a fresh session was created:
`remote:d7199318-5625-4278-b6a1-be6a11cf3dbc`. It is configured for
`minimouse`, has origin `SESSION_ORIGIN_REMOTE`, and is currently waiting at
Codex authentication. The initial local-login server flow was interrupted
because its `localhost:1455` callback belongs to the remote node; the same
session was then switched to `codex login --device-auth`. The terminal now
shows the official `https://auth.openai.com/codex/device` URL and a one-time
code. The code is intentionally not recorded in this document or any
effort-scoped file. It therefore contains no authenticated model-response
evidence yet. The operator must connect to this current session from Web
Console, complete the one-time device sign-in, and then perform the
hello-world response, browser reconnect, and node-agent-restart checks.

During this restart/reconnect validation, Web Console was hardened so enrolled
`LocalSession` authorization is refreshed from the shared operator-session
store immediately before Bridge HTTP and WebSocket requests. This removes the
15-minute static-token expiry window without introducing a second sign-in;
the focused refresh regression test and managed Web Console build pass.

The available runner paths were checked through the same governed command:

- Claude Code (`remote:9fc2c8d9-ab7b-4228-9868-0fb5f289ec29`) reached its
  welcome screen and stopped at `Missing API key`.
- Codex (`remote:4e06591c-a70e-40c6-b3a3-bd4f2896b2a5`) reached
  `gpt-5.1-codex-max` but stopped because its refresh token had already been
  used.
- Grok (`remote:a04dc713-4851-40b6-8719-c275978ba8e1`) and OpenCode
  (`remote:8afa01f6-c99f-485e-afc7-f86c936f498d`) were not installed on the
  node; the Codex OSS attempt
  (`remote:3686ab0a-9324-4e16-b69e-5f7c06ba2534`) had no configured local
  provider.

These are availability findings, not claimed coding-agent responses. No
simulator screenshot artifact exists because the typed probe found no usable
`xcodebuild` or `simctl`; the typed host-inventory operation
`b44c8516-ef1d-4c54-88eb-8191ad3f7015` is the current evidence reference for
that unsupported result. No platform-support row was promoted, so the matrix
retains its pre-plan tiers and no new artifact reference is asserted for an
unproven row. The root capability ledger's generic semantics are intentionally
left unchanged; node-level iOS status remains owned by the probe-derived
validation-matrix/`scenario-to-ios` path rather than being represented as a
static root capability declaration.
