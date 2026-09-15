/** Anonymous correlation ID, never authentication or an authorization token. */
export const VISITOR_ID_KEY = 'metrics_visitor_id';
export const isValidVisitorId = (value: unknown): value is string => typeof value === 'string' && /^[A-Za-z0-9_-]{1,128}$/.test(value);
let memoryVisitor: string | undefined;
function newVisitorId(): string {
  // getRandomValues also works on non-secure development origins.
  return `visitor_${Array.from(crypto.getRandomValues(new Uint8Array(16)), byte => byte.toString(16).padStart(2, '0')).join('')}`;
}

export function visitorIdFromCookie(cookie: string): string | undefined {
  const values = cookie.split(';').map(part => part.trimStart()).filter(part => part.startsWith(`${VISITOR_ID_KEY}=`)).map(part => part.slice(VISITOR_ID_KEY.length + 1));
  return values.length === 1 && isValidVisitorId(values[0]) ? values[0] : undefined;
}

export function getVisitorId(preferred?: unknown): string {
  let cookie: string | undefined; let stored: string | null = null;
  try { cookie = visitorIdFromCookie(document.cookie); } catch { /* blocked cookie access */ }
  try { stored = window.localStorage.getItem(VISITOR_ID_KEY); } catch { /* blocked storage */ }
  const id = isValidVisitorId(preferred) ? preferred : cookie ?? (isValidVisitorId(stored) ? stored : memoryVisitor ?? newVisitorId());
  memoryVisitor = id;
  try { if (stored !== id) window.localStorage.setItem(VISITOR_ID_KEY, id); } catch { /* memory identity remains stable */ }
  try {
    if (cookie !== id) document.cookie = `${VISITOR_ID_KEY}=${id}; Path=/; SameSite=Lax; Max-Age=31536000${window.location.protocol === 'https:' ? '; Secure' : ''}`;
  } catch { /* storage-independent callers retain the same ID */ }
  return id;
}
