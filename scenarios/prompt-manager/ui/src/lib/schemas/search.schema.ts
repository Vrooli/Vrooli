/**
 * Search-related Zod schemas for runtime validation.
 *
 * These schemas cover AI search functionality using Ollama embeddings and Qdrant.
 *
 * IMPORTANT: Go nil slices serialize to JSON `null`, not `[]`.
 * Array fields must handle both null and undefined.
 */

import { z } from 'zod'
import { DisplayFormatSchema } from './skill.schema'

/**
 * Helper for array fields that may be null (Go nil slice) or undefined (missing).
 */
const nullableStringArray = z
  .array(z.string())
  .nullable()
  .optional()
  .transform((val) => val ?? [])

/**
 * AI search result item.
 */
export const AISearchResultSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().nullable().optional().transform((val) => val ?? ''),
  folder: z.string(),
  tags: nullableStringArray,
  modes: nullableStringArray,
  score: z.number(),
  scorePercent: z.number(),
})

export type AISearchResult = z.infer<typeof AISearchResultSchema>

/**
 * Output mode for AI search.
 */
export const AISearchOutputSchema = z.enum(['results', 'combined', 'both'])
export type AISearchOutput = z.infer<typeof AISearchOutputSchema>

/**
 * Search method used.
 */
export const SearchMethodSchema = z.enum(['ai', 'text'])
export type SearchMethod = z.infer<typeof SearchMethodSchema>

/**
 * AI search response from the API.
 */
export const AISearchResponseSchema = z.object({
  results: z.array(AISearchResultSchema).nullable().optional().transform((val) => val ?? []),
  combined: z.string().nullable().optional(),
  skillCount: z.number().nullable().optional(),
  totalTokens: z.number().nullable().optional(),
  format: DisplayFormatSchema.nullable().optional(),
  total: z.number(),
  query: z.string(),
  method: SearchMethodSchema,
  output: AISearchOutputSchema.nullable().optional(),
})

export type AISearchResponse = z.infer<typeof AISearchResponseSchema>

/**
 * AI search request parameters.
 */
export const AISearchRequestSchema = z.object({
  query: z.string(),
  limit: z.number().optional(),
  output: AISearchOutputSchema.optional(),
  format: DisplayFormatSchema.optional(),
  renderLimit: z.number().optional(),
})

export type AISearchRequest = z.infer<typeof AISearchRequestSchema>

/**
 * AI search status response.
 */
export const AISearchStatusSchema = z.object({
  available: z.boolean(),
  ollama: z.boolean(),
  qdrant: z.boolean(),
  indexedCount: z.number(),
  message: z.string().optional(),
})

export type AISearchStatus = z.infer<typeof AISearchStatusSchema>

/**
 * AI reindex status response.
 */
export const AIReindexStatusSchema = z.object({
  running: z.boolean(),
  startedAt: z.string().optional(),
  finishedAt: z.string().optional(),
  indexed: z.number(),
  skipped: z.number(),
  errors: z.number(),
  total: z.number(),
  message: z.string().optional(),
  canceled: z.boolean().optional(),
  error: z.string().optional(),
})

export type AIReindexStatus = z.infer<typeof AIReindexStatusSchema>

/**
 * Content search match range.
 */
export const ContentMatchRangeSchema = z.object({
  start: z.number(),
  end: z.number(),
})

export type ContentMatchRange = z.infer<typeof ContentMatchRangeSchema>

/**
 * Content search match.
 */
export const ContentSearchMatchSchema = z.object({
  skillId: z.string(),
  skillName: z.string(),
  file: z.string(),
  folder: z.string(),
  lineNumber: z.number(),
  line: z.string(),
  matchRanges: z.array(ContentMatchRangeSchema).nullable().optional().transform((val) => val ?? []),
})

export type ContentSearchMatch = z.infer<typeof ContentSearchMatchSchema>

/**
 * Content search response.
 */
export const ContentSearchResponseSchema = z.object({
  matches: z.array(ContentSearchMatchSchema).nullable().optional().transform((val) => val ?? []),
  total: z.number(),
  query: z.string(),
})

export type ContentSearchResponse = z.infer<typeof ContentSearchResponseSchema>

