import { resolveApiBase, buildApiUrl } from '@vrooli/api-base';

export const API_BASE = resolveApiBase({ appendSuffix: true });

export async function apiCall<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const url = buildApiUrl(endpoint, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    credentials: 'include',
    ...options,
  });

  if (!res.ok) {
    const errorText =
      typeof (res as { text?: () => Promise<string | undefined> }).text === 'function'
        ? await (res as { text: () => Promise<string | undefined> }).text()
        : res.statusText || 'Unknown error';
    throw new Error(`API call failed (${res.status}): ${errorText}`);
  }

  if (typeof (res as { json?: () => Promise<unknown> }).json === 'function') {
    return res.json() as Promise<T>;
  }

  return Promise.resolve(undefined as unknown as T);
}
