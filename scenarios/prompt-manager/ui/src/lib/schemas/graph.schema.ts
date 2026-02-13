/**
 * Graph-related Zod schemas for runtime validation.
 *
 * These schemas match the Go API response shapes from api/graph/models.go
 * and api/graph/handlers.go.
 *
 * IMPORTANT: Go nil slices serialize to JSON `null`, not `[]`.
 * Array fields must use `.nullable().optional().transform(val => val ?? [])`
 * to handle both null (Go nil) and undefined (missing field).
 */

import { z } from 'zod'

/**
 * Helper for array fields that may be null (Go nil slice) or undefined (missing).
 * Transforms both null and undefined to empty array.
 */
function nullableArray<T extends z.ZodType>(schema: T) {
  return z
    .array(schema)
    .nullable()
    .optional()
    .transform((val) => val ?? [])
}

// ============================================================================
// Enums
// ============================================================================

export const NodeTypeSchema = z.enum(['team', 'agent', 'skill', 'cli'])
export type NodeType = z.infer<typeof NodeTypeSchema>

export const EdgeKindSchema = z.enum([
  'cli-read',
  'bold-listed',
  'default-scope',
  'path-ref',
  'membership',
  'code-usage',
])
export type EdgeKind = z.infer<typeof EdgeKindSchema>

// ============================================================================
// Core Graph Types (match api/graph/models.go)
// ============================================================================

export const HealthFactorSchema = z.record(z.string(), z.number())
export type HealthFactor = z.infer<typeof HealthFactorSchema>

export const HealthMessageSchema = z.object({
  key: z.string(),
  severity: z.enum(['info', 'warning', 'critical']),
  factor: z.string().optional().default(''),
  summary: z.string(),
  detail: z.string().optional().default(''),
  recommendation: z.string().optional().default(''),
  metricValue: z.number().optional(),
  target: z.string().optional().default(''),
})
export type HealthMessage = z.infer<typeof HealthMessageSchema>

export const HealthScoreSchema = z.object({
  nodeId: z.string(),
  score: z.number(),
  factors: HealthFactorSchema,
  messages: nullableArray(HealthMessageSchema),
})
export type HealthScore = z.infer<typeof HealthScoreSchema>

export const EntityHealthWeightsSchema = z.object({
  outgoingEdges: z.number(),
  incomingEdges: z.number(),
  codeUsage: z.number(),
  recentActivity: z.number(),
  skillContentLength: z.number(),
  agentContextLoad: z.number(),
  teamMemberCountBalance: z.number(),
  teamRoleCoverage: z.number(),
})
export type EntityHealthWeights = z.infer<typeof EntityHealthWeightsSchema>

export const CLIHealthConfigSchema = z.object({
  neutralCommands: z.array(z.string()),
  externalToolScore: z.number(),
  scenarioFallbackScore: z.number(),
})
export type CLIHealthConfig = z.infer<typeof CLIHealthConfigSchema>

export const GraphHealthConfigSchema = z.object({
  team: EntityHealthWeightsSchema,
  agent: EntityHealthWeightsSchema,
  skill: EntityHealthWeightsSchema,
  cli: CLIHealthConfigSchema,
})
export type GraphHealthConfig = z.infer<typeof GraphHealthConfigSchema>

export const GraphNodeSchema = z.object({
  id: z.string(),
  type: NodeTypeSchema,
  label: z.string(),
  description: z.string().optional().default(''),
  status: z.string().optional().default(''),
  tags: nullableArray(z.string()),
})
export type GraphNode = z.infer<typeof GraphNodeSchema>

export const GraphEdgeSchema = z.object({
  from: z.string(),
  to: z.string(),
  kind: EdgeKindSchema,
  category: z.string().optional().default(''),
  command: z.string().optional(),
  subcommand: z.string().optional(),
  sourceFile: z.string().optional().default(''),
  lineNumber: z.number().optional().default(0),
})
export type GraphEdge = z.infer<typeof GraphEdgeSchema>

export const GraphDataSchema = z.object({
  nodes: nullableArray(GraphNodeSchema),
  edges: nullableArray(GraphEdgeSchema),
  healthScores: nullableArray(HealthScoreSchema),
})
export type GraphData = z.infer<typeof GraphDataSchema>

export const GraphIndexSchema = z.object({
  generatedAt: z.string(),
  graph: GraphDataSchema,
})
export type GraphIndex = z.infer<typeof GraphIndexSchema>

// ============================================================================
// Node Detail (single node + adjacent edges)
// ============================================================================

export const NodeDetailSchema = z.object({
  node: GraphNodeSchema,
  adjacentEdges: nullableArray(GraphEdgeSchema),
})
export type NodeDetail = z.infer<typeof NodeDetailSchema>

// ============================================================================
// Query Response Types
// ============================================================================

export const PopularityEntrySchema = GraphNodeSchema
export type PopularityEntry = z.infer<typeof PopularityEntrySchema>

export const CircularRefSchema = z.array(z.string())
export type CircularRef = z.infer<typeof CircularRefSchema>

// ============================================================================
// API Response Schemas (match api/graph/handlers.go)
// ============================================================================

export const GraphResponseSchema = GraphIndexSchema
export type GraphResponse = z.infer<typeof GraphResponseSchema>

export const NodeDetailResponseSchema = NodeDetailSchema
export type NodeDetailResponse = z.infer<typeof NodeDetailResponseSchema>

// Regenerate returns the full GraphIndex
export const RegenerateResponseSchema = GraphIndexSchema
export type RegenerateResponse = z.infer<typeof RegenerateResponseSchema>

// Query endpoints return []Node
export const NodeListResponseSchema = nullableArray(GraphNodeSchema)
export type NodeListResponse = z.infer<typeof NodeListResponseSchema>

// Popular returns []Node
export const PopularityResponseSchema = nullableArray(GraphNodeSchema)
export type PopularityResponse = z.infer<typeof PopularityResponseSchema>

// Cycles returns [][]string
export const CircularRefResponseSchema = nullableArray(CircularRefSchema)
export type CircularRefResponse = z.infer<typeof CircularRefResponseSchema>

// Health returns []HealthScore
export const GraphHealthResponseSchema = nullableArray(HealthScoreSchema)
export type GraphHealthResponse = z.infer<typeof GraphHealthResponseSchema>

export const NodeHealthResponseSchema = HealthScoreSchema
export type NodeHealthResponse = z.infer<typeof NodeHealthResponseSchema>

export const GraphHealthConfigResponseSchema = GraphHealthConfigSchema
export type GraphHealthConfigResponse = z.infer<typeof GraphHealthConfigResponseSchema>
