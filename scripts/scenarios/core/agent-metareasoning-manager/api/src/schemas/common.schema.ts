import { z } from 'zod';

// Common schemas used across the API

export const uuidSchema = z.string().uuid();

export const paginationSchema = z.object({
  limit: z.coerce.number().min(1).max(100).default(20),
  offset: z.coerce.number().min(0).default(0)
});

export const categorySchema = z.enum(['decision', 'analysis', 'risk', 'strategy', 'review']);

export const statusSchema = z.enum(['draft', 'active', 'archived']);

export const prioritySchema = z.enum(['low', 'medium', 'high', 'critical']);

export const timestampSchema = z.object({
  created_at: z.date(),
  updated_at: z.date()
});

export const apiTokenSchema = z.object({
  id: uuidSchema,
  name: z.string().min(1).max(255),
  token_hash: z.string(),
  permissions: z.record(z.boolean()),
  expires_at: z.date().nullable(),
  created_at: z.date()
});

export type UUID = z.infer<typeof uuidSchema>;
export type Pagination = z.infer<typeof paginationSchema>;
export type Category = z.infer<typeof categorySchema>;
export type Status = z.infer<typeof statusSchema>;
export type Priority = z.infer<typeof prioritySchema>;
export type Timestamp = z.infer<typeof timestampSchema>;
export type ApiToken = z.infer<typeof apiTokenSchema>;