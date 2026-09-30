import { createServer, type IncomingMessage, type Server, type ServerResponse } from 'node:http';
import { once } from 'node:events';
import { randomUUID } from 'node:crypto';
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';

type Json = Record<string, unknown>;

function readRequestBody(request: IncomingMessage): Promise<string> {
  return new Promise((resolve, reject) => {
    let body = '';
    request.setEncoding('utf8');
    request.on('data', (chunk: string) => {
      body += chunk;
    });
    request.on('end', () => resolve(body));
    request.on('error', reject);
  });
}

function writeJson(response: ServerResponse, status: number, value: unknown): void {
  response.writeHead(status, {
    'content-type': 'application/json',
    'access-control-allow-origin': '*',
  });
  response.end(JSON.stringify(value));
}

describe('saved recording workflow replay in a fresh context', () => {
  const apiBase = process.env.BAS_REHAB_LIVE_API_BASE;
  const liveTest = apiBase ? it : it.skip;

  liveTest(
    'saves the generated alternating-tab workflow and replays its same-selector actions',
    async () => {
      const events: string[] = [];
      const fixture: Server = createServer((request, response) => {
        void (async (): Promise<void> => {
          if (request.url === '/events' && request.method === 'POST') {
            const payload = JSON.parse(await readRequestBody(request)) as { tab?: unknown };
            if (typeof payload.tab === 'string') events.push(payload.tab);
            writeJson(response, 200, { accepted: true });
            return;
          }
          const tab =
            new URL(request.url ?? '/', 'http://fixture.invalid').searchParams.get('tab') ?? 'main';
          response.writeHead(200, { 'content-type': 'text/html' });
          response.end(
            `<!doctype html><button id="same" style="position:absolute;left:20px;top:20px;width:160px;height:60px" onclick="fetch('/events',{method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify({tab:'${tab}'})})">same</button>`
          );
        })().catch((error: unknown) => {
          writeJson(response, 500, { error: String(error) });
        });
      });
      fixture.listen(0, '127.0.0.1');
      await once(fixture, 'listening');
      const address = fixture.address();
      if (!address || typeof address === 'string')
        throw new Error('Fixture server did not bind a TCP port');
      const fixtureBase = `http://127.0.0.1:${address.port}`;

      let sessionId: string | undefined;
      let workflowId: string | undefined;
      let projectId: string | undefined;
      let projectFolder: string | undefined;
      const connect = async <T extends Json>(serviceMethod: string, body: Json): Promise<T> => {
        const response = await fetch(`${apiBase}/browser_automation_studio.v1.${serviceMethod}`, {
          method: 'POST',
          headers: { 'content-type': 'application/json' },
          body: JSON.stringify(body),
        });
        const payload = (await response.json()) as T & { message?: string; code?: string };
        if (!response.ok)
          throw new Error(
            `${serviceMethod} failed (${response.status}): ${JSON.stringify(payload)}`
          );
        return payload;
      };
      const recording = async <T extends Json>(path: string, body?: Json): Promise<T> => {
        const response = await fetch(`${apiBase}/api/v1/recordings/live${path}`, {
          method: body ? 'POST' : 'GET',
          headers: { 'content-type': 'application/json' },
          ...(body ? { body: JSON.stringify(body) } : {}),
        });
        const payload = (await response.json()) as T & { message?: string; error?: string };
        if (!response.ok)
          throw new Error(
            `Recording ${path} failed (${response.status}): ${JSON.stringify(payload)}`
          );
        return payload;
      };
      const readTimeline = async (): Promise<{
        entries?: Array<{
          type?: string;
          pageId?: string;
          action?: { actionType?: string; pageId?: string; id?: string };
        }>;
        totalEntries?: number;
      }> => recording(`/${sessionId}/timeline?limit=1000`);
      const waitForRecordedClicks = async (expected: number): Promise<void> => {
        const deadline = Date.now() + 10000;
        while (Date.now() < deadline) {
          const timeline = await readTimeline();
          const clicks = (timeline.entries ?? []).filter(
            (entry) => entry.type === 'action' && entry.action?.actionType === 'click'
          );
          if (clicks.length >= expected) return;
          await new Promise((resolve) => setTimeout(resolve, 50));
        }
        const timeline = await readTimeline();
        throw new Error(
          `Expected ${expected} persisted click actions, got ${JSON.stringify(timeline.entries)}`
        );
      };

      try {
        projectFolder = await mkdtemp(join(tmpdir(), 'bas-rehab-fresh-context-'));
        const projectName = `rehab-fresh-context-${randomUUID()}`;
        const createdProject = await connect<{ project?: { id?: string } }>(
          'projects.ProjectsService/CreateProject',
          { name: projectName, folderPath: projectFolder }
        );
        if (!createdProject.project?.id)
          throw new Error(
            `Disposable project creation returned no ID: ${JSON.stringify(createdProject)}`
          );
        projectId = createdProject.project.id;

        const created = await recording<{ session_id?: string }>('/session', {
          initial_url: 'about:blank',
          viewport_width: 900,
          viewport_height: 700,
          restore_tabs: false,
        });
        if (!created.session_id) throw new Error('Recording session response omitted session_id');
        sessionId = created.session_id;
        await recording('/start', { session_id: sessionId, frame_fps: 1 });

        const mainUrl = `${fixtureBase}/?tab=main`;
        const initialPages = await recording<{
          pages?: Array<{ id?: string }>;
        }>(`/${sessionId}/pages`);
        const initialPageId = initialPages.pages?.[0]?.id;
        if (!initialPageId) throw new Error('Initial blank page has no logical identity');
        const mainReceipt = await recording<{ driverPageId?: string; activePageId?: string }>(
          `/${sessionId}/pages`,
          { url: mainUrl }
        );
        if (!mainReceipt.activePageId || !mainReceipt.driverPageId)
          throw new Error(`Main page receipt is incomplete: ${JSON.stringify(mainReceipt)}`);
        const mainPageId = mainReceipt.activePageId;
        await recording(`/${sessionId}/pages/${initialPageId}/close`, {});
        await recording(`/${sessionId}/pages/${mainPageId}/activate`, {});

        // Forward actual pointer input through BAS. The independent fixture logs
        // which logical page received each effect; the durable timeline is the
        // second oracle and must retain the same page identity.
        const baseTime = Date.now();
        await recording(`/${sessionId}/input`, {
          type: 'pointer',
          action: 'click',
          x: 70,
          y: 45,
          button: 'left',
        });
        await waitForRecordedClicks(1);

        const popupUrl = `${fixtureBase}/?tab=popup`;
        const popupReceipt = await recording<{ driverPageId?: string; activePageId?: string }>(
          `/${sessionId}/pages`,
          { url: popupUrl }
        );
        if (!popupReceipt.activePageId || !popupReceipt.driverPageId)
          throw new Error(`Popup page receipt is incomplete: ${JSON.stringify(popupReceipt)}`);

        await recording(`/${sessionId}/input`, {
          type: 'pointer',
          action: 'click',
          x: 70,
          y: 45,
          button: 'left',
        });
        await waitForRecordedClicks(2);
        await recording(`/${sessionId}/pages/${mainPageId}/activate`, {});
        await recording(`/${sessionId}/input`, {
          type: 'pointer',
          action: 'click',
          x: 70,
          y: 45,
          button: 'left',
        });
        await waitForRecordedClicks(3);

        const captured = await recording<{
          count?: number;
          actions?: Array<{
            id?: string;
            sessionId?: string;
            sequenceNum?: number;
            timestamp?: string;
            actionType?: string;
            confidence?: number;
            selector?: { primary?: string; candidates?: string[] };
            url?: string;
            pageId?: string;
            driverPageId?: string;
          }>;
        }>(`/${sessionId}/actions`);
        const clicks = (captured.actions ?? []).filter((action) => action.actionType === 'click');
        expect(clicks).toHaveLength(3);
        expect(clicks.every((action) => !action.pageId)).toBe(true);
        expect(clicks.map((action) => action.driverPageId)).toEqual([
          mainReceipt.driverPageId,
          popupReceipt.driverPageId,
          mainReceipt.driverPageId,
        ]);
        expect(clicks.every((action) => action.selector?.primary === '#same')).toBe(true);

        const persisted = await readTimeline();
        const persistedClicks = (persisted.entries ?? []).filter(
          (entry) => entry.type === 'action' && entry.action?.actionType === 'click'
        );
        expect(persistedClicks).toHaveLength(3);
        expect(persistedClicks.map((entry) => entry.pageId)).toEqual([
          mainPageId,
          popupReceipt.activePageId,
          mainPageId,
        ]);
        const persistedPageIdByActionId = new Map(
          persistedClicks.flatMap((entry) =>
            entry.action?.id && entry.pageId ? [[entry.action.id, entry.pageId] as const] : []
          )
        );
        const clicksWithPageIdentity = clicks.map((action) => ({
          ...action,
          pageId: action.id ? persistedPageIdByActionId.get(action.id) : undefined,
        }));
        expect(clicksWithPageIdentity.map((action) => action.pageId)).toEqual([
          mainPageId,
          popupReceipt.activePageId,
          mainPageId,
        ]);

        const actions = [
          {
            id: randomUUID(),
            sessionId,
            sequenceNum: 0,
            timestamp: new Date(baseTime).toISOString(),
            actionType: 'navigate',
            confidence: 1,
            url: mainUrl,
            pageId: mainPageId,
          },
          ...clicksWithPageIdentity,
        ];
        const workflowName = `rehab-fresh-context-${randomUUID()}`;
        const generated = await recording<{ workflow_id?: string; node_count?: number }>(
          `/${sessionId}/generate-workflow`,
          {
            project_id: projectId,
            name: workflowName,
            actions,
          }
        );
        if (!generated.workflow_id)
          throw new Error(
            `Workflow generation did not return workflow_id: ${JSON.stringify(generated)}`
          );
        workflowId = generated.workflow_id;

        type SavedWorkflow = {
          workflow?: {
            flowDefinition?: {
              nodes?: Array<{ action?: { type?: string; tabSwitch?: { action?: string } } }>;
            };
          };
        };
        const saved = await connect<SavedWorkflow>('WorkflowsService/GetWorkflow', { workflowId });
        const nodes = saved.workflow?.flowDefinition?.nodes ?? [];
        const operations = nodes.flatMap((node) => {
          const action = node.action;
          if (action?.type === 'ACTION_TYPE_NAVIGATE') return ['navigate'];
          if (action?.type === 'ACTION_TYPE_CLICK') return ['click'];
          if (action?.type === 'ACTION_TYPE_TAB_SWITCH')
            return [action.tabSwitch?.action ?? 'tab-switch'];
          return [];
        });
        expect(operations).toEqual([
          'navigate',
          'click',
          'TAB_SWITCH_ACTION_OPEN',
          'click',
          'TAB_SWITCH_ACTION_SWITCH',
          'click',
        ]);

        // End capture and close its browser before replay to prove that this is
        // a separate execution context rather than the recording session.
        await recording(`/session/${sessionId}/close`, {});
        sessionId = undefined;
        events.length = 0;
        const execution = await connect<{ executionId?: string; status?: string; error?: string }>(
          'WorkflowsService/ExecuteWorkflow',
          { workflowId, waitForCompletion: true }
        );
        if (!execution.executionId)
          throw new Error(
            `Saved workflow execution returned no execution ID: ${JSON.stringify(execution)}`
          );
        expect(execution.error).toBeFalsy();
        expect(events).toEqual(['main', 'popup', 'main']);

        const deleted = await connect<{ success?: boolean }>('WorkflowsService/DeleteWorkflow', {
          workflowId,
        });
        expect(deleted.success).toBe(true);
        await expect(connect('WorkflowsService/GetWorkflow', { workflowId })).rejects.toThrow(
          'GetWorkflow failed'
        );
        workflowId = undefined;
      } finally {
        if (sessionId) {
          await recording(`/session/${sessionId}/close`, {});
        }
        if (projectId) {
          await connect('projects.ProjectsService/DeleteProject', {
            id: projectId,
            deleteFiles: true,
          });
        }
        if (projectFolder) await rm(projectFolder, { recursive: true, force: true });
        await new Promise<void>((resolve) => fixture.close(() => resolve()));
      }
    },
    120000
  );

  liveTest(
    'persists 10000 native fixture clicks through the managed API journal',
    async () => {
      const fixtureEffects: number[] = [];
      const fixture: Server = createServer((request, response) => {
        if (request.url === '/effect' && request.method === 'POST') {
          fixtureEffects.push(fixtureEffects.length + 1);
          writeJson(response, 200, { effect: fixtureEffects.at(-1) });
          return;
        }
        if (request.url === '/count') {
          writeJson(response, 200, { count: fixtureEffects.length });
          return;
        }
        response.writeHead(200, { 'content-type': 'text/html' });
        response.end(`<!doctype html><button id="same" style="position:absolute;left:20px;top:20px;width:160px;height:60px" onclick="fetch('/effect',{method:'POST'})">same</button>`);
      });
      fixture.listen(0, '127.0.0.1');
      await once(fixture, 'listening');
      const address = fixture.address();
      if (!address || typeof address === 'string') throw new Error('Fixture server did not bind a TCP port');
      const fixtureBase = `http://127.0.0.1:${address.port}`;

      let sessionId: string | undefined;
      const healthBeforeResponse = await fetch(`${apiBase}/health`);
      if (!healthBeforeResponse.ok) throw new Error(`Managed BAS health failed before cohort: ${healthBeforeResponse.status}`);
      const healthBefore = (await healthBeforeResponse.json()) as { build_identity?: string };
      if (!healthBefore.build_identity) throw new Error('Managed BAS health omitted build_identity before cohort');
      let useRoutedTestStorage = false;
      const recording = async <T extends Json>(path: string, body?: Json): Promise<T> => {
        const response = await fetch(`${apiBase}/api/v1/recordings/live${path}`, {
          method: body ? 'POST' : 'GET',
          headers: {
            'content-type': 'application/json',
            ...(useRoutedTestStorage ? { 'X-Vrooli-Test-Mode': '1' } : {}),
          },
          ...(body ? { body: JSON.stringify(body) } : {}),
        });
        const payload = (await response.json()) as T & { error?: string };
        if (!response.ok) throw new Error(`Recording ${path} failed (${response.status}): ${JSON.stringify(payload)}`);
        return payload;
      };
      const readTimeline = (offset: number) => recording<{
        entries?: Array<{
          type?: string; sequence?: number; pageId?: string;
          action?: { actionType?: string; id?: string; sequenceNum?: number };
        }>;
        totalEntries?: number;
      }>(`/${sessionId}/timeline?limit=1000&offset=${offset}`);
      let timelineOffset = 0;
      const clickIds = new Set<string>();
      const clickSequences: number[] = [];
      const collectTimeline = async (): Promise<void> => {
        const timeline = await readTimeline(timelineOffset);
        const entries = timeline.entries ?? [];
        for (const entry of entries) {
          if (entry.type === 'action' && entry.action?.actionType === 'click') {
            if (entry.action.id) clickIds.add(entry.action.id);
            if (typeof entry.action.sequenceNum === 'number') clickSequences.push(entry.action.sequenceNum);
          }
        }
        timelineOffset += entries.length;
      };
      const waitForAcknowledgedBatch = async (target: number): Promise<void> => {
        const deadline = Date.now() + 10000;
        while (Date.now() < deadline) {
          await collectTimeline();
          const response = await fetch(`${fixtureBase}/count`);
          const { count } = (await response.json()) as { count: number };
          if (clickIds.size >= target && count >= target) return;
          await new Promise((resolve) => setTimeout(resolve, 25));
        }
        throw new Error(`Managed recording reached ${clickIds.size} journal clicks and ${fixtureEffects.length} fixture effects; expected ${target}`);
      };

      const routedStorageDir = await mkdtemp(join(tmpdir(), 'bas-passive-fidelity-'));
      const routedDatabasePath = join(routedStorageDir, 'recording-fixture.sqlite');
      const routedLeaseId = `bas-passive-fidelity-${randomUUID()}`;
      let routedPoolInstalled = false;
      let routedStorageStats: { testPoolRequests?: number | string; primaryDuringTestModeRequests?: number | string } | undefined;
      let cleanupFailure: string | undefined;

      try {
        const installResponse = await fetch(
          `${apiBase}/vrooli.dev_routing.v1.routing.RoutingService/InstallTestPool`,
          {
            method: 'POST',
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify({
              // Match the scenario's production SQLite tuning so this
              // durability owner measures the capture path, not rollback-
              // journal sync cost. In particular WAL is required for the
              // 10k-event cohort to complete in a useful feedback cycle.
              dsn: `file:${routedDatabasePath}?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=page_size(4096)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)&_pragma=mmap_size(268435456)&_time_format=sqlite`,
              leaseId: routedLeaseId,
              leaseTtlMs: 600000,
              emptyConfig: true,
            }),
          }
        );
        const install = (await installResponse.json()) as { activeLeaseId?: string; fileRootsInstalled?: boolean };
        if (!installResponse.ok || install.activeLeaseId !== routedLeaseId) {
          throw new Error(`Could not install isolated recording storage (${installResponse.status}): ${JSON.stringify(install)}`);
        }
        routedPoolInstalled = true;
        useRoutedTestStorage = true;
        // All durable recording mutations and reads in this cohort use the
        // temporary routed pool; the ordinary live BAS database stays untouched.

        const created = await recording<{ session_id?: string }>('/session', {
          initial_url: 'about:blank', viewport_width: 900, viewport_height: 700, restore_tabs: false,
        });
        if (!created.session_id) throw new Error('Recording session response omitted session_id');
        sessionId = created.session_id;
        await recording('/start', { session_id: sessionId, frame_fps: 1 });
        const initialPages = await recording<{ pages?: Array<{ id?: string }> }>(`/${sessionId}/pages`);
        const initialPageId = initialPages.pages?.[0]?.id;
        if (!initialPageId) throw new Error('Initial recording page has no logical identity');
        await recording(`/${sessionId}/pages`, { url: fixtureBase });
        await recording(`/${sessionId}/pages/${initialPageId}/close`, {});

        const appliedSequences: number[] = [];
        const total = 10000;
        const batchSize = 25;
        const startedAt = performance.now();
        for (let target = batchSize; target <= total; target += batchSize) {
          for (let index = 0; index < batchSize; index += 1) {
            const receipt = await recording<{ applied_sequence?: number }>(`/${sessionId}/input`, {
              type: 'pointer', action: 'click', x: 70, y: 45, button: 'left',
            });
            if (typeof receipt.applied_sequence !== 'number') throw new Error(`Input omitted applied_sequence: ${JSON.stringify(receipt)}`);
            appliedSequences.push(receipt.applied_sequence);
          }
          await waitForAcknowledgedBatch(target);
        }
        const elapsedMs = Math.round(performance.now() - startedAt);
        const allIds = new Set<string>();
        const allSequences: number[] = [];
        let offset = 0;
        let totalEntries = 0;
        do {
          const page = await readTimeline(offset);
          const entries = page.entries ?? [];
          totalEntries = page.totalEntries ?? totalEntries;
          for (const entry of entries) {
            if (entry.type === 'action' && entry.action?.actionType === 'click') {
              if (entry.action.id) allIds.add(entry.action.id);
              if (typeof entry.action.sequenceNum === 'number') allSequences.push(entry.action.sequenceNum);
            }
          }
          offset += entries.length;
          if (entries.length === 0) break;
        } while (offset < totalEntries);

        expect(fixtureEffects).toEqual(Array.from({ length: total }, (_, index) => index + 1));
        expect(allIds.size).toBe(total);
        expect(allSequences).toHaveLength(total);
        expect(allSequences.every((sequence, index) => index === 0 || sequence > (allSequences[index - 1] ?? -1))).toBe(true);
        expect(appliedSequences).toHaveLength(total);
        expect(appliedSequences.every((sequence, index) => index === 0 || sequence > (appliedSequences[index - 1] ?? -1))).toBe(true);
        const healthAfterResponse = await fetch(`${apiBase}/health`);
        if (!healthAfterResponse.ok) throw new Error(`Managed BAS health failed after cohort: ${healthAfterResponse.status}`);
        const healthAfter = (await healthAfterResponse.json()) as { build_identity?: string };
        if (healthAfter.build_identity !== healthBefore.build_identity) throw new Error('Managed BAS build changed during passive-fidelity cohort');
        await recording(`/session/${sessionId}/close`, {});
        sessionId = undefined;
        const clearResponse = await fetch(
          `${apiBase}/vrooli.dev_routing.v1.routing.RoutingService/ClearTestPool`,
          {
            method: 'POST',
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify({ leaseId: routedLeaseId }),
          }
        );
        const cleared = (await clearResponse.json()) as {
          stats?: { testPoolRequests?: number | string; primaryDuringTestModeRequests?: number | string };
        };
        if (!clearResponse.ok || !cleared.stats) {
          throw new Error(`Could not clear isolated recording storage (${clearResponse.status}): ${JSON.stringify(cleared)}`);
        }
        routedPoolInstalled = false;
        useRoutedTestStorage = false;
        routedStorageStats = cleared.stats;
        // Proto JSON omits scalar zero values unless default emission is
        // enabled, so absence of the primary counter is the expected zero.
        const testPoolRequests = Number(routedStorageStats.testPoolRequests ?? 0);
        const primaryRequestsDuringTestMode = Number(routedStorageStats.primaryDuringTestModeRequests ?? 0);
        if (testPoolRequests < 1 || primaryRequestsDuringTestMode !== 0) {
          throw new Error(`Recording test escaped its isolated routed pool: ${JSON.stringify(routedStorageStats)}`);
        }
        const ownerReceipt = {
          schemaVersion: 1,
          contractRow: 'passive-fidelity',
          capturedAt: new Date().toISOString(),
          managedBuildIdentityBefore: healthBefore.build_identity,
          managedBuildIdentityAfter: healthAfter.build_identity,
          actions: total,
          fixtureEffects: fixtureEffects.length,
          uniqueJournalIds: allIds.size,
          strictlyIncreasingJournalSequence: allSequences.every((sequence, index) => index === 0 || sequence > (allSequences[index - 1] ?? -1)),
          appliedInputReceipts: appliedSequences.length,
          strictlyIncreasingAppliedSequence: appliedSequences.every((sequence, index) => index === 0 || sequence > (appliedSequences[index - 1] ?? -1)),
          storageIsolation: {
            routedTestPool: true,
            testPoolRequests,
            primaryRequestsDuringTestMode,
            temporaryDatabase: true,
          },
          elapsedMs,
        };
        const receiptPath = process.env.BAS_PASSIVE_FIDELITY_RECEIPT?.trim();
        if (receiptPath) {
          await mkdir(dirname(receiptPath), { recursive: true });
          await writeFile(receiptPath, `${JSON.stringify(ownerReceipt, null, 2)}\n`, { flag: 'wx' });
        }
        // eslint-disable-next-line no-console
        console.log(`BAS_MANAGED_PASSIVE_FIDELITY ${JSON.stringify(ownerReceipt)}`);
      } finally {
        try {
          if (sessionId) {
            try {
              await recording(`/session/${sessionId}/close`, {});
            } catch (error) {
              cleanupFailure = `Could not close managed recording session: ${String(error)}`;
            }
          }
        } finally {
          try {
            if (routedPoolInstalled) {
              try {
                const cleanupResponse = await fetch(`${apiBase}/vrooli.dev_routing.v1.routing.RoutingService/ClearTestPool`, {
                  method: 'POST',
                  headers: { 'content-type': 'application/json' },
                  body: JSON.stringify({ leaseId: routedLeaseId }),
                });
                if (!cleanupResponse.ok) cleanupFailure ??= `Could not clear isolated recording storage (${cleanupResponse.status})`;
              } catch (error) {
                cleanupFailure ??= `Could not clear isolated recording storage: ${String(error)}`;
              }
            }
          } finally {
            try {
              await rm(routedStorageDir, { recursive: true, force: true });
            } catch (error) {
              cleanupFailure ??= `Could not remove isolated recording storage: ${String(error)}`;
            } finally {
              await new Promise<void>((resolve) => fixture.close((error) => {
                if (error) cleanupFailure ??= `Could not close fixture server: ${error.message}`;
                resolve();
              }));
            }
          }
        }
      }
      if (cleanupFailure) throw new Error(cleanupFailure);
    },
    600000
  );
});
