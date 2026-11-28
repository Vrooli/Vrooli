# Agent Detection Signals

Source-of-truth catalog for `packages/cli-core/cliutil/agent_context.go`'s
broad detector (`DetectCallerKind` / `IsLikelyAgentContext`).

Each row below names a *runtime* and the env-var pattern the detector
uses to recognize it. **Only env vars that the runtime itself sets on
the shell it spawns as a child are valid signals.** Vars set by Vrooli
wrappers, web-console PTY injection, or npm shims appear in the
ancestor chain of every shell — using them produces false positives
when a human opens a web-console terminal or runs the wrapped binary
manually. The "Excluded signals" section at the bottom enumerates the
ones that look tempting but must NOT be used.

The verification method for any new signal is:

1. Start the runtime in a controlled environment (e.g. `agent-manager
   run create --profile-id …`).
2. Get the PID of the runtime process itself and the PID of a tool
   shell it spawns.
3. Diff `/proc/<tool-pid>/environ` against `/proc/<runtime-pid>/environ`.
4. The candidate signal must appear in the tool-shell env and be ABSENT
   from the runtime's own env. Otherwise it's inherited from above and
   not a runtime-self signal.

When in doubt, document the run ID (`agent-manager runs ls`) used to
collect the evidence so the next person can re-verify.

## Reliable signals (use these)

