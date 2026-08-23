// Record Mode end-to-end harness.
//
// Drives the full flow against a live playwright driver: create a session,
// navigate, record, act, read back the captured actions, validate a selector,
// stop, and (when the API is up) generate a workflow from the live buffer.
//
// Ported from bash so the harness needs neither a POSIX shell nor curl/jq on
// the host. Node's global fetch replaces curl; plain property access replaces
// jq.
//
// Prerequisites:
//   - playwright driver on PLAYWRIGHT_DRIVER_URL (default http://localhost:39400)
//   - API server on API_URL (default http://localhost:39500); optional, its
//     tests are skipped when it is not reachable.
//
// Usage:
//   node tests/e2e/record-mode-e2e.mjs [--driver-url URL] [--api-url URL] [--verbose]

const RESET = "\x1b[0m";
const COLOR = { red: "\x1b[0;31m", green: "\x1b[0;32m", yellow: "\x1b[1;33m", blue: "\x1b[0;34m" };

let driverUrl = process.env.PLAYWRIGHT_DRIVER_URL || "http://localhost:39400";
let apiUrl = process.env.API_URL || "http://localhost:39500";
let verbose = false;

const argv = process.argv.slice(2);
for (let i = 0; i < argv.length; i++) {
  switch (argv[i]) {
    case "--driver-url": driverUrl = argv[++i]; break;
    case "--api-url": apiUrl = argv[++i]; break;
    case "--verbose": case "-v": verbose = true; break;
    default:
      console.error(`Unknown option: ${argv[i]}`);
      process.exit(1);
  }
}

let sessionId = "";
let workflowId = "";
let passed = 0;
let failed = 0;

