/**
 * Agent file template schemas for runtime validation.
 */

import { z } from 'zod'

export const AgentFileTemplateSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().optional(),
  fileName: z.string(),
  content: z.string(),
})

export type AgentFileTemplate = z.infer<typeof AgentFileTemplateSchema>

export const AgentFileTemplateListResponseSchema = z.object({
  templates: z.array(AgentFileTemplateSchema),
  count: z.number().int().nonnegative().optional(),
})

export type AgentFileTemplateListResponse = z.infer<typeof AgentFileTemplateListResponseSchema>
