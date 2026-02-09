import { z } from 'zod';
import { FlexibleTimestampSchema } from './common.schema';

export const RemoteProfileStatusSchema = z.enum(['unknown', 'active', 'expired', 'error']);

export const RemoteProfileSchema = z.object({
  id: z.number(),
  tag: z.string(),
  label: z.string().nullable().optional(),
  api_base: z.string(),
  connector_id: z.string().optional(),
  remote_session_id: z.string().nullable().optional(),
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

export const IncomingRemoteProfileSessionSchema = z.object({
  session_id: z.string(),
  admin_email: z.string(),
  connector_id: z.string(),
  profile_tag: z.string().optional(),
  origin: z.string().optional(),
  created_at: z.string(),
  last_activity: z.string(),
  expires_at: z.string(),
  ip_address: z.string().nullable().optional(),
  user_agent: z.string().nullable().optional(),
});

export const IncomingRemoteProfileSessionsResponseSchema = z.object({
  sessions: z.array(IncomingRemoteProfileSessionSchema).optional(),
});

export const RemoteProfileSessionLinksSchema = z.object({
  profile_id: z.number(),
  profile_tag: z.string(),
  connector_id: z.string(),
  local_has_session: z.boolean(),
  local_status: z.string(),
  local_session_expires_at: FlexibleTimestampSchema,
  remote_session_id: z.string().nullable().optional(),
  remote_sessions: z.array(IncomingRemoteProfileSessionSchema).optional(),
});

export type RemoteProfileStatus = z.infer<typeof RemoteProfileStatusSchema>;
export type RemoteProfile = z.infer<typeof RemoteProfileSchema>;
export type IncomingRemoteProfileSession = z.infer<typeof IncomingRemoteProfileSessionSchema>;
export type RemoteProfileSessionLinks = z.infer<typeof RemoteProfileSessionLinksSchema>;
