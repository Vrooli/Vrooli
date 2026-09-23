import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { once } from 'node:events';
import { createServer } from 'node:http';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

// Exercise the registered producer over HTTP. This controlled driver only tests
// the harness's verdicts; the separate live run supplies native browser proof.
async function runHarness(fault, { apiAvailable = true, requireApi = false } = {}) {
  const sessions = new Map();
  const projectID = randomUUID(), workflowID = randomUUID(), executionID = randomUUID();
  const observed = { close: 0, deleteWorkflow: 0, deleteProject: 0, ack: 0, owned: 0, execute: 0, retention: 0 };
  const server = createServer(async (req, res) => {
    try {
      let raw = '';
      for await (const part of req) raw += part;
      const body = raw ? JSON.parse(raw) : {};
      const path = req.url;
      let status = 200, result = {};
      const id = path.split('/')[2], session = sessions.get(id);
      const click = async () => {
        if (fault === 'click') { status = 500; result = { error: 'refused click' }; return; }
        if (session?.url) {
          await fetch(new URL('/effect', session.url), { method: 'POST', headers: { Cookie: session.cookie } });
          session.clicked = true;
        }
        result = { success: true };
      };
      if (path === '/health' || path === '/api/v1/health') result = { status: 'ok' };
      else if (path === '/session/start') {
        const sessionID = randomUUID(), lease = randomUUID();
        const prior = [...sessions.values()][0];
        sessions.set(sessionID, { lease, owner: body.execution_id, recording: false, cookie: fault === 'reuse' ? prior?.cookie : undefined });
        result = { session_id: sessionID, lease_id: lease, phase: 'ready' };
      } else if (path.endsWith('/run')) {
        const instruction = body.instruction;
        const url = instruction?.action?.navigate?.url ?? instruction?.params?.url;
        if (url) {
          if (url.startsWith('http://127.0.0.1:')) {
            const page = await fetch(url, { headers: { Cookie: session.cookie ?? '' } });
            session.url = url;
            session.cookie = page.headers.get('set-cookie')?.split(';')[0] ?? session.cookie ?? '';
          }
          result = { success: true };
        } else if (fault === 'outcome') result = { success: false, error: { code: 'REFUSED_REPLAY' } };
        else await click();
      } else if (path.endsWith('/record/input')) await click();
      else if (path.endsWith('/record/start')) {
        if (session.recording) { status = 409; result = { error: 'RECORDING_IN_PROGRESS' }; }
        else {
          session.recording = true; session.recordingID = randomUUID();
          result = { recording_id: session.recordingID, started_at: new Date().toISOString() };
        }
      } else if (path.endsWith('/record/status')) result = { is_recording: session.recording, recording_id: session.recordingID };
      else if (path.endsWith('/record/stop')) {
        session.recording = false;
        result = { recording_id: session.recordingID, action_count: 2, stopped_at: new Date().toISOString() };
      } else if (path.endsWith('/record/actions')) {
        if (fault === 'read') { status = 500; result = { error: 'refused recording read' }; }
        else result = { count: 2, entries: [
          { id: 'navigate', action: { type: 'ACTION_TYPE_NAVIGATE', navigate: { url: session.url } } },
          { id: 'click', action: { type: 'ACTION_TYPE_CLICK', click: { selector: '#counter' } } },
        ] };
      } else if (path.endsWith('/record/actions/ack')) {
        if (fault !== 'ack-owner' && (!body.execution_id || !body.lease_id)) {
          status = 400; result = { error: 'missing acknowledgement owner' };
        } else if (fault !== 'ack-owner' && (body.execution_id !== session.owner || body.lease_id !== session.lease)) {
          status = 404; result = { error: 'stale acknowledgement owner' };
        } else {
          observed.ack++; result = { entry_ids: fault === 'ack' ? [] : body.entry_ids };
        }
      } else if (path.endsWith('/record/validate-selector')) result = { valid: true, match_count: 1 };
      else if (path.endsWith('/generate-workflow')) {
        if (fault === 'generation') { status = 500; result = { error: 'refused generation' }; }
        else result = { workflow_id: workflowID, project_id: projectID, node_count: 2 };
      } else if (path.endsWith('/CreateProject')) result = { project: { id: projectID } };
      else if (path.endsWith('/GetWorkflow')) result = { workflow: { id: workflowID, version: 1, flowDefinition: { nodes: [{ id: 'navigate' }, { id: 'click' }] } } };
      else if (path.endsWith('/ExecuteWorkflow')) {
        observed.execute++;
        assert.equal(body.workflow_id, workflowID);
        assert.equal(body.workflow_version, 1);
        assert.equal(body.wait_for_completion, true);
        if (fault !== 'api-missing-effect' && fault !== 'api-execution') {
          const url = [...sessions.values()][0].url;
          const page = await fetch(url);
          const cookie = page.headers.get('set-cookie').split(';')[0];
          await fetch(new URL('/effect', url), { method: 'POST', headers: { Cookie: cookie } });
        }
        result = { executionId: executionID, status: fault === 'api-execution' ? 'EXECUTION_STATUS_FAILED' : 'EXECUTION_STATUS_COMPLETED' };
      } else if (path.endsWith('/GetExecutionTimeline')) result = {
        executionId: executionID, workflowId: workflowID, status: 'EXECUTION_STATUS_COMPLETED',
        entries: ['navigate', 'click'].map(nodeId => ({ nodeId, aggregates: { status: fault === 'api-timeline' ? 'STEP_STATUS_FAILED' : 'STEP_STATUS_COMPLETED' }, context: { success: fault !== 'api-timeline' } })),
      };
      else if (path.endsWith('/PreviewExecutionArtifactRetention') || path.endsWith('/RunExecutionArtifactRetention')) {
        assert.equal(body.workflow_id, workflowID);
        assert.equal(body.project_id, projectID);
        if (path.endsWith('/RunExecutionArtifactRetention')) {
          assert.equal(body.confirm, true); observed.retention++;
        }
        result = { removed: [{ executionId: fault === 'api-foreign-retention' ? randomUUID() : executionID }], removedCount: 1 };
      }
      else if (path.endsWith('/DeleteWorkflow')) { observed.deleteWorkflow++; result = { success: true }; }
      else if (path.endsWith('/DeleteProject')) { observed.deleteProject++; result = { filesDeleted: true }; }
      else if (path.endsWith('/close')) {
        observed.close++;
        if (fault === 'cleanup') { status = 500; result = { error: 'refused close' }; }
        else result = { success: true };
      } else { status = 404; result = { error: 'unexpected fixture route: ' + path }; }
      if (session && body.execution_id === session.owner && body.lease_id === session.lease) observed.owned++;
      res.writeHead(status, { 'Content-Type': 'application/json' });
      res.end(fault === 'malformed' && path.endsWith('/record/actions') ? 'not JSON' : JSON.stringify(result));
    } catch (error) { res.writeHead(500); res.end(JSON.stringify({ error: String(error) })); }
  });
  server.listen(0, '127.0.0.1');
  await once(server, 'listening');
  const url = `http://127.0.0.1:${server.address().port}`;
  let output = '';
  const child = spawn(process.execPath, [fileURLToPath(new URL('./record-mode-e2e.mjs', import.meta.url)), '--driver-url', url, '--api-url', apiAvailable ? url : url + '/unavailable', ...(requireApi ? ['--require-api'] : [])], { stdio: ['ignore', 'pipe', 'pipe'] });
  child.stdout.on('data', data => { output += data; });
  child.stderr.on('data', data => { output += data; });
  const timer = setTimeout(() => child.kill('SIGKILL'), 20000);
  try {
    const [code, signal] = await once(child, 'exit');
    assert.equal(signal, null, output);
    return { code, output, observed };
  } finally {
    clearTimeout(timer);
    server.closeAllConnections();
    await new Promise(resolve => server.close(resolve));
  }
}

