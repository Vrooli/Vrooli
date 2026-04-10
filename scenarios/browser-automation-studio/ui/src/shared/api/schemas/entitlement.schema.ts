import { z } from 'zod';

/**
 * Entitlement-related Zod schemas for API response validation.
 * These schemas provide runtime validation for subscription and usage data.
 */

// Subscription tier enum
export const SubscriptionTierSchema = z.enum(['free', 'solo', 'pro', 'studio', 'business']);

// Subscription status enum
export const SubscriptionStatusSchema = z.enum(['active', 'trialing', 'past_due', 'canceled', 'inactive']);

// Feature access summary schema
export const FeatureAccessSummarySchema = z.object({
  id: z.string(),
  label: z.string(),
  description: z.string(),
  required_tier: SubscriptionTierSchema.optional(),
  has_access: z.boolean(),
});

// Main entitlement status response schema
export const EntitlementStatusResponseSchema = z.object({
  user_identity: z.string(),
  status: SubscriptionStatusSchema,
  tier: SubscriptionTierSchema,
  is_active: z.boolean(),
  features: z.array(z.string()),
  feature_access: z.array(FeatureAccessSummarySchema).optional(),
  monthly_limit: z.number(), // -1 for unlimited
  monthly_used: z.number(),
  monthly_remaining: z.number(), // -1 for unlimited
  requires_watermark: z.boolean(),
  can_use_ai: z.boolean(),
  can_use_recording: z.boolean(),
  entitlements_enabled: z.boolean(),
  override_tier: SubscriptionTierSchema.optional(),

  // AI Credits
  ai_credits_used: z.number(),
  ai_credits_limit: z.number(), // -1 for unlimited
  ai_credits_remaining: z.number(), // -1 for unlimited
  ai_requests_count: z.number(),
  ai_reset_date: z.string(), // ISO date
});

// Usage period schema
export const UsagePeriodSchema = z.object({
  billing_month: z.string(),
  total_credits_used: z.number(),
  total_operations: z.number(),
  by_operation: z.record(z.string(), z.number()),
  operation_counts: z.record(z.string(), z.number()),
  credits_limit: z.number(),
  credits_remaining: z.number(),
  period_start: z.string(),
  period_end: z.string(),
  reset_date: z.string(),
});

// Operation log entry schema
export const OperationLogEntrySchema = z.object({
  id: z.string(),
  operation_type: z.string(),
  credits_charged: z.number(),
  success: z.boolean(),
  created_at: z.string(),
  metadata: z.record(z.string(), z.unknown()).optional(),
  error_message: z.string().optional(),
});

// Operation log page schema
export const OperationLogPageSchema = z.object({
  user_identity: z.string(),
  billing_month: z.string(),
  operations: z.array(OperationLogEntrySchema),
  total: z.number(),
  limit: z.number(),
  offset: z.number(),
  has_more: z.boolean(),
});

// Usage history response schema
export const UsageHistoryResponseSchema = z.object({
  periods: z.array(UsagePeriodSchema),
});

// Identity response schema (for getUserEmail)
export const IdentityResponseSchema = z.object({
  email: z.string().optional(),
});

// API source response schema
export const ApiSourceResponseSchema = z.object({
  source: z.enum(['production', 'local', 'disabled']),
  local_port: z.number().optional(),
});

// Export types from schemas
export type SubscriptionTier = z.infer<typeof SubscriptionTierSchema>;
export type SubscriptionStatus = z.infer<typeof SubscriptionStatusSchema>;
export type FeatureAccessSummary = z.infer<typeof FeatureAccessSummarySchema>;
export type EntitlementStatusResponse = z.infer<typeof EntitlementStatusResponseSchema>;
export type UsagePeriod = z.infer<typeof UsagePeriodSchema>;
export type OperationLogEntry = z.infer<typeof OperationLogEntrySchema>;
export type OperationLogPage = z.infer<typeof OperationLogPageSchema>;
export type UsageHistoryResponse = z.infer<typeof UsageHistoryResponseSchema>;
export type IdentityResponse = z.infer<typeof IdentityResponseSchema>;
export type ApiSourceResponse = z.infer<typeof ApiSourceResponseSchema>;
