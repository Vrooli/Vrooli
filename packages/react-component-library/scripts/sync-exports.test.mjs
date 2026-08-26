import { execFileSync } from "node:child_process";
import { test } from "node:test";
import assert from "node:assert/strict";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const scriptsRoot = dirname(fileURLToPath(import.meta.url));

test("sync-exports check rejects no live imports and reports the inverse count", () => {
  const output = execFileSync(process.execPath, [join(scriptsRoot, "sync-exports.mjs"), "--check"], {
    encoding: "utf8",
  });
  const report = JSON.parse(output.trim().split("\n").at(-1));
  assert.equal(report.checked, true);
  assert.equal(report.brokenVersionImports, 0);
  assert.equal(report.versionedExports, 318);
});
