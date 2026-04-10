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

export function listRemoteProfilesAdmin() {
  return apiCall<{ profiles: RemoteProfile[] }>('/admin/remote-profiles').then((resp) => {
    const validated = parseOrNull(RemoteProfilesListResponseSchema, resp, 'RemoteProfilesListResponse');
    if (!validated) {
      return { profiles: [] };
    }
    return validated;
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
  return apiCall<RemoteProfile>(`/admin/remote-profiles/${id}`, {
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
  return apiCall<{ success: boolean }>(`/admin/remote-profiles/${id}`, {
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
  return apiCall<RemoteProfile>(`/admin/remote-profiles/${id}/login`, {
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
  return apiCall<RemoteProfile>(`/admin/remote-profiles/${id}/logout`, {
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
  return apiCall<RemoteProfile>(`/admin/remote-profiles/${id}/test`, {
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
  return apiCall<RemoteProfileSessionLinks>(`/admin/remote-profiles/${id}/session-links`).then((resp) => {
    const validated = parseOrNull(RemoteProfileSessionLinksSchema, resp, 'RemoteProfileSessionLinks');
    if (!validated) {
      throw new Error('Invalid remote profile session links response from API');
    }
    return validated;
  });
}

export function revokeRemoteProfileSessionsAdmin(id: number) {
  return apiCall<RemoteProfileSessionLinks>(`/admin/remote-profiles/${id}/remote-revoke`, {
    method: 'POST',
  }).then((resp) => {
    const validated = parseOrNull(RemoteProfileSessionLinksSchema, resp, 'RemoteProfileSessionLinks');
    if (!validated) {
      throw new Error('Invalid remote profile session links response from API');
    }
    return validated;
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
