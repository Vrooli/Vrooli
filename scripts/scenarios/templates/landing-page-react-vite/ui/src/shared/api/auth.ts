import { apiCall } from './common';

export interface AdminSessionResponse {
  authenticated: boolean;
  email?: string;
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
