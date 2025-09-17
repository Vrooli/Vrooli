import {
  SystemStatus,
  HealthSummary,
  Scenario,
  Vulnerability,
  SecurityScan,
  HealthAlert,
  PerformanceMetric,
  AutomatedFixConfig,
  AutomatedFix,
  BreakingChange,
  ApiResponse
} from '@/types/api'

const API_BASE = '/api/v1'

class ApiService {
  private async fetch<T>(url: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${API_BASE}${url}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Request failed' }))
      throw new Error(error.message || `HTTP ${response.status}`)
    }

    return response.json()
  }

  // System Status
  async getSystemStatus(): Promise<SystemStatus> {
    const summary = await this.getHealthSummary()
    return {
      status: summary.status as any,
      health_score: summary.system_health_score || 0,
      scenarios: (typeof summary.scenarios === 'object' ? summary.scenarios?.total : summary.scenarios) || 0,
      vulnerabilities: (typeof summary.vulnerabilities === 'object' ? summary.vulnerabilities?.total : summary.vulnerabilities) || 0,
      endpoints: 0, // This field doesn't exist in current API response
      timestamp: summary.timestamp || new Date().toISOString(),
    }
  }

  // Health endpoints
  async getHealthSummary(): Promise<HealthSummary> {
    return this.fetch<HealthSummary>('/health/summary')
  }

  async getScenarioHealth(name: string): Promise<any> {
    return this.fetch(`/scenarios/${name}/health`)
  }

  async getHealthAlerts(): Promise<{ alerts: HealthAlert[], count: number }> {
    return this.fetch('/health/alerts')
  }

  async getHealthMetrics(scenario: string): Promise<any> {
    return this.fetch(`/health/metrics/${scenario}`)
  }

  // Scenarios
  async getScenarios(): Promise<Scenario[]> {
    const response = await this.fetch<{ scenarios: Scenario[] }>('/scenarios')
    return response.scenarios || []
  }

  async getScenario(name: string): Promise<Scenario> {
    const scenarios = await this.getScenarios()
    const scenario = scenarios.find(s => s.name === name)
    if (!scenario) throw new Error('Scenario not found')
    return scenario
  }

  async discoverScenarios(): Promise<{ discovered: number, scenarios: Scenario[] }> {
    return this.fetch('/scenarios/discover', { method: 'POST' })
  }

  // Vulnerabilities
  async scanScenario(name: string, options?: { type?: string }): Promise<SecurityScan> {
    return this.fetch(`/scenarios/${name}/scan`, {
      method: 'POST',
      body: JSON.stringify(options || {}),
    })
  }

  async getVulnerabilities(scenario?: string): Promise<Vulnerability[]> {
    const url = scenario ? `/vulnerabilities?scenario=${scenario}` : '/vulnerabilities'
    const response = await this.fetch<{ vulnerabilities: Vulnerability[] }>(url)
    return response.vulnerabilities || []
  }

  async getRecentScans(): Promise<SecurityScan[]> {
    const response = await this.fetch<{ scans: SecurityScan[] }>('/scans/recent')
    return response.scans || []
  }

  // Performance
  async createPerformanceBaseline(scenario: string, config: any): Promise<any> {
    return this.fetch(`/performance/baseline/${scenario}`, {
      method: 'POST',
      body: JSON.stringify(config),
    })
  }

  async getPerformanceMetrics(scenario: string): Promise<{
    scenario: string
    metrics: PerformanceMetric[]
    count: number
  }> {
    return this.fetch(`/performance/metrics/${scenario}`)
  }

  async getPerformanceAlerts(): Promise<{ alerts: any[], count: number }> {
    return this.fetch('/performance/alerts')
  }

  // Breaking Changes
  async detectChanges(scenario: string, options?: any): Promise<any> {
    return this.fetch(`/changes/detect/${scenario}`, {
      method: 'POST',
      body: JSON.stringify(options || {}),
    })
  }

  async getChangeHistory(scenario: string): Promise<{
    scenario: string
    history: BreakingChange[]
    count: number
  }> {
    return this.fetch(`/changes/history/${scenario}`)
  }

  // Automated Fixes
  async getFixConfig(): Promise<AutomatedFixConfig> {
    return this.fetch('/fix/config')
  }

  async enableAutomatedFixes(config: Partial<AutomatedFixConfig> & { confirmation_understood: boolean }): Promise<any> {
    return this.fetch('/fix/config/enable', {
      method: 'POST',
      body: JSON.stringify(config),
    })
  }

  async disableAutomatedFixes(): Promise<any> {
    return this.fetch('/fix/config/disable', { method: 'POST' })
  }

  async applyFix(scenario: string, vulnerabilityId: string): Promise<AutomatedFix> {
    return this.fetch(`/fix/apply/${scenario}`, {
      method: 'POST',
      body: JSON.stringify({ vulnerability_id: vulnerabilityId }),
    })
  }

  async rollbackFix(fixId: string): Promise<any> {
    return this.fetch(`/fix/rollback/${fixId}`, { method: 'POST' })
  }

  async getFixHistory(): Promise<AutomatedFix[]> {
    const response = await this.fetch<{ fixes: AutomatedFix[] }>('/fix/history')
    return response.fixes || []
  }

  // Standards Compliance
  async checkStandards(name: string, options?: { type?: string, standards?: string[] }): Promise<any> {
    return this.fetch(`/standards/check/${name}`, {
      method: 'POST',
      body: JSON.stringify(options || {}),
    })
  }

  async getStandardsViolations(scenario?: string): Promise<any[]> {
    const url = scenario ? `/standards/violations?scenario=${scenario}` : '/standards/violations'
    const response = await this.fetch<{ violations: any[] }>(url)
    return response.violations || []
  }
}

export const apiService = new ApiService()