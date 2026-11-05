import { apiCall } from './common';
import { parseOrNull } from './safeParse';
import {
  IncomingRemoteProfileSessionsResponseSchema,
  RemoteProfileSchema,
  RemoteProfilesListResponseSchema,
  RemoteProfileSessionLinksSchema,
} from './schemas/remoteProfiles.schema';
import { SuccessResponseSchema } from './schemas/common.schema';
import type { IncomingRemoteProfileSession, RemoteProfile, RemoteProfileSessionLinks } from './types';

export interface RemoteProfileCreatePayload {
  tag: string;
  label?: string;
  api_base: string;
}

export interface RemoteProfileUpdatePayload {
  tag?: string;
  label?: string;
  api_base?: string;
}

export interface RemoteProfileLoginPayload {
  email: string;
  password: string;
}

export interface RemoteProfileProxyPayload {
  method: string;
  path: string;
  query?: Record<string, string>;
  headers?: Record<string, string>;
  body?: unknown;
}

export interface RemoteAPIKey {
  id: string;
  provider: string;
  key_hint: string;
  is_active: boolean;
  last_verified_at?: string;
  created_at?: string;
  updated_at?: string;
}

export interface RemoteStripeSettings {
  publishable_key_preview?: string;
  publishable_key_set: boolean;
  secret_key_set: boolean;
  webhook_secret_set: boolean;
  anomaly_webhook_url_set: boolean;
  anomaly_webhook_enabled: boolean;
  dashboard_url?: string;
  updated_at?: string;
  source: string;
}

export interface RemoteStripeUpdatePayload {
  publishable_key?: string;
  secret_key?: string;
  webhook_secret?: string;
  dashboard_url?: string;
  anomaly_webhook_url?: string;
  anomaly_webhook_enabled?: boolean;
  anomaly_rate_limits?: string;
}

const REMOTE_ADMINISTRATION_PATH = '/landing_page_business_suite.v1.AdministrationService';
const REMOTE_STRIPE_PATH = '/landing_page_business_suite.v1.StripeSettingsService';

function recordValue(input: unknown): Record<string, unknown> {
  return input && typeof input === 'object' ? input as Record<string, unknown> : {};
}

function stringValue(record: Record<string, unknown>, snake: string, camel = snake): string | undefined {
  const value = record[camel] ?? record[snake];
  return typeof value === 'string' && value.length > 0 ? value : undefined;
}

function booleanValue(record: Record<string, unknown>, snake: string, camel = snake): boolean {
  return Boolean(record[camel] ?? record[snake]);
}

function normalizeRemoteAPIKey(input: unknown): RemoteAPIKey {
  const record = recordValue(input);
  return {
    id: stringValue(record, 'id') ?? '',
    provider: stringValue(record, 'provider') ?? '',
    key_hint: stringValue(record, 'key_hint', 'keyHint') ?? '',
    is_active: booleanValue(record, 'is_active', 'isActive'),
    last_verified_at: stringValue(record, 'last_verified_at', 'lastVerifiedAt'),
    created_at: stringValue(record, 'created_at', 'createdAt'),
    updated_at: stringValue(record, 'updated_at', 'updatedAt'),
  };
}

function normalizeRemoteStripeSettings(input: unknown): RemoteStripeSettings {
  const response = recordValue(input);
  const settings = recordValue(response.settings);
  const snapshot = recordValue(response.snapshot);
  const source = snapshot.source;
  return {
    publishable_key_preview: stringValue(snapshot, 'publishable_key_preview', 'publishableKeyPreview'),
    publishable_key_set: booleanValue(snapshot, 'publishable_key_set', 'publishableKeySet'),
    secret_key_set: booleanValue(snapshot, 'secret_key_set', 'secretKeySet'),
    webhook_secret_set: booleanValue(snapshot, 'webhook_secret_set', 'webhookSecretSet'),
    anomaly_webhook_url_set: booleanValue(settings, 'anomaly_webhook_url_set', 'anomalyWebhookUrlSet'),
    anomaly_webhook_enabled: booleanValue(settings, 'anomaly_webhook_enabled', 'anomalyWebhookEnabled'),
    dashboard_url: stringValue(settings, 'dashboard_url', 'dashboardUrl'),
    updated_at: stringValue(settings, 'updated_at', 'updatedAt'),
    source: typeof source === 'string' ? source : String(source ?? 'unknown'),
  };
}