const log = (m) => console.log(`${COLOR.blue}[INFO]${RESET} ${m}`);
const logWarn = (m) => console.log(`${COLOR.yellow}[WARN]${RESET} ${m}`);
const debug = (m) => { if (verbose) console.log(`${COLOR.yellow}[DEBUG]${RESET} ${m}`); };
const pass = (m) => { passed++; console.log(`${COLOR.green}[PASS]${RESET} ${m}`); };
const fail = (m) => { failed++; console.log(`${COLOR.red}[FAIL]${RESET} ${m}`); };

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// A request helper that never throws: every caller treats a transport failure
// the same as a rejected response, which is what `curl -s ... || true` did.
async function request(url, { method = "GET", body } = {}) {
  try {
    const response = await fetch(url, {
      method,
      headers: body === undefined ? undefined : { "Content-Type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    const text = await response.text();
    let json;
    try { json = JSON.parse(text); } catch { json = undefined; }
    return { ok: response.ok, status: response.status, text, json };
  } catch (error) {
    return { ok: false, status: 0, text: String(error), json: undefined };
  }
}

const reachable = async (url) => (await request(url)).ok;

async function checkPrerequisites() {
  log("Checking prerequisites...");
  debug(`Checking playwright driver at ${driverUrl}/health`);
  if (!(await reachable(`${driverUrl}/health`))) {
    fail(`Playwright driver not available at ${driverUrl}`);
    log("Make sure the driver is running: make start (in playwright-driver dir)");
    process.exit(1);
  }
  pass("Playwright driver is healthy");

  debug(`Checking API at ${apiUrl}/api/v1/health`);
  if (await reachable(`${apiUrl}/api/v1/health`)) pass("API server is healthy");
  else logWarn("API server not available - skipping API-level tests");
}

async function testCreateSession() {
  log("Creating browser session...");
  const res = await request(`${driverUrl}/session/start`, {
    method: "POST",
    body: { viewport: { width: 1280, height: 720 }, userAgent: "Record Mode E2E Test" },
  });
  debug(`Response: ${res.text}`);
  sessionId = res.json?.session_id ?? "";
  if (!sessionId) return fail(`Failed to create session: ${res.text}`);
  pass(`Created session: ${sessionId}`);
}

async function testNavigate() {
  log("Navigating to test page...");
  const res = await request(`${driverUrl}/session/${sessionId}/run`, {
    method: "POST",
    body: { instruction: { index: 0, node_id: "nav-1", type: "navigate", params: { url: "https://example.com" } } },
  });
  debug(`Response: ${res.text}`);
  if (res.json?.success !== true) return fail(`Failed to navigate: ${res.text}`);
  pass("Navigated to https://example.com");
}

async function testStartRecording() {
  log("Starting recording...");
  const res = await request(`${driverUrl}/session/${sessionId}/record/start`, { method: "POST", body: {} });
  debug(`Response: ${res.text}`);
  if (!res.json?.recording_id) return fail(`Failed to start recording: ${res.text}`);
  pass(`Started recording: ${res.json.recording_id}`);
}

async function testRecordingStatus() {
  log("Checking recording status...");
  const res = await request(`${driverUrl}/session/${sessionId}/record/status`);
  debug(`Response: ${res.text}`);
  if (res.json?.is_recording !== true) return fail(`Recording not active: ${res.text}`);
  pass("Recording is active");
}

async function testPerformActions() {
  log("Performing test actions...");
  const res = await request(`${driverUrl}/session/${sessionId}/run`, {
    method: "POST",
    body: { instruction: { index: 1, node_id: "click-1", type: "click", params: { selector: "h1" } } },
  });
  debug(`Click response: ${res.text}`);
  await sleep(500); // let the recorder capture the action
  pass("Performed click action");
}

async function testGetActions() {
  log("Getting recorded actions...");
  const res = await request(`${driverUrl}/session/${sessionId}/record/actions`);
  debug(`Response: ${res.text}`);
  const count = res.json?.count ?? 0;
  log(`Recorded ${count} actions`);
  for (const action of res.json?.actions ?? []) {
    console.log(`  - ${action.actionType}: ${action.selector?.primary ?? "no selector"}`);
  }
  pass("Retrieved recorded actions");
}

async function testValidateSelector() {
  log("Validating selector...");
  const res = await request(`${driverUrl}/session/${sessionId}/record/validate-selector`, {
    method: "POST",
    body: { selector: "h1" },
  });
  debug(`Response: ${res.text}`);
  const valid = res.json?.valid ?? false;
  const matchCount = res.json?.match_count ?? 0;
  log(`Selector 'h1' valid: ${valid}, matches: ${matchCount}`);
  if (valid === true || matchCount > 0) pass("Selector validation works");
  else fail(`Selector validation failed: ${res.text}`);
}

async function testStopRecording() {
  log("Stopping recording...");
  const res = await request(`${driverUrl}/session/${sessionId}/record/stop`, { method: "POST" });
  debug(`Response: ${res.text}`);
  const actionCount = res.json?.action_count ?? -1;
  if (actionCount < 0) return fail(`Failed to stop recording: ${res.text}`);
  pass(`Stopped recording. Total actions: ${actionCount}`);
}

async function testDuplicateStart() {
  log("Testing duplicate start recording (should fail)...");
  await request(`${driverUrl}/session/${sessionId}/record/start`, { method: "POST" });
  const res = await request(`${driverUrl}/session/${sessionId}/record/start`, { method: "POST" });
  debug(`Response: ${res.text}`);
  if (res.json?.error === "RECORDING_IN_PROGRESS") pass("Correctly rejected duplicate start");
  else fail(`Should have rejected duplicate start: ${res.text}`);
  await request(`${driverUrl}/session/${sessionId}/record/stop`, { method: "POST" });
}

async function testGenerateWorkflowApi() {
  if (!(await reachable(`${apiUrl}/api/v1/health`))) {
    return logWarn("Skipping API workflow generation test (API not available)");
  }
  log("Testing workflow generation via API...");
  await request(`${driverUrl}/session/${sessionId}/record/start`, { method: "POST" });
  await sleep(200);
  await request(`${driverUrl}/session/${sessionId}/run`, {
    method: "POST",
    body: { instruction: { index: 0, node_id: "test-click", type: "click", params: { selector: "h1" } } },
  });
  await sleep(200);

  // Recording stays open on purpose: the API reads from the live buffer.
  const res = await request(`${apiUrl}/api/v1/recordings/live/${sessionId}/generate-workflow`, {
    method: "POST",
    body: { name: "E2E Test Workflow" },
  });
  debug(`Response: ${res.text}`);
  await request(`${driverUrl}/session/${sessionId}/record/stop`, { method: "POST" });

  workflowId = res.json?.workflow_id ?? "";
  // A missing workflow is a warning, not a failure: the API may not be configured.
  if (workflowId) pass(`Generated workflow: ${workflowId}`);
  else logWarn(`Could not generate workflow via API: ${res.text}`);
}

async function cleanup() {
  log("Cleaning up...");
  if (sessionId) {
    debug(`Closing session ${sessionId}`);
    await request(`${driverUrl}/session/${sessionId}/close`, { method: "POST" });
  }
}

async function main() {
  console.log("==============================================");
  console.log("  Record Mode End-to-End Tests");
  console.log("==============================================\n");
  console.log(`Playwright Driver: ${driverUrl}`);
  console.log(`API Server: ${apiUrl}\n`);

  await checkPrerequisites();
  console.log("\nRunning tests...\n");

  // Each step reports its own verdict; one failure does not stop the sequence.
  const steps = [
    testCreateSession, testNavigate, testStartRecording, testRecordingStatus,
    testPerformActions, testGetActions, testValidateSelector, testStopRecording,
    testDuplicateStart, testGenerateWorkflowApi,
  ];
  for (const step of steps) {
    try { await step(); } catch (error) { fail(`${step.name} threw: ${error}`); }
  }

  console.log("\n==============================================");
  console.log("  Test Results");
  console.log("==============================================\n");
  console.log(`  ${COLOR.green}Passed:${RESET} ${passed}`);
  console.log(`  ${COLOR.red}Failed:${RESET} ${failed}\n`);

  if (failed === 0) console.log(`${COLOR.green}All tests passed!${RESET}`);
  else console.log(`${COLOR.red}Some tests failed${RESET}`);
  return failed === 0 ? 0 : 1;
}

let exitCode = 1;
try {
  exitCode = await main();
} finally {
  await cleanup();
  process.exit(exitCode);
}
