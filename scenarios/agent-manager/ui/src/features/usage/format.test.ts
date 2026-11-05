import assert from "node:assert/strict";
import { test } from "vitest";
import { evidenceText, formatDayRange, formatServerText, friendlyLabel, windowTitle } from "./format.js";

const HOUR = 3600_000;
const iso = (ms: number) => new Date(ms).toISOString();

test("quota scopes are named like the CLI: Claude/Codex and 5-hour/weekly, never raw ids", () => {
  assert.equal(windowTitle({ provider: "anthropic", pool: "five_hour", window: "300m" }), "Claude · 5-hour");
  assert.equal(windowTitle({ provider: "anthropic", pool: "seven_day", window: "10080m" }), "Claude · weekly");
  assert.equal(windowTitle({ provider: "openai", pool: "primary", window: "10080m" }), "Codex · weekly");
  assert.equal(windowTitle({ provider: "openai", pool: "primary", window: "300m" }), "Codex · 5-hour");
  assert.equal(windowTitle({ provider: "anthropic", pool: "five_hour", window: "five_hour" }), "Claude · 5-hour");
  assert.equal(windowTitle({ provider: "anthropic", pool: "seven_day_opus", window: "10080m" }), "Claude · weekly · Opus", "a pool with a name of its own keeps it");
});

test("windows whose titles collide on one surface append their pool id, like the CLI", () => {
  const primary = { provider: "openai", pool: "primary", window: "10080m" };
  const secondary = { provider: "openai", pool: "secondary", window: "10080m" };
  const claude = { provider: "anthropic", pool: "seven_day", window: "10080m" };
  const surface = [primary, secondary, claude];
  assert.equal(windowTitle(primary, surface), "Codex · weekly · primary");
  assert.equal(windowTitle(secondary, surface), "Codex · weekly · secondary");
  assert.equal(windowTitle(claude, surface), "Claude · weekly", "a title that does not collide is unchanged");
  assert.equal(windowTitle(primary, [primary, claude]), "Codex · weekly");
});

test("evidence text tells same-label siblings apart by short id", () => {
  assert.equal(evidenceText({ id: "00b65371-7359-42d6-b8f4-83218914f65f", label: "sub-agent" }), "sub-agent · 00b65371");
  assert.equal(evidenceText({ id: "76d970de-82d0-4c52-b559-ceadb6acbf61", label: "76d970de Agent manager token usage analysis" }), "76d970de Agent manager token usage analysis");
  assert.equal(evidenceText({ id: "00b65371-7359-42d6-b8f4-83218914f65f", label: "" }), "00b65371");
});

test("times inside server sentences and one-line ranges are formatted, not raw RFC 3339", () => {
  assert.doesNotMatch(formatServerText("the window reset at 2026-09-09T20:28:00Z"), /T20:28:00Z/);
  assert.match(formatServerText("the window reset at 2026-09-09T20:28:00Z"), /^the window reset at Sep 9, /);
  const start = new Date(2026, 9, 30, 19).toISOString();
  assert.equal(formatDayRange(start, new Date(2026, 9, 31, 15).toISOString()), "Oct 30–31");
  assert.equal(formatDayRange(start, new Date(2026, 10, 2, 15).toISOString()), "Oct 30 – Nov 2");
  assert.equal(formatDayRange(start), "Oct 30");
});

test("raw UUIDs never label a row", () => {
  assert.equal(friendlyLabel("Workflow 5f518188-6df0-48eb-a0f0-0085643e8634 node draft"), "Workflow 5f51… node draft");
});
