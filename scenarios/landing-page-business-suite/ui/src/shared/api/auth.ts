import { apiCall } from './common';

export interface AdminSessionResponse {
  authenticated: boolean;
  email?: string;
  reset_enabled?: boolean;
}

export interface AdminProfile {
  email: string;
  is_default_email: boolean;
  is_default_password: boolean;
}

export interface AdminProfileUpdatePayload {
  current_password: string;
  new_email?: string;
  new_password?: string;
}

export async function adminLogin(email: string, password: string) {
  return apiCall<AdminSessionResponse>('/admin/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}

export async function adminLogout() {
  return apiCall<{ success: boolean }>('/admin/logout', {
    method: 'POST',
  });
}

export async function checkAdminSession() {
  return apiCall<AdminSessionResponse>('/admin/session');
}

export async function getAdminProfile() {
  return apiCall<AdminProfile>('/admin/profile');
}

export async function updateAdminProfile(payload: AdminProfileUpdatePayload) {
  return apiCall<AdminProfile>('/admin/profile', {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}
