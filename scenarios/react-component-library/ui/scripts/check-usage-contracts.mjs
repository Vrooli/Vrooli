#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { Linter } from "eslint";
import parser from "@typescript-eslint/parser";
import usageContracts from "../eslint-rules/usage-contracts.js";

const repo = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../../..");
const flagships = ["web-console", "agent-manager", "notification-hub", "prompt-manager", "nutrition-planner", "personal-planner", "image-tools", "brand-manager"];
const args = process.argv.slice(2);
const rootsArg = args.indexOf("--roots");
const allReactUIs = args.includes("--all-react-uis");
const ratchetArg = args.indexOf("--ratchet-file");
const checkCatalogCoverage = allReactUIs || ratchetArg >= 0;
const reactScenarioRoots = () => fs.readdirSync(path.join(repo, "scenarios"), { withFileTypes: true })
  .filter((entry) => entry.isDirectory())
  .map((entry) => entry.name)
  .filter((name) => {
    const uiRoot = path.join(repo, "scenarios", name, "ui");
    const packagePath = path.join(uiRoot, "package.json");
    if (!fs.existsSync(path.join(uiRoot, "src")) || !fs.existsSync(packagePath)) return false;
    try {
      const pkg = JSON.parse(fs.readFileSync(packagePath, "utf8"));
      return Boolean(pkg.dependencies?.react || pkg.devDependencies?.react || pkg.peerDependencies?.react);
    } catch { return false; }
  })
  .sort()
  .map((name) => `scenarios/${name}/ui/src`);
const scenarioRoots = allReactUIs
  ? reactScenarioRoots()
  : rootsArg >= 0 ? args[rootsArg + 1].split(",") : flagships.map((name) => `scenarios/${name}/ui/src`);
const linter = new Linter({ configType: "flat" });
const config = [{
  files: ["**/*.{ts,tsx}"],
  languageOptions: { parser, parserOptions: { ecmaVersion: 2022, sourceType: "module", ecmaFeatures: { jsx: true } } },
  plugins: { "rcl-usage-contracts": { rules: { "usage-contract": usageContracts } } },
  rules: { "rcl-usage-contracts/usage-contract": "error" },
}];
const findings = [];
let checkedFiles = 0;

function catalogCoverageReport() {
  const script = path.join(repo, "scenarios/react-component-library/ui/scripts/check-catalog-usage-contracts.mjs");
  const run = spawnSync(process.execPath, [script, "--json"], {
    cwd: repo,
    encoding: "utf8",
    maxBuffer: 8 * 1024 * 1024,
  });
  if (run.error)
    return { status: "not-evaluated", eligible: 0, contracted: 0, errors: [`catalog coverage checker could not start: ${run.error.message}`] };
  let report;
  try {
    report = JSON.parse(run.stdout);
  } catch (error) {
    return { status: "not-evaluated", eligible: 0, contracted: 0, errors: [`catalog coverage checker returned invalid JSON: ${error.message}`] };
  }
  const eligible = report.coverage?.eligible;
  const contracted = report.coverage?.contracted;
  const errors = Array.isArray(report.errors) ? report.errors : [];
  const complete = run.status === 0 && Number.isInteger(eligible) && eligible > 0 && contracted === eligible && errors.length === 0;
  return {
    status: complete ? "passed" : run.status === null ? "not-evaluated" : "failed",
    eligible: Number.isInteger(eligible) ? eligible : 0,
    contracted: Number.isInteger(contracted) ? contracted : 0,
    errors,
  };
}

function walk(dir) {
  if (!fs.existsSync(dir)) return [];
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return walk(full);
    return /\.[jt]sx?$/.test(entry.name) && !/\.(test|spec)\.[jt]sx?$/.test(entry.name) ? [full] : [];
  });
}

for (const relativeRoot of scenarioRoots) {
  for (const file of walk(path.resolve(repo, relativeRoot))) {
    checkedFiles++;
    const source = fs.readFileSync(file, "utf8");
    // Flat-config file globs are matched against paths relative to the
    // repository root. Passing an absolute path outside the current working
    // directory causes ESLint to ignore the file silently, turning the
    // flagship check into an always-green report.
    const messages = linter.verify(source, config, { filename: path.relative(repo, file) });
    for (const message of messages) {
      if (message.severity !== 2 || message.ruleId !== "rcl-usage-contracts/usage-contract") continue;
      findings.push({
        file: path.relative(repo, file),
        line: message.line,
        code: message.messageId ?? "usage-contract",
        message: message.message,
      });
    }
  }
}

