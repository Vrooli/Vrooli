import { randomUUID } from 'node:crypto';
import { chromium, expect, type Page } from '@playwright/test';
import {
  env,
  measureJourney,
  recordJourneyReceipt,
  target,
  withJourneyResources,
  apiRoute,
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
      const browser = await chromium.launch({ headless: true });
      resources.add('close J30 UI browser', () => browser.close());
      const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
      resources.add('close J30 UI context', () => context.close());
      const firstPage = await context.newPage();
      const secondPage = await context.newPage();
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

      await measureJourney(
        'J30',
        'isolation.first_frames',
        async () => {
          await Promise.all([waitForFrame(firstPage), waitForFrame(secondPage)]);
          return true;
        },
        () => ({ session_ids: [firstSession, secondSession] })
      );
      const [firstBefore, secondBefore] = await Promise.all([
        stableTimeline(firstSession),
        stableTimeline(secondSession),
      ]);
      const isolationReceipt = await measureJourney(
        'J30',
        'isolation.interaction_scoped',
        async () => {
          await clickRemoteCounter(firstPage);
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
      recordJourneyReceipt('J30', 'isolation.owner_receipt', {
        session_ids: [firstSession, secondSession],
        page_ids: [firstPageId ?? '', secondPageId ?? ''],
        first_timeline_delta: isolationReceipt.firstTimelineDelta,
        second_timeline_unchanged: isolationReceipt.secondTimelineUnchanged,
      });
      for (const [page, sessionId] of [
        [firstPage, firstSession],
        [secondPage, secondSession],
      ] as const) {
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
