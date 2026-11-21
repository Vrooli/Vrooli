const BINDING_KEY = 'lpbs.sign-in.browser-binding';
let memoryBinding: string | null = null;

function randomBinding(): string {
  const bytes = new Uint8Array(24);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
}

/**
 * A random secret that identifies this browser to the sign-in service. A code
 * is only accepted from the browser that requested it, and an app's callback
 * details are only returned to it. Storage can be unavailable (private mode,
 * blocked site data), so the value degrades to this page's memory.
 */
export function getBrowserBinding(): string {
  try {
    const stored = window.localStorage.getItem(BINDING_KEY);
    if (stored && stored.length >= 32) return stored;
    const created = randomBinding();
    window.localStorage.setItem(BINDING_KEY, created);
    return created;
  } catch {
    memoryBinding ??= randomBinding();
    return memoryBinding;
  }
}
