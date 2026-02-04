import type { RemoteProfile } from '../../../shared/api';
import {
  createRemoteProfileAdmin,
  deleteRemoteProfileAdmin,
  listRemoteProfilesAdmin,
  loginRemoteProfileAdmin,
  logoutRemoteProfileAdmin,
  testRemoteProfileAdmin,
  updateRemoteProfileAdmin,
} from '../../../shared/api';

export interface RemoteProfileFormState {
  tag: string;
  label: string;
  apiBase: string;
}

export interface RemoteProfileLoginFormState {
  email: string;
  password: string;
}

export const DEFAULT_REMOTE_PROFILE_FORM: RemoteProfileFormState = {
  tag: '',
  label: '',
  apiBase: '',
};

export const DEFAULT_REMOTE_PROFILE_LOGIN: RemoteProfileLoginFormState = {
  email: '',
  password: '',
};

export type RemoteProfileStatusTone = 'success' | 'warning' | 'error' | 'info';

export interface RemoteProfileStatusMeta {
  label: string;
  tone: RemoteProfileStatusTone;
  description: string;
}

export function getRemoteProfileStatusMeta(profile: RemoteProfile): RemoteProfileStatusMeta {
  switch (profile.status) {
    case 'active':
      return {
        label: 'Active',
        tone: 'success',
        description: profile.has_session ? 'Session active' : 'Session missing',
      };
    case 'expired':
      return {
        label: 'Expired',
        tone: 'warning',
        description: 'Session expired',
      };
    case 'error':
      return {
        label: 'Error',
        tone: 'error',
        description: 'Remote error',
      };
    default:
      return {
        label: 'Unknown',
        tone: 'info',
        description: profile.has_session ? 'Session stored' : 'No session',
      };
  }
}

export function normalizeRemoteProfileForm(form: RemoteProfileFormState) {
  return {
    tag: form.tag.trim(),
    label: form.label.trim(),
    api_base: form.apiBase.trim(),
  };
}

export function validateRemoteProfileForm(form: RemoteProfileFormState): string | null {
  const tag = form.tag.trim();
  const apiBase = form.apiBase.trim();
  if (!tag) {
    return 'Tag is required';
  }
  if (!apiBase) {
    return 'API base is required';
  }
  if (!apiBase.includes('/api/v1')) {
    return 'API base must include /api/v1';
  }
  return null;
}

export function validateRemoteProfileLoginForm(form: RemoteProfileLoginFormState): string | null {
  if (!form.email.trim()) {
    return 'Email is required';
  }
  if (!form.password.trim()) {
    return 'Password is required';
  }
  return null;
}

export async function fetchRemoteProfiles(): Promise<RemoteProfile[]> {
  const response = await listRemoteProfilesAdmin();
  return response.profiles ?? [];
}

export async function createRemoteProfile(form: RemoteProfileFormState): Promise<RemoteProfile> {
  const payload = normalizeRemoteProfileForm(form);
  return createRemoteProfileAdmin(payload);
}

export async function updateRemoteProfile(id: number, form: RemoteProfileFormState): Promise<RemoteProfile> {
  const payload = normalizeRemoteProfileForm(form);
  return updateRemoteProfileAdmin(id, payload);
}

export async function deleteRemoteProfile(id: number): Promise<void> {
  await deleteRemoteProfileAdmin(id);
}

export async function loginRemoteProfile(id: number, form: RemoteProfileLoginFormState): Promise<RemoteProfile> {
  return loginRemoteProfileAdmin(id, {
    email: form.email.trim(),
    password: form.password,
  });
}

export async function logoutRemoteProfile(id: number): Promise<RemoteProfile> {
  return logoutRemoteProfileAdmin(id);
}

export async function testRemoteProfile(id: number): Promise<RemoteProfile> {
  return testRemoteProfileAdmin(id);
}
