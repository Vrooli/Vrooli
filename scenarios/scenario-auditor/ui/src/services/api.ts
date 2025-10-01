import {
  SystemStatus,
  HealthSummary,
  Scenario,
  Vulnerability,
  SecurityScan,
  SecurityScanStatus,
  HealthAlert,
  PerformanceMetric,
  AutomatedFixConfig,
  AutomatedFix,
  BreakingChange,
  AgentInfo,
  FixAgentResponse,
  FixPromptPreviewResponse,
  StandardsViolation,
  StandardsScanStatus,
  AutomatedFixJobSnapshot,
  RuleScenarioTestResult,
  RuleScenarioBatchTestResult,
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
    const scenariosMetric = typeof summary.scenarios === 'object' && summary.scenarios !== null
      ? summary.scenarios
      : undefined
    const vulnerabilitiesMetric = typeof summary.vulnerabilities === 'object' && summary.vulnerabilities !== null
      ? summary.vulnerabilities
      : undefined

    return {
      status: summary.status as any,
      health_score: summary.system_health_score || 0,
      scenarios: scenariosMetric?.total ?? (typeof summary.scenarios === 'number' ? summary.scenarios : 0),
      vulnerabilities: vulnerabilitiesMetric?.total ?? (typeof summary.vulnerabilities === 'number' ? summary.vulnerabilities : 0),
      standards_violations: (summary as any).standards_violations || 0,
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
  async scanScenario(
    name: string,
    options?: { type?: string }
  ): Promise<{ job_id: string; status: SecurityScanStatus }> {
    return this.fetch(`/scenarios/${name}/scan`, {
      method: 'POST',
      body: JSON.stringify(options || {}),
    })
  }

  async getSecurityScanStatus(jobId: string): Promise<SecurityScanStatus> {
    return this.fetch(`/scenarios/scan/jobs/${encodeURIComponent(jobId)}`)
  }

  async cancelSecurityScan(jobId: string): Promise<{ success: boolean; status: SecurityScanStatus; message?: string }> {
    return this.fetch(`/scenarios/scan/jobs/${encodeURIComponent(jobId)}/cancel`, {
      method: 'POST',
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

  async getDashboard(): Promise<any> {
    return this.fetch('/dashboard')
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

  async enableAutomatedFixes(config: Pick<AutomatedFixConfig, 'violation_types' | 'severities' | 'strategy' | 'loop_delay_seconds' | 'timeout_seconds' | 'max_fixes' | 'model'>): Promise<AutomatedFixConfig> {
    return this.fetch('/fix/config/enable', {
      method: 'POST',
      body: JSON.stringify(config),
    })
  }

  async disableAutomatedFixes(): Promise<AutomatedFixConfig> {
    return this.fetch('/fix/config/disable', { method: 'POST' })
  }

  async applyFix(
    scenario: string,
    overrides?: Partial<Pick<AutomatedFixConfig, 'violation_types' | 'severities'>>
  ): Promise<any> {
    return this.fetch(`/fix/apply/${scenario}`, {
      method: 'POST',
      body: JSON.stringify(overrides || {}),
    })
  }

  async rollbackFix(fixId: string): Promise<any> {
    return this.fetch(`/fix/rollback/${fixId}`, { method: 'POST' })
  }

  async getFixHistory(): Promise<AutomatedFix[]> {
    const response = await this.fetch<{ fixes: AutomatedFix[] }>('/fix/history')
    return response.fixes || []
  }

  async getFixJobs(): Promise<AutomatedFixJobSnapshot[]> {
    const response = await this.fetch<{ jobs: AutomatedFixJobSnapshot[] }>('/fix/jobs')
    return response.jobs || []
  }

  async getFixJob(jobId: string): Promise<AutomatedFixJobSnapshot> {
    const response = await this.fetch<{ job: AutomatedFixJobSnapshot }>(`/fix/jobs/${encodeURIComponent(jobId)}`)
    if (!response.job) {
      throw new Error('Automation job not found')
    }
    return response.job
  }

  async cancelFixJob(jobId: string): Promise<{ success: boolean; message: string }> {
    return this.fetch(`/fix/jobs/${encodeURIComponent(jobId)}/cancel`, { method: 'POST' })
  }

  // Standards Compliance
  async checkStandards(
    name: string,
    options?: { type?: string; standards?: string[] }
  ): Promise<{ job_id: string; status: StandardsScanStatus }> {
    return this.fetch(`/standards/check/${encodeURIComponent(name)}`, {
      method: 'POST',
      body: JSON.stringify(options || {}),
    })
  }

  async getStandardsCheckStatus(jobId: string): Promise<StandardsScanStatus> {
    return this.fetch(`/standards/check/jobs/${encodeURIComponent(jobId)}`)
  }

  async cancelStandardsScan(jobId: string): Promise<{ success: boolean; status: StandardsScanStatus; message?: string }> {
    return this.fetch(`/standards/check/jobs/${encodeURIComponent(jobId)}/cancel`, {
      method: 'POST',
    })
  }

  async getStandardsViolations(scenario?: string): Promise<StandardsViolation[]> {
    const url = scenario
      ? `/standards/violations?scenario=${encodeURIComponent(scenario)}`
      : '/standards/violations'
    const response = await this.fetch<{ violations: StandardsViolation[] }>(url)
    return response.violations || []
  }

  // Claude Fix
  async triggerClaudeFix(
    scenarioName: string,
    fixType: 'standards' | 'vulnerabilities',
    issueIds?: string[]
  ): Promise<FixAgentResponse> {
    return this.fetch('/claude/fix', {
      method: 'POST',
      body: JSON.stringify({
        scenario_name: scenarioName,
        fix_type: fixType,
        issue_ids: issueIds,
      }),
    })
  }

  async triggerBulkFix(
    fixType: 'standards' | 'vulnerabilities',
    targets: Array<{ scenario: string; issue_ids?: string[] }>,
    extraPrompt?: string,
    agentCount?: number,
    model?: string
  ): Promise<FixAgentResponse> {
    const payload: Record<string, unknown> = {
      fix_type: fixType,
      targets,
    }
    if (extraPrompt && extraPrompt.trim().length > 0) {
      payload.extra_prompt = extraPrompt.trim()
    }
    if (agentCount && agentCount > 0) {
      payload.agent_count = agentCount
    }
    if (model && model.trim().length > 0 && model.trim().toLowerCase() !== 'default') {
      payload.model = model.trim()
    }

    return this.fetch('/claude/fix', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
  }

  async previewBulkFix(
    fixType: 'standards' | 'vulnerabilities',
    targets: Array<{ scenario: string; issue_ids?: string[] }>,
    extraPrompt?: string,
    agentCount?: number,
    model?: string
  ): Promise<FixPromptPreviewResponse> {
    const payload: Record<string, unknown> = {
      fix_type: fixType,
      targets,
    }
    if (extraPrompt && extraPrompt.trim().length > 0) {
      payload.extra_prompt = extraPrompt.trim()
    }
    if (agentCount && agentCount > 0) {
      payload.agent_count = agentCount
    }
    if (model && model.trim().length > 0 && model.trim().toLowerCase() !== 'default') {
      payload.model = model.trim()
    }

    return this.fetch('/claude/fix/preview', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
  }

  async getClaudeFixStatus(fixId: string): Promise<{ success: boolean; status: string; agent: AgentInfo }> {
    return this.fetch(`/claude/fix/${fixId}/status`)
  }

  // Audit-specific endpoints
  async getRules(category?: string): Promise<any> {
    const url = category ? `/rules?category=${encodeURIComponent(category)}` : '/rules'
    return this.fetch(url)
  }

  async getRule(ruleId: string): Promise<any> {
    return this.fetch(`/rules/${encodeURIComponent(ruleId)}`)
  }

  async toggleRule(ruleId: string, enabled: boolean): Promise<any> {
    return this.fetch(`/rules/${encodeURIComponent(ruleId)}/toggle`, {
      method: 'POST',
      body: JSON.stringify({ enabled })
    })
  }

  async testRule(ruleId: string): Promise<any> {
    return this.fetch(`/rules/${encodeURIComponent(ruleId)}/test`, {
      method: 'POST'
    })
  }

  async testRuleOnScenario(ruleId: string, scenario: string): Promise<RuleScenarioTestResult> {
    return this.fetch(`/rules/${encodeURIComponent(ruleId)}/scenario-test`, {
      method: 'POST',
      body: JSON.stringify({ scenario })
    })
  }

  async testRuleOnScenarios(ruleId: string, scenarios: string[]): Promise<RuleScenarioBatchTestResult | RuleScenarioTestResult> {
    return this.fetch(`/rules/${encodeURIComponent(ruleId)}/scenario-test`, {
      method: 'POST',
      body: JSON.stringify({ scenarios })
    })
  }

  async validateRule(ruleId: string, code: string, language?: string): Promise<any> {
    return this.fetch(`/rules/${encodeURIComponent(ruleId)}/validate`, {
      method: 'POST',
      body: JSON.stringify({ code, language: language || 'go' })
    })
  }

  async startRuleAgent(ruleId: string, action: 'add_rule_tests' | 'fix_rule_tests', label?: string): Promise<{ success: boolean; agent: AgentInfo; message: string; prompt_len: number }> {
    const payload: Record<string, unknown> = { action, rule_id: ruleId }
    if (label) {
      payload.label = label
    }
    return this.fetch('/agents', {
      method: 'POST',
      body: JSON.stringify(payload)
    })
  }

  async createRuleWithAI(payload: { name: string; description: string; category: string; severity: string; motivation?: string }): Promise<{ success: boolean; message: string; agent: AgentInfo; metadata?: Record<string, string> }> {
    return this.fetch('/rules/ai/create', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
  }

  async getActiveAgents(): Promise<{ count: number; agents: AgentInfo[] }> {
    try {
      return await this.fetch('/agents')
    } catch (error) {
      console.warn('Active agents request failed:', error)
      return { count: 0, agents: [] }
    }
  }

  async stopAgent(agentId: string): Promise<{ success: boolean; message: string }> {
    return this.fetch(`/agents/${encodeURIComponent(agentId)}/stop`, {
      method: 'POST'
    })
  }

  async getAgentLogs(agentId: string): Promise<{ agent_id: string; logs: string }> {
    return this.fetch(`/agents/${encodeURIComponent(agentId)}/logs`)
  }

  async clearTestCache(ruleId?: string): Promise<any> {
    const url = ruleId ? `/rules/${encodeURIComponent(ruleId)}/test-cache` : '/rules/test-cache'
    return this.fetch(url, { method: 'DELETE' })
  }

  async getTestCoverage(): Promise<any> {
    return this.fetch('/rules/test-coverage')
  }

  async getResults(): Promise<any> {
    return this.fetch('/scan-results')
  }

  async getPreferences(): Promise<any> {
    return this.fetch('/preferences')
  }

  async updatePreferences(preferences: any): Promise<any> {
    return this.fetch('/preferences', {
      method: 'PUT',
      body: JSON.stringify(preferences)
    })
  }
}

export const apiService = new ApiService()
