import { apiCall } from './common';

export type AdminSecurityEvent = {
  event: string;
  created_at: string;
  ip_hint?: string;
  user_agent?: string;
  detail?: Record<string, unknown>;
};

export const getAdminSecurityEvents = () => apiCall<{ events: AdminSecurityEvent[] }>('/admin/security-events');
