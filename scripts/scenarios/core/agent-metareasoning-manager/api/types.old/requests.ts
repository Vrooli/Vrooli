/**
 * Request types for Agent Metareasoning Manager API endpoints
 */

// Prompt management requests
export interface CreatePromptRequest {
  name: string;
  description?: string;
  category: 'decision' | 'analysis' | 'evaluation';
  pattern: string;
  variables: string[];
  metadata?: Record<string, any>;
  tags?: string[];
}

export interface UpdatePromptRequest {
  name?: string;
  description?: string;
  category?: 'decision' | 'analysis' | 'evaluation';
  pattern?: string;
  variables?: string[];
  metadata?: Record<string, any>;
  tags?: string[];
}

export interface ListPromptsQuery {
  category?: string;
  pattern?: string;
  limit?: string;
  offset?: string;
}

// Workflow management requests
export interface ListWorkflowsQuery {
  platform?: 'n8n' | 'windmill';
  pattern?: string;
  limit?: string;
  offset?: string;
}

export interface ExecuteWorkflowRequest {
  input_data: Record<string, any>;
  timeout_seconds?: number;
  webhook_url?: string;
}

// Analysis requests
export interface AnalyzeDecisionRequest {
  input: string;
  context?: Record<string, any>;
  options?: {
    criteria?: string[];
    depth?: 'basic' | 'detailed' | 'comprehensive';
    include_alternatives?: boolean;
  };
}

export interface AnalyzeProsConsRequest {
  input: string;
  context?: Record<string, any>;
  options?: {
    max_items_per_side?: number;
    weight_factors?: string[];
    include_neutral_points?: boolean;
  };
}

export interface AnalyzeSwotRequest {
  input: string;
  context?: Record<string, any>;
  options?: {
    focus_area?: 'internal' | 'external' | 'balanced';
    include_action_plan?: boolean;
    timeframe?: 'short' | 'medium' | 'long';
  };
}

export interface AnalyzeRisksRequest {
  action: string;
  constraints?: string;
  options?: {
    risk_tolerance?: 'low' | 'medium' | 'high';
    focus_areas?: string[];
    include_mitigation_costs?: boolean;
  };
}

// Template management requests
export interface CreateTemplateRequest {
  name: string;
  description?: string;
  pattern: string;
  structure: Record<string, any>;
  example_usage?: string;
  best_practices?: string[];
  limitations?: string[];
  tags?: string[];
  is_public?: boolean;
}

export interface ListTemplatesQuery {
  pattern?: string;
  is_public?: string;
  limit?: string;
  offset?: string;
}

// Authentication requests
export interface CreateTokenRequest {
  name: string;
  permissions?: Record<string, any>;
  expires_at?: string;
}

// Generic query parameters
export interface PaginationQuery {
  limit?: string;
  offset?: string;
}

export interface FilterQuery extends PaginationQuery {
  [key: string]: string | undefined;
}

// Request validation helpers
export const VALID_CATEGORIES = ['decision', 'analysis', 'evaluation'] as const;
export const VALID_PLATFORMS = ['n8n', 'windmill'] as const;
export const VALID_ANALYSIS_TYPES = ['decision', 'pros_cons', 'swot', 'risk_assessment'] as const;
export const VALID_RISK_TOLERANCES = ['low', 'medium', 'high'] as const;
export const VALID_DEPTH_LEVELS = ['basic', 'detailed', 'comprehensive'] as const;