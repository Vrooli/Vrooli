/**
 * Experiment Zod schemas for runtime validation.
 *
 * These schemas match the Go API response shapes from api/skills/experiment_models.go.
 */

import { z } from 'zod'

/**
 * Helper for map fields that may be null (Go nil map) or undefined.
 */
const nullableIntMap = z
  .record(z.string(), z.number())
  .nullable()
  .optional()
  .transform((val) => val ?? {})

/**
 * An experiment arm with variant assignment and weight.
 */
export const ExperimentArmSchema = z.object({
  variantId: z.string(),
  variantName: z.string().nullable().optional().transform((val) => val ?? ''),
  weight: z.number(),
})

export type ExperimentArm = z.infer<typeof ExperimentArmSchema>

/**
 * Arm input for creating/updating experiments.
 */
export const ExperimentArmInputSchema = z.object({
  variantId: z.string(),
  weight: z.number().min(0).max(1),
})

export type ExperimentArmInput = z.infer<typeof ExperimentArmInputSchema>

/**
 * ExperimentResponse from the API.
 */
export const ExperimentSchema = z.object({
  id: z.string(),
  skillId: z.string(),
  name: z.string(),
  hypothesis: z.string().nullable().optional().transform((val) => val ?? ''),
  status: z.enum(['draft', 'running', 'concluded']),
  arms: z.array(ExperimentArmSchema).nullable().optional().transform((val) => val ?? []),
  outcomeCounts: nullableIntMap,
  startedAt: z.string().nullable().optional(),
  concludedAt: z.string().nullable().optional(),
  winnerVariantId: z.string().nullable().optional(),
  notes: z.string().nullable().optional().transform((val) => val ?? ''),
  createdAt: z.string(),
  updatedAt: z.string(),
  revision: z.number(),
})

export type Experiment = z.infer<typeof ExperimentSchema>

/**
 * Array of experiments.
 */
export const ExperimentArraySchema = z.array(ExperimentSchema)

/**
 * Request to create an experiment.
 */
export const CreateExperimentRequestSchema = z.object({
  id: z.string().optional(),
  skillId: z.string(),
  name: z.string().min(1, 'Name is required'),
  hypothesis: z.string().optional(),
  arms: z.array(ExperimentArmInputSchema).min(2, 'At least 2 arms required'),
})

export type CreateExperimentRequest = z.infer<typeof CreateExperimentRequestSchema>

/**
 * Request to conclude an experiment.
 */
export const ConcludeExperimentRequestSchema = z.object({
  winnerVariantId: z.string(),
  notes: z.string().optional(),
})

export type ConcludeExperimentRequest = z.infer<typeof ConcludeExperimentRequestSchema>
