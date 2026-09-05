# Remote session portability proof attempt — 2026-08-29

This record documents the live validation attempt for the remote-session
portability pair. It is evidence of the observed state, not a supported-cell
claim.

- Console: Linux amd64 (`web-console`)
- Node: `minimouse`, node id `25c7e426-c76c-421a-8351-aaf964589802`
- Node platform: Darwin amd64
- Bridge readiness after restart: `NODE_STATUS_NEEDS_UPDATE`; the agent reports
  no negotiated protocol revision, so the fail-closed compatibility rung does
  not admit a session.
- Bridge doctor before restart: PASS while the node was online and dispatchable.
- Attempted command:

  ```text
  web-console session create --target bridge-node:25c7e426-c76c-421a-8351-aaf964589802 --backend remote --launch-command 'printf REMOTE_MAC_PROOF' --execute-launch-command --label 'plan-live-darwin-proof' --json
  ```

- Result: refused with `failed_precondition: remote target unavailable:
  heartbeat freshness` after the Bridge restart, following an earlier attempt
  that reached the Bridge session manager and was refused because the running
  agent did not satisfy the new transport/protocol contract.

After the current Bridge restart and the architecture-contract wiring, a
repeat attempt is refused earlier and more specifically with:

```text
failed_precondition: remote target unavailable: protocol compatibility
```

This is the expected fail-closed result for the still-unrefreshed Darwin agent;
the control plane must not infer a protocol or machine architecture from the
legacy `arch` field.

## Completed live proof

The trusted operator identity was `matthalloran8`; onboarding with the default
`root` user was the earlier false lead. Working-tree onboarding then completed
SSH setup, tree synchronization, Darwin/amd64 artifact transfer, pairing,
service installation, and online verification. Its final readiness step still
reported unrelated host setup blockers (unavailable credentials and host
safeguards), so this is not a claim that the whole Mac host passed setup.

After refreshing the agent and Web Console, the repository's opt-in live probe
passed:

```text
WEB_CONSOLE_LIVE_URL=http://127.0.0.1:16382
WEB_CONSOLE_LIVE_TARGET_ID=bridge-node:25c7e426-c76c-421a-8351-aaf964589802
go test . -run '^TestLiveRemoteSessionThroughWebConsole$' -count=1 -v
--- PASS: TestLiveRemoteSessionThroughWebConsole (0.92s)
```

The authenticated transcript contained `Darwin`, the terminal resize was
observed as `100x30`, and `stty size` returned `30 100`. The Bridge registry
re-read after the run stored `machine_arch=amd64` and `binary_arch=amd64`; the
legacy `arch` projection is also `amd64`. No local Linux working directory or
shell was sent to the node.

The `linux-console → macos-node` declaration is therefore promoted to
`supported`, with this record and the live test as its evidence.

## Scoped conformance evidence

The repository-wide gate completed in 40 seconds against the current working
tree. It discovered 1,638 targets and reported 86 findings plus 60 warnings;
none of the findings belonged to `web-console`, `vrooli-bridge`, `nodeclient`,
or `session-core`. The nonzero exit is therefore caused by unrelated baseline
defects elsewhere in the repository, not by a touched plan component.

The plan-owned cross-platform matrix was run directly for Linux amd64 and
Darwin amd64 across the root module, Web Console API, Bridge API/agent/CLI,
and nodeclient. All twelve relevant checks passed. The focused Web Console UI
regression sets also passed: 11 files and 110 tests in total.

The session persistence regression was additionally run with a real five-second
detachment before reattachment: `TestHandleTerminalWS_ReconnectToSameSession`
passed in 5.21 seconds and replayed the retained output history.

The missing-grant mapping now preserves the required derived scope in the
operator message (`vrooli-bridge:write`) and names the machine-permissions
recovery action; its handler regression test passes.

The create-error recovery path now retains the complete failed launch request.
Retrying a failed remote launch therefore reuses its original target and agent
command instead of silently creating a local shell. The Workspace regression
test and full Workspace test file pass (33 tests).

The production Connect create boundary was also exercised with the node's
interactive write grant temporarily removed and then restored. Before the
fix, the endpoint exposed an internal node identity and Bridge error chain.
After the fix, the browser-facing response was:

```text
code=failed_precondition
message=remote target unavailable: remote node lacks required scope
        "vrooli-bridge:write"; manage the machine permissions
```

The restored node immediately passed `nodes doctor`; its original four grants
were preserved.

The same browser-facing redaction is now applied to unknown-node, expired
reauthentication, handshake, transport, and streaming failures. Handler
regressions cover these classes and reject node identities and internal error
chains in their returned messages.

Final validation after the error-boundary fix:

```text
vrooli capability conformance --declarations-only --json
{"findings":null,"targets":null}

vrooli capability conformance --json
1638 targets; 86 repository findings; 60 warnings; 0 findings in the four
plan-owned components (web-console, vrooli-bridge, nodeclient, session-core)
```
