// Shared type definitions for PRD Control Tower

export type EntityType = 'scenario' | 'resource'
export type DraftStatus = 'draft' | 'published'
export type ViewMode = 'split' | 'edit' | 'preview'

// Runtime constants for type checking and comparisons
export const EntityTypes = {
  SCENARIO: 'scenario' as const,
  RESOURCE: 'resource' as const,
}

export const DraftStatuses = {
  DRAFT: 'draft' as const,
  PUBLISHED: 'published' as const,
}

export const ViewModes = {
  SPLIT: 'split' as const,
  EDIT: 'edit' as const,
  PREVIEW: 'preview' as const,
}

export interface Draft {
  id: string
  entity_type: EntityType
  entity_name: string
  content: string
  owner?: string
  created_at: string
  updated_at: string
  status: DraftStatus
}

export interface DraftListResponse {
  drafts: Draft[]
  total: number
}

export interface CreateDraftRequest {
  entity_type: EntityType
  entity_name: string
  content: string
  owner?: string
}

export interface UpdateDraftRequest {
  content: string
}

export interface CatalogEntry {
  type: EntityType
  name: string
  has_prd: boolean
  prd_path: string
  has_draft: boolean
  description: string
}

export interface CatalogResponse {
  entries: CatalogEntry[]
  total: number
}

export interface PublishedPRDResponse {
  type: EntityType
  name: string
  content: string
  path: string
  content_html?: string
}

export interface ValidationRequest {
  use_cache?: boolean
}

/**
 * Represents a single validation violation from the auditor.
 */
export interface Violation {
  rule?: string
  severity?: 'error' | 'warning' | 'info'
  message?: string
  line?: number
  description?: string
}

/**
 * Error response when auditor is unavailable or encounters issues.
 */
export interface ValidationError {
  error: string
  message: string
  stderr?: string
}

/**
 * Error response when auditor output cannot be parsed as JSON.
 */
export interface ValidationParseError {
  output: string
  parse_error: string
}

/**
 * Validation result can be:
 * - An array of violations
 * - An object with categorized violations (by severity or rule)
 * - An error response when auditor is unavailable
 * - Raw text output when JSON parsing fails
 */
export type ValidationResult =
  | Violation[]
  | Record<string, Violation[]>
  | ValidationError
  | ValidationParseError

export interface ValidationResponse {
  draft_id: string
  entity_type: EntityType
  entity_name: string
  violations: ValidationResult
  cached_at?: string
  validated_at: string
  cache_used: boolean
}

export interface AIGenerateRequest {
  section: string
  context?: string
}

export interface AIGenerateResponse {
  draft_id: string
  section: string
  generated_text: string
  model: string
  success: boolean
  message?: string
}

export interface PublishRequest {
  create_backup?: boolean
}

export interface PublishResponse {
  success: boolean
  message: string
  published_to: string
  backup_path?: string
  published_at: string
}
