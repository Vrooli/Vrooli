// Local recording qualification against managed driver/API instances.
// The fixture owns its effect log; captured entries are never their own oracle.
// Usage: node tests/e2e/record-mode-e2e.mjs --driver-url URL [--api-url URL]
//        [--require-api] [--j02-input] [--verbose]
// Optional API coverage remains unqualified when unavailable. Selected API
// generation, persistence and cleanup failures always fail the run.
import assert from 'node:assert/strict';
import { randomUUID } from 'node:crypto';
import { EventEmitter, once } from 'node:events';
import { mkdir, mkdtemp, open, writeFile } from 'node:fs/promises';
import { createServer } from 'node:http';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

let driverUrl = process.env.PLAYWRIGHT_DRIVER_URL || 'http://localhost:39400';
let apiUrl = process.env.API_URL || 'http://localhost:39500';
let requireApi = false, verbose = false, j02Input = false;
for (let i = 2; i < process.argv.length; i++) {
  const arg = process.argv[i];
  if (arg === '--require-api') requireApi = true;
  else if (arg === '--j02-input') j02Input = true;
  else if (arg === '--verbose' || arg === '-v') verbose = true;
  else if (arg === '--driver-url' || arg === '--api-url') {
    const value = process.argv[++i];
    if (!value || value.startsWith('--')) throw new Error(`Missing value for ${arg}`);
    if (arg === '--driver-url') driverUrl = value; else apiUrl = value;
  } else throw new Error(`Unknown option: ${arg}`);
}
driverUrl = driverUrl.replace(/\/$/, '');
apiUrl = apiUrl.replace(/\/$/, '');

async function request(base, path, body, expectedStatus) {
  const response = await fetch(base + path, {
    method: body === undefined ? 'GET' : 'POST',
    headers: { 'Content-Type': 'application/json', 'Connect-Protocol-Version': '1' },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(30000),
  });
  const text = await response.text();
  if (verbose) console.log(`${body === undefined ? 'GET' : 'POST'} ${path}: ${response.status}`);
  assert.ok(expectedStatus === undefined ? response.ok : response.status === expectedStatus,
    `${path}: HTTP ${response.status}: ${text.slice(0, 500)}`);
  try { return JSON.parse(text); }
  catch { throw new Error(`${path}: response is not JSON`); }
}

const cases = [], sessions = [], effects = [], effectEvents = new EventEmitter();
const inputObservations = [], inputEvents = new EventEmitter();
const artifactDir = await mkdtemp(join(tmpdir(), 'bas-recording-e2e-'));
let executionID, workflowVersion, workflowNodeIDs, projectID, workflowID, recordingSession, committedEntries, apiSelected = false, acknowledgementAttempted = false;
const projects = '/browser_automation_studio.v1.projects.ProjectsService/';
const workflows = '/browser_automation_studio.v1.WorkflowsService/';
const fixture = createServer(async (req, res) => {
  if (req.method === 'POST' && req.url === '/input-observation') {
    const chunks = [];
    for await (const chunk of req) chunks.push(chunk);
    inputObservations.push({ ...JSON.parse(Buffer.concat(chunks).toString('utf8')), context: req.headers.cookie ?? '' });
    inputEvents.emit('input');
    res.writeHead(204); res.end();
  }
  else if (req.method === 'POST' && req.url === '/effect') {
    effects.push({ sequence: effects.length + 1, context: req.headers.cookie ?? '', received_at: new Date().toISOString() });
    effectEvents.emit('effect');
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ count: effects.length }));
  } else if (req.method === 'GET' && req.url === '/') {
    const headers = { 'Content-Type': 'text/html' };
    if (!/(^|;\s*)fixture_context=/.test(req.headers.cookie ?? '')) {
      headers['Set-Cookie'] = `fixture_context=${randomUUID()}; Path=/; SameSite=Lax`;
    }
    res.writeHead(200, headers);
      res.end(`<!doctype html><meta charset="utf-8"><title>Recording qualification fixture</title>
      <input id="fixture-input" value="initial" style="position:absolute;left:20px;top:100px;width:220px;height:30px" />
      <button id="clipboard" style="position:absolute;left:260px;top:100px;width:150px;height:32px">Write clipboard</button>
      <button id="compose" style="position:absolute;left:20px;top:150px;width:150px;height:32px">Commit IME fixture</button>
      <button id="counter" style="position:absolute;left:20px;top:20px;width:160px;height:60px"
      onclick="fetch('/effect',{method:'POST'}).then(r=>r.json()).then(r=>this.textContent='Effect '+r.count)">Click fixture</button>
      <script>
        const reportInput = (type, details = {}) => fetch('/input-observation', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({type, ...details})});
        const reportKey = (type, event) => reportInput(type, {key:event.key, ctrlKey:event.ctrlKey, metaKey:event.metaKey, altKey:event.altKey, shiftKey:event.shiftKey});
        document.addEventListener('keydown', event => { void reportKey('keydown', event); });
        document.addEventListener('pointerdown', event => { void reportKey('pointerdown', event); });
        document.addEventListener('pointermove', event => { void reportKey('pointermove', event); });
        document.addEventListener('pointerup', event => { void reportKey('pointerup', event); });
        const input = document.querySelector('#fixture-input');
        input.addEventListener('input', event => { void reportInput('field', {value:input.value, inputType:event.inputType}); });
        document.querySelector('#clipboard').addEventListener('click', () => {
          input.focus();
          input.value = 'pasted value';
          input.dispatchEvent(new InputEvent('input', {bubbles:true, data:input.value, inputType:'insertFromPaste'}));
          void reportInput('clipboard-ready');
        });
        document.querySelector('#compose').addEventListener('click', () => {
          const value = '東京';
          input.focus();
          input.value = value;
          input.dispatchEvent(new CompositionEvent('compositionstart', {bubbles:true, data:''}));
          input.dispatchEvent(new CompositionEvent('compositionupdate', {bubbles:true, data:value}));
          input.dispatchEvent(new InputEvent('input', {bubbles:true, data:value, inputType:'insertCompositionText', isComposing:true}));
          input.dispatchEvent(new CompositionEvent('compositionend', {bubbles:true, data:value}));
          input.dispatchEvent(new InputEvent('input', {bubbles:true, data:value, inputType:'insertFromComposition', isComposing:false}));
        });
      </script>`);
  } else { res.writeHead(404); res.end(); }
});

