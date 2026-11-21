import { randomUUID } from 'node:crypto';
import { chromium, expect, type Page } from '@playwright/test';
import { SessionProfilesService } from '@vrooli/proto-types/browser-automation-studio/v1/session_profiles/session_profiles_pb';
import {
  apiRoute,
  env,
  fixtureAfter,
  fixtureState,
  measureJourney,
  recordJourneyReceipt,
  rpc,
  withJourneyResources,
} from './support';

jest.setTimeout(120000);

type RecordingPages = { pages: Array<{ id: string; url: string }> };

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

async function clickRemotePoint(page: Page, x: number, y: number): Promise<void> {
  const canvas = page.locator('canvas:visible').first();
  await expect
    .poll(() => canvas.evaluate((el) => (el as HTMLCanvasElement).width), { timeout: 30000 })
    .toBeGreaterThan(0);
  const dimensions = await canvas.evaluate((el) => ({
    width: (el as HTMLCanvasElement).width,
    height: (el as HTMLCanvasElement).height,
  }));
  const box = await canvas.boundingBox();
  if (!box || dimensions.height <= 0)
    throw new Error('BAS preview canvas is not ready for browser input');
  const scale = Math.min(box.width / dimensions.width, box.height / dimensions.height);
  await page.mouse.click(
    box.x + (box.width - dimensions.width * scale) / 2 + x * scale,
    box.y + (box.height - dimensions.height * scale) / 2 + y * scale
  );
}

