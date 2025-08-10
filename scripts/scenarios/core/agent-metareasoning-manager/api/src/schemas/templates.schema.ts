import { z } from 'zod';
import { uuidSchema, categorySchema, paginationSchema } from './common.schema.js';

// Template entity schema
export const templateSchema = z.object({
  id: uuidSchema,
  name: z.string().min(1).max(255),
  description: z.string().max(1000).optional(),
  category: categorySchema,
  structure: z.object({
    prompts: z.array(uuidSchema),
    workflows: z.array(uuidSchema),
    sequence: z.array(z.object({
      type: z.enum(['prompt', 'workflow', 'analysis']),
      id: z.string(),
      config: z.record(z.any()).optional()
    }))
  }),
  variables: z.array(z.string()).default([]),
  tags: z.array(z.string()).default([]),
  usage_count: z.number().int().default(0),
  created_at: z.date(),
  updated_at: z.date()
});

// List templates query parameters
export const listTemplatesQuerySchema = z.object({
  query: paginationSchema.extend({
    category: categorySchema.optional(),
    tag: z.string().optional(),
    search: z.string().optional()
  })
});

// Create template request
export const createTemplateSchema = z.object({
  body: z.object({
    name: z.string().min(1).max(255),
    description: z.string().max(1000).optional(),
    category: categorySchema,
    structure: z.object({
      prompts: z.array(uuidSchema),
      workflows: z.array(uuidSchema),
      sequence: z.array(z.object({
        type: z.enum(['prompt', 'workflow', 'analysis']),
        id: z.string(),
        config: z.record(z.any()).optional()
      }))
    }),
    variables: z.array(z.string()).optional(),
    tags: z.array(z.string()).optional()
  })
});

// Apply template request
export const applyTemplateSchema = z.object({
  params: z.object({
    id: uuidSchema
  }),
  body: z.object({
    context: z.string().min(10).max(5000),
    variables: z.record(z.string()),
    options: z.object({
      async: z.boolean().default(false),
      includeIntermediateResults: z.boolean().default(false)
    }).optional()
  })
});

// Template application result
export const templateResultSchema = z.object({
  template_id: uuidSchema,
  execution_id: z.string(),
  status: z.enum(['completed', 'partial', 'failed']),
  results: z.array(z.object({
    step: z.string(),
    type: z.enum(['prompt', 'workflow', 'analysis']),
    result: z.any(),
    duration_ms: z.number()
  })),
  summary: z.string(),
  total_duration_ms: z.number()
});

export type Template = z.infer<typeof templateSchema>;
export type ListTemplatesQuery = z.infer<typeof listTemplatesQuerySchema>;
export type CreateTemplateRequest = z.infer<typeof createTemplateSchema>;
export type ApplyTemplateRequest = z.infer<typeof applyTemplateSchema>;
export type TemplateResult = z.infer<typeof templateResultSchema>;