| Runtime | Match rule | Confirmation |
|---|---|---|
| **Vrooli sandbox** | `VROOLI_SANDBOX_ID` or `VROOLI_SANDBOX_MERGED` set | Agent-manager `phases/env.go::SandboxEnvVars()` is the canonical setter. |
| **Vrooli agent-manager run** | `VROOLI_AGENT_IDENTITY_TOKEN` set | Agent-manager `phases/env.go::IdentityEnvVars()`. The signed token is the only run-attribution signal; an environment-provided run id is not trusted. |
| **Vrooli swarm-manager session** | `VROOLI_SWARM_MANAGER_SESSION_ID` set | Swarm-manager session bootstrap. |
| **Claude Code** | `CLAUDECODE=1` (presence; corroborating: `CLAUDE_CODE_SESSION_ID`, `CLAUDE_CODE_ENTRYPOINT`, `CLAUDE_CODE_EXECPATH`) | Live self-inspection 2026-05-21 inside an active Claude Code session: vars present in tool-shell env, absent from the `claude` runtime's parent env. |
| **Codex CLI** | `CODEX_CI=1` AND `CODEX_THREAD_ID` non-empty | Agent-manager run `da1ab31b-13fe-44d1-910e-cde0f8d8fb26` (2026-05-21). The two-var AND rule rules out CI shells that happen to set `CODEX_CI=1` for unrelated reasons. |
| **Opencode** | `OPENCODE_PID` set, AND the value must match a PID in the calling process's ancestor chain (`/proc/<pid>/status` PPid walk). Corroborating: `OPENCODE_RUN_ID`, `OPENCODE_PROCESS_ROLE`. | Static analysis of `/home/matthalloran8/.local/share/vrooli/resources/opencode/bin/opencode` 2026-05-21 (grep for `process.env.OPENCODE_*=` and `??=`). Observational re-confirmation in a real opencode-launched process is deferred — agent-manager's opencode runner shells to claude internally as of 2026-05-21. The PID-match rule is the anti-spoofing guard: a stale `OPENCODE_PID` env var that doesn't match any real ancestor will not classify. |
| **Grok CLI** | `GROK_AGENT` equals the exact value `1`. Match on the value, NOT mere presence — see the value-override note. | Live `/proc` env-diff 2026-06-28 (standalone `grok 0.2.72`, launched headless `--always-approve`, asked to run a terminal command that walks its own process-ancestor chain). `GROK_AGENT=1` was present in the bash tool shells grok spawned (ancestor depths 0–1) and **absent** from the `grok` runtime's own env (the parent of those shells) and every ancestor above it (incl. the launching `claude` shell). **Value-override proof:** a second run launched with `GROK_AGENT=vrooli-probe-customvalue` preset showed grok's own env keeping that custom value while the tool shell still received the fixed sentinel `GROK_AGENT=1` — so grok deterministically rewrites it for subprocesses regardless of the user's custom-agent config. Matching the exact `1` therefore detects every grok tool shell (no false negative for custom-agent users) without classifying a human who exported `GROK_AGENT=<agent-name>` in their rc (a non-`1` value). |
| **Antigravity CLI** (Google's `agy`; replaces Gemini CLI) | `ANTIGRAVITY_AGENT` equals the exact value `1`. Match on the value, NOT mere presence (mirrors the grok rule). | **Binary-confirmed 2026-06-29 (`agy 1.0.13`); live `/proc` confirmation pending.** The agy binary's command-execution path contains the literal env-injection constant `ANTIGRAVITY_AGENT=1` (adjacent to `context_engine_hook`), alongside per-run `ANTIGRAVITY_CONVERSATION_ID=%s` and `ANTIGRAVITY_TRAJECTORY_ID=` set on the shells agy spawns to run tool commands — the direct analog of grok's `GROK_AGENT=1`. A first live probe MISSED it due to a too-strict probe regex (`^ANTIGRAVITY=` instead of `^ANTIGRAVITY_AGENT=`); a later non-mutating probe stopped at the authentication wall, so a clean live env-diff remains pending. Wired now on the strong binary evidence + grok precedent. **To finish confirming:** after authentication is available, run a headless `agy -p` whose tool shell dumps `/proc/$$/environ`, verify `ANTIGRAVITY_AGENT=1` is present there and absent from the `agy` runtime's own env, then drop this "pending" note. |

## Deferred runtimes

| Runtime | Why deferred |
|---|---|
| **Cursor (background agent)** | No Cursor runner exists in agent-manager today. Same verification path applies once a harness is available. |

> **Grok CLI** graduated to the reliable table above on 2026-06-28 once the
> `GROK_AGENT=1` self-signal was confirmed by `/proc` env-diff. The earlier
> hook-only candidates (`GROK_SESSION_ID` / `GROK_WORKSPACE_ROOT`, injected for
> **hook** processes only per `~/.grok/docs/user-guide/10-hooks.md`) were
> correctly NOT used — they are absent from general `run_terminal_command`
> subprocesses, so they would have missed the gated case.

## Excluded signals (do NOT use)

The following env vars look agent-related but are set by Vrooli
infrastructure, npm wrappers, or resource bootstraps — they appear in
the ancestor chain of every PTY spawned by web-console (and most other
launch surfaces), so using them would classify a human opening a
web-console terminal as an agent.

| Excluded var | Source | Why excluded |
|---|---|---|
| `AI_AGENT=<runtime>_<ver>_agent` | Vrooli infrastructure (agent-manager / web-console wrapper). | Codex investigation run observed `AI_AGENT=claude-code_2-1-144_agent` even with codex active — proof it's not a runtime-self signal. |
| `CODEX_HOME`, `CODEX_PATH`, `CODEX_AGENT_TAG`, `CODEX_NON_INTERACTIVE`, `WC_CODEX_SESSIONS_DIR` | Vrooli wrappers / web-console PTY env injection. See `scenarios/web-console/docs/guides/CONVERSATION_TRACKING.md` ("the three attribution env vars are injected on every PTY spawn"). | Present in every PTY spawn regardless of caller. |
| `OPENCODE_PATH`, `OPENCODE_STATE_DIR`, `OPENCODE_LOG_DIR`, `OPENCODE_CACHE_DIR`, `OPENCODE_DATA_DIR`, `OPENCODE_CONFIG_DIR`, `OPENCODE_XDG_*` | `resources/opencode/lib/common.sh` lines 14–34. | Set before opencode is invoked; present in every PTY spawn. |
| `WC_*` family | web-console attribution. | Set on every PTY spawn. |
| `CODEX_MANAGED_BY_NPM`, `CODEX_MANAGED_PACKAGE_ROOT` | codex npm wrapper. | Inherited launch context, appears in both runtime and tool-shell env. |

## Operator overrides

| Var | Effect |
|---|---|
| `VROOLI_CALLER=human` | Forces `CallerKindHuman` regardless of signals. Recovery path for false positives. |
| `VROOLI_CALLER=agent` | Forces `CallerKindOverride` regardless of signals. Useful for test harnesses and intentional agent contexts that lack the usual env vars. |

## Adding a new runtime

1. Run the candidate runtime in a controlled environment and capture
   the env diff (see verification method above).
2. Document the confirmation evidence (run ID, date) in a new row.
3. Add a row to `knownAgentSignals` in `agent_context.go` with a
   matching `Match` predicate.
4. Add a test case to `agent_context_test.go` mirroring
   `TestDetectCallerKind_ClaudeCode` / `TestDetectCallerKind_CodexRequiresBothSignals`.
5. If the signal needs anything beyond simple env-presence checks (see
   the opencode PID-match rule), implement the check in the same file
   and unit-test it independently.