describe('[REQ:BAS-RH-J01] BAS UI profile persistence and restoration', () => {
  it('persists synthetic authentication on UI close and restores it in a new UI session', () =>
    withJourneyResources(async (resources) => {
      const profileName = `J29 ${randomUUID()}`;
      const created = await rpc(SessionProfilesService.method.create, { name: profileName });
      const profileId = created.profile?.id ?? '';
      if (!profileId) throw new Error('BAS did not create the J29 profile');
      resources.add('delete J29 session profile', () =>
        rpc(SessionProfilesService.method.delete, { id: profileId })
      );

      const browser = await chromium.launch({ headless: true });
      resources.add('close J29 UI browser', () => browser.close());
      const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
      resources.add('close J29 UI context', () => context.close());
      await context.addInitScript((selectedProfileId) => {
        localStorage.setItem('bas_last_selected_session_profile_id', selectedProfileId);
      }, profileId);
      const page = await context.newPage();

      let sessionId = '';
      let sessionClosed = false;
      let sessionReceipt: Promise<void> = Promise.resolve();
      resources.add('close retained J29 session through owner API', async () => {
        await sessionReceipt;
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
        sessionReceipt = response
          .json()
          .then((body: { session_id?: unknown }) => {
            if (typeof body.session_id === 'string') sessionId = body.session_id;
          })
          .catch(() => undefined);
      });

      await page.goto(`${env.ui}/record/new`, { waitUntil: 'domcontentloaded', timeout: 30000 });
      await page.getByTestId('app-ready').waitFor({ state: 'attached', timeout: 30000 });
      await page.getByTestId('browser-url-input').waitFor({ state: 'visible', timeout: 30000 });
      await page.waitForURL((url) => /^\/record\/[0-9a-f-]+$/i.test(url.pathname), {
        timeout: 30000,
      });
      sessionId = page.url().match(/\/record\/([0-9a-f-]+)/i)?.[1] ?? '';
      if (!sessionId) throw new Error(`BAS did not expose the first J29 session: ${page.url()}`);
      await page
        .getByText(profileName, { exact: true })
        .waitFor({ state: 'visible', timeout: 30000 });

      const beforeChallenge = await fixtureState();
      const challengeUrl = new URL('/auth-challenge', env.fixture).toString();
      await page.getByTestId('browser-url-input').fill(challengeUrl);
      await page.getByTestId('browser-url-input').press('Enter');
      await fixtureAfter('challenge-visit', beforeChallenge.challengeVisits.length);
      await waitFor(
        () => apiRoute<RecordingPages>(`/recordings/live/${sessionId}/pages`),
        (result) => result.pages.some(({ url }) => url === challengeUrl),
        'challenge page in BAS session registry'
      );
      await clickRemotePoint(page, 230, 202);
      const challenge = await fixtureAfter('challenge', beforeChallenge.challenges.length);
      expect(challenge.challenges.slice(beforeChallenge.challenges.length)).toEqual([
        expect.objectContaining({ completedBy: 'person' }),
      ]);

      const firstSessionId = sessionId;
      const persistReceipt = await measureJourney(
        'J29',
        'profile.close_persisted',
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
          const closed = await closeResponse;
          expect(closed.status()).toBe(200);
          const receipt = (await closed.json()) as {
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
          const saved = await rpc(SessionProfilesService.method.list, {});
          const profilePersisted =
            saved.profiles.find(({ id }) => id === profileId)?.hasStorageState === true;
          expect(profilePersisted).toBe(true);
          return { receipt, profilePersisted };
        },
        ({ receipt, profilePersisted }) => ({
          session_id: firstSessionId,
          profile_id: profileId,
          status: receipt.status ?? 'unknown',
          profile_persisted: profilePersisted,
        })
      );
      sessionClosed = true;
      recordJourneyReceipt('J29', 'profile.initial_session_closed', {
        session_id: firstSessionId,
        profile_id: profileId,
        status: persistReceipt.receipt.status ?? 'unknown',
        profile_persisted: persistReceipt.profilePersisted,
      });

      await sessionReceipt;
      sessionId = '';
      sessionClosed = false;
      const restored = await measureJourney(
        'J29',
        'profile.fresh_session_restore',
        async () => {
          await page.goto(`${env.ui}/record/new`, {
            waitUntil: 'domcontentloaded',
            timeout: 30000,
          });
          await page.getByTestId('app-ready').waitFor({ state: 'attached', timeout: 30000 });
          await page.waitForURL((url) => /^\/record\/[0-9a-f-]+$/i.test(url.pathname), {
            timeout: 30000,
          });
          const restoredSessionId = page.url().match(/\/record\/([0-9a-f-]+)/i)?.[1] ?? '';
          if (!restoredSessionId || restoredSessionId === firstSessionId)
            throw new Error('BAS did not create a fresh session for J29 restoration');
          sessionId = restoredSessionId;
          const restoredPages = await waitFor(
            () => apiRoute<RecordingPages>(`/recordings/live/${restored.session_id}/pages`),
            (result) => result.pages.some(({ url }) => url === challengeUrl),
            'profile tab restored into a fresh BAS UI session'
          );
          expect(restoredPages.pages.some(({ url }) => url === challengeUrl)).toBe(true);
          return { session_id: restoredSessionId, page_count: restoredPages.pages.length };
        },
        ({ session_id, page_count }) => ({
          profile_id: profileId,
          prior_session_id: firstSessionId,
          restored_session_id: session_id,
          restored_page_count: page_count,
        })
      );

      const authProbe = await measureJourney(
        'J29',
        'profile.auth_probe',
        async () => {
          const beforeProbe = await fixtureState();
          const probeUrl = new URL('/auth-probe', env.fixture).toString();
          await page.getByTestId('browser-url-input').fill(probeUrl);
          await page.getByTestId('browser-url-input').press('Enter');
          const observed = await fixtureAfter(
            'auth-observation',
            beforeProbe.authObservations.length
          );
          const newObservations = observed.authObservations.slice(
            beforeProbe.authObservations.length
          );
          expect(newObservations).toEqual([expect.objectContaining({ hasSyntheticAuth: true })]);
          return newObservations[0];
        },
        (observation) => ({
          restored_session_id: restored.session_id,
          has_synthetic_auth: observation.hasSyntheticAuth,
        })
      );
      recordJourneyReceipt('J29', 'profile.restoration_receipt', {
        profile_id: profileId,
        prior_session_id: firstSessionId,
        restored_session_id: restored.session_id,
        restored_page_count: restored.page_count,
        has_synthetic_auth: authProbe.hasSyntheticAuth,
      });

      const restoreCloseResponse = page.waitForResponse(
        (response) => {
          const url = new URL(response.url());
          return (
            response.request().method() === 'POST' &&
            url.origin === env.api &&
            url.pathname.endsWith(`/recordings/live/session/${restored.session_id}/close`)
          );
        },
        { timeout: 30000 }
      );
      await page.getByTitle('Close Record Mode').click();
      const restoredClosed = await restoreCloseResponse;
      expect(restoredClosed.status()).toBe(200);
      expect(await restoredClosed.json()).toMatchObject({
        session_id: restored.session_id,
        status: 'closed',
        profile_persisted: true,
      });
      sessionClosed = true;
    }));
});
