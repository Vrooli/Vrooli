import { randomUUID } from 'node:crypto';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { chromium, expect, type Page } from '@playwright/test';
import { ProjectsService } from '@vrooli/proto-types/browser-automation-studio/v1/projects/project_pb';
import { WorkflowsService } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import {
  env,
  canvasFrameSignature,
  elapsedSinceMonotonic,
  journeyResponsivenessBudget,
  measureJourney,
  recordJourneyMeasurement,
  recordJourneyRuntimeIdentity,
  recordJourneyReceipt,
  recordNotApplicableResponsivenessMetric,
  rpc,
  target,
  trackJourneySocketTraffic,
  withJourneyResources,
  apiRoute,
  waitForCanvasFrameChange,
} from './support';

jest.setTimeout(120000);

type RecordingPages = { pages: Array<{ id: string; url: string }> };
type RecordingTimeline = { entries: Array<{ type: string }> };

async function waitFor<T>(
  read: () => Promise<T>,
  ready: (value: T) => boolean,
  description: string
): Promise<T> {
  const deadline = Date.now() + 30000;
  let latest: T | undefined;
  while (Date.now() < deadline) {
    latest = await read();
    if (ready(latest)) return latest;
    await new Promise((resolve) => setTimeout(resolve, 200));
  }
  throw new Error(`Timed out waiting for ${description}; last value: ${JSON.stringify(latest)}`);
}

