/**
 * Core types for Agent Metareasoning Manager API
 */

// Database model types
export interface ReasoningPrompt {
  id: string;
  name: string;
  description?: string;
  category: 'decision' | 'analysis' | 'evaluation';
  pattern: string;
  variables: string[];
  metadata?: Record<string, any>;
  tags?: string[];
  usage_count: number;
  average_rating?: number;
  is_active: boolean;
  created_at: Date;
  updated_at: Date;
}

export interface Workflow {
  id: string;
  name: string;
  description?: string;
  platform: 'n8n' | 'windmill';
  pattern: string;
  input_schema?: Record<string, any>;
  output_schema?: Record<string, any>;
  dependencies?: string[];
  tags?: string[];
  execution_count: number;
  average_duration_ms?: number;
  is_active: boolean;
  created_at: Date;
  updated_at?: Date;
}

export interface AnalysisExecution {
  id: string;
  type: 'decision' | 'pros_cons' | 'swot' | 'risk_assessment';
  input_data: Record<string, any>;
  output_data?: Record<string, any>;
  status: 'pending' | 'running' | 'completed' | 'failed';
  execution_time_ms?: number;
  api_token_id?: string;
  created_at: Date;
  completed_at?: Date;
}

export interface Template {
  id: string;
  name: string;
  description?: string;
  pattern: string;
  structure: Record<string, any>;
  example_usage?: string;
  best_practices?: string[];
  limitations?: string[];
  tags?: string[];
  is_public: boolean;
  usage_count: number;
  created_at: Date;
  updated_at?: Date;
}

export interface ApiToken {
  id: string;
  name: string;
  token_hash: string;
  permissions?: Record<string, any>;
  expires_at?: Date;
  last_used_at?: Date;
  is_active: boolean;
  created_at: Date;
}

export interface ApiUsageStat {
  id: string;
  endpoint: string;
  method: string;
  response_code: number;
  execution_time_ms: number;
  api_token_id?: string;
  created_at: Date;
}

// Analysis result types
export interface DecisionAnalysis {
  input: string;
  context: string;
  factors_considered: string[];
  recommendation: string;
  confidence: number;
  reasoning: string;
}

export interface ProsCons {
  item: string;
  weight: number;
  explanation: string;
}

export interface ProsConsAnalysis {
  input: string;
  context: string;
  pros: ProsCons[];
  cons: ProsCons[];
  net_score: number;
  recommendation: string;
  confidence: number;
}

export interface SwotAnalysis {
  input: string;
  context: string;
  strengths: string[];
  weaknesses: string[];
  opportunities: string[];
  threats: string[];
  strategic_position: string;
  recommendations: string[];
}

export interface Risk {
  risk: string;
  category: string;
  probability: 'Low' | 'Medium' | 'High';
  impact: 'Low' | 'Medium' | 'High';
  risk_score: number;
  mitigation: string;
  early_warning: string;
}

export interface RiskAssessment {
  action: string;
  constraints?: string;
  risks: Risk[];
  overall_risk_posture: string;
  top_priorities: string[];
  mitigation_summary: string;
}

// Configuration types
export interface ServerConfig {
  port: number;
  host: string;
  cors: {
    origin: string[];
    credentials: boolean;
  };
  database: {
    host: string;
    port: number;
    database: string;
    user: string;
    password: string;
    max: number;
    idleTimeoutMillis: number;
  };
  redis: {
    url: string;
    keyPrefix: string;
    retryDelayOnFailover: number;
    socket: {
      connectTimeout: number;
      lazyConnect: boolean;
    };
  };
  n8n: {
    baseUrl: string;
    webhookBase: string;
  };
  windmill: {
    baseUrl: string;
    workspace: string;
  };
  rateLimiting: {
    windowMs: number;
    max: number;
    message: string;
  };
}

// Health check types
export interface ServiceHealth {
  database: 'connected' | 'disconnected';
  redis: 'connected' | 'disconnected';
  n8n: 'connected' | 'disconnected';
  windmill: 'connected' | 'disconnected';
  [key: string]: string;
}

export type HealthStatus = 'healthy' | 'degraded' | 'unhealthy';