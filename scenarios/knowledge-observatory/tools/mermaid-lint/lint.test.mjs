import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import test from "node:test";

function lint(blocks) {
  const run = spawnSync(process.execPath, ["lint.mjs"], {
    cwd: new URL(".", import.meta.url),
    input: JSON.stringify({ blocks }),
    encoding: "utf8",
  });
  assert.equal(run.status, 0, run.stderr);
  return JSON.parse(run.stdout);
}

test("returns real parser verdicts for supported and malformed diagrams", () => {
  const output = lint([
    { id: "valid", content: "sequenceDiagram\n  A->>B: hello" },
    { id: "semicolon", content: "sequenceDiagram\n  A->>B: hello; goodbye" },
    { id: "headerless", content: "A --> B" },
    { id: "brackets", content: "flowchart TD\n  A[broken --> B" },
    { id: "empty", content: "" },
  ]);
  assert.equal(output.engine, "mermaid@11.13.0");
  assert.equal(output.results[0].valid, true);
  for (const result of output.results.slice(1)) {
    assert.equal(result.valid, false);
    assert.equal(typeof result.error, "string");
  }
  assert.equal(output.results[1].line, 2);
});
