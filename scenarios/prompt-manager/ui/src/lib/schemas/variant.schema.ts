/**
 * Variant Zod schemas for runtime validation.
 *
 * These schemas match the Go API response shapes from api/skills/variant_models.go.
 */

import { z } from 'zod'

/**
 * VariantResponse from the API.
 */
export const VariantSchema = z.object({
  id: z.string(),
  skillId: z.string(),
  name: z.string(),
  description: z.string().nullable().optional().transform((val) => val ?? ''),
  content: z.string().nullable().optional().transform((val) => val ?? ''),
  createdAt: z.string(),
  updatedAt: z.string(),
  revision: z.number(),
})

export type Variant = z.infer<typeof VariantSchema>

/**
 * Array of variants.
 */
export const VariantArraySchema = z.array(VariantSchema)

/**
 * Request to create a variant.
 */
export const CreateVariantRequestSchema = z.object({
  id: z.string().optional(),
  name: z.string().min(1, 'Name is required'),
  description: z.string().optional(),
  content: z.string().min(1, 'Content is required'),
})

export type CreateVariantRequest = z.infer<typeof CreateVariantRequestSchema>
