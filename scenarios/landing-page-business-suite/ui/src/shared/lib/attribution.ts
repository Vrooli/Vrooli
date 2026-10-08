import { getVisitorId } from './visitorIdentity';

export { getVisitorId } from './visitorIdentity';

const SESSION_KEY = 'metrics_session_id';
const CAMPAIGN_KEY = 'metrics_campaign';
let memorySession: string | undefined;

function id(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 11)}`;
}

function storage(kind: 'local' | 'session'): Storage | undefined {
  if (typeof window === 'undefined') return undefined;
  try { return kind === 'session' ? window.sessionStorage : window.localStorage; } catch { return undefined; }
}

export function getSessionId(): string {
  const store = storage('session');
  if (store) {
    try {
      const existing = store.getItem(SESSION_KEY);
      if (existing) { memorySession = existing; return existing; }
      const created = id('session'); store.setItem(SESSION_KEY, created); memorySession = created; return created;
    } catch { /* use memory fallback */ }
  }
  memorySession ??= id('session');
  return memorySession;
}

export interface AttributionContext {
  visitor_id: string;
  session_id: string;
  variant_slug: string;
  utm_source: string;
  utm_medium: string;
  utm_campaign: string;
  landing_path: string;
  referrer: string;
}

export function getAttributionContext(variantSlug: string | null | undefined, preferredVisitor?: unknown): AttributionContext {
  const path = typeof window === 'undefined' ? '/' : window.location.pathname;
  const empty: AttributionContext = { visitor_id: getVisitorId(preferredVisitor), session_id: getSessionId(), variant_slug: variantSlug ?? '', utm_source: '', utm_medium: '', utm_campaign: '', landing_path: path, referrer: '' };
  const store = storage('session');
  if (!store || typeof window === 'undefined') return empty;
  try {
    const prior = store.getItem(CAMPAIGN_KEY);
    if (prior) return { ...empty, ...JSON.parse(prior) as Partial<AttributionContext>, variant_slug: variantSlug ?? '' };
    const params = new URLSearchParams(window.location.search);
    const first: AttributionContext = { ...empty, utm_source: params.get('utm_source') ?? '', utm_medium: params.get('utm_medium') ?? '', utm_campaign: params.get('utm_campaign') ?? '', referrer: document.referrer };
    store.setItem(CAMPAIGN_KEY, JSON.stringify(first));
    return first;
  } catch (error) { console.warn('[attribution] session storage unavailable:', error); return empty; }
}
