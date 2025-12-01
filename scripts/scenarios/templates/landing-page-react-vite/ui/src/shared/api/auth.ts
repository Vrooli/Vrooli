import { apiCall } from './common';

export async function adminLogin(email: string, password: string) {
  return apiCall<{ success: boolean; message: string }>('/admin/login', {
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
  return apiCall<{ authenticated: boolean; email?: string }>('/admin/session');
}
