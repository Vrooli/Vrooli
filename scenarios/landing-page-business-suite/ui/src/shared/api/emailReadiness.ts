import { apiCall } from './common';

export type EmailReadinessCheck = { Name: string; Status: 'pass' | 'warn' | 'fail'; Detail: string };
export type EmailReadinessReport = { Checks: EmailReadinessCheck[]; CheckedAt: string };
export type SignInDeliveryAdminReport = { window_hours: number; delivery_24h?: { sent: number; failed: number; delivered: number; bounced: number; deferred: number; dropped: number; last_error?: string; last_webhook_event?: string } };

export const getEmailReadiness = () => apiCall<EmailReadinessReport>('/admin/auth/email-readiness');
export const getSignInDeliveryAdminReport = () => apiCall<SignInDeliveryAdminReport>('/admin/auth/delivery');
