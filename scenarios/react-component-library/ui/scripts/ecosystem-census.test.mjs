import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { inspectTidiness } from "./ecosystem-census.mjs";

const script = new URL("./ecosystem-census.mjs", import.meta.url);

test("ecosystem census walks the library and reports stable count fields", () => {
  const result = spawnSync(process.execPath, [fileURLToPath(script)], { encoding: "utf8" });
  assert.equal(result.status, 0, result.stderr);
  const report = JSON.parse(result.stdout);

  assert.equal(report.inventory.componentCount, report.inventory.components.length);
  assert.equal(report.inventory.versionCount, report.inventory.versions.length);
  assert.ok(report.inventory.componentCount > 200);
  assert.ok(report.inventory.versionCount >= report.inventory.componentCount);
  assert.equal(report.package.versionedExports, report.package.exportNames.filter((name) => /\/\d+\.\d+\.\d+/.test(name)).length);
  assert.equal(report.tidiness.hashNamedTestFiles.length, 0);
  assert.equal(report.tidiness.unreferencedToolFiles.length, 0);
});

test("tidiness checks fail for reintroduced hash tests and tool residue", () => {
  const root = mkdtempSync(path.join(tmpdir(), "rcl-census-"));
  const hashTest = path.join(root, "scenarios/react-component-library/library/components/Fake/versions/1.0.0/Fake.deadbeef.test.tsx");
  mkdirSync(path.dirname(hashTest), { recursive: true });
  writeFileSync(hashTest, "test('fixture', () => {});\n");
  const residue = path.join(root, "scenarios/react-component-library/tools/leftover.mjs");
  mkdirSync(path.dirname(residue), { recursive: true });
  writeFileSync(residue, "export {};\n");

  const result = inspectTidiness(root);
  assert.deepEqual(result.hashNamedTestFiles, ["scenarios/react-component-library/library/components/Fake/versions/1.0.0/Fake.deadbeef.test.tsx"]);
  assert.deepEqual(result.unreferencedToolFiles, ["scenarios/react-component-library/tools/leftover.mjs"]);
});

test("tidiness checks fail for dated generated artifacts under reserved paths", () => {
  const root = mkdtempSync(path.join(tmpdir(), "rcl-census-artifact-"));
  const artifact = path.join(
    root,
    "scenarios/react-component-library/docs/evidence/2026-08-28-run.json",
  );
  mkdirSync(path.dirname(artifact), { recursive: true });
  writeFileSync(artifact, "{}\n");

  const result = inspectTidiness(root);
  assert.deepEqual(result.datedArtifactFiles, [
    "scenarios/react-component-library/docs/evidence/2026-08-28-run.json",
  ]);
});
