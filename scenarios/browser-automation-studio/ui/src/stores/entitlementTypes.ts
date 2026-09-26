import type {
  OperationLogEntry as ProtoOperationLogEntry,
  OperationLogPage as ProtoOperationLogPage,
  UsageSummary as ProtoUsageSummary,
} from '@vrooli/proto-types/browser-automation-studio/v1/entitlement/entitlement_pb';
import type { Timestamp } from '@bufbuild/protobuf/wkt';

import { PLAN_CONFIG, type PlanTier } from '@components/MonetizationAccount';

export type SubscriptionTier = PlanTier;
export type SubscriptionStatus = 'active' | 'trialing' | 'past_due' | 'canceled' | 'inactive';

export interface FeatureAccessSummary {
  id: string;
  label: string;
  description: string;
  required_tier?: SubscriptionTier;
  has_access: boolean;
}

export interface EntitlementStatusResponse {
  user_identity: string;
  status: SubscriptionStatus;
  tier: SubscriptionTier;
  is_active: boolean;
  features: string[];
  feature_access?: FeatureAccessSummary[];
  monthly_limit: number;
  monthly_used: number;
  monthly_remaining: number;
  requires_watermark: boolean;
  can_use_ai: boolean;
  can_use_recording: boolean;
  entitlements_enabled: boolean;
  ai_credits_used: number;
  ai_credits_limit: number;
  ai_credits_remaining: number;
  ai_requests_count: number;
  ai_reset_date: string;
}

export interface UsagePeriod {
  billing_month: string;
  total_credits_used: number;
  total_operations: number;
  by_operation: Record<string, number>;
  operation_counts: Record<string, number>;
  credits_limit: number;
  credits_remaining: number;
  period_start: string;
  period_end: string;
  reset_date: string;
}

export interface OperationLogEntry {
  id: string;
  operation_type: string;
  credits_charged: number;
  success: boolean;
  created_at: string;
  metadata?: Record<string, unknown>;
  error_message?: string;
}

export interface OperationLogPage {
  user_identity: string;
  billing_month: string;
  operations: OperationLogEntry[];
  total: number;
  limit: number;
  offset: number;
  has_more: boolean;
}

export const TIER_CONFIG = PLAN_CONFIG;

export const tsToIso = (ts?: Timestamp): string => {
  if (!ts) return '';
  const seconds = typeof ts.seconds === 'bigint' ? Number(ts.seconds) : Number(ts.seconds ?? 0);
  const nanos = Number(ts.nanos ?? 0);
  return new Date(seconds * 1000 + Math.floor(nanos / 1_000_000)).toISOString();
};

export const toUsagePeriod = (u: ProtoUsageSummary): UsagePeriod => ({
  billing_month: u.billingMonth,
  total_credits_used: u.totalCreditsUsed,
  total_operations: u.totalOperations,
  by_operation: { ...u.byOperation },
  operation_counts: { ...u.operationCounts },
  credits_limit: u.creditsLimit,
  credits_remaining: u.creditsRemaining,
  period_start: tsToIso(u.periodStart),
  period_end: tsToIso(u.periodEnd),
  reset_date: tsToIso(u.resetDate),
});

export const toOperationLogPage = (p: ProtoOperationLogPage): OperationLogPage => ({
  user_identity: p.userIdentity,
  billing_month: p.billingMonth,
  operations: (p.operations ?? []).map((entry: ProtoOperationLogEntry) => ({
    id: entry.id,
    operation_type: entry.operationType,
    credits_charged: entry.creditsCharged,
    success: entry.success,
    created_at: tsToIso(entry.createdAt),
    metadata: entry.metadata ? (entry.metadata.fields as unknown as Record<string, unknown>) : undefined,
    error_message: entry.errorMessage || undefined,
  })),
  total: p.total,
  limit: p.limit,
  offset: p.offset,
  has_more: p.hasMore,
});
