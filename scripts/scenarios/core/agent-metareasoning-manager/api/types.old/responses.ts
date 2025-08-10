/**
 * Response types for Agent Metareasoning Manager API endpoints
 */

import {
  ReasoningPrompt,
  Workflow,
  Template,
  AnalysisExecution,
  DecisionAnalysis,
  ProsConsAnalysis,
  SwotAnalysis,
  RiskAssessment,
  ServiceHealth,
  HealthStatus
} from './index.js';

// Generic API response wrapper
export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
  requestId?: string;
  timestamp: string;
}

// Health endpoints
export interface HealthResponse {
  status: HealthStatus;
  timestamp: string;
  version: string;
  services: ServiceHealth;
}

export interface InfoResponse {
  name: string;
  version: string;
  description: string;
  endpoints: Record<string, string>;
  timestamp: string;
}

// Prompt management responses
export interface ListPromptsResponse {
  prompts: ReasoningPrompt[];
  count: number;
  total: number;
  offset: number;
  limit: number;
}

export interface GetPromptResponse {
  prompt: ReasoningPrompt;
}

export interface CreatePromptResponse {
  prompt: ReasoningPrompt;
  created: true;
}

// Workflow management responses
export interface ListWorkflowsResponse {
  workflows: Workflow[];
  count: number;
  total: number;
  offset: number;
  limit: number;
}

export interface GetWorkflowResponse {
  workflow: Workflow;
}

export interface ExecuteWorkflowResponse {
  execution_id: string;
  workflow_name: string;
  status: 'triggered' | 'running' | 'completed' | 'failed';
  webhook_url?: string;
  estimated_duration_ms?: number;
}

// Analysis responses
export interface AnalyzeDecisionResponse {
  decision_analysis: DecisionAnalysis;
  execution_id: string;
  status: 'completed' | 'failed';
  analysis_type: 'decision';
}

export interface AnalyzeProsConsResponse {
  pros_cons_analysis: ProsConsAnalysis;
  execution_id: string;
  status: 'completed' | 'failed';
  analysis_type: 'pros_cons';
}

export interface AnalyzeSwotResponse {
  swot_analysis: SwotAnalysis;
  execution_id: string;
  status: 'completed' | 'failed';
  analysis_type: 'swot';
}

export interface AnalyzeRisksResponse {
  risk_assessment: RiskAssessment;
  execution_id: string;
  status: 'completed' | 'failed';
  analysis_type: 'risk_assessment';
}

// Template management responses
export interface ListTemplatesResponse {
  templates: Template[];
  count: number;
  total: number;
  offset: number;
  limit: number;
}

export interface GetTemplateResponse {
  template: Template;
}

export interface CreateTemplateResponse {
  template: Template;
  created: true;
}

// Execution status responses
export interface GetExecutionStatusResponse {
  execution: AnalysisExecution;
  progress?: {
    current_step?: string;
    total_steps?: number;
    completed_steps?: number;
    estimated_remaining_ms?: number;
  };
}

// Statistics and usage responses
export interface UsageStatsResponse {
  period: {
    start: string;
    end: string;
  };
  total_requests: number;
  requests_by_endpoint: Record<string, number>;
  average_response_time_ms: number;
  error_rate: number;
  top_analysis_types: Array<{
    type: string;
    count: number;
    success_rate: number;
  }>;
}

// Error response types
export interface ValidationError {
  field: string;
  message: string;
  code: string;
}

export interface ErrorResponse {
  error: string;
  code?: string;
  details?: string;
  validation_errors?: ValidationError[];
  requestId?: string | undefined;
  timestamp: string;
}

// Authentication responses
export interface TokenResponse {
  token: string;
  token_id: string;
  expires_at?: string;
  permissions?: Record<string, any>;
}

// Pagination helpers
export interface PaginatedResponse<T> {
  data: T[];
  count: number;
  total: number;
  offset: number;
  limit: number;
  has_more: boolean;
}

// Status codes mapping
export const HTTP_STATUS = {
  OK: 200,
  CREATED: 201,
  BAD_REQUEST: 400,
  UNAUTHORIZED: 401,
  FORBIDDEN: 403,
  NOT_FOUND: 404,
  CONFLICT: 409,
  UNPROCESSABLE_ENTITY: 422,
  INTERNAL_SERVER_ERROR: 500,
  SERVICE_UNAVAILABLE: 503
} as const;