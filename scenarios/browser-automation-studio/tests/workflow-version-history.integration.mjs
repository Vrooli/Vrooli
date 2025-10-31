import assert from 'node:assert/strict';
import { randomUUID } from 'node:crypto';
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const apiBase = process.env.BAS_INTEGRATION_API_URL?.replace(/\/?$/, '');
if (!apiBase) {
  throw new Error('BAS_INTEGRATION_API_URL must be defined to run workflow version integration tests');
}

const tempDir = mkdtempSync(join(tmpdir(), 'bas-version-history-'));
const cleanupTasks = [];

const log = (message, extra = {}) => {
  const payload = Object.keys(extra).length ? ` ${JSON.stringify(extra)}` : '';
  console.log(`ℹ️  [version-history] ${message}${payload}`);
};

async function request(path, { expectStatus, ...options } = {}) {
  const response = await fetch(`${apiBase}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    ...options,
  });

  if (expectStatus && response.status !== expectStatus) {
    const body = await response.text();
    throw new Error(`Unexpected status ${response.status} for ${path} (expected ${expectStatus})\n${body}`);
  }

  return response;
}

const projectName = `version-history-${randomUUID().slice(0, 8)}`;
log('Creating project', { projectName, folder: tempDir });

const projectResponse = await request('/projects', {
  method: 'POST',
  body: JSON.stringify({
    name: projectName,
    folder_path: tempDir,
  }),
  expectStatus: 201,
});

const projectPayload = await projectResponse.json();
const projectId = projectPayload.project_id || projectPayload.project?.id;
assert(projectId, 'Project ID should be returned');
cleanupTasks.push(async () => {
  await request(`/projects/${projectId}`, { method: 'DELETE', expectStatus: 200 }).catch(() => {});
});

const baseNodes = [
  {
    id: 'node-1',
    type: 'navigate',
    position: { x: 0, y: 0 },
    data: { url: 'https://example.com' },
  },
];
const baseEdges = [];

log('Creating workflow baseline');
const workflowCreateResponse = await request('/workflows/create', {
  method: 'POST',
  body: JSON.stringify({
    project_id: projectId,
    name: `Version History Demo ${randomUUID().slice(0, 6)}`,
    folder_path: '/',
    flow_definition: { nodes: baseNodes, edges: baseEdges },
  }),
  expectStatus: 201,
});

const createdWorkflow = await workflowCreateResponse.json();
const workflowId = createdWorkflow.workflow_id;
assert(workflowId, 'Workflow ID should exist');
cleanupTasks.push(async () => {
  await request(`/workflows/${workflowId}`, { method: 'DELETE', expectStatus: 200 }).catch(() => {});
});

assert.equal(createdWorkflow.version, 1, 'Initial workflow version should be 1');

const updatedNodes = [
  ...baseNodes,
  {
    id: 'node-2',
    type: 'click',
    position: { x: 200, y: 48 },
    data: { selector: '#hero' },
  },
];
const updatedEdges = [
  {
    id: 'edge-1',
    source: 'node-1',
    target: 'node-2',
    type: 'smoothstep',
    data: {},
  },
];

log('Saving workflow to create new version');
const updateResponse = await request(`/workflows/${workflowId}`, {
  method: 'PUT',
  body: JSON.stringify({
    name: createdWorkflow.name,
    description: createdWorkflow.description || '',
    folder_path: createdWorkflow.folder_path || '/',
    nodes: updatedNodes,
    edges: updatedEdges,
    flow_definition: { nodes: updatedNodes, edges: updatedEdges },
    expected_version: createdWorkflow.version,
    change_description: 'Integration update',
    source: 'integration-test',
  }),
  expectStatus: 200,
});

const updatedWorkflow = await updateResponse.json();
assert.equal(updatedWorkflow.version, createdWorkflow.version + 1, 'Version should increment after update');
assert.equal(updatedWorkflow.last_change_source, 'integration-test');

log('Fetching workflow versions list');
const versionsResponse = await request(`/workflows/${workflowId}/versions?limit=10`, { expectStatus: 200 });
const versionsPayload = await versionsResponse.json();
assert(Array.isArray(versionsPayload.versions), 'Versions payload should be an array');
assert(versionsPayload.versions.length >= 2, 'Should contain at least two versions');

const [latestVersion, initialVersion] = versionsPayload.versions;
assert.equal(latestVersion.version, updatedWorkflow.version, 'Latest version should match updated workflow version');
assert(latestVersion.created_by, 'Latest version should record created_by');
assert.equal(initialVersion.version, 1, 'First version should be 1');

log('Triggering optimistic-lock conflict');
const conflictResponse = await request(`/workflows/${workflowId}`, {
  method: 'PUT',
  body: JSON.stringify({
    name: updatedWorkflow.name,
    folder_path: updatedWorkflow.folder_path,
    nodes: updatedNodes,
    edges: updatedEdges,
    flow_definition: { nodes: updatedNodes, edges: updatedEdges },
    expected_version: createdWorkflow.version, // stale version on purpose
    source: 'integration-conflict',
  }),
});

assert.equal(conflictResponse.status, 409, 'Conflict save should return 409');
const conflictBody = await conflictResponse.text();
assert(/expected/i.test(conflictBody), 'Conflict message should mention expected version');

log('Restoring historical version', { restoreVersion: initialVersion.version });
const restoreResponse = await request(`/workflows/${workflowId}/versions/${initialVersion.version}/restore`, {
  method: 'POST',
  body: JSON.stringify({ change_description: 'Integration restore' }),
  expectStatus: 200,
});

const restorePayload = await restoreResponse.json();
assert(restorePayload.workflow, 'Restore response should include workflow');
assert.equal(
  restorePayload.workflow.version,
  updatedWorkflow.version + 1,
  'Restoring should create a new version entry'
);
assert.equal(
  restorePayload.workflow.last_change_description,
  'Integration restore',
  'Restore should set change description'
);

log('Verifying versions after restore');
const postRestoreVersions = await request(`/workflows/${workflowId}/versions?limit=5`, { expectStatus: 200 })
  .then((res) => res.json());

assert.equal(postRestoreVersions.versions[0].version, restorePayload.workflow.version, 'Latest entry should be restore');
assert.equal(postRestoreVersions.versions[0].change_description, 'Integration restore');

log('Workflow version history integration passed');

for (const task of cleanupTasks.reverse()) {
  await task();
}

rmSync(tempDir, { recursive: true, force: true });
