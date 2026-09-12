const configured = (import.meta.env as { VITE_LANDING_PAGE_URL?: unknown }).VITE_LANDING_PAGE_URL;

// One resolver keeps local storefront routing consistent across paid-surface
// refusals and upgrade prompts. The server refusal payload may override this
// destination when a request supplies an explicit upgrade_path.
export const LANDING_PAGE_URL = typeof configured === 'string' && configured.length > 0 ? configured : 'https://vrooli.com';

export function resolveUpgradeDestination(path: string | undefined, fallback = '/pricing'): string {
  try { return new URL(path || fallback, LANDING_PAGE_URL).toString(); } catch { return new URL(fallback, LANDING_PAGE_URL).toString(); }
}
