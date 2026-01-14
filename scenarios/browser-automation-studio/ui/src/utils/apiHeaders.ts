/**
 * API Header Utilities
 *
 * Provides utilities for getting headers for API requests,
 * including BYOK (Bring Your Own Key) headers for AI requests.
 */

// Header name for BYOK OpenRouter API key
export const BYOK_HEADER_NAME = 'X-BYOK-OpenRouter-Key';

/**
 * Gets the BYOK OpenRouter API key from settings store.
 * Returns null if no key is configured.
 */
export async function getBYOKKey(): Promise<string | null> {
  try {
    const { useSettingsStore } = await import('@/stores/settingsStore');
    const settings = useSettingsStore.getState();
    const key = settings.apiKeys.openrouterApiKey;
    return key && key.trim().length > 0 ? key : null;
  } catch {
    // Settings store unavailable
    return null;
  }
}

/**
 * Gets the BYOK OpenRouter API key synchronously from settings store.
 * Returns null if no key is configured or store is unavailable.
 *
 * NOTE: This requires settingsStore to already be imported/loaded.
 */
export function getBYOKKeySync(): string | null {
  try {
    // Dynamic import won't work synchronously, so we need to check if store is available
    const storedKeys = window.localStorage.getItem('browserAutomation.settings.apiKeys');
    if (!storedKeys) return null;

    const parsed = JSON.parse(storedKeys);
    const key = parsed?.openrouterApiKey;
    return key && typeof key === 'string' && key.trim().length > 0 ? key : null;
  } catch {
    return null;
  }
}

/**
 * Gets headers for AI-related API requests.
 * Includes BYOK header if an OpenRouter API key is configured.
 */
export async function getAIRequestHeaders(): Promise<HeadersInit> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  const byokKey = await getBYOKKey();
  if (byokKey) {
    headers[BYOK_HEADER_NAME] = byokKey;
  }

  return headers;
}

/**
 * Gets headers for AI-related API requests synchronously.
 * Includes BYOK header if an OpenRouter API key is configured.
 */
export function getAIRequestHeadersSync(): HeadersInit {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  const byokKey = getBYOKKeySync();
  if (byokKey) {
    headers[BYOK_HEADER_NAME] = byokKey;
  }

  return headers;
}

/**
 * Adds BYOK header to existing headers if an OpenRouter API key is configured.
 */
export async function addBYOKHeader(headers: HeadersInit): Promise<HeadersInit> {
  const byokKey = await getBYOKKey();
  if (!byokKey) return headers;

  if (headers instanceof Headers) {
    headers.set(BYOK_HEADER_NAME, byokKey);
    return headers;
  }

  if (Array.isArray(headers)) {
    return [...headers, [BYOK_HEADER_NAME, byokKey]];
  }

  return { ...headers, [BYOK_HEADER_NAME]: byokKey };
}

/**
 * Adds BYOK header to existing headers synchronously.
 */
export function addBYOKHeaderSync(headers: HeadersInit): HeadersInit {
  const byokKey = getBYOKKeySync();
  if (!byokKey) return headers;

  if (headers instanceof Headers) {
    headers.set(BYOK_HEADER_NAME, byokKey);
    return headers;
  }

  if (Array.isArray(headers)) {
    return [...headers, [BYOK_HEADER_NAME, byokKey]];
  }

  return { ...headers, [BYOK_HEADER_NAME]: byokKey };
}
