import assert from "node:assert/strict";
import { test } from "vitest";
import { canLimitContext, launchHarness, withContextLimit } from "./contextLimit";

test("standard adds nothing, so a usage mode or the harness default still applies", () => {
  for (const command of ["codex --yolo --no-alt-screen", "vrooli agent launch --runner claude --arg=--dangerously-skip-permissions", "opencode"]) {
    assert.equal(withContextLimit(command, "standard"), command);
  }
});

test("a Claude launch gets the limit as an environment prefix, for that launch only", () => {
  assert.equal(withContextLimit("vrooli agent launch --runner claude --arg=--dangerously-skip-permissions", "tight"), "CLAUDE_CODE_AUTO_COMPACT_WINDOW=150000 vrooli agent launch --runner claude --arg=--dangerously-skip-permissions");
  assert.equal(withContextLimit("claude --dangerously-skip-permissions", "keep_long"), "CLAUDE_CODE_AUTO_COMPACT_WINDOW=1000000 claude --dangerously-skip-permissions");
});

test("a Codex launch gets the limit as a launch argument", () => {
  assert.equal(withContextLimit("codex --yolo --no-alt-screen", "tight"), "codex -c model_auto_compact_token_limit=150000 --yolo --no-alt-screen");
  assert.equal(withContextLimit("vrooli agent launch --runner codex", "tight"), "vrooli agent launch --runner codex --arg=-c --arg=model_auto_compact_token_limit=150000");
});

test("tight never goes below Claude's 100k minimum", () => {
  assert.ok(Number(/=(\d+)/.exec(withContextLimit("claude", "tight"))?.[1]) >= 100_000);
});

test("other agents, wrappers and an explicit limit are left exactly as they are", () => {
  for (const command of ["opencode", "grok", "if true; then codex; fi", "codex | tee log", "CLAUDE_CODE_AUTO_COMPACT_WINDOW=300000 claude", "codex -c model_auto_compact_token_limit=300000"]) {
    assert.equal(withContextLimit(command, "tight"), command, command);
  }
  assert.equal(launchHarness("vrooli agent launch --runner opencode"), null);
  assert.equal(canLimitContext("codex && rm -rf x"), false);
});

test("nothing typed into a running session ever persists a harness setting", () => {
  for (const choice of ["keep_long", "tight"] as const) {
    const command = withContextLimit("claude", choice);
    assert.doesNotMatch(command, /\/(model|autocompact)/);
  }
});