function waitForEffects(count) {
  if (effects.length >= count) return Promise.resolve();
  return new Promise((resolve, reject) => {
    const finish = error => {
      clearTimeout(timer); effectEvents.removeListener('effect', observed);
      if (error) reject(error); else resolve();
    };
    const observed = () => { if (effects.length >= count) finish(); };
    const timer = setTimeout(() => finish(new Error(`Expected ${count} independent effects, observed ${effects.length}`)), 5000);
    effectEvents.on('effect', observed);
  });
}

function waitForInputObservation(predicate) {
  if (inputObservations.some(predicate)) return Promise.resolve(inputObservations.find(predicate));
  return new Promise((resolve, reject) => {
    const finish = error => {
      clearTimeout(timer); inputEvents.removeListener('input', observed);
      if (error) reject(error); else resolve(inputObservations.find(predicate));
    };
    const observed = () => { if (inputObservations.some(predicate)) finish(); };
    const timer = setTimeout(() => finish(new Error(`Timed out waiting for a browser input observation; recent events ${JSON.stringify(inputObservations.slice(-12))}`)), 5000);
    inputEvents.on('input', observed);
  });
}

async function check(name, operation) {
  try { await operation(); cases.push({ name, status: 'passed' }); console.log(`[PASS] ${name}`); }
  catch (error) { cases.push({ name, status: 'failed', error: String(error) }); console.error(`[FAIL] ${name}: ${error}`); throw error; }
}

async function createSession({ apiManaged = false, initialUrl } = {}) {
  if (apiManaged) {
    const response = await request(apiUrl, '/api/v1/recordings/live/session', {
      viewport_width: 640, viewport_height: 480, initial_url: initialUrl, restore_tabs: false,
    });
    const session = { id: response.session_id ?? response.sessionId, apiManaged: true, sequence: 0 };
    assert.ok(session.id, 'API did not return a recording session identity');
    sessions.push(session);
    return session;
  }
  const execution_id = randomUUID();
  const response = await request(driverUrl, '/session/start', {
    execution_id, workflow_id: randomUUID(), reuse_mode: 'fresh', viewport: { width: 640, height: 480 },
  });
  assert.ok(response.session_id, 'Driver did not return a session identity');
  const session = { id: response.session_id, execution_id, lease_id: response.lease_id, sequence: 0 };
  sessions.push(session);
  assert.ok(session.lease_id, 'Driver did not return its lease');
  return session;
}

