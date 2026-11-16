import { resolveApiBase, createScenarioConnectTransport } from '@vrooli/api-base';

import { useAuthStore } from '../stores/authStore';

/**
 * Shared Connect-Web transport for every generated proto client in BAS.
 *
 * Connect-RPC services are mounted at the chi root (see api/main.go
 * `connectx.RegisterChi`) — NOT under /api/v1 — so the transport uses
 * the bare API base URL (no suffix).
 */
export const API_BASE = resolveApiBase();

export const authenticatedFetch: typeof fetch = async (input, init) => {
  // Use the same auth source as the UI: Electron's native auth manager in a
  // packaged app, and the short-lived browser token in a web session.
  const token = await useAuthStore.getState().getAccessToken();
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
  const aiRequest = requestURL.pathname.includes('VisionNavigationService/StartNavigation') || requestURL.pathname.includes('.v1.ai.AIService/');
  if (aiRequest && typeof window !== 'undefined') {
    const receipt = window.sessionStorage.getItem('bas.ai.remote-consent-receipt');
    const source = window.sessionStorage.getItem('bas.ai.remote-consent-source');
    // Source selection is a routing preference, while the opaque receipt is
    // independent consent authority. Forward the preference by itself so a
    // missing/revoked receipt is refused by Gateway for the selected source
    // instead of silently falling back to another route.
    if (source) headers.set('X-Vrooli-AI-Consent-Source', source);
    if (receipt) {
      headers.set('X-Vrooli-AI-Consent-Grant', receipt);
    }
  }
  return fetch(input, { ...init, headers });
};

export const transport = createScenarioConnectTransport({ baseUrl: API_BASE, fetch: authenticatedFetch });
