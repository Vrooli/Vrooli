#!/usr/bin/env node
import { randomUUID } from "node:crypto";
import { execFile } from "node:child_process";
import { createServer } from "node:http";
import { appendFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { promisify } from "node:util";
import { fileURLToPath } from "node:url";

const execFileAsync = promisify(execFile);
const sourcePath = fileURLToPath(import.meta.url);
const scenarioRoot = resolve(dirname(sourcePath), "../../..");
const repositoryRoot = resolve(scenarioRoot, "../..");
const api = process.env.BAS_API_URL || "http://127.0.0.1:17116";
const observationsPath = process.env.BAS_J07_OBSERVATIONS?.trim();
const executionRPC =
  "/browser_automation_studio.v1.ExecutionsService/GetExecution";
const adhocRPC =
  "/browser_automation_studio.v1.WorkflowsService/ExecuteAdhocWorkflow";

function assertLoopbackURL(value) {
  const url = new URL(value);
  if (
    url.protocol !== "http:" ||
    !["127.0.0.1", "localhost", "[::1]"].includes(url.hostname)
  ) {
    throw new Error(
      `BAS_API_URL must target a loopback HTTP address, got ${url.origin}`,
    );
  }
  return url.origin;
}

const apiOrigin = assertLoopbackURL(api);
const managedRestartTimeoutSeconds = 240;
const heldEffectTimeoutMs = managedRestartTimeoutSeconds * 1000 + 60_000;
if (!observationsPath) {
  throw new Error(
    "BAS_J07_OBSERVATIONS must name the owner-observation JSONL file",
  );
}

async function get(path) {
  const response = await fetch(`${apiOrigin}${path}`, {
    signal: AbortSignal.timeout(5000),
  });
  const body = await response.json();
  if (!response.ok)
    throw new Error(`${path}: ${response.status}: ${JSON.stringify(body)}`);
  return body;
}

async function post(path, body) {
  const response = await fetch(`${apiOrigin}${path}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Connect-Protocol-Version": "1",
    },
    body: JSON.stringify(body),
    signal: AbortSignal.timeout(10000),
  });
  const result = await response.json();
  if (!response.ok)
    throw new Error(`${path}: ${response.status}: ${JSON.stringify(result)}`);
  return result;
}

async function buildIdentity() {
  const health = await get("/health");
  const identity = String(health.build_identity || "").trim();
  if (!identity) throw new Error("/health returned no build_identity");
  return identity;
}

async function waitForEffect(promise, timeoutMs, label) {
  let timer;
  try {
    return await Promise.race([
      promise,
      new Promise((_, reject) => {
        timer = setTimeout(
          () => reject(new Error(`${label} timed out after ${timeoutMs}ms`)),
          timeoutMs,
        );
      }),
    ]);
  } finally {
    clearTimeout(timer);
  }
}

async function waitForTerminal(executionId, timeoutMs) {
  const started = performance.now();
  while (performance.now() - started < timeoutMs) {
    try {
      const body = await post(executionRPC, { executionId });
      const execution = body.execution;
      const status = String(execution?.status || "").toLowerCase();
      if (status.includes("failed") || status.includes("cancelled")) {
        return { execution, status, observedAt: performance.now() };
      }
      if (status.includes("completed"))
        throw new Error("restarted execution completed unexpectedly");
    } catch (error) {
      if (
        error instanceof Error &&
        error.message.includes("completed unexpectedly")
      )
        throw error;
      // Startup can temporarily refuse requests while the managed scenario returns.
    }
    await new Promise((resolveDelay) => setTimeout(resolveDelay, 100));
  }
  throw new Error(
    `execution ${executionId} did not recover to a terminal state within ${timeoutMs}ms`,
  );
}

async function waitForHealthDown(timeoutMs) {
  const deadline = performance.now() + timeoutMs;
  while (performance.now() < deadline) {
    try {
      const health = await get("/health");
      if (health.status !== "healthy" || health.readiness === false)
        return performance.now();
    } catch {
      return performance.now();
    }
    await new Promise((resolveDelay) => setTimeout(resolveDelay, 50));
  }
  throw new Error(
    `managed API stayed available during restart for ${timeoutMs}ms`,
  );
}

async function managedStopStartedAt() {
  const { stdout } = await execFileAsync(
    "vrooli",
    ["scenario", "status", "browser-automation-studio", "--json"],
    { cwd: repositoryRoot, timeout: 10000, maxBuffer: 2 * 1024 * 1024 },
  );
  const status = JSON.parse(stdout);
  const stopStep = status.scenario?.start_operation?.steps?.find(
    (step) => step.name === "stop" && step.status === "done",
  );
  if (!stopStep?.started_at)
    throw new Error(
      "managed scenario operation has no completed stop-step timestamp",
    );
  return Date.parse(stopStep.started_at);
}

async function restartManagedScenario() {
  try {
    return await execFileAsync(
      "vrooli",
      [
        "scenario",
        "restart",
        "browser-automation-studio",
        "--timeout",
        String(managedRestartTimeoutSeconds),
        "--json",
      ],
      {
        cwd: repositoryRoot,
        timeout: managedRestartTimeoutSeconds * 1000 + 10000,
        maxBuffer: 4 * 1024 * 1024,
      },
    );
  } catch (error) {
    if (error.code !== 124) throw error;
    return await execFileAsync(
      "vrooli",
      [
        "scenario",
        "wait",
        "browser-automation-studio",
        "--timeout",
        "120",
        "--json",
      ],
      { cwd: repositoryRoot, timeout: 130000, maxBuffer: 4 * 1024 * 1024 },
    );
  }
}

const effectState = { count: 0, live: 0, acceptedAt: 0, releasedAt: 0 };
let resolveEffect;
let resolveRelease;
const effectAccepted = new Promise((resolvePromise) => {
  resolveEffect = resolvePromise;
});
const effectReleased = new Promise((resolvePromise) => {
  resolveRelease = resolvePromise;
});
let server;
let beforeBuild;
let executionId;
let restartStartedAt;
let restartStartedWall;
let restartFinishedAt;
let outcome;

try {
  beforeBuild = await buildIdentity();
  server = createServer((request, response) => {
    if (request.method !== "GET" || request.url !== "/effect") {
      response.writeHead(404).end();
      return;
    }
    effectState.count += 1;
    effectState.live += 1;
    effectState.acceptedAt = performance.now();
    resolveEffect();
    let released = false;
    response.on("close", () => {
      if (released || response.writableEnded) return;
      released = true;
      effectState.live -= 1;
      effectState.releasedAt = performance.now();
      resolveRelease();
    });
    // Hold the navigation response across the managed API/driver restart.
  });
  await new Promise((resolveListen, rejectListen) => {
    server.once("error", rejectListen);
    server.listen(0, "127.0.0.1", resolveListen);
  });
  const fixturePort = server.address().port;
  const nodeId = randomUUID();
  const result = await post(adhocRPC, {
    metadata: { name: `J07 managed restart ${randomUUID()}` },
    flowDefinition: {
      metadata: {
        name: "J07 managed restart owner",
        executionMode: "EXECUTION_MODE_OBSERVER",
      },
      settings: { headless: true, timeoutMs: heldEffectTimeoutMs },
      nodes: [
        {
          id: nodeId,
          action: {
            type: "ACTION_TYPE_NAVIGATE",
            navigate: {
              url: `http://127.0.0.1:${fixturePort}/effect`,
              timeoutMs: heldEffectTimeoutMs,
            },
          },
          executionSettings: { timeoutMs: heldEffectTimeoutMs },
        },
      ],
      edges: [],
    },
    waitForCompletion: false,
    parameters: { headless: true, viewportWidth: 640, viewportHeight: 480 },
  });
  executionId = String(result.executionId || "").trim();
  if (!executionId)
    throw new Error(
      `adhoc execution returned no execution_id: ${JSON.stringify(result)}`,
    );

  await waitForEffect(effectAccepted, 30000, "fixture external effect");
  const resourcesBeforeClose = effectState.live;
  if (effectState.count !== 1 || resourcesBeforeClose !== 1) {
    throw new Error(
      `fixture before restart observed ${effectState.count} effects and ${effectState.live} live resources, want1/1`,
    );
  }
  const before = await post(executionRPC, { executionId });
  const beforeStatus = String(before.execution?.status || "").toLowerCase();
  if (!beforeStatus.includes("running")) {
    throw new Error(
      `execution was not running while the external effect was held: ${beforeStatus}`,
    );
  }

  restartStartedAt = performance.now();
  restartStartedWall = Date.now();
  // Observe the interrupted execution as soon as the API can serve it again.
  // Waiting for `vrooli scenario restart` to finish first includes the outer
  // health-gate tail in the recovery measurement even when the durable
  // terminal result was already available.
  const outcomePromise = waitForTerminal(executionId, 370000).then(
    (value) => ({ value }),
    (error) => ({ error }),
  );
  const restartPromise = restartManagedScenario();
  const [healthDownAt] = await Promise.all([
    waitForHealthDown(370000),
    waitForEffect(
      effectReleased,
      370000,
      "fixture resource release after managed restart",
    ),
  ]);
  const releaseAfterRestart = effectState.releasedAt;
  const restart = await restartPromise;
  restartFinishedAt = performance.now();
  const restartResult = JSON.parse(restart.stdout);
  if (!restartResult.success) {
    throw new Error(
      `managed restart did not succeed: ${JSON.stringify(restartResult)}`,
    );
  }

  const afterBuild = await buildIdentity();
  if (afterBuild !== beforeBuild) {
    throw new Error(
      `managed restart changed candidate identity from ${beforeBuild} to ${afterBuild}; discard this observation`,
    );
  }
  const stopStartedWall = await managedStopStartedAt();
  const stopStartedAt =
    restartStartedAt + (stopStartedWall - restartStartedWall);
  const observedOutcome = await outcomePromise;
  if (observedOutcome.error) throw observedOutcome.error;
  outcome = observedOutcome.value;
  if (!outcome.status.includes("failed")) {
    throw new Error(
      `interrupted execution status was ${outcome.status}, want failed`,
    );
  }
  if (effectState.count !== 1 || effectState.live !== 0) {
    throw new Error(
      `fixture after restart observed ${effectState.count} effects and ${effectState.live} live resources, want1/0`,
    );
  }

  const observation = {
    ownerTest: "cancellation-restart-cohort/qualification.mjs",
    observed: true,
    passed: false,
    externalEffects: effectState.count,
    terminalStatus: "failed",
    liveResourcesBeforeClose: resourcesBeforeClose,
    liveResourcesAfterClose: effectState.live,
    inputStoppedMs: releaseAfterRestart - effectState.acceptedAt,
    cleanupMs: releaseAfterRestart - stopStartedAt,
    recoveryMs: outcome.observedAt - stopStartedAt,
    uncertainEffect: true,
    retryAdmitted: effectState.count !== 1,
  };
  if (
    observation.cleanupMs < 0 ||
    observation.cleanupMs > 5000 ||
    observation.recoveryMs > 10000
  ) {
    throw new Error(
      `J07 restart bands exceeded: ${JSON.stringify(observation)}`,
    );
  }
  observation.passed = true;
  await appendFile(
    observationsPath,
    `${JSON.stringify({ case: "apiRestart", observation })}\n`,
    { mode: 0o600 },
  );
  console.log(
    JSON.stringify(
      {
        executionId,
        beforeBuild,
        afterBuild,
        apiUnavailableMs: restartFinishedAt - healthDownAt,
        restartElapsedMs: restartFinishedAt - restartStartedAt,
        observation,
      },
      null,
      2,
    ),
  );
} finally {
  if (server) {
    server.closeAllConnections();
    await new Promise((resolveClose) => server.close(resolveClose));
  }
}
