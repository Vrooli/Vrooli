/**
 * Database-specific types for Agent Metareasoning Manager
 */

import { Pool } from 'pg';
import { RedisClientType } from 'redis';

// Database connection types
export type DatabasePool = Pool;
export type RedisClient = RedisClientType;

// Database query result types
export interface QueryResult<T = any> {
  rows: T[];
  rowCount: number;
  command: string;
}

// Database row types (raw from database)
export interface PromptsRow {
  id: string;
  name: string;
  description: string | null;
  category: string;
  pattern: string;
  variables: string[];
  metadata: any | null;
  tags: string[] | null;
  usage_count: number;
  average_rating: number | null;
  is_active: boolean;
  created_at: Date;
  updated_at: Date;
}

export interface WorkflowsRow {
  id: string;
  name: string;
  description: string | null;
  platform: string;
  pattern: string;
  input_schema: any | null;
  output_schema: any | null;
  dependencies: string[] | null;
  tags: string[] | null;
  execution_count: number;
  average_duration_ms: number | null;
  is_active: boolean;
  created_at: Date;
  updated_at: Date | null;
}

export interface AnalysisExecutionsRow {
  id: string;
  type: string;
  input_data: any;
  output_data: any | null;
  status: string;
  execution_time_ms: number | null;
  api_token_id: string | null;
  created_at: Date;
  completed_at: Date | null;
}

export interface TemplatesRow {
  id: string;
  name: string;
  description: string | null;
  pattern: string;
  structure: any;
  example_usage: string | null;
  best_practices: string[] | null;
  limitations: string[] | null;
  tags: string[] | null;
  is_public: boolean;
  usage_count: number;
  created_at: Date;
  updated_at: Date | null;
}

export interface ApiTokensRow {
  id: string;
  name: string;
  token_hash: string;
  permissions: any | null;
  expires_at: Date | null;
  last_used_at: Date | null;
  is_active: boolean;
  created_at: Date;
}

export interface ApiUsageStatsRow {
  id: string;
  endpoint: string;
  method: string;
  response_code: number;
  execution_time_ms: number;
  api_token_id: string | null;
  created_at: Date;
}

// Database query parameter types
export type QueryParams = (string | number | boolean | Date | any[] | null)[];

// Database transaction types
export interface DatabaseTransaction {
  query<T = any>(text: string, params?: QueryParams): Promise<QueryResult<T>>;
  release(): void;
}

// Common database operations
export interface DatabaseOperations {
  // Prompts
  createPrompt(data: Partial<PromptsRow>): Promise<PromptsRow>;
  getPrompt(id: string): Promise<PromptsRow | null>;
  listPrompts(filters: {
    category?: string;
    pattern?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ prompts: PromptsRow[]; total: number }>;
  updatePrompt(id: string, data: Partial<PromptsRow>): Promise<PromptsRow>;
  deletePrompt(id: string): Promise<boolean>;

  // Workflows
  getWorkflow(id: string): Promise<WorkflowsRow | null>;
  listWorkflows(filters: {
    platform?: string;
    pattern?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ workflows: WorkflowsRow[]; total: number }>;

  // Analysis executions
  createAnalysisExecution(data: Partial<AnalysisExecutionsRow>): Promise<AnalysisExecutionsRow>;
  getAnalysisExecution(id: string): Promise<AnalysisExecutionsRow | null>;
  updateAnalysisExecution(id: string, data: Partial<AnalysisExecutionsRow>): Promise<AnalysisExecutionsRow>;

  // Templates
  listTemplates(filters: {
    pattern?: string;
    is_public?: boolean;
    limit?: number;
    offset?: number;
  }): Promise<{ templates: TemplatesRow[]; total: number }>;

  // API tokens
  validateApiToken(tokenHash: string): Promise<ApiTokensRow | null>;
  updateTokenUsage(tokenId: string): Promise<void>;

  // Usage stats
  logApiUsage(data: Partial<ApiUsageStatsRow>): Promise<void>;
}

// Redis operations
export interface RedisOperations {
  // Cache operations
  get<T = any>(key: string): Promise<T | null>;
  set(key: string, value: any, ttlSeconds?: number): Promise<void>;
  del(key: string): Promise<boolean>;
  exists(key: string): Promise<boolean>;

  // Session management
  setSession(sessionId: string, data: any, ttlSeconds?: number): Promise<void>;
  getSession<T = any>(sessionId: string): Promise<T | null>;
  deleteSession(sessionId: string): Promise<boolean>;

  // Rate limiting
  incrementRateLimit(key: string, windowMs: number, limit: number): Promise<{
    current: number;
    remaining: number;
    resetTime: Date;
  }>;
}

// Database error types
export class DatabaseError extends Error {
  constructor(
    message: string,
    public code?: string,
    public constraint?: string,
    public detail?: string
  ) {
    super(message);
    this.name = 'DatabaseError';
  }
}

export class ValidationError extends Error {
  constructor(
    message: string,
    public field: string,
    public code: string
  ) {
    super(message);
    this.name = 'ValidationError';
  }
}