async function openWorkspace(page: Page, route: string): Promise<string> {
  await page.goto(route, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.getByTestId('app-ready').waitFor({ state: 'attached', timeout: 30000 });
  await page.getByTestId('browser-url-input').waitFor({ state: 'visible', timeout: 30000 });
  await page.waitForURL((url) => /^\/record\/[0-9a-f-]+$/i.test(url.pathname), { timeout: 30000 });
  const sessionId = page.url().match(/\/record\/([0-9a-f-]+)/i)?.[1] ?? '';
  if (!sessionId) throw new Error(`BAS did not expose a session id in ${page.url()}`);
  return sessionId;
}

async function clickRemoteCounter(page: Page): Promise<void> {
  const canvas = page.locator('canvas:visible').first();
  const deadline = Date.now() + 30000;
  let dimensions = { width: 0, height: 0 };
  while (Date.now() < deadline) {
    dimensions = await canvas.evaluate((element) => {
      const targetCanvas = element as HTMLCanvasElement;
      return { width: targetCanvas.width, height: targetCanvas.height };
    });
    if (dimensions.width > 0 && dimensions.height > 0) break;
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  if (dimensions.width <= 0 || dimensions.height <= 0)
    throw new Error('BAS did not stream the isolated session frame');
  const box = await canvas.boundingBox();
  if (!box || box.width <= 0 || box.height <= 0)
    throw new Error('BAS session canvas is not visible');
  const point = target('counter');
  const scale = Math.min(box.width / dimensions.width, box.height / dimensions.height);
  await page.mouse.click(
    box.x + (box.width - dimensions.width * scale) / 2 + point.x * scale,
    box.y + (box.height - dimensions.height * scale) / 2 + point.y * scale
  );
}

async function waitForFrame(page: Page): Promise<void> {
  const canvas = page.locator('canvas:visible').first();
  const deadline = Date.now() + 30000;
  while (Date.now() < deadline) {
    const dimensions = await canvas.evaluate((element) => {
      const targetCanvas = element as HTMLCanvasElement;
      return { width: targetCanvas.width, height: targetCanvas.height };
    });
    if (dimensions.width > 0 && dimensions.height > 0) return;
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  throw new Error('BAS did not paint a frame for both concurrent sessions');
}

async function stableTimeline(sessionId: string): Promise<RecordingTimeline> {
  const deadline = Date.now() + 30000;
  let previous: RecordingTimeline | undefined;
  while (Date.now() < deadline) {
    const current = await apiRoute<RecordingTimeline>(
      `/recordings/live/${sessionId}/timeline?limit=100`
    );
    if (previous && JSON.stringify(current.entries) === JSON.stringify(previous.entries))
      return current;
    previous = current;
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  throw new Error(`Timeline for ${sessionId} did not settle before the interaction`);
}

describe('[REQ:BAS-RH-J30] BAS concurrent workspace isolation', () => {
  it('keeps parallel workspace sessions and recording timelines isolated', () =>
    withJourneyResources(async (resources) => {
      const projectFolder = await mkdtemp(join(tmpdir(), 'bas-j30-'));
      resources.add('remove temporary J30 project directory', () =>
        rm(projectFolder, { recursive: true, force: true })
      );
      const projectName = `J30 recorded workflows ${randomUUID()}`;
      const { project } = await rpc(ProjectsService.method.createProject, {
        name: projectName,
        folderPath: projectFolder,
      });
      const projectId = project?.id ?? '';
      if (!projectId) throw new Error('BAS did not create the J30 recording project');
      resources.add('delete J30 recording project', () =>
        rpc(ProjectsService.method.deleteProject, { id: projectId, deleteFiles: true })
      );

      const browser = await chromium.launch({ headless: true });
      resources.add('close J30 UI browser', () => browser.close());
      await recordJourneyRuntimeIdentity('J30', browser.version());
      const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
      resources.add('close J30 UI context', () => context.close());
      const firstPage = await context.newPage();
      const secondPage = await context.newPage();
      const firstSocketTraffic = trackJourneySocketTraffic(firstPage);
      const secondSocketTraffic = trackJourneySocketTraffic(secondPage);
      const sessions = new Set<string>();
      const closed = new Set<string>();
      const sessionReceipts: Promise<void>[] = [];
      for (const page of [firstPage, secondPage]) {
        page.on('response', (response) => {
          const url = new URL(response.url());
          if (
            response.request().method() !== 'POST' ||
            !url.pathname.endsWith('/recordings/live/session') ||
            !response.ok()
          )
            return;
          sessionReceipts.push(
            response
              .json()
              .then((body: { session_id?: unknown }) => {
                if (typeof body.session_id === 'string') sessions.add(body.session_id);
              })
              .catch(() => undefined)
          );
        });
      }
      resources.add('close remaining J30 BAS sessions', async () => {
        await Promise.all(sessionReceipts);
        for (const sessionId of sessions) {
          if (closed.has(sessionId)) continue;
          await apiRoute(`/recordings/live/session/${sessionId}/close`, {});
          closed.add(sessionId);
        }
      });

      const firstUrl = new URL(`/complex?owner=${randomUUID()}`, env.fixture).toString();
      const secondUrl = new URL(`/complex?owner=${randomUUID()}`, env.fixture).toString();
      const { firstSession, secondSession } = await measureJourney(
        'J30',
        'isolation.workspaces_ready',
        async () => {
          const firstSession = await openWorkspace(
            firstPage,
            `${env.ui}/record/new?url=${encodeURIComponent(firstUrl)}`
          );
          sessions.add(firstSession);
          const secondSession = await openWorkspace(
            secondPage,
            `${env.ui}/record/new?url=${encodeURIComponent(secondUrl)}`
          );
          sessions.add(secondSession);
          await Promise.all([waitForFrame(firstPage), waitForFrame(secondPage)]);
          return { firstSession, secondSession };
        },
        () => ({ workspace_count: 2 }),
        { budgetMetrics: ['setup_to_ready_ms'] }
      );
      sessions.add(secondSession);
      expect(secondSession).not.toBe(firstSession);

      const firstPages = await waitFor(
        () => apiRoute<RecordingPages>(`/recordings/live/${firstSession}/pages`),
        (value) => value.pages?.some(({ url }) => url === firstUrl) ?? false,
        'first workspace page'
      );
      const secondPages = await waitFor(
        () => apiRoute<RecordingPages>(`/recordings/live/${secondSession}/pages`),
        (value) => value.pages?.some(({ url }) => url === secondUrl) ?? false,
        'second workspace page'
      );
      const firstPageId = firstPages.pages.find(({ url }) => url === firstUrl)?.id;
      const secondPageId = secondPages.pages.find(({ url }) => url === secondUrl)?.id;
      expect(firstPageId).toBeTruthy();
      expect(secondPageId).toBeTruthy();
      expect(secondPageId).not.toBe(firstPageId);

      const [firstBefore, secondBefore] = await Promise.all([
        stableTimeline(firstSession),
        stableTimeline(secondSession),
      ]);
      const canvasBeforeClick = await canvasFrameSignature(firstPage);
      const firstTrafficBefore = firstSocketTraffic.snapshot(firstSession);
      const secondTrafficBefore = secondSocketTraffic.snapshot(secondSession);
      const eventStarted = process.hrtime.bigint();
      await measureJourney(
        'J30',
        'isolation.click_to_paint',
        async () => {
          await clickRemoteCounter(firstPage);
          return waitForCanvasFrameChange(
            firstPage,
            canvasBeforeClick,
            journeyResponsivenessBudget('J30', 'click_to_paint_ms')
          );
        },
        (signature) => ({ session_id: firstSession, frame_changed: Boolean(signature) }),
        { budgetMetrics: ['click_to_paint_ms'] }
      );
      const eventDeadline = Date.now() + journeyResponsivenessBudget('J30', 'event_fanout_ms');
      let firstTrafficAfter = firstSocketTraffic.snapshot(firstSession);
      while (
        Date.now() < eventDeadline &&
        firstTrafficAfter.received_frames <= firstTrafficBefore.received_frames
      ) {
        await new Promise((resolve) => setTimeout(resolve, 25));
        firstTrafficAfter = firstSocketTraffic.snapshot(firstSession);
      }
      const secondTrafficAfter = secondSocketTraffic.snapshot(secondSession);
      const firstEventFrames =
        firstTrafficAfter.received_frames - firstTrafficBefore.received_frames;
      const firstEventBytes = firstTrafficAfter.received_bytes - firstTrafficBefore.received_bytes;
      const secondEventFrames =
        secondTrafficAfter.received_frames - secondTrafficBefore.received_frames;
      expect(firstEventFrames).toBeGreaterThan(0);
      expect(secondEventFrames).toBe(0);
      if (firstTrafficAfter.last_received_at_ns === null)
        throw new Error('J30 event has no receive timestamp');
      const fanoutMs = elapsedSinceMonotonic(eventStarted, firstTrafficAfter.last_received_at_ns);
      expect(firstEventBytes).toBeLessThanOrEqual(
        journeyResponsivenessBudget('J30', 'event_bytes')
      );
      recordJourneyMeasurement('J30', 'isolation.event_fanout', fanoutMs, 'event_fanout_ms', {
        receiving_session_id: firstSession,
        unrelated_session_id: secondSession,
        receiving_frames: firstEventFrames,
        unrelated_frames: secondEventFrames,
        received_bytes: firstEventBytes,
      });
      recordJourneyReceipt('J30', 'isolation.event_bytes', {
        receiving_session_id: firstSession,
        unrelated_session_id: secondSession,
        receiving_frames: firstEventFrames,
        unrelated_frames: secondEventFrames,
        received_bytes: firstEventBytes,
        event_bytes_budget: journeyResponsivenessBudget('J30', 'event_bytes'),
        event_bytes_budget_met:
          firstEventBytes <= journeyResponsivenessBudget('J30', 'event_bytes'),
      });
      const isolationReceipt = await measureJourney(
        'J30',
        'isolation.interaction_scoped',
        async () => {
          const firstAfter = await waitFor(
            () =>
              apiRoute<RecordingTimeline>(`/recordings/live/${firstSession}/timeline?limit=100`),
            (value) => value.entries.length > firstBefore.entries.length,
            'interaction in first session timeline'
          );
          const secondAfter = await apiRoute<RecordingTimeline>(
            `/recordings/live/${secondSession}/timeline?limit=100`
          );
          expect(firstAfter.entries.length).toBeGreaterThan(firstBefore.entries.length);
          expect(secondAfter.entries).toEqual(secondBefore.entries);
          return {
            firstTimelineDelta: firstAfter.entries.length - firstBefore.entries.length,
            secondTimelineUnchanged:
              JSON.stringify(secondAfter.entries) === JSON.stringify(secondBefore.entries),
          };
        },
        (receipt) => ({
          session_ids: [firstSession, secondSession],
          page_ids: [firstPageId ?? '', secondPageId ?? ''],
          first_timeline_delta: receipt.firstTimelineDelta,
          second_timeline_unchanged: receipt.secondTimelineUnchanged,
        })
      );

      // Turn the actual first-session recording into a saved artifact through BAS's UI form.
      const workflowName = `J30 captured workflow ${randomUUID()}`;
      await firstPage.getByTestId('timeline-create-workflow-button').click();
      await firstPage.getByTestId('workflow-creation-name-input').fill(workflowName);
      await firstPage.getByTestId('recording-project-selector').click();
      await firstPage.getByRole('heading', { name: 'Select Project' }).waitFor({
        state: 'visible',
        timeout: 30000,
      });
      await firstPage.getByRole('button', { name: projectName, exact: true }).click();
      await firstPage.getByTestId('workflow-creation-submit-button').click();
      await firstPage.waitForURL(
        (url) =>
          url.pathname === `/projects/${projectId}` && Boolean(url.searchParams.get('preview')),
        { timeout: 30000 }
      );
      closed.add(firstSession);
      await firstPage.getByTestId('project-workflows-surface').waitFor({
        state: 'visible',
        timeout: 30000,
      });
      const workflowId = new URL(firstPage.url()).searchParams.get('preview') ?? '';
      if (!workflowId)
        throw new Error('BAS did not route to the workflow generated from J30 recording');
      const { workflow } = await rpc(WorkflowsService.method.getWorkflow, { workflowId });
      if (!workflow?.id || workflow.id !== workflowId)
        throw new Error('BAS did not retain the J30 UI-generated workflow');
      expect(workflow.flowDefinition?.nodes.length ?? 0).toBeGreaterThan(0);
      resources.add('delete J30 UI-generated workflow', () =>
        rpc(WorkflowsService.method.deleteWorkflow, { workflowId })
      );
      recordJourneyReceipt('J30', 'isolation.recording_artifact_saved', {
        session_id: firstSession,
        authoring_surface: 'record-mode-create-workflow',
        workflow_id: workflowId,
        project_id: projectId,
        recorded_action_count: workflow.flowDefinition?.nodes.length ?? 0,
      });

      recordJourneyReceipt('J30', 'isolation.owner_receipt', {
        session_ids: [firstSession, secondSession],
        page_ids: [firstPageId ?? '', secondPageId ?? ''],
        first_timeline_delta: isolationReceipt.firstTimelineDelta,
        second_timeline_unchanged: isolationReceipt.secondTimelineUnchanged,
      });
      recordNotApplicableResponsivenessMetric(
        'J30',
        'isolation.tab_switch_not_sampled',
        'tab_switch_to_first_frame_ms',
        'The isolation journey opens two separate workspace pages and does not switch page selection within one recording session.'
      );
      for (const [page, sessionId] of [
        [firstPage, firstSession],
        [secondPage, secondSession],
      ] as const) {
        if (closed.has(sessionId)) continue;
        const closedResponse = page.waitForResponse(
          (response) => {
            const url = new URL(response.url());
            return (
              response.request().method() === 'POST' &&
              url.origin === env.api &&
              url.pathname.endsWith(`/recordings/live/session/${sessionId}/close`)
            );
          },
          { timeout: 30000 }
        );
        await page.getByTitle('Close Record Mode').click();
        const response = await closedResponse;
        expect(response.status()).toBe(200);
        const receipt = (await response.json()) as {
          session_id?: string;
          status?: string;
          profile_persisted?: boolean;
        };
        expect(receipt).toMatchObject({
          session_id: sessionId,
          status: 'closed',
          profile_persisted: true,
        });
        closed.add(sessionId);
      }
    }));
});