function proxyRemoteAdmin<T>(id: number, procedure: string, body: unknown): Promise<T> {
  return proxyRemoteProfileAdmin(id, {
    method: 'POST',
    path: procedure,
    body,
  }) as Promise<T>;
}

export async function listRemoteAPIKeysAdmin(id: number): Promise<{ keys: RemoteAPIKey[] }> {
  const response = recordValue(await proxyRemoteAdmin(id, `${REMOTE_ADMINISTRATION_PATH}/ListAPIKeys`, {}));
  const keys = Array.isArray(response.keys) ? response.keys.map(normalizeRemoteAPIKey) : [];
  return { keys };
}

export async function createRemoteAPIKeyAdmin(id: number, provider: string, key: string): Promise<RemoteAPIKey> {
  const response = recordValue(await proxyRemoteAdmin(id, `${REMOTE_ADMINISTRATION_PATH}/CreateAPIKey`, { provider, key }));
  return normalizeRemoteAPIKey(response.key);
}

export async function deleteRemoteAPIKeyAdmin(id: number, provider: string): Promise<void> {
  await proxyRemoteAdmin(id, `${REMOTE_ADMINISTRATION_PATH}/DeleteAPIKey`, { provider });
}

export async function testRemoteAPIKeyAdmin(id: number, provider: string): Promise<{ success: boolean; message: string }> {
  const response = recordValue(await proxyRemoteAdmin(id, `${REMOTE_ADMINISTRATION_PATH}/TestAPIKey`, { provider }));
  return { success: Boolean(response.success), message: stringValue(response, 'message') ?? 'Remote API key test completed' };
}

export async function setRemoteAPIKeyActiveAdmin(id: number, provider: string, active: boolean): Promise<void> {
  await proxyRemoteAdmin(id, `${REMOTE_ADMINISTRATION_PATH}/SetAPIKeyActive`, { provider, active });
}

export function getRemoteStripeSettingsAdmin(id: number): Promise<RemoteStripeSettings> {
  return proxyRemoteAdmin(id, `${REMOTE_STRIPE_PATH}/GetStripeSettings`, {}).then(normalizeRemoteStripeSettings);
}

export function updateRemoteStripeSettingsAdmin(id: number, payload: RemoteStripeUpdatePayload): Promise<RemoteStripeSettings> {
  return proxyRemoteAdmin(id, `${REMOTE_STRIPE_PATH}/UpdateStripeSettings`, payload).then(normalizeRemoteStripeSettings);
}

export function listRemoteProfilesAdmin() {
  return apiCall<{ profiles: RemoteProfile[] }>('/admin/remote-profiles').then((resp) => {
    const validated = parseOrNull(RemoteProfilesListResponseSchema, resp, 'RemoteProfilesListResponse');
    if (!validated) {
      return { profiles: [] };
    }
    return { profiles: validated.profiles ?? [] };
  });
}

export function createRemoteProfileAdmin(payload: RemoteProfileCreatePayload) {
  return apiCall<RemoteProfile>('/admin/remote-profiles', {
    method: 'POST',
    body: JSON.stringify(payload),
    headers: {
      'Content-Type': 'application/json',
    },
  }).then((resp) => {
    const validated = parseOrNull(RemoteProfileSchema, resp, 'RemoteProfile');
    if (!validated) {
      throw new Error('Invalid remote profile response from API');
    }
    return validated;
  });
}

