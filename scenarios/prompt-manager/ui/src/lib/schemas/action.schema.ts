/**
 * Action-related Zod schemas for runtime validation.
 *
 * These schemas match the Go API response shapes from api/actions and
 * api/store/models.go. Actions are executable contracts, so they deliberately
 * do not reuse Skill schemas.
 *
 * IMPORTANT: Go nil slices/maps serialize as JSON null. Collection fields use
 * defaults so UI code can treat them as arrays/objects.
 */

import { z } from 'zod'

function nullableArray<T extends z.ZodType>(schema: T) {
  return z
    .array(schema)
    .nullable()
    .optional()
    .transform((val) => val ?? [])
}

function nullableRecord<T extends z.ZodType>(schema: T) {
  return z
    .record(z.string(), schema)
    .nullable()
    .optional()
    .transform((val) => val ?? {})
}

export const ActionStatusSchema = z.enum(['active', 'draft', 'archived'])
export type ActionStatus = z.infer<typeof ActionStatusSchema>

export const ActionOwnerSchema = z.object({
  type: z.enum(['project', 'scenario', 'resource', 'team']),
  id: z.string(),
})
export type ActionOwner = z.infer<typeof ActionOwnerSchema>

export const ActionCommandSchema = z.object({
  argv: nullableArray(z.string()),
})
export type ActionCommand = z.infer<typeof ActionCommandSchema>

export const ActionInputTypeSchema = z.enum([
  'string',
  'number',
  'integer',
  'boolean',
  'file',
  'path',
  'scenario',
  'team',
  'action',
])
export type ActionInputType = z.infer<typeof ActionInputTypeSchema>

export const ActionOutputTypeSchema = z.enum([
  'string',
  'number',
  'integer',
  'boolean',
  'file',
  'path',
  'json',
])
export type ActionOutputType = z.infer<typeof ActionOutputTypeSchema>

export const ActionInputSchema = z.object({
  type: ActionInputTypeSchema,
  description: z.string().nullable().optional().transform((val) => val ?? ''),
  required: z.boolean().nullable().optional().transform((val) => val ?? false),
  enum: nullableArray(z.string()),
  default: z.unknown().optional(),
  pattern: z.string().nullable().optional().transform((val) => val ?? ''),
  min: z.number().nullable().optional(),
  max: z.number().nullable().optional(),
  maxLength: z.number().nullable().optional(),
  allowMultiline: z.boolean().nullable().optional().transform((val) => val ?? false),
})
export type ActionInput = z.infer<typeof ActionInputSchema>

export const ActionOutputSchema = z.object({
  type: ActionOutputTypeSchema,
  description: z.string().nullable().optional().transform((val) => val ?? ''),
})
export type ActionOutput = z.infer<typeof ActionOutputSchema>

export const ActionPermissionsSchema = z.object({
  filesystemRead: z.boolean().nullable().optional().transform((val) => val ?? false),
  filesystemWrite: z.boolean().nullable().optional().transform((val) => val ?? false),
  localhostNetwork: z.boolean().nullable().optional().transform((val) => val ?? false),
  externalNetwork: z.boolean().nullable().optional().transform((val) => val ?? false),
  apiRead: z.boolean().nullable().optional().transform((val) => val ?? false),
  apiWrite: z.boolean().nullable().optional().transform((val) => val ?? false),
  processStart: z.boolean().nullable().optional().transform((val) => val ?? false),
  processStop: z.boolean().nullable().optional().transform((val) => val ?? false),
  hostConfigure: z.boolean().nullable().optional().transform((val) => val ?? false),
  secretRead: z.boolean().nullable().optional().transform((val) => val ?? false),
  secretWrite: z.boolean().nullable().optional().transform((val) => val ?? false),
  destructive: z.boolean().nullable().optional().transform((val) => val ?? false),
})
export type ActionPermissions = z.infer<typeof ActionPermissionsSchema>

export const ActionExampleSchema = z.object({
  description: z.string().nullable().optional().transform((val) => val ?? ''),
  input: z.record(z.string(), z.unknown()).nullable().optional().transform((val) => val ?? {}),
})
export type ActionExample = z.infer<typeof ActionExampleSchema>

export const ActionExecutionSchema = z.object({
  timeoutSeconds: z.number().nullable().optional(),
  outputMode: z.enum(['', 'stdout', 'json']).nullable().optional().transform((val) => val ?? ''),
  runEligible: z.boolean().nullable().optional(),
})
export type ActionExecution = z.infer<typeof ActionExecutionSchema>

export const ActionValidationSchema = z.object({
  mode: z.enum(['', 'contract', 'owner']).nullable().optional().transform((val) => val ?? ''),
  argv: nullableArray(z.string()),
})
export type ActionValidation = z.infer<typeof ActionValidationSchema>

