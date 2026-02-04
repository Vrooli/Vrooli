import { apiCall } from './common';
import { parseOrNull } from './safeParse';
import { RemoteProfileSchema, RemoteProfilesListResponseSchema } from './schemas/remoteProfiles.schema';
import { SuccessResponseSchema } from './schemas/common.schema';
import type { RemoteProfile } from './types';

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
