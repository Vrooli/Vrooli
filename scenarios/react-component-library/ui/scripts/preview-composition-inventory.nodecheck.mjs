import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";

const script = path.resolve(import.meta.dirname, "preview-composition-inventory.mjs");

function run(...args) {
  return JSON.parse(
    execFileSync(process.execPath, [script, ...args], {
      encoding: "utf8",
      maxBuffer: 64 * 1024 * 1024,
    }),
  );
}

function stories(report) {
  return report.entries.flatMap((entry) => entry.storyRecords);
}

test("inventory emits unique stable keys and classifies the full corpus", () => {
  const report = run();
  const records = stories(report);
  const keys = records.map((record) => record.storyKey);
  assert.equal(report.summary.storyCount, records.length);
  assert.equal(report.summary.batchSize, 50);
  assert.ok(report.summary.batchCount > 1);
  assert.equal(report.resume.stateFile, "docs/evidence/preview-composition-state.json");
  assert.equal(new Set(keys).size, keys.length);
  assert.ok(report.summary.contractCount > 0);
  assert.equal(report.summary.contractCount, report.entries.length);
  assert.ok(
    records.every((record) => !record.diagnostics.some((item) => item.code === "raw-child-node")),
  );
  assert.ok(
    records.every((record) => record.reviewSet === "core" || typeof record.reviewSet === "string"),
  );
  assert.equal(
    records.filter((record) =>
      record.diagnostics.some((item) => item.code === "missing-review-set"),
    ).length,
    0,
  );
});

test("bounded batches are deterministic, disjoint, and resumable", () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "rcl-inventory-"));
  const statePath = path.join(directory, "state.json");
  const first = run("--batch-size", "25", "--batch-index", "0", "--state", statePath);
  const second = run("--batch-size", "25", "--batch-index", "1", "--state", statePath);
  const firstKeys = first.resume.nextBatchStoryKeys;
  const secondKeys = second.resume.nextBatchStoryKeys;
  assert.equal(firstKeys.length, 25);
  assert.equal(secondKeys.length, 25);
  assert.equal(firstKeys.filter((key) => secondKeys.includes(key)).length, 0);

  const marked = run(
    "--batch-size",
    "25",
    "--batch-index",
    "0",
    "--state",
    statePath,
    "--mark-complete",
    firstKeys[0],
  );
  assert.equal(marked.summary.completedCount, 1);
  fs.writeFileSync(statePath, JSON.stringify({ completedStoryKeys: firstKeys }));
  const resumed = run("--batch-size", "25", "--batch-index", "0", "--state", statePath);
  assert.equal(resumed.summary.completedCount, 25);
  assert.equal(resumed.resume.nextBatchStoryKeys[0], secondKeys[0]);
  fs.rmSync(directory, { recursive: true, force: true });
});
