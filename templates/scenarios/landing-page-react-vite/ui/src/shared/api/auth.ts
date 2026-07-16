import { createClient } from '@connectrpc/connect';
import { AdminAuthService } from '@vrooli/proto-types/landing-page-react-vite/v1/admin_pb';
import type { AdminSessionResponse } from '@vrooli/proto-types/landing-page-react-vite/v1/admin_pb';

import { transport } from './client';

const authClient = createClient(AdminAuthService, transport);

/** Logs in an admin; the server sets an HMAC session cookie on success. */
export function adminLogin(email: string, password: string): Promise<AdminSessionResponse> {
  return authClient.login({ email, password });
}

/** Logs out the current admin session. */
export async function adminLogout(): Promise<boolean> {
  const resp = await authClient.logout({});
  return resp.success;
}

/** Reports whether the current session is an authenticated admin. */
export function checkAdminSession(): Promise<AdminSessionResponse> {
  return authClient.session({});
}

export type { AdminSessionResponse };