const ownership = session => ({ execution_id: session.execution_id, lease_id: session.lease_id });
const record = (session, operation, body = {}) => request(driverUrl, `/session/${session.id}/record/${operation}`, { ...body, ...ownership(session) });
const sendInput = (session, body) => session.apiManaged
  ? request(apiUrl, `/api/v1/recordings/live/${session.id}/input`, body)
  : record(session, 'input', body);
async function run(session, action) {
  const sequence = ++session.sequence;
  const outcome = await request(driverUrl, `/session/${session.id}/run`, {
    ...ownership(session), operation_sequence: sequence, invocation_id: `${session.execution_id}:${sequence}`, attempt: 1,
    instruction: { index: sequence - 1, node_id: `fixture-${sequence}`, action },
  });
  assert.equal(outcome.success, true, `Driver rejected action: ${JSON.stringify(outcome.error)}`);
}

async function commitEntries(entries) {
  const file = await open(join(artifactDir, 'entries.json'), 'w');
  try { await file.writeFile(JSON.stringify(entries, null, 2)); await file.sync(); }
  finally { await file.close(); }
}

async function acknowledgeEntries() {
  const ids = committedEntries.map(entry => entry.id);
  if (recordingSession.apiManaged) {
    const result = await request(apiUrl, `/api/v1/recordings/live/${recordingSession.id}/actions?clear=true`);
    const acknowledged = (result.entries ?? result.actions ?? []).map(entry => entry.id);
    assert.deepEqual(acknowledged, ids, 'API did not commit and acknowledge the exact entries');
    acknowledgementAttempted = true;
    return;
  }
  const ack = await record(recordingSession, 'actions/ack', { entry_ids: ids });
  assert.deepEqual(ack.entry_ids, ids, 'Driver did not acknowledge the committed entries');
  acknowledgementAttempted = true;
}

async function generateWorkflow(session) {
  const folder = join(artifactDir, 'project');
  await mkdir(folder);
  const project = await request(apiUrl, projects + 'CreateProject', {
    name: 'Recording qualification ' + randomUUID(), folder_path: folder,
  });
  projectID = project.project?.id;
  assert.ok(projectID, 'API did not return a project identity');
  const generated = await request(apiUrl, `/api/v1/recordings/live/${session.id}/generate-workflow`, {
    name: 'Recorded fixture click', project_id: projectID,
  });
  workflowID = generated.workflow_id;
  assert.ok(workflowID && generated.node_count > 0, 'API did not persist a generated workflow');
  const saved = await request(apiUrl, workflows + 'GetWorkflow', { workflow_id: workflowID });
  assert.equal(saved.workflow?.id, workflowID, 'Persisted workflow identity changed');
  workflowVersion = saved.workflow.version;
  workflowNodeIDs = saved.workflow.flowDefinition?.nodes?.map(node => node.id);
  assert.ok(saved.workflow.flowDefinition?.nodes?.length > 0, 'Persisted workflow has no definition');
}

