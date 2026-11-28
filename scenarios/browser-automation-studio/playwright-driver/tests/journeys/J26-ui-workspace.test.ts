import { chromium } from '@playwright/test';
import {
  apiRoute,
  env,
  measureJourney,
  canvasFrameSignature,
  elapsedSinceMonotonic,
  waitForCanvasFrameChange,
  journeyResponsivenessBudget,
  recordJourneyMeasurement,
  recordJourneyRuntimeIdentity,
  recordJourneyReceipt,
  trackJourneySocketTraffic,
  withJourneyResources,
} from './support';

jest.setTimeout(120000);

type RecordingPages = {
  pages: Array<{ id: string; url: string; title: string; status?: string; openerId?: string }>;
};
type RecordingTimeline = {
  entries: Array<{ type: string; entry?: { telemetry?: { framePath?: string[] } } }>;
};

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

async function clickRemotePoint(
  page: import('@playwright/test').Page,
  x: number,
  y: number
): Promise<void> {
  const canvas = page.locator('canvas:visible').first();
  const deadline = Date.now() + 30000;
  let dimensions = { width: 0, height: 0 };
  while (Date.now() < deadline) {
    dimensions = await canvas.evaluate((element) => {
      const target = element as HTMLCanvasElement;
      return { width: target.width, height: target.height };
    });
    if (dimensions.width > 0 && dimensions.height > 0) break;
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  if (dimensions.width <= 0 || dimensions.height <= 0)
    throw new Error('BAS preview did not receive a browser frame');
  const box = await canvas.boundingBox();
  if (!box || box.width <= 0 || box.height <= 0)
    throw new Error('BAS preview canvas has no visible bounds');
  const scale = Math.min(box.width / dimensions.width, box.height / dimensions.height);
  const letterboxX = (box.width - dimensions.width * scale) / 2;
  const letterboxY = (box.height - dimensions.height * scale) / 2;
  await page.mouse.click(box.x + letterboxX + x * scale, box.y + letterboxY + y * scale);
}

describe('[REQ:BAS-RH-J26] BAS UI workspace lifecycle', () => {
  it('opens a public read-only page, adds a fixture tab, persists page events, and closes from the workspace', () =>
    withJourneyResources(async (resources) => {
      const browser = await chromium.launch({ headless: true });
      resources.add('close BAS UI journey browser', () => browser.close());
      await recordJourneyRuntimeIdentity('J26', browser.version());
      const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
      resources.add('close BAS UI journey context', () => context.close());
      const page = await context.newPage();
      const socketTraffic = trackJourneySocketTraffic(page);
      const canvas = page.locator('canvas:visible').first();
      let sessionId = '';
      let sessionClosed = false;
      let captureSessionReceipt: Promise<void> = Promise.resolve();
      resources.add('close retained BAS UI session', async () => {
        await captureSessionReceipt;
        if (!sessionId || sessionClosed) return;
        await apiRoute(`/recordings/live/session/${sessionId}/close`, {});
        sessionClosed = true;
      });
      page.on('response', (response) => {
        const url = new URL(response.url());
        if (
          response.request().method() !== 'POST' ||
          !url.pathname.endsWith('/recordings/live/session') ||
          !response.ok()
        )
          return;
        captureSessionReceipt = response
          .json()
          .then((body: { session_id?: unknown }) => {
            if (typeof body.session_id === 'string') sessionId = body.session_id;
          })
          .catch(() => undefined);
      });

      const publicUrl = 'https://example.com/';
      sessionId = await measureJourney(
        'J26',
        'workspace.canvas_ready',
        async () => {
          await page.goto(`${env.ui}/record/new?url=${encodeURIComponent(publicUrl)}`, {
            waitUntil: 'domcontentloaded',
            timeout: 30000,
          });
          await page.getByTestId('app-ready').waitFor({ state: 'attached', timeout: 30000 });
          await page.getByTestId('browser-url-input').waitFor({ state: 'visible', timeout: 30000 });
          await page.waitForURL((url) => /^\/record\/[0-9a-f-]+$/i.test(url.pathname), {
            timeout: 30000,
          });
          const routedSessionId = page.url().match(/\/record\/([0-9a-f-]+)/i)?.[1] ?? '';
          if (!routedSessionId)
            throw new Error(`BAS did not expose the created session in the route: ${page.url()}`);
          await waitFor(
            async () =>
              canvas.evaluate((element) => ({
                width: (element as HTMLCanvasElement).width,
                height: (element as HTMLCanvasElement).height,
              })),
            ({ width, height }) => width > 0 && height > 0,
            'first browser frame in the BAS workspace'
          );
          return routedSessionId;
        },
        (id) => ({ session_id: id, creation_surface: 'record-mode-ui' }),
        { budgetMetrics: ['setup_to_ready_ms'] }
      );

      const firstPages = await waitFor(
        () => apiRoute<RecordingPages>(`/recordings/live/${sessionId}/pages`),
        (result) => result.pages?.some(({ url }) => url === publicUrl) ?? false,
        'first public page in BAS page registry'
      );
      expect(firstPages.pages).toHaveLength(1);

      const fixtureUrl = new URL('/complex', env.fixture).toString();
      const address = page.getByTestId('browser-url-input');
      const beforeSecondTab = await canvasFrameSignature(page);
      const socketBeforeSecondTab = socketTraffic.snapshot(sessionId);
      const secondTabStarted = process.hrtime.bigint();
      const pages = await measureJourney(
        'J26',
        'workspace.second_tab_ready',
        async () => {
          await page.getByRole('button', { name: 'Open new tab' }).click();
          await address.fill(fixtureUrl);
          await address.press('Enter');
          const result = await waitFor(
            () => apiRoute<RecordingPages>(`/recordings/live/${sessionId}/pages`),
            (result) =>
              result.pages?.length === 2 && result.pages.some(({ url }) => url === fixtureUrl),
            'second fixture tab in BAS page registry'
          );
          await waitForCanvasFrameChange(
            page,
            beforeSecondTab,
            journeyResponsivenessBudget('J26', 'click_to_paint_ms')
          );
          return result;
        },
        (result) => ({ session_id: sessionId, page_count: result.pages.length }),
        { budgetMetrics: ['click_to_paint_ms'] }
      );
      const secondTabDeadline = Date.now() + journeyResponsivenessBudget('J26', 'event_fanout_ms');
      let secondTabTraffic = socketTraffic.snapshot(sessionId);
      while (
        Date.now() < secondTabDeadline &&
        secondTabTraffic.received_frames <= socketBeforeSecondTab.received_frames
      ) {
        await new Promise((resolve) => setTimeout(resolve, 25));
        secondTabTraffic = socketTraffic.snapshot(sessionId);
      }
      const fanoutFrames = secondTabTraffic.received_frames - socketBeforeSecondTab.received_frames;
      const fanoutBytes = secondTabTraffic.received_bytes - socketBeforeSecondTab.received_bytes;
      expect(fanoutFrames).toBeGreaterThan(0);
      if (secondTabTraffic.last_received_at_ns === null)
        throw new Error('Recording event has no receive timestamp');
      const fanoutMs = elapsedSinceMonotonic(
        secondTabStarted,
        secondTabTraffic.last_received_at_ns
      );
      expect(fanoutBytes).toBeLessThanOrEqual(journeyResponsivenessBudget('J26', 'event_bytes'));
      recordJourneyMeasurement('J26', 'workspace.event_fanout', fanoutMs, 'event_fanout_ms', {
        session_id: sessionId,
        received_frames: fanoutFrames,
        received_bytes: fanoutBytes,
      });
      recordJourneyReceipt('J26', 'workspace.event_bytes', {
        session_id: sessionId,
        received_frames: fanoutFrames,
        received_bytes: fanoutBytes,
        event_bytes_budget: journeyResponsivenessBudget('J26', 'event_bytes'),
        event_bytes_budget_met: fanoutBytes <= journeyResponsivenessBudget('J26', 'event_bytes'),
      });
      expect(pages.pages.map(({ url }) => url)).toEqual(
        expect.arrayContaining([publicUrl, fixtureUrl])
      );
      await page.getByRole('tablist').getByRole('tab').nth(1).waitFor({ state: 'visible' });
      const opener = pages.pages.find(({ url }) => url === fixtureUrl);
      if (!opener) throw new Error('BAS did not register the fixture opener tab');

      // Click the fixture's popup control through BAS's streamed canvas and input path.
      const withPopup = await measureJourney(
        'J26',
        'workspace.popup_registered',
        async () => {
          await clickRemotePoint(page, 200, 100);
          return waitFor(
            () => apiRoute<RecordingPages>(`/recordings/live/${sessionId}/pages`),
            (result) =>
              result.pages?.length === 3 &&
              result.pages.some(
                ({ url }) => url === new URL('/same-popup', env.fixture).toString()
              ),
            'popup opened from the active BAS tab'
          );
        },
        (result) => ({ session_id: sessionId, page_count: result.pages.length })
      );
      const popup = withPopup.pages.find(
        ({ url }) => url === new URL('/same-popup', env.fixture).toString()
      );
      expect(popup?.openerId).toBe(opener.id);

      // Return to the fixture tab, then click inside its iframe through the same UI canvas.
      const frameUrl = new URL('/frames', env.fixture).toString();
      const timeline = await measureJourney(
        'J26',
        'workspace.iframe_timeline',
        async () => {
          const beforeTabSwitch = await canvasFrameSignature(page);
          await measureJourney(
            'J26',
            'workspace.tab_switch_to_first_frame',
            async () => {
              await page.getByRole('tablist').getByRole('tab').nth(1).click();
              return waitForCanvasFrameChange(
                page,
                beforeTabSwitch,
                journeyResponsivenessBudget('J26', 'tab_switch_to_first_frame_ms')
              );
            },
            (signature) => ({ session_id: sessionId, frame_changed: Boolean(signature) }),
            { budgetMetrics: ['tab_switch_to_first_frame_ms'] }
          );
          await address.fill(frameUrl);
          await address.press('Enter');
          await waitFor(
            () => apiRoute<RecordingPages>(`/recordings/live/${sessionId}/pages`),
            (result) =>
              result.pages?.some(({ id, url }) => id === opener.id && url === frameUrl) ?? false,
            'fixture iframe host navigation'
          );
          await clickRemotePoint(page, 110, 125);
          return waitFor(
            () => apiRoute<RecordingTimeline>(`/recordings/live/${sessionId}/timeline?limit=100`),
            (result) =>
              result.entries?.some(({ entry }) =>
                entry?.telemetry?.framePath?.includes('#fixture-frame')
              ) ?? false,
            'recorded interaction attributed to the fixture iframe'
          );
        },
        (result) => ({
          session_id: sessionId,
          frame_path_recorded: result.entries.some(({ entry }) =>
            entry?.telemetry?.framePath?.includes('#fixture-frame')
          ),
        })
      );
      expect(
        timeline.entries.some(({ entry }) =>
          entry?.telemetry?.framePath?.includes('#fixture-frame')
        )
      ).toBe(true);
      expect(timeline.entries.some(({ type }) => type === 'page_event')).toBe(true);

      // Reload the BAS workspace while the owned browser session remains live.
      // The route is the durable UI handle; the backend page registry and
      // journal must reconstruct the visible workspace from that identity.
      const workspaceRoute = page.url();
      const reloadedTimeline = await measureJourney(
        'J26',
        'workspace.reload_restored',
        async () => {
          await page.reload({ waitUntil: 'domcontentloaded', timeout: 30000 });
          await page.getByTestId('app-ready').waitFor({ state: 'attached', timeout: 30000 });
          await page.waitForURL(workspaceRoute, { timeout: 30000 });
          await waitFor(
            () => apiRoute<RecordingPages>(`/recordings/live/${sessionId}/pages`),
            (result) =>
              result.pages?.length === 3 &&
              result.pages.some(({ id, url }) => id === opener.id && url === frameUrl),
            'workspace browser pages to remain available after UI reload'
          );
          await page.getByRole('tablist').getByRole('tab').nth(1).waitFor({ state: 'visible' });
          return waitFor(
            () => apiRoute<RecordingTimeline>(`/recordings/live/${sessionId}/timeline?limit=100`),
            (result) =>
              result.entries?.some(({ entry }) =>
                entry?.telemetry?.framePath?.includes('#fixture-frame')
              ) ?? false,
            'recording journal frame attribution to remain available after UI reload'
          );
        },
        (result) => ({
          session_id: sessionId,
          frame_path_recorded: result.entries.some(({ entry }) =>
            entry?.telemetry?.framePath?.includes('#fixture-frame')
          ),
        })
      );
      expect(
        reloadedTimeline.entries.some(({ entry }) =>
          entry?.telemetry?.framePath?.includes('#fixture-frame')
        )
      ).toBe(true);

      const closeReceipt = await measureJourney(
        'J26',
        'workspace.close_receipt',
        async () => {
          const closeResponse = page.waitForResponse(
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
          const response = await closeResponse;
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
          await page.waitForURL((url) => url.pathname === '/', { timeout: 30000 });
          return receipt;
        },
        (receipt) => ({
          session_id: sessionId,
          status: receipt.status ?? 'unknown',
          profile_persisted: receipt.profile_persisted === true,
        })
      );
      sessionClosed = true;
      recordJourneyReceipt('J26', 'workspace.session_closed', {
        session_id: sessionId,
        status: closeReceipt.status ?? 'unknown',
        profile_persisted: closeReceipt.profile_persisted === true,
      });
    }));
});
