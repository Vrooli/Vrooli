import assert from "node:assert/strict";
import { test } from "vitest";
import { detectLanguage, normalizeLanguage } from "../../src/components/markdown/utils/languageDetection.js";

test("detectLanguage recognizes the supported code and document dialects", () => {
  const cases: Array<[string, string]> = [
    ['{"run_id": "abc", "status": "failed"}', "json"],
    ["interface Report { status: string }\nimport x from 'x'", "typescript"],
    ["const report = () => { console.log(report); }", "javascript"],
    ["def investigate(run):\n  return run", "python"],
    ["package report\nfunc Build() {\n if err != nil {}\n}", "go"],
    ["SELECT status FROM runs JOIN tasks ON tasks.id = runs.task_id", "sql"],
    ["#!/usr/bin/env bash\necho ${RUN_ID}", "bash"],
    ["<div class=\"report\">status</div>", "html"],
    [".report { color: red; margin: 1rem; }", "css"],
    ["run:\n  - status: failed", "yaml"],
    ["# Report\n- [details](./report)", "markdown"],
  ];
  for (const [source, expected] of cases) assert.equal(detectLanguage(source), expected);
});

test("detectLanguage falls back safely and normalizeLanguage handles aliases", () => {
  assert.equal(detectLanguage(""), "text");
  assert.equal(detectLanguage("  \n\t"), "text");
  assert.equal(detectLanguage("unstructured words"), "text");
  assert.equal(normalizeLanguage(" TS "), "typescript");
  assert.equal(normalizeLanguage("shell"), "bash");
  assert.equal(normalizeLanguage("C#"), "csharp");
  assert.equal(normalizeLanguage("GraphQL"), "graphql");
});
