import assert from "node:assert/strict";
import fs from "node:fs";
import { spawnSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const fixtures = path.join(scriptDir, "fixtures/usage-contracts");
let repo = scriptDir;
while (repo !== path.dirname(repo) && !fs.existsSync(path.join(repo, "scenarios/react-component-library/ui/scripts/check-usage-contracts.mjs")))
  repo = path.dirname(repo);
assert.ok(fs.existsSync(path.join(repo, "scenarios/react-component-library/ui/scripts/check-usage-contracts.mjs")), "could not locate repository-owned usage-contract checker");
const checker = path.join(repo, "scenarios/react-component-library/ui/scripts/check-usage-contracts.mjs");

function scan(fixture) {
  const fixtureRoot = path.join(fixtures, fixture);
  return spawnSync(process.execPath, [checker, "--roots", path.relative(repo, fixtureRoot), "--json"], {
    cwd: repo,
    encoding: "utf8",
  });
}

test("react-vite consumer contract fixture fails on a raw overlay footer", () => {
  const result = scan("fail");
  assert.equal(result.status, 1, result.stderr || result.stdout);
  const report = JSON.parse(result.stdout);
  assert.equal(report.findings.length, 1);
  assert.equal(report.findings[0].code, "rebuild");
  assert.match(report.findings[0].message, /OverlayActionBar/);
});

test("react-vite consumer contract fixture passes with the shared action bar", () => {
  const result = scan("pass");
  assert.equal(result.status, 0, result.stderr || result.stdout);
  const report = JSON.parse(result.stdout);
  assert.equal(report.findings.length, 0);
});
