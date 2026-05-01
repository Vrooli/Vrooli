import { test } from "vitest";
import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import { createElement } from "react";
import { DiffViewer } from "../../src/components/DiffViewer.js";
import { parseHunks } from "../../src/lib/diffHunks.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import type { RunDiff } from "../../src/types.js";

test("parseHunks returns empty array for empty input", () => {
  assert.deepEqual(parseHunks(""), []);
});

test("parseHunks parses a single hunk with adds and removes", () => {
  const patch = [
    "@@ -1,3 +1,4 @@",
    " line1",
    "-line2",
    "+line2-modified",
    "+line2b-new",
    " line3",
  ].join("\n");

  const hunks = parseHunks(patch);
  assert.equal(hunks.length, 1);
  assert.equal(hunks[0]?.oldStart, 1);
  assert.equal(hunks[0]?.newStart, 1);
  assert.equal(hunks[0]?.lines.length, 5);
  assert.deepEqual(hunks[0]?.lines[0], {
    type: "context",
    content: "line1",
    oldLine: 1,
    newLine: 1,
  });
  assert.deepEqual(hunks[0]?.lines[1], {
    type: "remove",
    content: "line2",
    oldLine: 2,
  });
  assert.deepEqual(hunks[0]?.lines[2], {
    type: "add",
    content: "line2-modified",
    newLine: 2,
  });
  assert.deepEqual(hunks[0]?.lines[3], {
    type: "add",
    content: "line2b-new",
    newLine: 3,
  });
  assert.deepEqual(hunks[0]?.lines[4], {
    type: "context",
    content: "line3",
    oldLine: 3,
    newLine: 4,
  });
});

test("parseHunks parses multiple hunks", () => {
  const patch = [
    "@@ -1,2 +1,2 @@",
    "-old1",
    "+new1",
    " same",
    "@@ -10,2 +10,2 @@",
    "-old10",
    "+new10",
    " same10",
  ].join("\n");

  const hunks = parseHunks(patch);
  assert.equal(hunks.length, 2);
  assert.equal(hunks[0]?.oldStart, 1);
  assert.equal(hunks[0]?.newStart, 1);
  assert.equal(hunks[1]?.oldStart, 10);
  assert.equal(hunks[1]?.newStart, 10);
});

test("parseHunks captures function context from the hunk header", () => {
  const patch = "@@ -5,3 +5,4 @@ function foo() {\n+  bar();\n";
  const hunks = parseHunks(patch);
  assert.equal(hunks.length, 1);
  assert.equal(hunks[0]?.context, "function foo() {");
});

test("parseHunks handles a header with no lines", () => {
  const hunks = parseHunks("@@ -1,0 +1,0 @@");
  assert.equal(hunks.length, 1);
  assert.deepEqual(hunks[0]?.lines, []);
});

test("parseHunks handles no-newline markers", () => {
  const patch = [
    "@@ -1,1 +1,1 @@",
    "-old",
    "\\ No newline at end of file",
    "+new",
  ].join("\n");

  const hunks = parseHunks(patch);
  assert.deepEqual(hunks[0]?.lines[1], {
    type: "no-newline",
    content: "\\ No newline at end of file",
  });
});

test("parseHunks preserves diff content that starts with +/-", () => {
  const patch = [
    "@@ -1,1 +1,1 @@",
    "-+1",
    "+-1",
  ].join("\n");

  const hunks = parseHunks(patch);
  assert.deepEqual(hunks[0]?.lines[0], {
    type: "remove",
    content: "+1",
    oldLine: 1,
  });
  assert.deepEqual(hunks[0]?.lines[1], {
    type: "add",
    content: "-1",
    newLine: 1,
  });
});

test("parseHunks ignores diff metadata before the first header", () => {
  const patch = [
    "diff --git a/foo.ts b/foo.ts",
    "index abc123..def456 100644",
    "--- a/foo.ts",
    "+++ b/foo.ts",
    "@@ -1,1 +1,1 @@",
    "-old",
    "+new",
  ].join("\n");

  const hunks = parseHunks(patch);
  assert.equal(hunks.length, 1);
  assert.equal(hunks[0]?.lines.length, 2);
});

test("DiffViewer renders file summary and parsed patch content", () => {
  renderWithProviders(
    createElement(DiffViewer, {
      diff: {
        files: [
          {
            path: "src/example.ts",
            changeType: "modified",
            additions: 1,
            deletions: 1,
            patch: ["@@ -1,1 +1,1 @@", "-old value", "+new value"].join("\n"),
          },
        ],
      } as RunDiff,
    }),
  );

  assert.equal(screen.getAllByText("src/example.ts").length, 2);
  assert.equal(screen.getByText("1 file").textContent, "1 file");
  assert.equal(screen.getByText("old value").textContent, "old value");
  assert.equal(screen.getByText("new value").textContent, "new value");
});