for (const fault of ['click', 'read', 'generation', 'cleanup']) {
  test(`the recording harness fails when ${fault} fails`, async () => {
    const result = await runHarness(fault);
    assert.notEqual(result.code, 0, result.output);
    assert.match(result.output, new RegExp('refused ' + (fault === 'read' ? 'recording read' : fault === 'cleanup' ? 'close' : fault)));
    if (fault === 'generation') assert.equal(result.observed.ack, 1, result.output);
    if (fault === 'cleanup') assert.equal(result.observed.deleteProject, 1, result.output);
  });
}

test('the recording harness rejects an incomplete acknowledgement', async () => {
  const result = await runHarness('ack');
  assert.notEqual(result.code, 0, result.output);
});

test('the recording harness rejects an acknowledgement accepted from an invalid owner', async () => {
  const result = await runHarness('ack-owner');
  assert.notEqual(result.code, 0, result.output);
  assert.match(result.output, /FAIL.*invalid owners leave recorded entries available/);
});

test('a successful HTTP response cannot hide a rejected replay action', async () => {
  const result = await runHarness('outcome');
  assert.notEqual(result.code, 0, result.output);
  assert.match(result.output, /REFUSED_REPLAY/);
});

test('malformed recording JSON fails qualification', async () => {
  const result = await runHarness('malformed');
  assert.notEqual(result.code, 0, result.output);
  assert.match(result.output, /response is not JSON/);
});

test('the recording harness detects a reused context during fresh replay', async () => {
  const result = await runHarness('reuse');
  assert.notEqual(result.code, 0, result.output);
  assert.match(result.output, /Replay reused recording context identity/);
});

test('an unavailable optional API is visibly unqualified', async () => {
  const result = await runHarness(undefined, { apiAvailable: false });
  assert.equal(result.code, 0, result.output);
  assert.match(result.output, /\[UNQUALIFIED\]/);
  assert.equal(result.observed.deleteProject, 0);
  assert.equal(result.observed.close, 2);
});

test('an unavailable required API fails qualification', async () => {
  const result = await runHarness(undefined, { apiAvailable: false, requireApi: true });
  assert.notEqual(result.code, 0, result.output);
  assert.match(result.output, /required API prerequisite/);
  assert.equal(result.observed.close, 0);
});

for (const fault of ['api-execution', 'api-missing-effect', 'api-timeline', 'api-foreign-retention']) {
  test(`saved workflow qualification rejects ${fault}`, async () => {
    const result = await runHarness(fault);
    assert.notEqual(result.code, 0, result.output);
    assert.equal(result.observed.execute, 1, result.output);
    if (fault === 'api-foreign-retention') assert.equal(result.observed.retention, 0, result.output);
  });
}

test('the complete controlled journey cleans up both sessions and API data', async () => {
  const result = await runHarness();
  assert.equal(result.code, 0, result.output);
  assert.equal(result.observed.close, 2, result.output);
  assert.equal(result.observed.deleteWorkflow, 1, result.output);
  assert.equal(result.observed.deleteProject, 1, result.output);
  assert.equal(result.observed.ack, 1, result.output);
  assert.ok(result.observed.owned >= 6, result.output);
  assert.equal(result.observed.execute, 1, result.output);
  assert.equal(result.observed.retention, 1, result.output);
});
