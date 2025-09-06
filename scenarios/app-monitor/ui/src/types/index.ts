// Application Types
export interface App {
  id: string;
  name: string;
  scenario_name: string;
  path: string;
  created_at: string;
  updated_at: string;
  status: 'running' | 'stopped' | 'error' | 'degraded' | 'healthy';
  port_mappings: Record<string, number>;
  environment: Record<string, any>;
  config: Record<string, any>;
  // Legacy fields for compatibility
  port?: string | number;
  uptime?: string;
  cpu?: number;
  memory?: number;
  description?: string;
  type?: 'scenario' | 'resource';
  lastActivity?: string;
}

// Resource Types
export interface Resource {
  id: string;
  name: string;
  type: string;
  status: 'online' | 'offline' | 'error';
  description?: string;
  icon?: string;
}

// System Metrics
export interface SystemMetrics {
  cpu: number;
  memory: number;
  disk: number;
  network: number;
  timestamp?: string;
}

// Log Entry
export interface LogEntry {
  timestamp: string;
  level: 'info' | 'warning' | 'error' | 'debug' | 'success' | 'log';
  message: string;
  app?: string;
  type?: 'lifecycle' | 'background';
}

// WebSocket Message Types
export interface WSMessage {
  type: 'connection' | 'app_update' | 'metric_update' | 'log_entry' | 'command_response' | 'error';
  payload: any;
}

// API Response Types
export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

// App Logs Response
export interface AppLogsResponse {
  logs: string[];
  hasMore?: boolean;
  error?: string;
}

// Terminal Command
export interface TerminalCommand {
  command: string;
  timestamp: string;
  output?: string;
  error?: string;
}

// View Types
export type ViewType = 'apps' | 'metrics' | 'logs' | 'resources' | 'terminal';
export type AppViewMode = 'grid' | 'list';

// Filter/Sort Options
export interface FilterOptions {
  search?: string;
  status?: string[];
  type?: string[];
}

export interface SortOptions {
  field: 'name' | 'status' | 'port' | 'cpu' | 'memory';
  direction: 'asc' | 'desc';
}