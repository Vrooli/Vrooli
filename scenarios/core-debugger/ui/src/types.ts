export type HealthStatus = 'healthy' | 'degraded' | 'critical';

export interface ComponentHealth {
  component: string;
  status: string;
  last_check: string;
  response_time_ms: number;
  error_count?: number;
  details?: Record<string, unknown>;
}

export interface IssueWorkaround {
  id: string;
  issue_id?: string;
  description: string;
  commands?: string[];
  validated?: boolean;
  success_rate?: number;
}

export interface IssueRecord {
  id: string;
  component: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  status?: string;
  description: string;
  error_signature?: string;
  first_seen: string;
  last_seen?: string;
  occurrence_count?: number;
  workarounds?: IssueWorkaround[];
}

export interface HealthResponse {
  status: HealthStatus;
  components: ComponentHealth[];
  active_issues?: number;
  last_check?: string;
}

export interface IssuesResponse {
  issues: IssueRecord[];
}

export interface ScenarioConfig {
  scenario?: string;
  apiUrl: string | null;
  apiPort?: string | null;
  refreshIntervalMs: number;
}
