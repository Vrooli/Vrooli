#!/usr/bin/env node
import { execFile } from "node:child_process";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { promisify } from "node:util";
import { fileURLToPath } from "node:url";
import { randomUUID } from "node:crypto";

const execFileAsync = promisify(execFile);
const sourcePath = fileURLToPath(import.meta.url);
const scenarioRoot = resolve(dirname(sourcePath), "../../..");
const apiRoot = resolve(scenarioRoot, "api");
const repositoryRoot = resolve(scenarioRoot, "../..");
const driverRoot = resolve(scenarioRoot, "playwright-driver");
const api = process.env.BAS_API_URL || "http://127.0.0.1:17116";
const apiURL = new URL(api);
if (
  apiURL.protocol !== "http:" ||
  !["127.0.0.1", "localhost", "[::1]"].includes(apiURL.hostname)
) {
  throw new Error(
    `BAS_API_URL must target loopback HTTP, got ${apiURL.origin}`,
  );
}
const executionTests =
  "^(TestStopExecutionRetainsUncertainOutcomeAndJoinsLeasedDriverClose|TestExecuteTimeoutDuringLiveInstructionClosesSessionWithoutReplay|TestExecuteDriverDeathRetainsUncertainEffectWithoutReplay)$";
const retryTest =
  "close keeps a completed click uncertain while denying retry admission";

async function buildIdentity() {
  const response = await fetch(`${api}/health`, {
    signal: AbortSignal.timeout(5000),
  });
  if (!response.ok) throw new Error(`BAS health returned ${response.status}`);
  const health = await response.json();
  const identity = String(health.build_identity || "").trim();
  if (!identity) throw new Error("BAS health returned no build_identity");
  return identity;
}

async function run(command, args, cwd, observationsPath) {
  let result;
  try {
    result = await execFileAsync(command, args, {
      cwd,
      timeout: 420000,
      maxBuffer: 8 * 1024 * 1024,
      env: {
        ...process.env,
        GOTOOLCHAIN: "local",
        GOPROXY: "off",
        BAS_J07_OBSERVATIONS: observationsPath,
      },
    });
  } catch (error) {
    // execFile buffers output and discards it from the terminal on a failing
    // child unless the coordinator explicitly forwards both streams. Keeping
    // the owner's measured band values visible makes failures diagnosable.
    if (error.stdout?.trim()) process.stdout.write(error.stdout);
    if (error.stderr?.trim()) process.stderr.write(error.stderr);
    throw new Error(`${command} ${args.join(" ")} failed: ${error.message}`, {
      cause: error,
    });
  }
  if (result.stdout.trim()) process.stdout.write(result.stdout);
  if (result.stderr.trim()) process.stderr.write(result.stderr);
}

const temporaryDirectory = await mkdtemp(join(tmpdir(), "bas-j07-"));
const observationsPath = join(temporaryDirectory, "observations.jsonl");
const buildBefore = await buildIdentity();
const outputPath =
  process.env.BAS_J07_RECEIPT ||
  resolve(
    scenarioRoot,
    ".vrooli/runtime/rehabilitation-evidence",
    `cancellation-recovery-${new Date().toISOString().replace(/[:.]/g, "-")}-${randomUUID().slice(0, 8)}.json`,
  );

try {
  await run(
    "go",
    [
      "test",
      "-race",
      "-timeout",
      "30s",
      "-p",
      "1",
      "-run",
      executionTests,
      "-count=1",
      "./services/workflow",
      "./automation/executor",
    ],
    apiRoot,
    observationsPath,
  );
  await run(
    "pnpm",
    [
      "exec",
      "jest",
      "tests/integration/typed-action-semantics.test.ts",
      "--runInBand",
      "--coverage=false",
      `--testNamePattern=${retryTest}`,
    ],
    driverRoot,
    observationsPath,
  );
  await run(
    "node",
    ["cmd/cancellation-restart-cohort/qualification.mjs"],
    apiRoot,
    observationsPath,
  );

  const buildAfter = await buildIdentity();
  if (buildAfter !== buildBefore) {
    throw new Error(
      `managed BAS identity changed during owner cohort: ${buildBefore} -> ${buildAfter}`,
    );
  }
  await run(
    "go",
    [
      "run",
      "./cmd/cancellation-qualification",
      "--build",
      buildBefore,
      "--observations",
      observationsPath,
      "--output",
      outputPath,
    ],
    apiRoot,
    observationsPath,
  );

  const receipt = JSON.parse(await readFile(outputPath, "utf8"));
  if (receipt.runtime?.buildIdentity !== buildBefore) {
    throw new Error(
      "written J07 receipt does not bind to the measured managed build",
    );
  }
  const cases = Object.keys(receipt.cases || {}).sort();
  const required = [
    "apiRestart",
    "cancellation",
    "driverDeath",
    "retriedStart",
    "timeout",
  ];
  if (JSON.stringify(cases) !== JSON.stringify(required)) {
    throw new Error(
      `receipt case set was ${cases.join(",")}, want ${required.join(",")}`,
    );
  }
  console.log(
    JSON.stringify(
      {
        receipt: outputPath,
        buildIdentity: buildBefore,
        cases: Object.fromEntries(
          Object.entries(receipt.cases).map(([name, observation]) => [
            name,
            {
              ownerTest: observation.ownerTest,
              externalEffects: observation.externalEffects,
              terminalStatus: observation.terminalStatus,
              resources: [
                observation.liveResourcesBeforeClose,
                observation.liveResourcesAfterClose,
              ],
              inputStoppedMs: observation.inputStoppedMs,
              cleanupMs: observation.cleanupMs,
              recoveryMs: observation.recoveryMs,
              uncertainEffect: observation.uncertainEffect,
              retryAdmitted: observation.retryAdmitted,
            },
          ]),
        ),
      },
      null,
      2,
    ),
  );
} finally {
  await rm(temporaryDirectory, { recursive: true, force: true });
}
