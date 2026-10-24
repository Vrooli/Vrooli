/**
 * Notification deep-link, iOS-correct.
 *
 * On iOS a home-screen web app tap never carries the notification's target into
 * navigate()/openWindow() — the OS just foregrounds the app at its last page.
 * So the service worker STASHES the target (a cache entry + a postMessage) and
 * this module, running in the foregrounded app, reads it and performs the hop.
 * The same path works on desktop/Android, so there is one code path, not two.
 *
 * The hop is guarded to the operator's own site (same registrable domain) so a
 * crafted notification can't turn the hub into an open redirect; an off-site
 * target is dropped rather than followed.
 */

const PENDING_NAV_CACHE = "notification-hub-pending-nav";
const PENDING_NAV_KEY = "/__pending_nav";
// A stash older than this is stale (the operator tapped long ago and moved on);
// acting on it would yank them away unexpectedly.
const MAX_AGE_MS = 120_000;

function baseDomain(host: string): string {
  const parts = host.split(".");
  return parts.length <= 2 ? host : parts.slice(-2).join(".");
}

/**
 * Resolve a raw target to an absolute URL IF it is on this site (same host, or
 * a subdomain of the same registrable domain — which is how fleet scenarios are
 * exposed, e.g. web-console.itsagitime.com from notification-hub.itsagitime.com).
 * Returns null for anything else, including non-http(s) schemes.
 */
export function sameSiteTarget(rawUrl: string): string | null {
  try {
    const target = new URL(rawUrl, window.location.href);
    if (target.protocol !== "https:" && target.protocol !== "http:") return null;
    if (target.hostname === window.location.hostname) return target.href;
    const base = baseDomain(window.location.hostname);
    if (target.hostname === base || target.hostname.endsWith("." + base)) return target.href;
    return null;
  } catch {
    return null;
  }
}

/** Read and CLEAR the stashed target; null if absent, stale, or off-site. */
export async function takePendingNav(
  now: number = Date.now(),
  expectedTarget?: string,
): Promise<string | null> {
  if (typeof caches === "undefined") return null;
  try {
    const cache = await caches.open(PENDING_NAV_CACHE);
    const response = await cache.match(PENDING_NAV_KEY);
    if (!response) return null;
    const { url, ts } = (await response.json()) as { url?: string; ts?: number };
    const target = typeof url === "string" ? sameSiteTarget(url) : null;
    // A worker message may arrive after a newer click has replaced the stash.
    // Leave that newer target for its own message or the next foreground event.
    if (expectedTarget && target !== expectedTarget) return null;
    await cache.delete(PENDING_NAV_KEY);
    if (!target || typeof ts !== "number" || now - ts > MAX_AGE_MS) return null;
    return target;
  } catch {
    return null;
  }
}

/** Perform the navigation. Same-origin stays in the app; cross-origin is a top-level nav. */
export function goTo(href: string): void {
  window.location.assign(href);
}

/**
 * Act on a destination that just arrived (via SW postMessage) or was stashed.
 * `messageUrl` is validated the same way as a stashed one. Returns the URL it
 * navigated to, or null when there was nothing valid to do.
 */
async function act(messageUrl?: string): Promise<string | null> {
  // A backgrounded Home Screen app can initialize before the user sees it
  // again. Keep the durable worker stash until a foreground lifecycle signal
  // gives this page a safe time to navigate.
  if (document.visibilityState !== "visible") return null;
  const href = messageUrl ? sameSiteTarget(messageUrl) : await takePendingNav();
  if (!href) return null;
  if (messageUrl) await takePendingNav(Date.now(), href);
  // Don't bounce if we're already there (avoids a reload loop on same-origin).
  if (href === window.location.href) return null;
  goTo(href);
  return href;
}

/**
 * Wire the foreground handlers: on boot, when the app returns to the foreground
 * (iOS's reliable signal after a tap), and when the worker posts a target. Safe
 * to call once at startup; returns a disposer for tests.
 */
export function initPendingNavigation(): () => void {
  const onForeground = () => {
    if (document.visibilityState === "visible") void act();
  };
  const onMessage = (event: MessageEvent) => {
    const data = event.data as { type?: string; url?: string } | null;
    if (data?.type === "PENDING_NAV") void act(data.url);
  };

  document.addEventListener("visibilitychange", onForeground);
  window.addEventListener("focus", onForeground);
  window.addEventListener("pageshow", onForeground);
  if ("serviceWorker" in navigator) {
    navigator.serviceWorker.addEventListener("message", onMessage);
  }
  // Boot: a cold launch from a notification tap lands here with a fresh stash.
  void act();

  return () => {
    document.removeEventListener("visibilitychange", onForeground);
    window.removeEventListener("focus", onForeground);
    window.removeEventListener("pageshow", onForeground);
    if ("serviceWorker" in navigator) {
      navigator.serviceWorker.removeEventListener("message", onMessage);
    }
  };
}
