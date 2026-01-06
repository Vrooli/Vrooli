export type InvestigationStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

export interface Investigation {
  id: string;
  deployment_id: string;
  deployment_run_id?: string;
  status: InvestigationStatus;
  findings?: string;
  progress: number;
  details?: InvestigationDetails;
  agent_run_id?: string;
  error_message?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface InvestigationDetails {
  source: string;
  run_id?: string;
  duration_seconds?: number;
  tokens_used?: number;
  cost_estimate?: number;
  operation_mode: string;
  trigger_reason: string;
  deployment_step?: string;
  source_investigation_id?: string;
  source_findings?: string;
}

export interface InvestigationSummary {
  id: string;
  deployment_id: string;
  deployment_run_id?: string;
  status: InvestigationStatus;
  progress: number;
  has_findings: boolean;
  error_message?: string;
  source_investigation_id?: string;
  created_at: string;
  completed_at?: string;
}

export interface CreateInvestigationRequest {
  auto_fix?: boolean;
  note?: string;
  include_contexts?: string[];
}

export interface ApplyFixesRequest {
  immediate: boolean;
  permanent: boolean;
  prevention: boolean;
  note?: string;
}

export interface AgentManagerStatus {
  enabled: boolean;
  available: boolean;
  message?: string;
}
