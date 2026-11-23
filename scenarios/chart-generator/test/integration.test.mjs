import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { CHART_URL, createBASChartClient } from './bas-client.mjs';
import { runChartIntegration } from './integration-core.mjs';

const OPEN = '/browser_automation_studio.v1.agentaccess.BrowserSessionService/OpenSession';
const CLOSE = '/browser_automation_studio.v1.agentaccess.BrowserSessionService/CloseSession';
const ref = n => `e${n}`;
const node = (role, name, id, children = []) => ({ role, name, ref: ref(id), children });
const chartTypes = ['bar', 'line', 'pie', 'scatter', 'area'];
const styles = ['professional', 'minimal', 'vibrant'];

async function startFakeOwner({ denyClickAt = 0, redirectAt = 0 } = {}) {
  const calls = [];
  let currentPath = '/';
  let screenshots = 0;
  let clicks = 0;
  let chart = 'Bar chart';
  let redirected = false;
  let tab = 'charts';
  const server = createServer(async (request, response) => {
    const chunks = [];
    for await (const chunk of request) chunks.push(chunk);
    const body = chunks.length ? JSON.parse(Buffer.concat(chunks).toString()) : {};
    calls.push({ path: request.url, headers: request.headers, body });
    const reply = (status, value) => { response.writeHead(status, { 'Content-Type': 'application/json' }); response.end(JSON.stringify(value)); };
    if (request.url === OPEN) return reply(200, { sessionId: 'chart-session' });
    if (request.url === CLOSE) return reply(200, { closed: true });
    if (request.url !== '/api/v1/browser/sessions/chart-session/task') return reply(404, {});
    if (body.operation === 'screenshot') {
      screenshots++;
      if (redirectAt === screenshots) redirected = true;
      return reply(200, { url: `http://127.0.0.1:37957${redirected ? '/login' : currentPath}`, title: 'Professional Chart Generator' });
    }
    if (body.operation === 'read') {
      const children = [node('region', 'Chart preview', 100)];
      if (tab === 'charts') {
        children.push(node('tab', 'Chart library', 10), node('tab', 'Styles', 11));
        chartTypes.forEach((type, index) => children.push(node('button', {
          bar: 'Bar chart Compare categories', line: 'Line chart Track performance', pie: 'Pie chart Show proportional',
          scatter: 'Scatter plot Highlight correlations', area: 'Area chart Emphasise magnitude',
        }[type], index + 1)));
      } else {
        children.push(node('tab', 'Chart library', 10), node('tab', 'Styles', 11));
        styles.forEach((style, index) => children.push(node('button', {
          professional: 'Professional Executive-ready styling', minimal: 'Minimal Muted neutrals', vibrant: 'Vibrant High-energy palette',
        }[style], index + 7)));
      }
      children.push(node('img', chart, 20));
      return reply(200, { tree: { url: 'http://127.0.0.1:37957', root: node('document', 'Chart Generator', 99, children) } });
    }
    if (body.operation === 'click') {
      clicks++;
      if (denyClickAt === clicks) return reply(403, { error: 'BROWSER_PERMISSION_DENIED' });
      const clickMap = new Map([
        [ref(1), ['chart', 'Bar chart']], [ref(2), ['chart', 'Line chart']], [ref(3), ['chart', 'Pie chart']],
        [ref(4), ['chart', 'Scatter plot']], [ref(5), ['chart', 'Area chart']],
        [ref(7), ['style', 'Professional']], [ref(8), ['style', 'Minimal']], [ref(9), ['style', 'Vibrant']],
      ]);
      if (body.ref === ref(11)) tab = 'styles';
      if (body.ref === ref(10)) tab = 'charts';
      const value = clickMap.get(body.ref);
      if (value?.[0] === 'chart') chart = value[1];
      return reply(200, { outcome: { success: true } });
    }
    return reply(400, { error: 'unsupported operation' });
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  return {
    apiURL: `http://127.0.0.1:${server.address().port}`, calls,
    close: () => new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())),
  };
}

const makeClient = owner => createBASChartClient({ apiURL: owner.apiURL, identityToken: 'am-id', grantReceipt: 'am-grant' });

for (const type of chartTypes) {
  test(`${type} chart generation uses the BAS-observed accessible preview`, async () => {
    const owner = await startFakeOwner();
    try {
      const result = await runChartIntegration({ brokerClient: makeClient(owner), kind: 'chart', value: type });
      assert.equal(result.check, `${type}-chart`);
      const open = owner.calls.find(call => call.path === OPEN);
      assert.deepEqual(open.body, { startUrl: CHART_URL, reuseMode: 'fresh', viewportWidth: 1440, viewportHeight: 1000 });
      assert.equal(Object.hasOwn(open.body, 'permissionRef'), false);
      assert.equal(owner.calls.at(-1).path, CLOSE);
      assert.ok(owner.calls.some(call => call.body.operation === 'click'));
      for (const call of owner.calls) {
        assert.equal(call.headers['x-agent-identity-token'], 'am-id');
        assert.equal(call.headers['x-vrooli-browser-grant-receipt'], 'am-grant');
        assert.equal(JSON.stringify(call.body).includes('am-grant'), false);
      }
    } finally { await owner.close(); }
  });
}

for (const style of styles) {
  test(`${style} style applies while chart preview remains accessible`, async () => {
    const owner = await startFakeOwner();
    try {
      const result = await runChartIntegration({ brokerClient: makeClient(owner), kind: 'style', value: style });
      assert.equal(result.check, `${style}-style`);
      assert.equal(owner.calls.at(-1).path, CLOSE);
    } finally { await owner.close(); }
  });
}

test('UI load retains the visible preview assertion and page title', async () => {
  const owner = await startFakeOwner();
  try {
    assert.deepEqual(await runChartIntegration({ brokerClient: makeClient(owner), kind: 'load' }), { check: 'preview-and-title' });
    assert.equal(owner.calls.at(-1).path, CLOSE);
  } finally { await owner.close(); }
});

test('denied click and redirected page fail closed and always close the session', async () => {
  for (const options of [{ denyClickAt: 1 }, { redirectAt: 2 }]) {
    const owner = await startFakeOwner(options);
    try {
      await assert.rejects(runChartIntegration({ brokerClient: makeClient(owner), kind: 'chart', value: 'bar' }), /denied|fixed application route/);
      assert.equal(owner.calls.at(-1).path, CLOSE);
    } finally { await owner.close(); }
  }
});

test('missing AM grant denies access before the BAS API call', () => {
  assert.throws(() => createBASChartClient({ apiURL: 'http://127.0.0.1:20000', identityToken: 'am-id' }), /identity and browser grant/);
});
