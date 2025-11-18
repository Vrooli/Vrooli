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
  source_backlog_id?: string  // Optional link to originating backlog entry
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

export interface DraftSaveStatus {
  type: 'success' | 'error'
  message: string
}

export interface CatalogEntry {
  type: EntityType
  name: string
  has_prd: boolean
  prd_path: string
  has_draft: boolean
  has_requirements: boolean
  description: string
  requirements_summary?: CatalogRequirementSummary
}

export interface CatalogResponse {
  entries: CatalogEntry[]
  total: number
}

export interface CatalogRequirementSummary {
  total: number
  completed: number
  in_progress: number
  pending: number
  p0: number
  p1: number
  p2: number
}

export interface PublishedPRDResponse {
  type: EntityType
  name: string
  content: string
  path: string
  content_html?: string
}

export interface PublishResponse {
  success: boolean
  message: string
  published_to: string
  backup_path?: string
  published_at: string
  draft_removed?: boolean
  created_scenario?: boolean
  scenario_id?: string
  scenario_type?: string
  scenario_path?: string
}

export interface ScenarioTemplateVar {
  name: string
  flag?: string
  description: string
  default?: string
  required: boolean
}

export interface ScenarioTemplate {
  name: string
  display_name: string
  description: string
  stack: string[]
  required_vars: ScenarioTemplateVar[]
  optional_vars: ScenarioTemplateVar[]
}

export interface ScenarioTemplateListResponse {
  templates: ScenarioTemplate[]
}

export type IssueSeverity = 'info' | 'low' | 'medium' | 'high' | 'critical' | 'p0' | 'p1' | 'p2'

export interface TrackerIssueSummary {
  id: string
  title: string
  status: string
  priority?: string
  created_at?: string
  updated_at?: string
  reporter?: string
  issue_url?: string
  local_issue_url?: string
}

export interface ScenarioIssuesSummary {
  entity_type: EntityType
  entity_name: string
  issues: TrackerIssueSummary[]
  open_count: number
  active_count: number
  total_count: number
  tracker_url?: string
  local_tracker_url?: string
  last_fetched: string
  from_cache: boolean
  stale: boolean
}

export interface IssueReportSelectionInput {
  id: string
  title: string
  detail: string
  category: string
  severity: IssueSeverity
  reference?: string
  notes?: string
}

export interface IssueReportAttachmentInput {
  name: string
  content: string
  content_type?: string
  encoding?: string
  category?: string
  description?: string
}

export interface ScenarioIssueReportRequest {
  entity_type: EntityType
  entity_name: string
  source: string
  title: string
  description: string
  priority?: string
  summary?: string
  tags?: string[]
  labels?: Record<string, string>
  metadata?: Record<string, string>
  selections: IssueReportSelectionInput[]
  attachments?: IssueReportAttachmentInput[]
}

export interface ScenarioIssueReportResponse {
  issue_id: string
  issue_url?: string
  message: string
}

export interface BulkIssueReportResult {
  request: ScenarioIssueReportRequest
  response?: ScenarioIssueReportResponse
  error?: string
}

export interface IssueReportCategorySeed {
  id: string
  title: string
  description?: string
  severity?: IssueSeverity
  defaultSelected?: boolean
  items: IssueReportSelectionInput[]
}

export interface IssueReportSeed {
  entity_type: EntityType
  entity_name: string
  source: string
  title: string
  description: string
  summary?: string
  display_name?: string
  tags?: string[]
  labels?: Record<string, string>
  metadata?: Record<string, string>
  attachments?: IssueReportAttachmentInput[]
  categories: IssueReportCategorySeed[]
}