async function main() {
  fixture.listen(0, '127.0.0.1');
  await once(fixture, 'listening');
  const fixtureURL = `http://127.0.0.1:${fixture.address().port}/`;
  await check('driver prerequisite', () => request(driverUrl, '/health'));
  try { await request(apiUrl, '/health'); apiSelected = true; }
  catch (error) {
    if (requireApi) await check('required API prerequisite', async () => { throw error; });
    cases.push({ name: 'API workflow generation', status: 'skipped', reason: String(error) });
    console.log('[UNQUALIFIED] Optional API workflow generation is unavailable');
  }
  const finalInputValue = 'final J02 value';
  let session, entries;
  await check('owned recording captures the independent fixture click', async () => {
    session = await createSession({ apiManaged: j02Input && apiSelected, initialUrl: fixtureURL });
    recordingSession = session;
    let start;
    if (session.apiManaged) {
      start = await request(apiUrl, '/api/v1/recordings/live/start', { session_id: session.id });
      const active = await request(apiUrl, `/api/v1/recordings/live/${session.id}/status`);
      assert.equal(active.isRecording ?? active.is_recording, true);
    } else {
      await run(session, { type: 'ACTION_TYPE_NAVIGATE', navigate: { url: fixtureURL } });
      start = await record(session, 'start');
      assert.ok(start.recording_id && Number.isFinite(Date.parse(start.started_at)), 'Invalid recording start receipt');
      const active = await request(driverUrl, `/session/${session.id}/record/status`);
      assert.equal(active.is_recording, true);
      assert.equal(active.recording_id, start.recording_id);
      const duplicate = await request(driverUrl, `/session/${session.id}/record/start`, ownership(session), 409);
      assert.equal(duplicate.error, 'RECORDING_IN_PROGRESS');
    }

    if (j02Input) {
      const waitForFieldValue = value => waitForInputObservation(event => event.type === 'field' && event.value === value)
        .catch(error => { throw new Error(`${error.message}; expected field value ${JSON.stringify(value)}, observed ${JSON.stringify(inputObservations)}`); });
      const inputPause = () => new Promise(resolve => setTimeout(resolve, 650));
      await sendInput(session, { type: 'pointer', action: 'click', x: 80, y: 115, button: 'left' });
      await sendInput(session, { type: 'keyboard', text: 'typed before pause' });
      await inputPause();
      await sendInput(session, { type: 'keyboard', key: 'a', modifiers: ['Control'] });
      await sendInput(session, { type: 'keyboard', text: 'replacement' });
      await waitForFieldValue('replacement');
      await inputPause();
      await sendInput(session, { type: 'keyboard', key: 'Home' });
      await sendInput(session, { type: 'keyboard', key: 'ArrowRight', modifiers: ['Shift'] });
      await sendInput(session, { type: 'keyboard', key: 'ArrowRight', modifiers: ['Shift'] });
      await sendInput(session, { type: 'keyboard', key: 'Backspace' });
      await waitForFieldValue('placement');
      await inputPause();

      await sendInput(session, { type: 'pointer', action: 'click', x: 320, y: 115, button: 'left' });
      const clipboard = await waitForInputObservation(event => event.type === 'clipboard-ready' || event.type === 'clipboard-failed');
      assert.equal(clipboard.type, 'clipboard-ready', `Fixture clipboard write failed: ${clipboard.message ?? 'unknown error'}`);
      await waitForFieldValue('pasted value');
      await inputPause();

      await sendInput(session, { type: 'pointer', action: 'click', x: 80, y: 165, button: 'left' });
      await waitForFieldValue('東京');
      await inputPause();
      await sendInput(session, { type: 'pointer', action: 'click', x: 80, y: 115, button: 'left' });
      await sendInput(session, { type: 'keyboard', key: 'a', modifiers: ['Control'] });
      await sendInput(session, { type: 'keyboard', key: 'Backspace' });
      await waitForFieldValue('');
      await inputPause();
      await sendInput(session, { type: 'keyboard', text: finalInputValue });
      await waitForFieldValue(finalInputValue);
      await inputPause();
    }

    await sendInput(session, { type: 'pointer', action: 'click', x: 70, y: 45, button: 'left' });
    await waitForEffects(1);
    assert.equal(effects.length, 1);
    if (!session.apiManaged) {
      const selector = await record(session, 'validate-selector', { selector: '#counter' });
      assert.equal(selector.valid, true); assert.equal(selector.match_count, 1);
    }
    const stopped = session.apiManaged
      ? await request(apiUrl, `/api/v1/recordings/live/${session.id}/stop`, {})
      : await record(session, 'stop');
    if (session.apiManaged) assert.ok(stopped.recordingId ?? stopped.recording_id, 'API did not return a stopped recording identity');
    else {
      assert.equal(stopped.recording_id, start.recording_id);
      assert.ok(Number.isFinite(Date.parse(stopped.stopped_at)), 'Invalid recording stop receipt');
    }
    const captured = session.apiManaged
      ? await request(apiUrl, `/api/v1/recordings/live/${session.id}/actions`)
      : await request(driverUrl, `/session/${session.id}/record/actions`);
    entries = captured.entries ?? captured.actions;
    assert.ok(Array.isArray(entries) && entries.length >= 2, 'Missing recorded navigation/click');
    assert.equal(captured.count, entries.length);
    assert.equal(new Set(entries.map(entry => entry.id)).size, entries.length, 'Duplicate entry identity');
    assert.ok(entries.every(entry => typeof entry.id === 'string' && entry.id && entry.action), 'Invalid typed entry');
    assert.equal(entries.filter(entry => entry.action.type === 'ACTION_TYPE_CLICK' && entry.action.click?.selector === '#counter').length, 1);
    assert.ok(entries.some(entry => entry.action.navigate?.url === fixtureURL), 'Missing fixture navigation');
    if (j02Input) {
      const inputSnapshots = entries.map(entry => entry.action?.input?.value).filter(value => typeof value === 'string');
      assert.ok(inputSnapshots.includes(finalInputValue), `Recorded input missed the final full-value snapshot: ${JSON.stringify(inputSnapshots)}`);
      assert.equal(inputSnapshots.at(-1), finalInputValue, 'The final recorded input snapshot is not the fixture value');
    }
    await commitEntries(entries);
    committedEntries = entries;
  });
  if (!session.apiManaged) await check('modifier chords and pointer modifiers reach real Chromium events', async () => {
    const keyboardBaseline = inputObservations.length;
    await sendInput(session, { type: 'keyboard', key: 'a', modifiers: ['Control'] });
    const chord = await waitForInputObservation(event => inputObservations.indexOf(event) >= keyboardBaseline && event.type === 'keydown' && event.key === 'a');
    assert.equal(chord.ctrlKey, true, 'Control+A arrived as unmodified text');

    await sendInput(session, { type: 'pointer', action: 'down', x: 350, y: 250, button: 'left', modifiers: ['Shift'] });
    const down = await waitForInputObservation(event => event.type === 'pointerdown' && event.shiftKey === true);
    assert.equal(down.shiftKey, true, 'Shift was not held for pointer down');
    await sendInput(session, { type: 'pointer', action: 'move', x: 370, y: 250, modifiers: ['Shift'] });
    const move = await waitForInputObservation(event => event.type === 'pointermove' && event.shiftKey === true);
    assert.equal(move.shiftKey, true, 'Shift was not held through pointer drag');
    await sendInput(session, { type: 'pointer', action: 'up', x: 370, y: 250, button: 'left', modifiers: ['Shift'] });
    const up = await waitForInputObservation(event => event.type === 'pointerup' && event.shiftKey === true);
    assert.equal(up.shiftKey, true, 'Shift was released before pointer up');
  });
  if (apiSelected) await check('API persists and reads the generated workflow', () => generateWorkflow(session));
  if (!session.apiManaged) await check('invalid owners leave recorded entries available', async () => {
    const ids = entries.map(entry => entry.id);
    for (const [identity, status] of [[{}, 400], [{ execution_id: randomUUID(), lease_id: randomUUID() }, 404]]) {
      await request(driverUrl, `/session/${session.id}/record/actions/ack`, { ...identity, entry_ids: ids }, status);
      const preserved = await request(driverUrl, `/session/${session.id}/record/actions`);
      assert.deepEqual(preserved.entries.map(entry => entry.id), ids, 'Rejected acknowledgement hid recorded entries');
    }
  });
  await check('committed entries receive an exact acknowledgement', acknowledgeEntries);
  await check('captured typed actions replay in a fresh browser context', async () => {
    const replay = await createSession();
    for (const entry of entries) await run(replay, entry.action);
    await waitForEffects(2);
    assert.equal(effects.length, 2, 'Replay duplicated or lost an effect');
    assert.ok(effects[0].context && effects[1].context, 'Fixture identity cookie was missing');
    assert.notEqual(effects[0].context, effects[1].context, 'Replay reused recording context identity');
    if (j02Input) {
      assert.ok(inputObservations.some(event => event.type === 'field' && event.value === finalInputValue && event.context === effects[1].context),
        'Direct fresh-context replay did not reproduce the final fixture input');
    }
  });
  if (apiSelected) await check('saved workflow executes through the API owner', async () => {
    assert.ok(Number.isInteger(workflowVersion) && workflowVersion > 0, 'Missing exact workflow revision');
    const execution = await request(apiUrl, workflows + 'ExecuteWorkflow', {
      workflow_id: workflowID, workflow_version: workflowVersion, wait_for_completion: true,
    });
    executionID = execution.executionId;
    await writeFile(join(artifactDir, 'execution-receipt.json'), JSON.stringify(execution, null, 2));
    assert.ok(executionID, 'No execution identity');
    assert.equal(execution.status, 'EXECUTION_STATUS_COMPLETED', 'Owner did not confirm completion');
    await waitForEffects(3);
    assert.equal(effects.length, 3, 'Saved workflow lost or duplicated the fixture effect');
    if (j02Input) {
      await waitForInputObservation(event => event.type === 'field' && event.value === finalInputValue && event.context === effects[2].context);
    }
    const timeline = await request(apiUrl, '/browser_automation_studio.v1.ExecutionsService/GetExecutionTimeline', { execution_id: executionID });
    await writeFile(join(artifactDir, 'execution-timeline.json'), JSON.stringify(timeline, null, 2));
    assert.equal(timeline.executionId, executionID);
    assert.equal(timeline.workflowId, workflowID);
    assert.equal(timeline.status, 'EXECUTION_STATUS_COMPLETED');
    assert.deepEqual(timeline.entries?.map(entry => entry.nodeId).sort(), [...workflowNodeIDs].sort(), 'Missing or duplicate workflow node evidence');
    assert.ok(timeline.entries.every(entry => entry.aggregates?.status === 'STEP_STATUS_COMPLETED' && entry.context?.success === true), 'Timeline does not confirm successful steps');
  });
}

