import { apiCall } from './common';

export type EmailReadinessCheck = { Provider: string; Status: 'pass' | 'warn' | 'fail'; Detail: string; Record?: string; Cost?: number };
export type EmailReadinessReport = { Domain: string; Providers: EmailReadinessCheck[]; DMARC: { Provider: string; Status: 'pass' | 'warn' | 'fail'; Detail: string }; CheckedAt: string };
export type SignInDeliveryAdminReport = {
  window_hours: number;
  delivery_24h?: { sent: number; failed: number; delivered: number; bounced: number; deferred: number; dropped: number; last_error?: string; last_webhook_event?: string };
  outbox?: Record<string, number>;
  providers?: Array<{ id: string; transport: string; enabled: boolean; cost_rank: number; recommended: boolean; credential: { ok: boolean; detail: string }; dns: { ok: boolean; detail: string }; state: string; remedy: string; skip_reason?: string }>;
  routing?: Array<{ provider_id: string; cost_rank: number; eligible: boolean; skip_reason?: string }>;
  chosen_provider?: string;
  quotas?: Array<{ provider_id: string; window: string; used: number; ceiling: number; started_at: string; ends_at: string }>;
  queue_health?: { by_priority: Array<{ priority: number; depth: number; oldest_age_seconds: number }>; oldest_wait_seconds: number; median_acceptance_seconds_24h?: number | null };
  history?: Array<{ id: string; purpose: string; recipient: string; requested_at: string; provider_id?: string; accepted_at?: string; acceptance_seconds?: number; status: string; last_error?: string; provider_message_id?: string; attempts?: Array<{ attempt: number; provider_id: string; outcome: string; diagnostic?: string; started_at: string; finished_at?: string }> }>;
};

export const getEmailReadiness = () => apiCall<EmailReadinessReport>('/admin/auth/email-readiness');
export const getSignInDeliveryAdminReport = (recipient = '') => apiCall<SignInDeliveryAdminReport>(`/admin/auth/delivery${recipient ? `?recipient=${encodeURIComponent(recipient)}` : ''}`);
export const sendDeliveryProbe = (to: string) => apiCall<{ request_id: string; expires_at: string }>('/admin/auth/delivery-probe', { method: 'POST', body: JSON.stringify({ to }) });
