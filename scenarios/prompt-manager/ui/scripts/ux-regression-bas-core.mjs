import { WORLD_URL } from './bas-ux-client.mjs';

const VIEWPORT = { width: 1280, height: 800 };
const ROUTES = {
  world: /^\/world\/?$/,
  agent: /^\/agents\/[^/]+\/?$/,
  team: /^\/teams\/[^/]+\/?$/,
};

function sameWorldOrigin(raw) {
  try {
    const url = new URL(raw);
    return url.origin === new URL(WORLD_URL).origin && !url.username && !url.password && !url.search && !url.hash;
  } catch { return false; }
}

function nodes(root) {
  const all = [];
  const visit = (node) => {
    if (!node || typeof node !== 'object') return;
    all.push(node);
    for (const child of node.children ?? []) visit(child);
  };
  visit(root);
  return all;
}

function findNode(tree, predicate) { return nodes(tree?.root).find(predicate); }

function matchNode(tree, predicate, label) {
  const node = findNode(tree, predicate);
  if (!node?.ref) throw new Error(`BAS accessibility read did not expose a safe reference for ${label}`);
  return node;
}

async function waitForNode(client, sessionId, predicate, label, routePattern = ROUTES.world, routeLabel = 'world') {
  let lastTree;
  for (let attempt = 0; attempt < 20; attempt++) {
    const read = await readWorld(client, sessionId, routePattern, routeLabel);
    lastTree = read.tree;
    const found = findNode(lastTree, predicate);
    if (found?.ref) return { tree: lastTree, node: found };
    await new Promise(resolve => setTimeout(resolve, 150));
  }
  throw new Error(`Prompt Manager did not expose ${label} in the accessibility tree`);
}

function namedRole(tree, role, name, label = `${role} ${name}`) {
  return matchNode(tree, node => node.role === role && node.name === name, label);
}

function requireRoute(raw, pattern, label) {
  if (!sameWorldOrigin(raw) || !pattern.test(new URL(raw).pathname)) throw new Error(`Prompt Manager left the expected ${label} route`);
}

async function readWorld(client, sessionId, routePattern = ROUTES.world, routeLabel = 'world') {
  const location = await client.task(sessionId, 'screenshot');
  requireRoute(location?.url, routePattern, routeLabel);
  const result = await client.task(sessionId, 'read');
  const tree = result?.tree;
  if (tree?.url !== new URL(WORLD_URL).origin) throw new Error('Prompt Manager accessibility read left the fixed origin');
  const layout = tree?.layout;
  if (!layout || layout.viewportWidthPx !== VIEWPORT.width || layout.viewportHeightPx !== VIEWPORT.height ||
      !Number.isSafeInteger(layout.documentHeightPx) || layout.documentHeightPx <= 0) {
    throw new Error('BAS accessibility read omitted trusted vertical layout metrics');
  }
  if (layout.documentHeightPx > layout.viewportHeightPx) {
    throw new Error(`Prompt Manager app shell overflows viewport (${layout.documentHeightPx}px document height)`);
  }
  return { tree, result };
}

async function clickRef(client, sessionId, ref) {
  await client.task(sessionId, 'click', { ref });
}

export async function runUXRegression({ brokerClient }) {
  if (!brokerClient?.openWorldSession || !brokerClient?.task || !brokerClient?.closeSession) {
    throw new Error('a BAS Agent Access client is required; local browser fallback is unavailable');
  }
  const opened = await brokerClient.openWorldSession();
  const sessionId = opened?.sessionId;
  if (!sessionId) throw new Error('BAS did not return a session ID');
  try {
    const initial = await readWorld(brokerClient, sessionId);
    const agentsTab = namedRole(initial.tree, 'tab', 'Agents');
    await clickRef(brokerClient, sessionId, agentsTab.ref);
    const { node: agentRow } = await waitForNode(brokerClient, sessionId, node => node.role === 'button' && node.name && node.name !== 'New Agent', 'first agent row');
    await clickRef(brokerClient, sessionId, agentRow.ref);
    await readWorld(brokerClient, sessionId, ROUTES.agent, 'agent detail');

    await brokerClient.task(sessionId, 'navigate', { url: WORLD_URL });
    const worldForTeams = await readWorld(brokerClient, sessionId);
    const teamsTab = namedRole(worldForTeams.tree, 'tab', 'Teams');
    await clickRef(brokerClient, sessionId, teamsTab.ref);
    const { node: teamRow } = await waitForNode(brokerClient, sessionId, node => node.role === 'button' && node.name && node.name !== 'New Team', 'first team row');
    await clickRef(brokerClient, sessionId, teamRow.ref);
    await readWorld(brokerClient, sessionId, ROUTES.team, 'team detail');

    await brokerClient.task(sessionId, 'navigate', { url: WORLD_URL });
    const worldForRuns = await readWorld(brokerClient, sessionId);
    const runsTab = namedRole(worldForRuns.tree, 'tab', 'Runs');
    await clickRef(brokerClient, sessionId, runsTab.ref);
    const runs = await waitForNode(brokerClient, sessionId, node =>
      node.name === 'No runs yet' || node.name === 'No matching runs' ||
      (node.role === 'button' && node.name && !['Agents', 'Teams', 'Runs'].includes(node.name)), 'run list');
    if (nodes(runs.tree.root).some(node => node.name === 'Failed to load runs')) {
      throw new Error('Prompt Manager reported that runs failed to load');
    }
    return { checks: ['shell-fit', 'agent-detail', 'team-detail', 'runs-loaded'], sessionId };
  } finally {
    await brokerClient.closeSession(sessionId);
  }
}

export { VIEWPORT, ROUTES };
