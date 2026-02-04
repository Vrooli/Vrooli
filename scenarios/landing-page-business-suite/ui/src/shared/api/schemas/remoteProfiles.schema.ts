import { z } from 'zod';
import { FlexibleTimestampSchema } from './common.schema';

export const RemoteProfileStatusSchema = z.enum(['unknown', 'active', 'expired', 'error']);

export const RemoteProfileSchema = z.object({
  id: z.number(),
  tag: z.string(),
  label: z.string().nullable().optional(),
  api_base: z.string(),
  status: RemoteProfileStatusSchema,
  has_session: z.boolean(),
  session_expires_at: FlexibleTimestampSchema,
  last_login_at: FlexibleTimestampSchema,
  last_used_at: FlexibleTimestampSchema,
  created_by: z.number().nullable().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const RemoteProfilesListResponseSchema = z.object({
  profiles: z.array(RemoteProfileSchema).optional(),
});

export type RemoteProfileStatus = z.infer<typeof RemoteProfileStatusSchema>;
export type RemoteProfile = z.infer<typeof RemoteProfileSchema>;
