import { resolveApiBase, createScenarioConnectTransport } from '@vrooli/api-base';

import { getWebAccessToken } from '../stores/authStore';

/**
 * Shared Connect-Web transport for every generated proto client in BAS.
 *
 * Connect-RPC services are mounted at the chi root (see api/main.go
 * `connectx.RegisterChi`) — NOT under /api/v1 — so the transport uses
 * the bare API base URL (no suffix).
 */
export const API_BASE = resolveApiBase();

const authenticatedFetch: typeof fetch = (input, init) => {
  const token = getWebAccessToken();
  if (!token) return fetch(input, init);

  // Only attach the consumer token to BAS's own API origin. This prevents a
  // future client reuse from forwarding subscription material to an arbitrary
  // URL selected by a caller.
  let requestURL: URL;
  try {
    requestURL = new URL(typeof input === 'string' ? input : input instanceof URL ? input.href : input.url, API_BASE);
  } catch {
    return fetch(input, init);
  }
  if (requestURL.origin !== new URL(API_BASE).origin) return fetch(input, init);

  const headers = new Headers(init?.headers);
  if (!headers.has('Authorization')) headers.set('Authorization', `Bearer ${token}`);
  return fetch(input, { ...init, headers });
};

export const transport = createScenarioConnectTransport({ baseUrl: API_BASE, fetch: authenticatedFetch });
