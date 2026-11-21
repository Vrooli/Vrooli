import assert from "node:assert/strict";
import { test } from "vitest";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { makeRun } from "../test-utils/fixtures/runs";
import { RunnerType, type Run } from "../types";
import { readableTitle, runHeading, runOrigin } from "./runPresentation";

test("runHeading gives people a title and keeps machine ids as a short reference", () => {
  const workflow = makeRun({ id: "c030e0ce-3593-4d77-a49f-0ef6735e3929", label: "Workflow e6b4c7c2-da57-416c-b2c2-8c0b5e942874 node draft", labelSource: "derived" });
  assert.deepEqual(runHeading(workflow), { title: "Workflow step · draft", reference: "e6b4c7c2" });

  const attached = makeRun({ label: "claude-code session claude-code-e91179ac8302966ef21403d99ec34ebb", labelSource: "harness", harnessKind: "claude-code", harnessSessionId: "claude-code-e91179ac8302966ef21403d99ec34ebb" });
  assert.deepEqual(runHeading(attached), { title: "Claude Code session", reference: "e91179ac" });

  const labelled = makeRun({ id: "abcdef12-0000-0000-0000-000000000000", label: "Fix the flaky dashboard test", labelSource: "user" });
  assert.deepEqual(runHeading(labelled), { title: "Fix the flaky dashboard test", reference: "abcdef12" });

  assert.equal(runHeading(makeRun({ label: "" }), "Task title").title, "Task title");
  assert.equal(runHeading(makeRun({ label: "", promptPreview: "Investigate the\\nfailing build" })).title, "Investigate the\\nfailing build");
  assert.equal(runHeading(makeRun({ id: "99887766-aaaa", label: "" })).title, "Run 99887766");
});

test("readableTitle strips prompt boilerplate and keeps a sensible first line", () => {
  assert.equal(readableTitle("Fork started — that fork is you. <fork-boilerplate>You are a worker fork.</fork-boilerplate>\nMore"), "Fork started — that fork is you.");
  assert.equal(readableTitle("\n\n## Heading line\nbody"), "Heading line");
  assert.equal(readableTitle("Run 3f1c9a2e-1111-2222-3333-444455556666 again"), "Run 3f1c9a2e again");
  const long = readableTitle("word ".repeat(40));
  assert.ok(long.length <= 91 && long.endsWith("…"), long);
  assert.equal(readableTitle("x".repeat(120)).length, 91, "an unbroken string is cut at the limit");
  assert.equal(readableTitle("<only-markup></only-markup>"), "");
});

test("runOrigin names the profile, or where an unprofiled run came from", () => {
  assert.equal(runOrigin(makeRun({ agentProfileId: "p1" }), "code-reviewer"), "code-reviewer");
  assert.equal(runOrigin(makeRun({ harnessKind: "codex" })), "Codex · attached");
  assert.equal(runOrigin(makeRun({ importedAt: timestampFromMs(0), importSourceHarness: "claude-code" })), "Imported · Claude Code");
  assert.equal(runOrigin(makeRun({ importedAt: timestampFromMs(0) })), "Imported");
  assert.equal(runOrigin(makeRun({ resolvedConfig: { runnerType: RunnerType.OPENCODE } as Run["resolvedConfig"] })), "OpenCode");
  assert.equal(runOrigin(makeRun({ agentProfileId: "gone" })), "Profile unavailable");
  assert.equal(runOrigin(makeRun({})), "No profile");
  });
