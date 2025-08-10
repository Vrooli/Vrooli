import { z } from 'zod';
import { uuidSchema, categorySchema, statusSchema, paginationSchema } from './common.schema.js';

// Prompt entity schema
export const promptSchema = z.object({
  id: uuidSchema,
  name: z.string().min(1).max(255),
  description: z.string().max(1000).optional(),
  category: categorySchema,
  template: z.string().min(10).max(10000),
  variables: z.array(z.string()).default([]),
  tags: z.array(z.string()).default([]),
  status: statusSchema,
  version: z.number().int().positive().default(1),
  created_at: z.date(),
  updated_at: z.date()
});

// List prompts query parameters
export const listPromptsQuerySchema = z.object({
  query: paginationSchema.extend({
    category: categorySchema.optional()
  }),
  params: z.object({}).optional(),
  body: z.object({}).optional()
});

// Create prompt request
export const createPromptSchema = z.object({
  body: z.object({
    name: z.string().min(1).max(255),
    description: z.string().max(1000).optional(),
    category: categorySchema,
    template: z.string().min(10).max(10000),
    variables: z.array(z.string()).optional(),
    tags: z.array(z.string()).optional()
  })
});

// Update prompt request
export const updatePromptSchema = z.object({
  params: z.object({
    id: uuidSchema
  }),
  body: z.object({
    name: z.string().min(1).max(255).optional(),
    description: z.string().max(1000).optional(),
    category: categorySchema.optional(),
    template: z.string().min(10).max(10000).optional(),
    variables: z.array(z.string()).optional(),
    tags: z.array(z.string()).optional(),
    status: statusSchema.optional()
  })
});

// Delete prompt request
export const deletePromptSchema = z.object({
  params: z.object({
    id: uuidSchema
  })
});

// Get single prompt request
export const getPromptSchema = z.object({
  params: z.object({
    id: uuidSchema
  })
});

// Test prompt request
export const testPromptSchema = z.object({
  params: z.object({
    id: uuidSchema
  }),
  body: z.object({
    variables: z.record(z.string()),
    context: z.string().optional()
  })
});

export type Prompt = z.infer<typeof promptSchema>;
export type ListPromptsQuery = z.infer<typeof listPromptsQuerySchema>;
export type CreatePromptRequest = z.infer<typeof createPromptSchema>;
export type UpdatePromptRequest = z.infer<typeof updatePromptSchema>;
export type DeletePromptRequest = z.infer<typeof deletePromptSchema>;
export type GetPromptRequest = z.infer<typeof getPromptSchema>;
export type TestPromptRequest = z.infer<typeof testPromptSchema>;