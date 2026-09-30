import assert from "node:assert/strict";
import { test } from "vitest";
import { NetworkAccess, RunnerType, SandboxMode } from "../../src/types.js";
import {
  cn, formatDuration, jsonObjectToPlain, jsonValueToPlain, networkAccessLabel,
  profileSandboxModeFormValue, runnerTypeFromSlug, runnerTypeLabel, runnerTypeToSlug,
  sandboxModeLabel, truncate,
} from "../../src/lib/utils.js";

test("utility labels cover supported values and safe defaults", () => {
  for (const [value, slug, label] of [[RunnerType.CLAUDE_CODE, "claude-code", "Claude Code"], [RunnerType.CODEX, "codex", "Codex"], [RunnerType.OPENCODE, "opencode", "OpenCode"], [RunnerType.GROK, "grok", "Grok"]] as const) {
    assert.equal(runnerTypeToSlug(value), slug);
    assert.equal(runnerTypeFromSlug(slug), value);
    assert.equal(runnerTypeLabel(value), label);
  }
  assert.equal(runnerTypeToSlug(undefined), "claude-code");
  assert.equal(runnerTypeFromSlug("other"), undefined);
  assert.equal(runnerTypeLabel(undefined), "Unknown");
  assert.equal(networkAccessLabel(NetworkAccess.NONE), "None");
  assert.equal(networkAccessLabel(NetworkAccess.LOCALHOST), "Localhost");
  assert.equal(networkAccessLabel(NetworkAccess.FULL), "Full");
  assert.equal(networkAccessLabel(undefined), "Localhost");
  assert.equal(sandboxModeLabel(SandboxMode.OFF), "Off");
  assert.equal(sandboxModeLabel(SandboxMode.TRACKING), "Tracking");
  assert.equal(sandboxModeLabel(SandboxMode.PROTECTED), "Protected");
  assert.equal(sandboxModeLabel(undefined), "Default");
});

test("utility formatting and protobuf-JSON conversion preserve primitive and nested values", () => {
  assert.equal(cn("p-2", false && "p-3", "text-sm"), "p-2 text-sm");
  assert.equal(formatDuration(999), "999ms"); assert.equal(formatDuration(1500), "1.5s");
  assert.equal(formatDuration(61_000), "1m 1s"); assert.equal(formatDuration(3_660_000), "1h 1m");
  assert.equal(truncate("short", 8), "short"); assert.equal(truncate("abcdefgh", 6), "abc...");
  assert.equal(profileSandboxModeFormValue({ sandboxConfig: { mode: SandboxMode.OFF } }), "off");
  assert.equal(profileSandboxModeFormValue({ sandboxConfig: { mode: SandboxMode.TRACKING } }), "tracking");
  assert.equal(profileSandboxModeFormValue({ sandboxConfig: { mode: SandboxMode.PROTECTED } }), "protected");
  assert.equal(profileSandboxModeFormValue({}), "protected");
  const value = { kind: { case: "objectValue", value: { fields: {
    enabled: { kind: { case: "boolValue", value: true } }, count: { kind: { case: "intValue", value: 4n } },
    values: { kind: { case: "listValue", value: { values: [{ kind: { case: "stringValue", value: "ok" } }, { kind: { case: "nullValue", value: 0 } }] } } },
  } } } } as never;
  assert.deepEqual(jsonValueToPlain(value), { enabled: true, count: 4, values: ["ok", null] });
  assert.equal(jsonValueToPlain(undefined), undefined);
  assert.equal(jsonObjectToPlain(undefined), undefined);
});
