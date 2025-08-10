import { z } from 'zod';
import { uuidSchema, statusSchema, paginationSchema } from './common.schema.js';

// Workflow entity schema
export const workflowSchema = z.object({
  id: uuidSchema,
  name: z.string().min(1).max(255),
  description: z.string().max(1000).optional(),
  type: z.enum(['n8n', 'windmill']),
  workflow_id: z.string(), // External workflow ID
  input_schema: z.record(z.any()).optional(),
  output_schema: z.record(z.any()).optional(),
  tags: z.array(z.string()).default([]),
  status: statusSchema,
  execution_count: z.number().int().default(0),
  last_executed: z.date().nullable(),
  created_at: z.date(),
  updated_at: z.date()
});

// List workflows query parameters
export const listWorkflowsQuerySchema = z.object({
  query: paginationSchema.extend({
    type: z.enum(['n8n', 'windmill']).optional(),
    status: statusSchema.optional(),
    tag: z.string().optional()
  })
});

// Execute workflow request
export const executeWorkflowSchema = z.object({
  params: z.object({
    id: uuidSchema
  }),
  body: z.object({
    input: z.record(z.any()),
    async: z.boolean().default(false),
    timeout: z.number().min(1000).max(300000).default(30000) // 30 seconds default
  })
});

// Get workflow execution status
export const getExecutionStatusSchema = z.object({
  params: z.object({
    workflowId: uuidSchema,
    executionId: z.string()
  })
});

// Workflow execution result
export const workflowExecutionSchema = z.object({
  execution_id: z.string(),
  workflow_id: uuidSchema,
  status: z.enum(['pending', 'running', 'completed', 'failed', 'timeout']),
  input: z.record(z.any()),
  output: z.record(z.any()).nullable(),
  error: z.string().nullable(),
  started_at: z.date(),
  completed_at: z.date().nullable(),
  duration_ms: z.number().nullable()
});

export type Workflow = z.infer<typeof workflowSchema>;
export type ListWorkflowsQuery = z.infer<typeof listWorkflowsQuerySchema>;
export type ExecuteWorkflowRequest = z.infer<typeof executeWorkflowSchema>;
export type GetExecutionStatusRequest = z.infer<typeof getExecutionStatusSchema>;
export type WorkflowExecution = z.infer<typeof workflowExecutionSchema>;