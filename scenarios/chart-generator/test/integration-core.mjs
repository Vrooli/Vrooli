import { CHART_URL } from './bas-client.mjs';

const chartLabels = new Map([
  ['bar', 'Bar chart'], ['line', 'Line chart'], ['pie', 'Pie chart'],
  ['scatter', 'Scatter plot'], ['area', 'Area chart'],
]);
const styleLabels = new Map([
  ['professional', 'Professional'], ['minimal', 'Minimal'], ['vibrant', 'Vibrant'],
]);

function treeNodes(root) {
  const result = [];
  const visit = node => {
    if (!node || typeof node !== 'object') return;
    result.push(node);
    for (const child of node.children ?? []) visit(child);
  };
  visit(root);
  return result;
}

function assertChartLocation(observation) {
  let url;
  try { url = new URL(observation?.url); } catch { throw new Error('BAS did not return a valid Chart Generator URL'); }
  if (url.origin !== new URL(CHART_URL).origin || url.pathname !== '/' || url.search || url.hash || url.username || url.password) {
    throw new Error('Chart Generator session left the fixed application route');
  }
  if (!/Chart Generator/i.test(observation?.title ?? '')) throw new Error('Chart Generator page title was not observed');
}

async function observeCurrent(client, sessionId) {
  const observation = await client.task(sessionId, 'screenshot');
  assertChartLocation(observation);
  const read = await client.task(sessionId, 'read');
  if (read?.tree?.url !== new URL(CHART_URL).origin) throw new Error('Chart Generator accessibility read left the fixed origin');
  return read.tree;
}

function findRef(tree, predicate, label) {
  const match = treeNodes(tree?.root).find(predicate);
  if (!match?.ref) throw new Error(`BAS tree omitted an actionable accessibility ref for ${label}`);
  return match.ref;
}

async function clickNamed(client, sessionId, role, startsWith, label) {
  const tree = await observeCurrent(client, sessionId);
  const ref = findRef(tree, node => node.role === role && node.name?.startsWith(startsWith), label);
  await client.task(sessionId, 'click', { ref });
}

async function requireRenderedChart(client, sessionId, expectedLabel) {
  for (let attempt = 0; attempt < 12; attempt++) {
    const tree = await observeCurrent(client, sessionId);
    if (treeNodes(tree.root).some(node => node.role === 'img' && node.name === expectedLabel)) return;
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  throw new Error(`Chart preview did not expose the rendered ${expectedLabel}`);
}

async function requirePreviewRegion(client, sessionId) {
  const tree = await observeCurrent(client, sessionId);
  if (!treeNodes(tree.root).some(node => node.role === 'region' && node.name === 'Chart preview')) {
    throw new Error('Chart preview container was not exposed');
  }
}

export async function runChartIntegration({ brokerClient, kind, value }) {
  if (!brokerClient?.openChartSession || !brokerClient?.task || !brokerClient?.closeSession) {
    throw new Error('a BAS Agent Access client is required; local browser fallback is unavailable');
  }
  if (!['load', 'chart', 'style'].includes(kind)) throw new Error('unsupported Chart Generator integration case');
  if (kind === 'chart' && !chartLabels.has(value)) throw new Error('unknown chart type');
  if (kind === 'style' && !styleLabels.has(value)) throw new Error('unknown chart style');
  const opened = await brokerClient.openChartSession();
  const sessionId = opened?.sessionId;
  if (!sessionId) throw new Error('BAS did not return a session ID');
  try {
    await requirePreviewRegion(brokerClient, sessionId);
    if (kind === 'load') return { check: 'preview-and-title' };
    if (kind === 'chart') {
      await clickNamed(brokerClient, sessionId, 'button', chartLabels.get(value), `${value} chart type`);
      await requireRenderedChart(brokerClient, sessionId, chartLabels.get(value));
      return { check: `${value}-chart` };
    }
    await clickNamed(brokerClient, sessionId, 'tab', 'Styles', 'Styles tab');
    await clickNamed(brokerClient, sessionId, 'button', styleLabels.get(value), `${value} style`);
    await clickNamed(brokerClient, sessionId, 'tab', 'Chart library', 'Charts tab');
    await requireRenderedChart(brokerClient, sessionId, 'Bar chart');
    return { check: `${value}-style` };
  } finally {
    await brokerClient.closeSession(sessionId);
  }
}

export { chartLabels, styleLabels };