const byContract = Object.fromEntries([...findings.reduce((counts, finding) => {
  const contract = /^([A-Z][A-Za-z0-9]*)(?:\.([A-Za-z][A-Za-z0-9]*))?/.exec(finding.message);
  const key = contract ? `${contract[1]}${contract[2] ? `.${contract[2]}` : ""}` : "overlay-footer";
  counts.set(key, (counts.get(key) ?? 0) + 1);
  return counts;
}, new Map())].sort(([left], [right]) => left.localeCompare(right)));
const fleet = { scenarios: scenarioRoots.length, files: checkedFiles, findings: findings.length, byContract };
const catalogCoverage = checkCatalogCoverage ? catalogCoverageReport() : null;
const catalogCoverageFailed = catalogCoverage !== null && catalogCoverage.status !== "passed";
let ratchet = null;
let ratchetFailed = false;
if (ratchetArg >= 0) {
  const ratchetPath = path.resolve(repo, args[ratchetArg + 1] ?? "");
  try {
    const baseline = JSON.parse(fs.readFileSync(ratchetPath, "utf8"));
    const failures = [];
    if (!Number.isInteger(baseline.scenarios) || baseline.scenarios < 1) failures.push("ratchet baseline must declare a positive scenario count");
    else if (fleet.scenarios !== baseline.scenarios) failures.push(`checked ${fleet.scenarios} React UI trees; ratchet requires ${baseline.scenarios}`);
    if (!baseline.byContract || typeof baseline.byContract !== "object" || Array.isArray(baseline.byContract)) failures.push("ratchet baseline must declare per-contract counts");
    const comparisons = {};
    for (const contract of new Set([...Object.keys(baseline.byContract ?? {}), ...Object.keys(byContract)])) {
      const before = baseline.byContract?.[contract] ?? 0;
      const after = byContract[contract] ?? 0;
      comparisons[contract] = { baseline: before, current: after, delta: after - before };
      if (!Number.isInteger(before) || before < 0) failures.push(`invalid ratchet count for ${contract}`);
      else if (after > before) failures.push(`${contract} findings increased from ${before} to ${after}`);
    }
    ratchet = { status: failures.length ? "failed" : "passed", baseline: path.relative(repo, ratchetPath), comparisons, failures };
    ratchetFailed = failures.length > 0;
  } catch (error) {
    ratchet = { status: "not-evaluated", baseline: path.relative(repo, ratchetPath), comparisons: {}, failures: [`ratchet baseline could not be read: ${error.message}`] };
    ratchetFailed = true;
  }
}

if (args.includes("--json")) console.log(JSON.stringify({ fleet, ratchet, catalogCoverage, findings }, null, 2));
else {
  for (const finding of findings) console.log(`${finding.file}:${finding.line} ${finding.code}: ${finding.message}`);
  console.log(`Usage-contract fleet: ${fleet.scenarios} UI trees, ${fleet.files} source files, ${fleet.findings} findings.`);
  if (catalogCoverage)
    console.log(`Catalog usage-contract coverage ${catalogCoverage.status}: ${catalogCoverage.contracted}/${catalogCoverage.eligible} eligible assets.`);
  for (const [contract, count] of Object.entries(byContract)) console.log(`${contract}\t${count}`);
  if (ratchet) {
    console.log(`Fleet ratchet ${ratchet.status}: ${ratchet.baseline}`);
    for (const [contract, row] of Object.entries(ratchet.comparisons)) console.log(`${contract}\t${row.baseline} → ${row.current}\t${row.delta >= 0 ? "+" : ""}${row.delta}`);
    for (const failure of ratchet.failures) console.error(failure);
  }
}
if (ratchetArg >= 0 ? ratchetFailed || catalogCoverageFailed : findings.length > 0 || catalogCoverageFailed) process.exitCode = 1;
