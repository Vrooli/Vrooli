import { test, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';
import ts from 'typescript';

const require = createRequire(import.meta.url);
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const storeSourcePath = path.resolve(__dirname, '../src/stores/workflowStore.ts');

const noop = () => {};

if (typeof globalThis.crypto?.randomUUID !== 'function') {
  globalThis.crypto = { randomUUID: () => 'test-id' };
}

let cleanupFetch;

beforeEach(() => {
  cleanupFetch = undefined;
});

afterEach(() => {
  if (cleanupFetch) {
    cleanupFetch();
  }
  delete require.cache[storeSourcePath];
});

function loadWorkflowStore(overrides = {}) {
  const sourceCode = fs.readFileSync(storeSourcePath, 'utf8');
  const transpiled = ts.transpileModule(sourceCode, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2020,
      esModuleInterop: true,
    },
  });

  const exportsObj = {};
  const moduleObj = { exports: exportsObj };
  const customRequire = (specifier) => {
    switch (specifier) {
      case '../config':
        return overrides.configModule ?? {
          getConfig: async () => ({ API_URL: 'http://localhost:3000/api/v1' }),
        };
      case '../utils/logger':
        return overrides.loggerModule ?? {
          logger: {
            info: noop,
            warn: noop,
            error: noop,
            debug: noop,
          },
        };
      case '../utils/workflowNormalizers':
        return overrides.normalizersModule ?? {
          normalizeNodes: (nodes) => nodes ?? [],
          normalizeEdges: (edges) => edges ?? [],
        };
      case 'zustand':
        return require('zustand');
      default:
        return require(specifier);
    }
  };
  const compiled = new Function('exports', 'require', 'module', '__filename', '__dirname', transpiled.outputText);
  compiled(exportsObj, customRequire, moduleObj, storeSourcePath, path.dirname(storeSourcePath));
  return moduleObj.exports.useWorkflowStore;
}

function installFetchStub(factory) {
  const originalFetch = global.fetch;
  const calls = [];
  global.fetch = async (...args) => {
    calls.push(args);
    return factory(args, calls.length);
  };
  cleanupFetch = () => {
    global.fetch = originalFetch;
  };
  return calls;
}

test('scheduleAutosave persists draft after debounce', async (t) => {
  const calls = installFetchStub(([, options]) => ({
    ok: true,
    status: 200,
    json: async () => ({
      id: 'wf-1',
      name: 'Test Workflow',
      folder_path: '/',
      flow_definition: { nodes: [], edges: [] },
      nodes: [],
      edges: [],
      version: 2,
      updated_at: new Date().toISOString(),
      created_at: new Date().toISOString(),
    }),
    text: async () => '',
  }));

  const useWorkflowStore = loadWorkflowStore();
  const created = new Date();
  useWorkflowStore.setState((state) => ({
    ...state,
    currentWorkflow: {
      id: 'wf-1',
      name: 'Test Workflow',
      description: '',
      folderPath: '/',
      nodes: [],
      edges: [],
      tags: [],
      version: 1,
      createdAt: created,
      updatedAt: created,
    },
    nodes: [],
    edges: [],
    isDirty: true,
    draftFingerprint: 'draft-1',
    lastSavedFingerprint: 'saved-0',
    hasVersionConflict: false,
  }), true);

  useWorkflowStore.getState().scheduleAutosave({ debounceMs: 10, changeDescription: 'Autosave debounce test' });
  await new Promise((resolve) => setTimeout(resolve, 320));

  assert.equal(calls.length, 1, 'expected one save request');
  const [, options] = calls[0];
  assert.equal(options.method, 'PUT');

  const state = useWorkflowStore.getState();
  assert.equal(state.isDirty, false);
  assert.equal(state.currentWorkflow?.version, 2);
});

test('scheduleAutosave skips when conflict present', async (t) => {
  const calls = installFetchStub(() => ({ ok: true, json: async () => ({}), text: async () => '' }));
  const useWorkflowStore = loadWorkflowStore();
  const now = new Date();
  useWorkflowStore.setState((state) => ({
    ...state,
    currentWorkflow: {
      id: 'wf-2',
      name: 'Has Conflict',
      description: '',
      folderPath: '/',
      nodes: [],
      edges: [],
      tags: [],
      version: 4,
      createdAt: now,
      updatedAt: now,
    },
    nodes: [],
    edges: [],
    isDirty: true,
    hasVersionConflict: true,
  }), true);

  useWorkflowStore.getState().scheduleAutosave({ debounceMs: 5 });
  await new Promise((resolve) => setTimeout(resolve, 15));

  assert.equal(calls.length, 0, 'autosave should not fire when conflict exists');
});

