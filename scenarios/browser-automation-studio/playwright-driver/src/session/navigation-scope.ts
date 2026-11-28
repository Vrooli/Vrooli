import type { BrowserContext, Page, Request, Route } from 'rebrowser-playwright';
import type { BrowserPermissionProfile } from '../types/session';
import { authorizeOperation } from './permission-policy';

const redirectGuards = new WeakMap<Page, Promise<void>>();

/**
 * Enforce the admitted session profile for document navigations initiated by
 * page content (links, script, redirects, popups and child frames). The route
 * belongs to this exact BrowserContext, whose lifetime is already bound to
 * the owning session/lease. Redirects are independently routed by Playwright,
 * so every hop is checked before its request is sent.
 */
export async function installNavigationScopeGuard(
  context: BrowserContext,
  profile: BrowserPermissionProfile | undefined
): Promise<void> {
  if (!profile) return;
  const attach = (page: Page): Promise<void> => {
    const current = redirectGuards.get(page);
    if (current) return current;
    const pending = attachRedirectGuard(context, page, profile);
    redirectGuards.set(page, pending);
    return pending;
  };
  context.on('page', (page) => {
    void attach(page).catch(() => undefined);
  });
  for (const page of context.pages()) await attach(page);
  await context.route('**/*', async (route: Route, request: Request) => {
    if (!request.isNavigationRequest() || request.resourceType() !== 'document') {
      await route.continue();
      return;
    }

    const target = request.url();
    // The policy's navigate check selects the target URL and also applies the
    // admitted effect/control and deny-by-default URL rules. Use target as the
    // current URL because popup initial requests may not have an attached frame.
    const decision = authorizeNavigationTarget(profile, target);
    if (!decision) {
      await route.abort('blockedbyclient');
      return;
    }
    // Attach the response-stage guard before releasing the request. It checks
    // each Location header before Chromium follows the redirect, without
    // issuing a second request to the allowed destination (which could repeat
    // an application side effect on an otherwise permitted GET).
    let page: Page | undefined;
    try {
      page = request.frame().page();
    } catch {
      const candidates = context
        .pages()
        .filter((candidate) => candidate.url() === target || candidate.url() === 'about:blank');
      page = candidates.length === 1 ? candidates[0] : undefined;
    }
    if (!page) {
      await route.abort('blockedbyclient');
      return;
    }
    try {
      await attach(page);
    } catch {
      await route.abort('blockedbyclient');
      return;
    }
    await route.continue();
  });
}

/** Wait until a newly-created session page has its response-stage redirect guard. */
export async function ensureNavigationScopeForPage(page: Page): Promise<void> {
  await redirectGuards.get(page);
}

async function attachRedirectGuard(
  context: BrowserContext,
  page: Page,
  profile: BrowserPermissionProfile
): Promise<void> {
  const cdp = await context.newCDPSession(page);
  const protocol = cdp as unknown as {
    on: (event: string, listener: (event: any) => void) => void;
    send: (method: string, params?: Record<string, unknown>) => Promise<unknown>;
  };
  await protocol.send('Fetch.enable', {
    patterns: [{ urlPattern: '*', resourceType: 'Document', requestStage: 'Response' }],
  });
  protocol.on('Fetch.requestPaused', (event) => {
    void (async () => {
      try {
        const location = (event.responseHeaders ?? []).find(
          (header: { name: string; value: string }) => header.name.toLowerCase() === 'location'
        )?.value;
        if (event.responseStatusCode >= 300 && event.responseStatusCode < 400 && location) {
          const redirected = new URL(location, event.request.url).toString();
          if (!authorizeNavigationTarget(profile, redirected)) {
            await protocol.send('Fetch.failRequest', {
              requestId: event.requestId,
              errorReason: 'BlockedByClient',
            });
            return;
          }
        }
        await protocol.send('Fetch.continueRequest', { requestId: event.requestId });
      } catch {
        await protocol
          .send('Fetch.failRequest', { requestId: event.requestId, errorReason: 'BlockedByClient' })
          .catch(() => undefined);
      }
    })();
  });
}

export function authorizeNavigationTarget(
  profile: BrowserPermissionProfile | undefined,
  target: string
): boolean {
  return authorizeOperation(profile, 'navigate', target, target).allowed;
}