// --- Agent AI search ---

/**
 * AI agent search result item.
 */
export const AIAgentSearchResultSchema = z.object({
  id: z.string(),
  displayName: z.string(),
  description: z.string().nullable().optional().transform((val) => val ?? ''),
  status: z.string(),
  tags: nullableStringArray,
  score: z.number(),
  scorePercent: z.number(),
})

export type AIAgentSearchResult = z.infer<typeof AIAgentSearchResultSchema>

/**
 * AI agent search response from the API.
 */
export const AIAgentSearchResponseSchema = z.object({
  results: z.array(AIAgentSearchResultSchema).nullable().optional().transform((val) => val ?? []),
  total: z.number(),
  query: z.string(),
  method: SearchMethodSchema,
})

export type AIAgentSearchResponse = z.infer<typeof AIAgentSearchResponseSchema>

/**
 * AI agent search request parameters.
 */
export const AIAgentSearchRequestSchema = z.object({
  query: z.string(),
  limit: z.number().optional(),
})

export type AIAgentSearchRequest = z.infer<typeof AIAgentSearchRequestSchema>

// --- Team AI search ---

/**
 * AI team search result item.
 */
export const AITeamSearchResultSchema = z.object({
  id: z.string(),
  displayName: z.string(),
  mission: z.string().nullable().optional().transform((val) => val ?? ''),
  enabled: z.boolean(),
  memberCount: z.number(),
  score: z.number(),
  scorePercent: z.number(),
})

export type AITeamSearchResult = z.infer<typeof AITeamSearchResultSchema>

/**
 * AI team search response from the API.
 */
export const AITeamSearchResponseSchema = z.object({
  results: z.array(AITeamSearchResultSchema).nullable().optional().transform((val) => val ?? []),
  total: z.number(),
  query: z.string(),
  method: SearchMethodSchema,
})

export type AITeamSearchResponse = z.infer<typeof AITeamSearchResponseSchema>

/**
 * AI team search request parameters.
 */
export const AITeamSearchRequestSchema = z.object({
  query: z.string(),
  limit: z.number().optional(),
})

export type AITeamSearchRequest = z.infer<typeof AITeamSearchRequestSchema>

// --- Agent content search ---

/**
 * Agent content search match.
 */
export const AgentContentSearchMatchSchema = z.object({
  agentId: z.string(),
  agentName: z.string(),
  file: z.string(),
  lineNumber: z.number(),
  line: z.string(),
  matchRanges: z.array(ContentMatchRangeSchema).nullable().optional().transform((val) => val ?? []),
})

export type AgentContentSearchMatch = z.infer<typeof AgentContentSearchMatchSchema>

/**
 * Agent content search response.
 */
export const AgentContentSearchResponseSchema = z.object({
  matches: z.array(AgentContentSearchMatchSchema).nullable().optional().transform((val) => val ?? []),
  total: z.number(),
  query: z.string(),
})

export type AgentContentSearchResponse = z.infer<typeof AgentContentSearchResponseSchema>

// --- Team content search ---

/**
 * Team content search match.
 */
export const TeamContentSearchMatchSchema = z.object({
  teamId: z.string(),
  teamName: z.string(),
  file: z.string(),
  lineNumber: z.number(),
  line: z.string(),
  matchRanges: z.array(ContentMatchRangeSchema).nullable().optional().transform((val) => val ?? []),
})

export type TeamContentSearchMatch = z.infer<typeof TeamContentSearchMatchSchema>

/**
 * Team content search response.
 */
export const TeamContentSearchResponseSchema = z.object({
  matches: z.array(TeamContentSearchMatchSchema).nullable().optional().transform((val) => val ?? []),
  total: z.number(),
  query: z.string(),
})

export type TeamContentSearchResponse = z.infer<typeof TeamContentSearchResponseSchema>

/**
 * Link preview data from OpenGraph metadata.
 */
export const LinkPreviewDataSchema = z.object({
  title: z.string().optional(),
  description: z.string().optional(),
  image: z.string().optional(),
  favicon: z.string().optional(),
  site_name: z.string().optional(),
})

export type LinkPreviewData = z.infer<typeof LinkPreviewDataSchema>
