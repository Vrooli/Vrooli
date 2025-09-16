export interface SystemStatus {
  status: 'healthy' | 'degraded' | 'unhealthy'
  health_score: number
  scenarios: number
  vulnerabilities: number
  endpoints: number
  timestamp: string
}

export interface HealthSummary {
  status: string
  system_health_score: number
  health_trend?: 'up' | 'down' | 'stable'
  scenarios: {
    total: number
    active: number
    healthy: number
    degraded: number
    unhealthy: number
  }
  vulnerabilities: {
    total: number
    critical: number
    high: number
    medium: number
    low: number
    trend?: 'up' | 'down' | 'stable'
  }
  endpoints: {
    total: number
    monitored: number
    unmonitored: number
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
  id: string
  scenario_name: string
  scan_type: 'full' | 'quick' | 'targeted'
  status: 'running' | 'completed' | 'failed'
  vulnerabilities: {
    total: number
    critical: number
    high: number
    medium: number
    low: number
  }
  started_at: string
  completed_at?: string
  duration_seconds?: number
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