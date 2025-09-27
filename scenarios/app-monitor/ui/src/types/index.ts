// Application Types
export interface App {
  id: string;
  name: string;
  scenario_name: string;
  path: string;
  created_at: string;
  updated_at: string;
  status: 'running' | 'stopped' | 'error' | 'degraded' | 'healthy' | 'unknown' | 'unhealthy';
  port_mappings: Record<string, number>;
  environment: Record<string, unknown>;
  config: Record<string, unknown>;
  // Legacy fields for compatibility
  port?: string | number;
  uptime?: string;
  description?: string;
  type?: 'scenario' | 'resource';
  runtime?: string;
  tags?: string[];
  health_status?: string;
  is_partial?: boolean;
  view_count?: number;
  first_viewed_at?: string | null;
  last_viewed_at?: string | null;
}

export interface AppViewStats {
  scenario_name: string;
  view_count: number;
  first_viewed_at?: string | null;
  last_viewed_at?: string | null;
}

// Resource Types
export interface Resource {
  id: string;
  name: string;
  type: string;
  status: 'online' | 'offline' | 'error' | 'stopped' | 'unknown' | 'unregistered';
  description?: string;
  icon?: string;
  enabled?: boolean;
  running?: boolean;
  enabled_known?: boolean;
  status_detail?: string;
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
  payload: unknown;
}

// API Response Types
export interface ApiResponse<T = unknown> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

export interface AppLogStream {
  key: string;
  label: string;
  type: 'lifecycle' | 'background';
  phase?: string;
  step?: string;
  command?: string;
  lines: string[];
}

// App Logs Response
export interface AppLogsResponse {
  logs: string[];
  streams?: AppLogStream[];
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
export type ViewType = 'apps' | 'logs' | 'resources' | 'terminal';
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