export const ActionSchema = z.object({
  kind: z.literal('action'),
  schemaVersion: z.number(),
  id: z.string(),
  name: z.string(),
  description: z.string().nullable().optional().transform((val) => val ?? ''),
  status: ActionStatusSchema,
  owner: ActionOwnerSchema,
  command: ActionCommandSchema,
  inputs: nullableRecord(ActionInputSchema),
  outputs: nullableRecord(ActionOutputSchema),
  permissions: ActionPermissionsSchema.nullable().optional().transform(
    (val) => val ?? ActionPermissionsSchema.parse({})
  ),
  examples: nullableArray(ActionExampleSchema),
  tags: nullableArray(z.string()),
  execution: ActionExecutionSchema.nullable().optional(),
  validation: ActionValidationSchema.nullable().optional(),
  revision: z.number(),
  createdAt: z.string(),
  updatedAt: z.string(),
})
export type Action = z.infer<typeof ActionSchema>

export const ActionArraySchema = z.array(ActionSchema)

export const CreateActionRequestSchema = ActionSchema.omit({
  revision: true,
  createdAt: true,
  updatedAt: true,
}).extend({
  pack: z.enum(['core', 'local', 'drafts']).optional(),
  revision: z.number().optional(),
  createdAt: z.string().optional(),
  updatedAt: z.string().optional(),
})
export type CreateActionRequest = z.infer<typeof CreateActionRequestSchema>

export const UpdateActionRequestSchema = ActionSchema.partial().extend({
  id: z.string().optional(),
})
export type UpdateActionRequest = z.infer<typeof UpdateActionRequestSchema>

export const CommandCertaintySchema = z.enum(['none', 'owner-only', 'command', 'operation'])
export type CommandCertainty = z.infer<typeof CommandCertaintySchema>

export const CommandEffectSchema = z.enum(['read', 'write', 'destructive', 'admin'])
export type CommandEffect = z.infer<typeof CommandEffectSchema>

export const CommandResolutionSchema = z.object({
  certainty: CommandCertaintySchema,
  owner: z.object({
    type: z.string(),
    id: z.string(),
  }),
  target: z.string(),
  commandPath: nullableArray(z.string()),
  effect: CommandEffectSchema.nullable().optional(),
  permissions: nullableArray(z.string()),
  runSurfaces: nullableArray(z.string()),
  requiresConfirmation: z.boolean().nullable().optional().transform((val) => val ?? false),
  message: z.string().nullable().optional().transform((val) => val ?? ''),
})
export type CommandResolution = z.infer<typeof CommandResolutionSchema>

export const ActionValidationCheckSchema = z.object({
  code: z.string(),
  status: z.enum(['passed', 'warning', 'failed']),
  message: z.string(),
  path: z.string().nullable().optional().transform((val) => val ?? ''),
})
export type ActionValidationCheck = z.infer<typeof ActionValidationCheckSchema>

export const ActionValidationResponseSchema = z.object({
  actionId: z.string().nullable().optional().transform((val) => val ?? ''),
  valid: z.boolean(),
  runnable: z.boolean().nullable().optional().transform((val) => val ?? false),
  unvalidated: z.boolean().nullable().optional().transform((val) => val ?? false),
  requiresConfirmation: z.boolean().nullable().optional().transform((val) => val ?? false),
  status: z.string().nullable().optional().transform((val) => val ?? ''),
  command: CommandResolutionSchema.nullable().optional(),
  checks: nullableArray(ActionValidationCheckSchema),
  action: ActionSchema.nullable().optional(),
})
export type ActionValidationResponse = z.infer<typeof ActionValidationResponseSchema>

export const ActionMutationResponseSchema = z.object({
  action: ActionSchema,
  validation: ActionValidationResponseSchema,
})
export type ActionMutationResponse = z.infer<typeof ActionMutationResponseSchema>

export const ActionRunRequestSchema = z.object({
  input: z.record(z.string(), z.unknown()).optional(),
  dryRun: z.boolean().optional(),
})
export type ActionRunRequest = z.infer<typeof ActionRunRequestSchema>

export const ActionRunStatusSchema = z.enum([
  'completed',
  'failed',
  'timed-out',
  'rejected',
  'throttled',
  'dry-run',
])
export type ActionRunStatus = z.infer<typeof ActionRunStatusSchema>

export const ActionRunResponseSchema = z.object({
  actionId: z.string(),
  status: ActionRunStatusSchema,
  exitCode: z.number().nullable().optional(),
  durationMs: z.number().nullable().optional().transform((val) => val ?? 0),
  argv: nullableArray(z.string()),
  stdout: z.string().nullable().optional().transform((val) => val ?? ''),
  stderr: z.string().nullable().optional().transform((val) => val ?? ''),
  stdoutTruncated: z.boolean().nullable().optional().transform((val) => val ?? false),
  stderrTruncated: z.boolean().nullable().optional().transform((val) => val ?? false),
  output: z.record(z.string(), z.unknown()).nullable().optional(),
  validation: ActionValidationResponseSchema,
  error: z.string().nullable().optional().transform((val) => val ?? ''),
})
export type ActionRunResponse = z.infer<typeof ActionRunResponseSchema>
