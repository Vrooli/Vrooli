import { z } from 'zod';
import { FlexibleTimestampSchema } from './common.schema';

/**
 * Auth-related Zod schemas for API response validation.
 */

// ===== Admin Auth Schemas =====

// Admin session response schema
export const AdminSessionResponseSchema = z.object({
  authenticated: z.boolean(),
  email: z.string().optional(),
  reset_enabled: z.boolean().optional(),
});

// Admin profile schema
export const AdminProfileSchema = z.object({
  email: z.string(),
  is_default_email: z.boolean(),
  is_default_password: z.boolean(),
});

// ===== User Auth Schemas =====

// User auth user schema
export const UserAuthUserSchema = z.object({
  id: z.string(),
  email: z.string(),
  email_verified: z.boolean(),
  stripe_customer_id: z.string().optional(),
  created_at: FlexibleTimestampSchema,
  last_login_at: z.string().nullable().optional(),
});

// User auth tokens schema
export const UserAuthTokensSchema = z.object({
  access_token: z.string(),
  refresh_token: z.string(),
  expires_at: z.string(),
  token_type: z.string(),
});

// Magic link response schema
export const MagicLinkResponseSchema = z.object({
  message: z.string(),
});

// Verify magic link response schema
export const VerifyMagicLinkResponseSchema = UserAuthTokensSchema.extend({
  user: UserAuthUserSchema,
});

// User auth me response schema
export const UserAuthMeResponseSchema = z.object({
  user: UserAuthUserSchema,
});

// Export inferred types
export type AdminSessionResponse = z.infer<typeof AdminSessionResponseSchema>;
export type AdminProfile = z.infer<typeof AdminProfileSchema>;
export type UserAuthUser = z.infer<typeof UserAuthUserSchema>;
export type UserAuthTokens = z.infer<typeof UserAuthTokensSchema>;
export type MagicLinkResponse = z.infer<typeof MagicLinkResponseSchema>;
export type VerifyMagicLinkResponse = z.infer<typeof VerifyMagicLinkResponseSchema>;
export type UserAuthMeResponse = z.infer<typeof UserAuthMeResponseSchema>;