try { await main(); }
catch (error) {
  if (!cases.some(entry => entry.status === 'failed')) {
    cases.push({ name: 'harness setup', status: 'failed', error: String(error) });
    console.error(error);
  }
} finally {
  const cleanup = async (name, operation) => { try { await check(name, operation); } catch { /* Keep cleaning other owned resources. */ } };
  if (recordingSession && !acknowledgementAttempted) await cleanup('stop and acknowledge owned recording before cleanup', async () => {
    const status = recordingSession.apiManaged
      ? await request(apiUrl, `/api/v1/recordings/live/${recordingSession.id}/status`)
      : await request(driverUrl, `/session/${recordingSession.id}/record/status`);
    if (status.isRecording ?? status.is_recording) {
      if (recordingSession.apiManaged) await request(apiUrl, `/api/v1/recordings/live/${recordingSession.id}/stop`, {});
      else await record(recordingSession, 'stop');
    }
    const captured = recordingSession.apiManaged
      ? await request(apiUrl, `/api/v1/recordings/live/${recordingSession.id}/actions`)
      : await request(driverUrl, `/session/${recordingSession.id}/record/actions`);
    const pending = captured.entries ?? captured.actions ?? [];
    if (pending.length) {
      committedEntries = pending;
      await acknowledgeEntries();
    } else {
      acknowledgementAttempted = true;
    }
  });
  for (const session of sessions.reverse()) await cleanup('close owned session ' + session.id, async () => {
    if (session.apiManaged) {
      const closed = await request(apiUrl, `/api/v1/recordings/live/session/${session.id}/close`, {});
      assert.equal(closed.status, 'closed');
    } else {
      const closed = await request(driverUrl, `/session/${session.id}/close`, ownership(session));
      assert.equal(closed.success, true);
    }
  });
  if (executionID) await cleanup('remove only this fixture execution and artifacts', async () => {
    const filter = { workflow_id: workflowID, project_id: projectID, max_age_days: 0, keep_latest: 0 };
    const prefix = '/browser_automation_studio.v1.ExecutionsService/';
    const preview = await request(apiUrl, prefix + 'PreviewExecutionArtifactRetention', filter);
    await writeFile(join(artifactDir, 'retention-preview.json'), JSON.stringify(preview, null, 2));
    assert.deepEqual((preview.removed ?? []).map(row => row.executionId), [executionID], 'Retention preview exceeded the fixture execution');
    const removed = await request(apiUrl, prefix + 'RunExecutionArtifactRetention', { ...filter, confirm: true });
    await writeFile(join(artifactDir, 'retention-result.json'), JSON.stringify(removed, null, 2));
    assert.deepEqual((removed.removed ?? []).map(row => row.executionId), [executionID]);
    assert.equal(removed.errorCount ?? 0, 0);
  });
  if (workflowID) await cleanup('delete fixture workflow', async () => {
    const deleted = await request(apiUrl, workflows + 'DeleteWorkflow', { workflow_id: workflowID });
    assert.equal(deleted.success, true);
  });
  if (projectID) await cleanup('delete fixture project and files', async () => {
    const deleted = await request(apiUrl, projects + 'DeleteProject', { id: projectID, delete_files: true });
    assert.equal(deleted.filesDeleted, true);
  });
  fixture.closeAllConnections();
  await new Promise(resolve => fixture.close(resolve));
  const result = { cases, effects, execution_id: executionID, artifact_dir: artifactDir, api_selected: apiSelected,
    passed: cases.filter(entry => entry.status === 'passed').length,
    failed: cases.filter(entry => entry.status === 'failed').length,
    skipped: cases.filter(entry => entry.status === 'skipped').length };
  await writeFile(join(artifactDir, 'result.json'), JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result));
  process.exitCode = result.failed ? 1 : 0;
}
