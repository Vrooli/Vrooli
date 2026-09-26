#!/usr/bin/env node
import { execFile } from "node:child_process";
import { createHash, randomUUID } from "node:crypto";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { basename, dirname, isAbsolute, join, relative, resolve } from "node:path";
import { promisify } from "node:util";
import { fileURLToPath } from "node:url";

const execFileAsync = promisify(execFile);
const scenarioRoot = resolve(dirname(fileURLToPath(import.meta.url)), "../../..");
const apiRoot = join(scenarioRoot, "api");
const api = process.env.BAS_API_URL || "http://127.0.0.1:17116";
const apiURL = new URL(api);
if (apiURL.protocol !== "http:" || !["127.0.0.1", "localhost", "[::1]"].includes(apiURL.hostname)) {
  throw new Error(`BAS_API_URL must target loopback HTTP, got ${apiURL.origin}`);
}

const sourceFiles = [
  "docs/internal/REFRACTOR_CONTRACT.json",
  "api/automation/execution-writer/file_writer.go",
  "api/automation/execution-writer/file_writer_test.go",
  "api/automation/execution-writer/external_artifacts.go",
  "api/automation/execution-writer/external_artifacts_test.go",
  "api/automation/execution-writer/evidence_manifest.go",
  "api/automation/execution-writer/evidence_manifest_test.go",
  "api/services/retention/retention.go",
  "api/services/retention/retention_test.go",
  "api/internal/evidencecompletenessqualification/receipt.go",
  "api/internal/evidencecompletenessqualification/receipt_test.go",
  "api/handlers/profilevalidation/provider.go",
  "api/handlers/profilevalidation/provider_test.go",
  "api/internal/testutil/testutil.go",
  ".vrooli/test-genie.json",
  ".vrooli/program-runtime/setpoint-read.py",
  "api/cmd/evidence-completeness-cohort/qualification.mjs",
];
const suites = [
  {
    package: "./automation/execution-writer",
    tests: [
      "TestScreenshotAndOutcomeWriteFailuresBothSurvive",
      "TestInlineTelemetryRemainsAttributableWhenSnapshotStorageFails",
      "TestExternalArtifactsRejectMissingOrUncommittedEvidence",
    ],
  },
  {
    package: "./services/retention",
    tests: ["TestActiveEvidenceRefusesDeletionUntilExportFinishes"],
  },
];

const evidenceRoot = join(scenarioRoot, ".vrooli/runtime/rehabilitation-evidence");
await mkdir(evidenceRoot, { recursive: true });

async function sha256(path) {
  return createHash("sha256").update(await readFile(path)).digest("hex");
}

async function buildIdentity() {
  const response = await fetch(`${api}/health`, { signal: AbortSignal.timeout(5000) });
  if (!response.ok) throw new Error(`BAS health returned ${response.status}`);
  const health = await response.json();
  const identity = String(health.build_identity || "").trim();
  if (!identity) throw new Error("BAS health returned no build_identity");
  return identity;
}

const buildBefore = await buildIdentity();
const artifacts = [];
const ownerTests = [];
for (const suite of suites) {
  const selector = `^(${suite.tests.join("|")})$`;
  const args = ["test", "-json", "-run", selector, "-count=1", suite.package];
  const outputPath = join(evidenceRoot, `evidence-completeness-owner-${randomUUID()}.jsonl`);
  let result;
  try {
    result = await execFileAsync("go", args, {
      cwd: apiRoot,
      timeout: 240000,
      maxBuffer: 12 * 1024 * 1024,
      env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off" },
    });
  } catch (error) {
    const output = `${error.stdout || ""}${error.stderr || ""}`;
    await writeFile(outputPath, output, { mode: 0o600 });
    process.stderr.write(output);
    throw new Error(`focused owner failed for ${suite.package}: ${error.message}`);
  }
  const output = result.stdout || "";
  if (result.stderr) process.stderr.write(result.stderr);
  await writeFile(outputPath, output, { mode: 0o600 });
  const passed = new Set();
  for (const line of output.split(/\r?\n/)) {
    if (!line) continue;
    const event = JSON.parse(line);
    if (event.Action === "pass" && suite.tests.includes(event.Test)) passed.add(event.Test);
  }
  for (const name of suite.tests) {
    if (!passed.has(name)) throw new Error(`owner test did not report pass: ${name}`);
    ownerTests.push({ name, passed: true });
  }
  artifacts.push({
    path: `.vrooli/runtime/rehabilitation-evidence/${basename(outputPath)}`,
    sha256: await sha256(outputPath),
    tests: suite.tests,
  });
  process.stdout.write(`${suite.package}: passed ${suite.tests.join(", ")}\n`);
}

const buildAfter = await buildIdentity();
if (buildAfter !== buildBefore) throw new Error(`managed BAS identity changed during owners: ${buildBefore} -> ${buildAfter}`);
const sourceSHA256 = {};
for (const relative of sourceFiles) sourceSHA256[relative] = await sha256(join(scenarioRoot, relative));
const receipt = {
  schemaVersion: 1,
  contractRow: "evidence-completeness",
  result: "passed",
  managedBuildIdentity: buildBefore,
  sourceSha256: sourceSHA256,
  artifacts,
};
const outputPath = resolve(
  process.env.BAS_EVIDENCE_COMPLETENESS_RECEIPT ||
    join(evidenceRoot, `evidence-completeness-${new Date().toISOString().replace(/[:.]/g, "-")}-${randomUUID().slice(0, 8)}.json`),
);
const outputRelative = relative(evidenceRoot, outputPath);
if (isAbsolute(outputRelative) || outputRelative === ".." || outputRelative.startsWith(`..${process.platform === "win32" ? "\\" : "/"}`)) {
  throw new Error("receipt path must stay under the ignored rehabilitation evidence directory");
}
await writeFile(outputPath, `${JSON.stringify(receipt, null, 2)}\n`, { flag: "wx", mode: 0o600 });
console.log(JSON.stringify({ receipt: outputPath, buildIdentity: buildBefore, ownerTests: ownerTests.length, rawArtifacts: artifacts.length }, null, 2));
