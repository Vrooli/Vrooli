export interface SystemStatus {
  status: 'healthy' | 'degraded' | 'unhealthy'
  health_score: number
  scenarios: number
  vulnerabilities: number
  standards_violations?: number
  endpoints: number
  timestamp: string
}

export interface HealthSummary {
  status: string
  system_health_score: number | null  // null when no scans have been performed
  health_trend?: 'up' | 'down' | 'stable'
  scenarios: number // Updated to match API response
  scenarios_detail?: {
    total: number
    available: number
    running: number
    healthy: number
    critical: number
  }
  vulnerabilities: number // Updated to match API response
  vulnerabilities_detail?: {
    total: number
    critical: number
    high: number
  }
  endpoints?: {
    total: number
    monitored: number
    unmonitored: number
  }
  scan_status?: {
    has_scans: boolean
    total_scans: number
    last_scan: string | null
    message: string
  }
  timestamp: string
}

export interface Scenario {
  name: string
  status: 'healthy' | 'degraded' | 'unhealthy' | 'unknown'
  description: string
  endpoint_count: number
  vulnerability_count: number
  critical_count: number
  last_scan: string | null
  created_at: string
  updated_at: string
  metadata?: Record<string, any>
}

export interface Vulnerability {
  id: string
  scenario_name: string
  type: string
  severity: 'critical' | 'high' | 'medium' | 'low'
  title: string
  description: string
  file_path: string
  line_number: number
  code_snippet?: string
  recommendation: string
  status: 'open' | 'fixed' | 'ignored'
  discovered_at: string
  fixed_at?: string
}

export interface SecurityScan {
  id?: string
  scan_id?: string
  scenario_name?: string
  scenarios_scanned?: number | string
  scan_type: 'full' | 'quick' | 'targeted'
  status: 'running' | 'completed' | 'failed'
  vulnerabilities: {
    total: number
    critical: number
    high: number
    medium: number
    low: number
  }
  files_scanned?: {
    total: number
    code_files: number
    config_files: number
  }
  started_at: string
  completed_at?: string
  duration_seconds?: number
  message?: string
  scan_notes?: string
}

export interface SecurityScanStatus {
  id: string
  scenario: string
  scan_type: 'quick' | 'full' | 'targeted'
  status: 'running' | 'cancelling' | 'completed' | 'failed' | 'cancelled'
  started_at: string
  completed_at?: string | null
  elapsed_seconds: number
  total_scenarios: number
  processed_scenarios: number
  processed_files: number
  total_files: number
  current_scenario?: string
  current_scanner?: string
  message?: string
  error?: string
  result?: SecurityScan
}

export interface StandardsViolation {
  id: string
  scenario_name: string
  type: string
  severity: 'critical' | 'high' | 'medium' | 'low'
  title: string
  description: string
  file_path: string
  line_number: number
  code_snippet?: string
  recommendation: string
  standard: string
  discovered_at: string
}

export interface StandardsCheckResult {
  check_id: string
  status: 'completed' | 'failed' | 'cancelled' | 'running'
  scan_type: 'full' | 'quick' | 'targeted'
  started_at: string
  completed_at?: string
  duration_seconds: number
  files_scanned: number
  violations: StandardsViolation[]
  statistics: Record<string, number>
  message?: string
  scenario_name?: string
}

export interface StandardsScanStatus {
  id: string
  scenario: string
  scan_type: string
  status: 'running' | 'cancelling' | 'completed' | 'failed' | 'cancelled'
  started_at: string
  completed_at?: string | null
  elapsed_seconds: number
  total_scenarios: number
  processed_scenarios: number
  processed_files: number
  total_files: number
  current_scenario?: string
  current_file?: string
  message?: string
  error?: string
  result?: StandardsCheckResult
}

export interface HealthAlert {
  id: string
  level: 'critical' | 'warning' | 'info'
  category: string
  title: string
  description: string
  scenario?: string
  created_at: string
  resolved_at?: string
}

export interface PerformanceMetric {
  scenario_name: string
  type: string
  value: number
  unit: string
  measured_at: string
  metadata?: Record<string, any>
}

export interface AutomatedFixConfig {
  enabled: boolean
  allowed_categories: string[]
  max_confidence: 'low' | 'medium' | 'high'
  require_approval: boolean
  backup_enabled: boolean
  rollback_window: number
  safety_status: string
}

export interface AutomatedFix {
  id: string
  vulnerability_id: string
  scenario_name: string
  category: string
  confidence: 'low' | 'medium' | 'high'
  status: 'pending' | 'approved' | 'applied' | 'rolled_back'
  description: string
  changes: {
    file_path: string
    before: string
    after: string
  }[]
  applied_at?: string
  rolled_back_at?: string
}

export interface BreakingChange {
  id: string
  scenario_name: string
  type: 'breaking' | 'minor'
  category: string
  description: string
  impact: string
  recommendation: string
  detected_at: string
}

export interface ApiResponse<T> {
  data?: T
  error?: string
  message?: string
  timestamp: string
}
export interface AgentInfo {
  id: string
  name: string
  label: string
  action: string
  rule_id?: string
  scenario?: string
  model: string
  status: string
  started_at: string
  ended_at?: string
  duration_seconds: number
  command: string[]
  prompt_length: number
  pid?: number
  metadata?: Record<string, string>
  issue_ids?: string[]
  error?: string
}

export interface RuleTestStatus {
  total: number
  passed: number
  failed: number
  warning?: string
  error?: string
  has_issues: boolean
}

export interface RuleImplementationStatus {
  valid: boolean
  error?: string
  details?: string
}

export interface FixAgentResponse {
  success: boolean
  message: string
  fix_id: string
  started_at: string
  agent?: AgentInfo
  scenarios?: string[]
  user_instructions?: string
  error?: string
  fix_ids?: string[]
  agents?: AgentInfo[]
  agent_count?: number
  issue_count?: number
  issues_per_agent_cap?: number
}