export function updateRemoteProfileAdmin(id: number, payload: RemoteProfileUpdatePayload) {
  return apiCall<RemoteProfile>(`/admin/remote-profiles/${String(id)}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
    headers: {
      'Content-Type': 'application/json',
    },
  }).then((resp) => {
    const validated = parseOrNull(RemoteProfileSchema, resp, 'RemoteProfile');
    if (!validated) {
      throw new Error('Invalid remote profile response from API');
    }
    return validated;
  });
}

export function deleteRemoteProfileAdmin(id: number) {
  return apiCall<{ success: boolean }>(`/admin/remote-profiles/${String(id)}`, {
    method: 'DELETE',
  }).then((resp) => {
    const validated = parseOrNull(SuccessResponseSchema, resp, 'RemoteProfileDeleteResponse');
    if (!validated) {
      throw new Error('Invalid delete remote profile response from API');
    }
    return validated;
  });
}

export function loginRemoteProfileAdmin(id: number, payload: RemoteProfileLoginPayload) {
  return apiCall<RemoteProfile>(`/admin/remote-profiles/${String(id)}/login`, {
    method: 'POST',
    body: JSON.stringify(payload),
    headers: {
      'Content-Type': 'application/json',
    },
  }).then((resp) => {
    const validated = parseOrNull(RemoteProfileSchema, resp, 'RemoteProfile');
    if (!validated) {
      throw new Error('Invalid remote profile response from API');
    }
    return validated;
  });
}

export function logoutRemoteProfileAdmin(id: number) {
  return apiCall<RemoteProfile>(`/admin/remote-profiles/${String(id)}/logout`, {
    method: 'POST',
  }).then((resp) => {
    const validated = parseOrNull(RemoteProfileSchema, resp, 'RemoteProfile');
    if (!validated) {
      throw new Error('Invalid remote profile response from API');
    }
    return validated;
  });
}

export function testRemoteProfileAdmin(id: number) {
  return apiCall<RemoteProfile>(`/admin/remote-profiles/${String(id)}/test`, {
    method: 'POST',
  }).then((resp) => {
    const validated = parseOrNull(RemoteProfileSchema, resp, 'RemoteProfile');
    if (!validated) {
      throw new Error('Invalid remote profile response from API');
    }
    return validated;
  });
}

export function getRemoteProfileSessionLinksAdmin(id: number) {
  return apiCall<RemoteProfileSessionLinks>(`/admin/remote-profiles/${String(id)}/session-links`).then((resp) => {
    const validated = parseOrNull(RemoteProfileSessionLinksSchema, resp, 'RemoteProfileSessionLinks');
    if (!validated) {
      throw new Error('Invalid remote profile session links response from API');
    }
    return validated;
  });
}

export function revokeRemoteProfileSessionsAdmin(id: number) {
  return apiCall<RemoteProfileSessionLinks>(`/admin/remote-profiles/${String(id)}/remote-revoke`, {
    method: 'POST',
  }).then((resp) => {
    const validated = parseOrNull(RemoteProfileSessionLinksSchema, resp, 'RemoteProfileSessionLinks');
    if (!validated) {
      throw new Error('Invalid remote profile session links response from API');
    }
    return validated;
  });
}

/**
 * Calls one of the server-enforced remote owner settings procedures through a
 * stored remote profile. The response is intentionally untyped here because
 * the allowlisted procedures have different generated response messages; the
 * owning settings client should validate its procedure-specific response.
 */
export function proxyRemoteProfileAdmin(id: number, payload: RemoteProfileProxyPayload) {
  return apiCall<unknown>(`/admin/remote-profiles/${String(id)}/proxy`, {
    method: 'POST',
    body: JSON.stringify(payload),
    headers: {
      'Content-Type': 'application/json',
    },
  });
}

export function listIncomingRemoteProfileSessionsAdmin(connectorID?: string) {
  const query = connectorID?.trim() ? `?connector_id=${encodeURIComponent(connectorID.trim())}` : '';
  return apiCall<{ sessions: IncomingRemoteProfileSession[] }>(`/admin/remote-profile-sessions${query}`).then((resp) => {
    const validated = parseOrNull(IncomingRemoteProfileSessionsResponseSchema, resp, 'IncomingRemoteProfileSessionsResponse');
    if (!validated) {
      return { sessions: [] };
    }
    return { sessions: validated.sessions ?? [] };
  });
}

export function revokeIncomingRemoteProfileSessionAdmin(sessionID: string) {
  return apiCall<{ success: boolean }>(`/admin/remote-profile-sessions/${encodeURIComponent(sessionID)}`, {
    method: 'DELETE',
  }).then((resp) => {
    const validated = parseOrNull(SuccessResponseSchema, resp, 'IncomingRemoteProfileSessionRevokeResponse');
    if (!validated) {
      throw new Error('Invalid revoke incoming remote profile session response from API');
    }
    return validated;
  });
}