test('loadWorkflowVersions sorts results by version desc', async (t) => {
  const calls = installFetchStub(([url]) => {
    if (!/versions/.test(url)) {
      return { ok: true, json: async () => ({}), text: async () => '' };
    }
    return {
      ok: true,
      json: async () => ({
        versions: [
          { version: 1, workflow_id: 'wf-3', created_at: '2024-01-01T00:00:00Z', created_by: 'manual' },
          { version: 3, workflow_id: 'wf-3', created_at: '2024-01-03T00:00:00Z', created_by: 'autosave' },
          { version: 2, workflow_id: 'wf-3', created_at: '2024-01-02T00:00:00Z', created_by: 'manual' },
        ],
      }),
      text: async () => '',
    };
  });

  const useWorkflowStore = loadWorkflowStore();
  await useWorkflowStore.getState().loadWorkflowVersions('wf-3');

  const history = useWorkflowStore.getState().versionHistory;
  assert.deepEqual(history.map((item) => item.version), [3, 2, 1]);
  assert.equal(calls.length, 1);
});

test('restoreWorkflowVersion resets restoring flag and refreshes history', async (t) => {
  let callIndex = 0;
  const calls = installFetchStub(([url, options]) => {
    callIndex += 1;
    if (options?.method === 'POST') {
      return {
        ok: true,
        json: async () => ({
          workflow: {
            id: 'wf-4',
            name: 'Restore Target',
            folder_path: '/',
            flow_definition: { nodes: [], edges: [] },
            nodes: [],
            edges: [],
            version: 7,
            updated_at: new Date().toISOString(),
            created_at: new Date().toISOString(),
          },
          restored_version: {
            version: 7,
            workflow_id: 'wf-4',
            created_at: new Date().toISOString(),
            created_by: 'manual',
          },
        }),
        text: async () => '',
      };
    }
    return {
      ok: true,
      json: async () => ({ versions: [] }),
      text: async () => '',
    };
  });

  const useWorkflowStore = loadWorkflowStore();
  const stamp = new Date();
  useWorkflowStore.setState((state) => ({
    ...state,
    currentWorkflow: {
      id: 'wf-4',
      name: 'Restore Target',
      description: '',
      folderPath: '/',
      nodes: [],
      edges: [],
      tags: [],
      version: 5,
      createdAt: stamp,
      updatedAt: stamp,
    },
  }), true);

  await useWorkflowStore.getState().restoreWorkflowVersion('wf-4', 7, 'Restored in test');

  const state = useWorkflowStore.getState();
  assert.equal(state.currentWorkflow?.version, 7);
  assert.equal(state.restoringVersion, null);
  assert.equal(state.versionHistoryLoadedFor, 'wf-4');
  assert.ok(callIndex >= 2);
  assert.ok(calls.length >= 2, 'expected restore + refresh requests');
});

test('saveWorkflow captures conflict snapshot', async () => {
  const now = new Date();
  const conflictPayload = {
    id: 'wf-conflict',
    name: 'Conflict Workflow',
    folder_path: '/',
    nodes: [],
    edges: [],
    version: 2,
    updated_at: now.toISOString(),
    created_at: now.toISOString(),
    last_change_source: 'autosave',
    last_change_description: 'Remote autosave',
  };

  const calls = installFetchStub(([, options], index) => {
    if (index === 1) {
      return { ok: false, status: 409, text: async () => 'conflict' };
    }
    if (index === 2) {
      return { ok: true, status: 200, json: async () => conflictPayload };
    }
    return { ok: true, status: 200, json: async () => ({}) };
  });

  const useWorkflowStore = loadWorkflowStore();
  const created = new Date(now.getTime() - 10_000);
  useWorkflowStore.setState((state) => ({
    ...state,
    currentWorkflow: {
      id: 'wf-conflict',
      name: 'Conflict Workflow',
      description: '',
      folderPath: '/',
      nodes: [],
      edges: [],
      tags: [],
      version: 1,
      createdAt: created,
      updatedAt: created,
      lastChangeSource: 'manual',
      lastChangeDescription: 'Local edit',
    },
    nodes: [],
    edges: [],
    isDirty: true,
    draftFingerprint: 'draft-conflict',
    lastSavedFingerprint: 'saved-conflict',
  }), true);

  await assert.rejects(useWorkflowStore.getState().saveWorkflow(), /conflict/i);
  assert.equal(calls.length >= 2, true);

  const state = useWorkflowStore.getState();
  assert.equal(state.hasVersionConflict, true, 'conflict flag should be set');
  assert(state.conflictWorkflow, 'conflict workflow snapshot should be stored');
  assert.equal(state.conflictWorkflow?.version, 2);
  assert(state.conflictMetadata, 'conflict metadata should be present');
  assert.equal(state.conflictMetadata?.changeSource, 'autosave');
});

