import { API_BASE, authenticatedFetch } from './client';

export type RemoteAISource = 'own-key' | 'vrooli-credits';
export interface RemoteAIConsentGrant {
  id: string;
  source: RemoteAISource;
  created_at: string;
  expires_at: string;
  revoked_at?: string;
}

export const AI_CONSENT_RECEIPT_KEY = 'bas.ai.remote-consent-receipt';
export const AI_CONSENT_SOURCE_KEY = 'bas.ai.remote-consent-source';

async function consentRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json');
  const response = await authenticatedFetch(new URL(path, API_BASE), { ...init, headers, credentials: 'include' });
  if (!response.ok) throw new Error(`AI consent request failed (${response.status})`);
  if (response.status === 204) return undefined as T;
  return await response.json() as T;
}

export const listRemoteAIConsent = () => consentRequest<{ grants: RemoteAIConsentGrant[] }>('/api/v1/ai/privacy-consent');
export const createRemoteAIConsent = (source: RemoteAISource) => consentRequest<{ grant: RemoteAIConsentGrant; receipt: string }>('/api/v1/ai/privacy-consent', {
  method: 'POST', body: JSON.stringify({ source, lifetime_days: 30 }),
});
export const revokeRemoteAIConsent = (id: string) => consentRequest<void>(`/api/v1/ai/privacy-consent/${encodeURIComponent(id)}`, { method: 'DELETE' });
