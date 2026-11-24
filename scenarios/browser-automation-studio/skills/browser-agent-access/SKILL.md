---
name: "browser-agent-access"
description: "Find and use the BAS-owned permissioned browser broker through the full CLI, Vrooli stdio MCP, or Program Runtime while preserving Agent Manager grants and the existing session owner."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["browser", "agents", "permissions", "sessions", "trace"]
  status: "active"
  revision: 3
  createdAt: "2026-11-19T00:00:00Z"
  updatedAt: "2026-11-22T00:00:00Z"
  requires:
    scenarios: ["browser-automation-studio", "agent-manager", "search-hub", "prompt-manager", "program-runtime"]
    commands: ["search-hub query query", "prompt-manager skill read", "browser-automation-studio browser", "vrooli browser"]
  origin:
    kind: "authored"
---
## Tools focus: browser-agent-access

Find the BAS browser surface through Search Hub, read this skill, and use one BAS-owned session through the CLI, Vrooli stdio MCP, or Program Runtime under the current Agent Manager grant.

## Discover the surface

Start without guessing a provider or path:

```sh
search-hub query query "I need instructions for an agent to inspect a website and interact with it through a permissioned browser session" --type skill --limit 5
prompt-manager skill read browser-agent-access
```

Search Hub registers this intent as `browser-agent-session` and expects `browser-agent-access`. Read `browser-automation-studio` for workflow authoring, not session access.

## Choose an entry point

| Need | Use | Boundary |
|---|---|---|
| Full BAS browser operations | `browser-automation-studio browser --help` | Supports open, observe, typed act/task, attach, trace, unsaved `draft-trace`, gated promotion/replay, close and harness qualification. |
| Compact host CLI | `vrooli browser --help` | Supports MCP, capabilities, open, observe, act, trace, promotion request and close. It does not expose task, attach, replay or draft-trace. |
| Tool-calling agent | `vrooli browser mcp` over persistent stdio | Tools: `browser_capabilities`, `browser_open`, `browser_observe`, `browser_act`, `browser_trace`, `browser_draft_trace`, `browser_promote_trace`, `browser_close`. `browser_draft_trace` is an unsaved preview on the existing session. |
| Governed program | Program Runtime browser bindings | Includes `browser.session.open`, `browser.observe`, `browser.act`, `browser.session.attach`, `browser.session.read_trace`, `browser.run_task`, `browser.draft_trace`, `browser.promote_trace`, `browser.replay_flow`, `browser.session.close`. Only declared BAS session bindings receive the invocation-local grant relay. |

Use `BAS_BROWSER_API_URL` for the broker API origin. Run the CLI/MCP inside an Agent Manager run that already injects `VROOLI_AGENT_IDENTITY_TOKEN` and `VROOLI_BROWSER_GRANT_RECEIPT`. Program Runtime resolves a fresh browser grant for each invocation. Keep credentials in the environment; never put them in requests, files or program inputs.

## Operate one authorized session

1. Read the current capability/grant summary before opening. `browser_capabilities` or `vrooli browser capabilities` reports the current mode and backend limits; it does not grant new permission.
2. Open a session only when the run's existing grant allows it. BAS selects the permission profile from trusted Agent Manager context; a caller-supplied profile is not authority.
3. Observe before acting. Use typed actions or bounded `browser.run_task` instructions. The broker rechecks the session lease, profile, URL scope and action policy on every effect.
4. Reuse the returned session ID for observation, action, trace and close. A session ID addresses a BAS-owned session; it does not grant authority.
5. Use MCP `browser_draft_trace` or BAS/Program Runtime `draft-trace`/`draft_trace` to review a deterministic V2 draft. A draft is unsaved and takes no project ID.

Attached-tab use requires the existing consent owner and a user action. Never start a private browser, create a second session to finish an action, or pass CDP endpoints, lease IDs, caller/run IDs, credentials, approvals or consent receipts as task data. Upstream Playwright MCP/Browser Use adapters remain unqualified for live tasks; the Vrooli stdio MCP adapter is the supported MCP entry point.

## Trace and project-scope gate

Read trace history only when the active profile grants it. `browser_promote_trace` and `browser.promote_trace` request project persistence, but the production Agent Manager project-scope receipt verifier is not wired. BAS therefore fails promotion/replay closed before persistence or replay actions. A project ID alone is not authorization. Use the unsaved draft path until the existing project-scope owner supplies and verifies the required receipt. Do not claim a saved workflow ID/version or deterministic replay from draft evidence.

## Permission and qualification limits

BAS and Agent Manager own session, consent, profile, grant and extension authority. Observe-only grants cannot click or type. A missing, expired or revoked identity/grant, a denied action, or an out-of-scope URL means stop and ask the existing owner to resolve access; do not copy credentials or broaden a profile. Remote browser mode and live adapter/session qualification remain unsupported here. No bot-detection realism or bypass claim is supported without dated, per-mode and per-site qualification.

Generated extensions use a ready owner-issued build/artifact reference and digest. Installation, reload and validation use the same existing BAS managed session and require separate Agent Manager install/load grants. Live extension qualification still requires an admitted target.

### Troubleshooting & Edge Cases

| Symptom | Next action |
|---|---|
| Search Hub finds another skill first | Keep the query scoped to a permissioned browser session and use `--type skill --limit 5`; read `browser-agent-access` explicitly. |
| Missing identity or browser grant | Stop. Ask the Agent Manager run owner to provide the supported run context. Do not pass tokens or grant receipts manually. |
| Observe works but act is denied | Treat the current profile as observe-only or revoked. Reopen only through an authorized run with the required control grant. |
| URL scope or consent check fails | Do not retry against a different tab/site. Use the BAS/consent owner path for the exact session and requested scope. |
| Promotion or replay fails on project scope | Preserve the draft and stop. The current production project-scope receipt verifier is absent; do not treat `project_id` as proof. |
| Extension install/load is denied | Keep the failure evidence. The separate Agent Manager opt-ins and an admitted browser target are required. |