test('forceSaveWorkflow uses remote version when overwriting', async () => {
  const updatedPayload = {
    id: 'wf-force',
    name: 'Force Workflow',
    folder_path: '/',
    nodes: [],
    edges: [],
    version: 4,
    updated_at: new Date().toISOString(),
    created_at: new Date().toISOString(),
  };

  const calls = installFetchStub(([, options]) => ({
    ok: true,
    status: 200,
    json: async () => updatedPayload,
    text: async () => JSON.stringify(updatedPayload),
  }));

  const useWorkflowStore = loadWorkflowStore();
  const created = new Date();
  useWorkflowStore.setState((state) => ({
    ...state,
    currentWorkflow: {
      id: 'wf-force',
      name: 'Force Workflow',
      description: '',
      folderPath: '/',
      nodes: [],
      edges: [],
      tags: [],
      version: 2,
      createdAt: created,
      updatedAt: created,
    },
    conflictWorkflow: {
      id: 'wf-force',
      name: 'Force Workflow',
      description: '',
      folderPath: '/',
      nodes: [],
      edges: [],
      tags: [],
      version: 3,
      createdAt: created,
      updatedAt: created,
      lastChangeSource: 'autosave',
      lastChangeDescription: 'Remote edit',
    },
    conflictMetadata: {
      detectedAt: created,
      remoteVersion: 3,
      remoteUpdatedAt: created,
      changeDescription: 'Remote edit',
      changeSource: 'autosave',
      nodeCount: 0,
      edgeCount: 0,
    },
    nodes: [],
    edges: [],
    isDirty: true,
    draftFingerprint: 'draft-force',
    lastSavedFingerprint: 'saved-force',
    hasVersionConflict: true,
  }), true);

  await useWorkflowStore.getState().forceSaveWorkflow({ changeDescription: 'Force save test' });

  assert.equal(calls.length >= 1, true);
  const body = JSON.parse(calls[0][1].body);
  assert.equal(body.expected_version, 3, 'should use remote version when forcing save');
  assert.equal(body.change_description, 'Force save test');

  const state = useWorkflowStore.getState();
  assert.equal(state.hasVersionConflict, false, 'conflict flag should clear after force save');
  assert.equal(state.conflictWorkflow, null, 'conflict snapshot should clear after force save');
});

test('transient workflow metadata does not mark workflow dirty', async () => {
  const now = new Date();
  installFetchStub(([, options]) => {
    const body = options?.body ? JSON.parse(options.body) : {};
    return {
      ok: true,
      status: 200,
      json: async () => ({
        id: 'wf-transient',
        name: 'Transient',
        folder_path: '/',
        flow_definition: body.flow_definition ?? { nodes: body.nodes ?? [], edges: body.edges ?? [] },
        nodes: body.nodes ?? [],
        edges: body.edges ?? [],
        version: 2,
        updated_at: now.toISOString(),
        created_at: now.toISOString(),
      }),
      text: async () => '',
    };
  });

  const useWorkflowStore = loadWorkflowStore();
  const baseNode = {
    id: 'node-1',
    type: 'navigate',
    position: { x: 0, y: 0 },
    data: { url: 'https://example.com' },
  };
  const baseEdge = {
    id: 'edge-1',
    source: 'node-1',
    target: 'node-1',
    type: 'default',
  };

  useWorkflowStore.setState((state) => ({
    ...state,
    currentWorkflow: {
      id: 'wf-transient',
      name: 'Transient',
      description: '',
      folderPath: '/',
      nodes: [baseNode],
      edges: [baseEdge],
      tags: [],
      version: 1,
      createdAt: now,
      updatedAt: now,
    },
    nodes: [baseNode],
    edges: [baseEdge],
    isDirty: true,
    draftFingerprint: null,
    lastSavedFingerprint: null,
  }), true);

  await useWorkflowStore.getState().saveWorkflow({ force: true });

  const mutatedNode = {
    ...baseNode,
    selected: true,
    positionAbsolute: { x: 0, y: 0 },
  };
  const mutatedEdge = {
    ...baseEdge,
    selected: true,
    sourceX: 10,
    sourceY: 20,
    targetX: 30,
    targetY: 40,
  };

  useWorkflowStore.getState().updateWorkflow({
    nodes: [mutatedNode],
    edges: [mutatedEdge],
  });

  const state = useWorkflowStore.getState();
  assert.equal(state.isDirty, false, 'transient UI metadata should not mark workflow dirty');
  assert.equal(state.draftFingerprint, state.lastSavedFingerprint, 'fingerprints should remain aligned');
});