export interface ScenarioExistenceResponse {
  exists: boolean
  path?: string
  last_modified?: string
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

export interface ValidationSummary {
  total_violations: number
  errors: number
  warnings: number
  info: number
}

export interface DraftValidationResult {
  violations: Violation[]
  template_compliance?: PRDTemplateValidationResult
  template_compliance_v2?: PRDValidationResultV2
  target_linkage_issues?: TargetLinkageIssue[]
  summary?: ValidationSummary
}

/**
 * PRD Template validation violation
 */
export interface PRDTemplateViolation {
  section: string
  level: number
  message: string
  severity: 'error' | 'warning'
  suggestion?: string
  line_number?: number
}

/**
 * PRD Template validation result
 */
export interface PRDTemplateValidationResult {
  compliant_sections: string[]
  missing_sections: string[]
  violations: PRDTemplateViolation[]
  compliance_percent: number
  is_compliant: boolean
}

export interface PRDContentIssue {
  section: string
  issue_type: string
  message: string
  severity: string
  line_number?: number
  suggestion?: string
}

export interface PRDValidationResultV2 {
  compliant_sections: string[]
  missing_sections: string[]
  violations: PRDTemplateViolation[]
  content_issues: PRDContentIssue[]
  structure_score: number
  content_score: number
  overall_score: number
  is_fully_compliant: boolean
  missing_subsections: Record<string, string[]>
  required_sections: number
  completed_sections: number
  unexpected_sections: string[]
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

export interface EcosystemTaskSummary {
  id: string
  title: string
  status: string
  priority: string
  category: string
  operation: string
  target: string
  view_url?: string
  created_at?: string
  updated_at?: string
}

export interface EcosystemTaskStatusResponse {
  configured: boolean
  supported: boolean
  error?: string
  manage_url?: string
  task?: EcosystemTaskSummary
}

export interface TargetLinkageIssue {
  title: string
  criticality: string
  message: string
}

export interface QualityIssueCounts {
  missing_prd: number
  missing_template_sections: number
  target_coverage: number
  requirement_coverage: number
  prd_ref: number
  total: number
  blocking: number
}

export interface RequirementSummaryIssue {
  id: string
  title: string
  prd_ref: string
  criticality: string
  status: string
  category: string
  file_path?: string
  issue: string
}

export interface ScenarioQualityReport {
  entity_type: EntityType
  entity_name: string
  has_prd: boolean
  has_requirements: boolean
  prd_path?: string
  requirements_path?: string
  validated_at: string
  cache_used: boolean
  status: 'healthy' | 'needs_attention' | 'blocked' | 'missing_prd' | 'error'
  message?: string
  error?: string
  issue_counts: QualityIssueCounts
  template_compliance?: PRDTemplateValidationResult
  template_compliance_v2?: PRDValidationResultV2
  target_linkage_issues?: TargetLinkageIssue[]
  requirements_without_targets?: RequirementSummaryIssue[]
  prd_ref_issues?: PRDValidationIssue[]
  requirement_count: number
  target_count: number
}

export interface QualityScanEntity {
  type: EntityType
  name: string
}

export interface QualityScanResponse {
  reports: ScenarioQualityReport[]
  generated_at: string
  duration_ms: number
}

export interface QualitySummaryEntity {
  entity_type: EntityType
  entity_name: string
  status: ScenarioQualityReport['status']
  issue_counts: QualityIssueCounts
}

export interface QualitySummary {
  total_entities: number
  scanned: number
  with_issues: number
  missing_prd: number
  last_generated: string
  duration_ms: number
  cache_used: boolean
  samples?: QualitySummaryEntity[]
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

export type BacklogStatus = 'pending' | 'converted' | 'archived'

export interface BacklogEntry {
  id: string
  idea_text: string
  entity_type: EntityType
  suggested_name: string
  notes?: string
  status: BacklogStatus
  created_at: string
  updated_at: string
  converted_draft_id?: string
}

export interface BacklogListResponse {
  entries: BacklogEntry[]
  total: number
}

export interface BacklogCreateItem {
  idea_text: string
  entity_type?: EntityType
  suggested_name?: string
  notes?: string
}

export interface BacklogCreateRequest {
  raw_input?: string
  entity_type?: EntityType
  entries?: BacklogCreateItem[]
}

export interface BacklogCreateResponse {
  entries: BacklogEntry[]
}

export interface BacklogConvertRequest {
  entry_ids: string[]
}

export interface BacklogConvertResult {
  entry: BacklogEntry
  draft?: Draft
  error?: string
}

export interface RequirementValidation {
  type: string
  ref: string
  phase: string
  status: string
  notes?: string
}

export interface TestFileReference {
  file_path: string
  requirement_id: string
  lines: number[]
  test_names: string[]
}

export interface PRDValidationIssue {
  requirement_id: string
  prd_ref: string
  issue_type: 'missing_section' | 'ambiguous_match' | 'no_checkbox' | 'invalid_format'
  message: string
  suggestions?: string[]
}

export interface RequirementRecord {
  id: string
  category?: string
  prd_ref?: string
  title: string
  description?: string
  status?: string
  criticality?: string
  file_path?: string
  validation?: RequirementValidation[]
  linked_operational_target_ids?: string[]
  test_files?: TestFileReference[]
  prd_ref_issue?: PRDValidationIssue
}

export interface RequirementGroup {
  id: string
  name: string
  description?: string
  file_path?: string
  requirements: RequirementRecord[]
  children?: RequirementGroup[]
}

export interface RequirementsResponse {
  entity_type: EntityType
  entity_name: string
  updated_at: string
  groups: RequirementGroup[]
}

export interface OperationalTarget {
  id: string
  entity_type: EntityType
  entity_name: string
  category: string
  criticality?: string
  title: string
  notes?: string
  status: 'complete' | 'pending'
  path: string
  linked_requirement_ids?: string[]
}

export interface OperationalTargetsResponse {
  entity_type: EntityType
  entity_name: string
  targets: OperationalTarget[]
  unmatched_requirements: RequirementRecord[]
}

export interface BacklogConvertResponse {
  results: BacklogConvertResult[]
}
