import { AgentInfo, RuleImplementationStatus, RuleScenarioTestResult, RuleTestStatus, Scenario } from '@/types/api'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, Brain, CheckCircle, ChevronDown, CircleStop, Clock, Code, Eye, EyeOff, FileText, Info, Play, Plus, RefreshCw, Search, Shield, Target, Terminal, TestTube, X, XCircle } from 'lucide-react'
import { Highlight, themes } from 'prism-react-renderer'
import React, { useEffect, useMemo, useState } from 'react'
import { CodeEditor } from '../components/CodeEditor'
import ReportIssueDialog from '../components/ReportIssueDialog'
import type { ReportPayload } from '../components/ReportIssueDialog'
import { apiService } from '../services/api'

const TARGET_CATEGORY_CONFIG: Array<{ id: string; label: string; description: string }> = [
  { id: 'api', label: 'API Files', description: 'Rules applied to files within the scenario\'s api/ directory.' },
  { id: 'main_go', label: 'main.go', description: 'Rules evaluated specifically against api/main.go or other lifecycle entrypoints.' },
  { id: 'ui', label: 'UI Files', description: 'Rules focused on assets within ui/ such as React components or static markup.' },
  { id: 'cli', label: 'CLI Files', description: 'Rules covering the CLI implementation under cli/.' },
  { id: 'test', label: 'Test Files', description: 'Rules that target files under test/ and other testing utilities.' },
  { id: 'service_json', label: 'service.json', description: 'Rules that run against .vrooli/service.json lifecycle configuration.' },
  { id: 'makefile', label: 'Makefile', description: 'Rules focused on the scenario Makefile lifecycle wrapper.' },
  { id: 'structure', label: 'Scenario Structure', description: 'Rules that validate high-level scenario layout and required assets.' },
  { id: 'misc', label: 'Miscellaneous', description: 'Rules missing targets; update the rule metadata so it runs during scans.' },
]

const TARGET_BADGE_CLASSES: Record<string, string> = {
  api: 'bg-blue-100 text-blue-800',
  main_go: 'bg-indigo-100 text-indigo-800',
  ui: 'bg-pink-100 text-pink-800',
  cli: 'bg-green-100 text-green-800',
  test: 'bg-yellow-100 text-yellow-800',
  service_json: 'bg-purple-100 text-purple-800',
  makefile: 'bg-orange-100 text-orange-800',
  structure: 'bg-slate-100 text-slate-800',
  misc: 'bg-gray-100 text-gray-700 border border-gray-300',
}

// Code Block Component with syntax highlighting and line numbers
function CodeBlock({ code }: { code: string }) {
  if (!code) return <div className="text-gray-500 italic">No code available</div>

  return (
    <Highlight theme={themes.vsDark} code={code} language="go">
      {({ className, style, tokens, getLineProps, getTokenProps }) => (
        <div className="relative">
          <pre
            className={`${className} text-sm rounded-lg p-4 overflow-auto max-h-[500px]`}
            style={style}
          >
            <code className="block">
              {tokens.map((line, i) => {
                const lineProps = getLineProps({ line, key: i })
                const lineKey = lineProps.key as React.Key | null | undefined
                const { className: lineClassName = '', key: _lineKey, ...restLineProps } = lineProps

                return (
                  <div
                    key={lineKey ?? i}
                    className={`table-row ${lineClassName}`.trim()}
                    {...restLineProps}
                  >
                    <span className="table-cell text-right pr-4 select-none text-gray-500 text-xs" style={{ minWidth: '3ch' }}>
                      {i + 1}
                    </span>
                    <span className="table-cell">
                      {line.map((token, key) => {
                        const tokenProps = getTokenProps({ token, key })
                        const tokenKey = tokenProps.key as React.Key | null | undefined
                        const { className: tokenClassName = '', key: _tokenKey, ...restTokenProps } = tokenProps
                        return <span key={tokenKey ?? key} className={tokenClassName} {...restTokenProps} />
                      })}
                    </span>
                  </div>
                )
              })}
            </code>
          </pre>
        </div>
      )}
    </Highlight>
  )
}

// Rule Card Component
function RuleCard({ rule, status, onViewRule, onToggleRule }: {
  rule: any,
  status?: RuleTestStatus,
  onViewRule: (ruleId: string) => void,
  onToggleRule: (ruleId: string, enabled: boolean) => void
}) {
  const handleToggle = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.stopPropagation()
    onToggleRule(rule.id, e.target.checked)
  }

  const handleActivate = () => {
    onViewRule(rule.id)
  }

  const handleCardClick = (event: React.MouseEvent<HTMLDivElement>) => {
    event.preventDefault()
    handleActivate()
  }

  const handleCardKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      handleActivate()
    }
  }

  const testStatus: RuleTestStatus | undefined = status || rule?.test_status
  const implementationStatus: RuleImplementationStatus | undefined = rule?.implementation
  const displayTargets: string[] = Array.isArray(rule.displayTargets) ? rule.displayTargets : Array.isArray(rule.targets) ? rule.targets : []
  const missingTargets = Boolean(rule.missingTargets)
  const implementationError = Boolean(implementationStatus && implementationStatus.valid === false)
  const showWarning = Boolean(testStatus?.has_issues || implementationError || missingTargets)
  const isToggleDisabled = implementationError
  const tooltipLines: string[] = []
  if (testStatus?.warning) {
    tooltipLines.push(testStatus.warning)
  }
  if (!testStatus?.warning && typeof testStatus?.failed === 'number' && typeof testStatus?.total === 'number' && testStatus.total > 0) {
    tooltipLines.push(`${testStatus.failed}/${testStatus.total} tests are failing`)
  }
  if (typeof testStatus?.total === 'number' && testStatus.total === 0 && !tooltipLines.some(line => line.toLowerCase().includes('test cases'))) {
    tooltipLines.push('No test cases defined for this rule')
  }
  if (testStatus?.error) {
    tooltipLines.push(`Test execution error: ${testStatus.error}`)
  }
  if (implementationError) {
    tooltipLines.push(implementationStatus?.error || 'Rule implementation failed to load')
  }
  if (missingTargets) {
    tooltipLines.push('Rule has no targets and will not run during standards scans')
  }

  return (
    <div
      className="bg-white rounded-lg shadow-sm border border-gray-200 hover:shadow-md transition-shadow cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-400"
      data-testid={`rule-${rule.id}`}
      role="button"
      tabIndex={0}
      onClick={handleCardClick}
      onKeyDown={handleCardKeyDown}
    >
      <div className="p-6">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <h3 className="text-lg font-medium text-gray-900">{rule.name}</h3>
            <p className="mt-1 text-sm text-gray-500">{rule.description}</p>
            {implementationError && (
              <p className="mt-2 text-xs text-red-600">
                Implementation error: {implementationStatus?.error || 'Rule implementation unavailable'}
              </p>
            )}
          </div>
          <div className="ml-4 flex items-center space-x-2">
            {showWarning && (
              <div className="relative group">
                <AlertTriangle className="h-5 w-5 text-amber-500" aria-hidden="true" />
                <div className="absolute right-0 z-20 mt-2 hidden w-64 rounded-md border border-amber-200 bg-white p-3 text-xs text-gray-700 shadow-lg group-hover:block">
                  <p className="font-semibold text-amber-700">Rule enforcement paused</p>
                  {tooltipLines.map((line, idx) => (
                    <p key={idx} className="mt-1 text-amber-800">{line}</p>
                  ))}
                  <p className="mt-2 text-[0.7rem] text-amber-700">Fix the failing tests to restore enforcement.</p>
                </div>
              </div>
            )}
            <button
              className="text-gray-400 hover:text-gray-600 transition-colors"
              onClick={(event) => {
                event.stopPropagation()
                onViewRule(rule.id)
              }}
              title="View rule file"
              type="button"
            >
              <Terminal className="h-5 w-5" />
            </button>
          </div>
        </div>

        <div className="mt-4 flex items-center justify-between">
          <div className="flex flex-wrap items-center gap-2">
            {displayTargets.length > 0 ? (
              displayTargets.map(target => {
                const badgeClass = TARGET_BADGE_CLASSES[target] || 'bg-gray-100 text-gray-700'
                const category = TARGET_CATEGORY_CONFIG.find(cfg => cfg.id === target)
                return (
                  <span key={`${rule.id}-${target}`} className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${badgeClass}`}>
                    {category?.label || target}
                  </span>
                )
              })
            ) : (
              <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${TARGET_BADGE_CLASSES.misc}`}>
                No Targets
              </span>
            )}
            {implementationError && (
              <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold bg-red-100 text-red-800 border border-red-200" title={implementationStatus?.error || 'Rule implementation failed to load'}>
                Loader Error
              </span>
            )}
            {missingTargets && (
              <span className="text-xs text-red-600">Add a Targets: metadata line to enable this rule.</span>
            )}
            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${rule.severity === 'critical' ? 'bg-red-100 text-red-800' :
                rule.severity === 'high' ? 'bg-orange-100 text-orange-800' :
                  rule.severity === 'medium' ? 'bg-yellow-100 text-yellow-800' :
                    rule.severity === 'low' ? 'bg-green-100 text-green-800' :
                      'bg-gray-100 text-gray-800'
              }`}>
              {rule.severity}
            </span>
          </div>
          <div className="flex items-center">
            <label
              className="relative inline-flex items-center cursor-pointer"
              onClick={(event) => event.stopPropagation()}
            >
              <input
                type="checkbox"
                checked={rule.enabled}
                onChange={handleToggle}
                className="sr-only peer"
                data-testid={`rule-toggle-${rule.id}`}
                disabled={isToggleDisabled}
              />
              <div className={`w-11 h-6 ${isToggleDisabled ? 'bg-gray-100 cursor-not-allowed' : 'bg-gray-200'} peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all ${isToggleDisabled ? 'peer-checked:bg-gray-300' : 'peer-checked:bg-blue-600'}`}></div>
            </label>
          </div>
        </div>
      </div>
    </div>
  )
}

export default function RulesManager() {
  const [selectedCategory, setSelectedCategory] = useState<string>('')
  const [selectedRule, setSelectedRule] = useState<string | null>(null)
  const [localRules, setLocalRules] = useState<Record<string, any>>({})
  const [searchTerm, setSearchTerm] = useState<string>('')
  const [activeTab, setActiveTab] = useState<'implementation' | 'tests' | 'playground'>('implementation')
  const [testResults, setTestResults] = useState<any>(null)
  const [isRunningTests, setIsRunningTests] = useState(false)
  const [playgroundCode, setPlaygroundCode] = useState('')
  const [playgroundResult, setPlaygroundResult] = useState<any>(null)
  const [isRunningPlayground, setIsRunningPlayground] = useState(false)
  const [isLaunchingAgent, setIsLaunchingAgent] = useState(false)
  const [agentError, setAgentError] = useState<string | null>(null)
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [isCreatingRule, setIsCreatingRule] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)
  const [createSuccess, setCreateSuccess] = useState<string | null>(null)
  const [createForm, setCreateForm] = useState({
    name: '',
    description: '',
    category: 'api',
    severity: 'medium',
    motivation: '',
  })
  const [isScenarioTestModalOpen, setIsScenarioTestModalOpen] = useState(false)
  const [isRunningScenarioTest, setIsRunningScenarioTest] = useState(false)
  const [scenarioTestResults, setScenarioTestResults] = useState<RuleScenarioTestResult[]>([])
  const [scenarioTestError, setScenarioTestError] = useState<string | null>(null)
  const [scenarioTestCompletedAt, setScenarioTestCompletedAt] = useState<Date | null>(null)
  const [scenarioSearchTerm, setScenarioSearchTerm] = useState('')
  const [selectedScenarios, setSelectedScenarios] = useState<Set<string>>(new Set())
  const [isReportDialogOpen, setIsReportDialogOpen] = useState(false)
  const queryClient = useQueryClient()

  const { data: activeAgentsData } = useQuery({
    queryKey: ['activeAgents'],
    queryFn: () => apiService.getActiveAgents(),
    refetchInterval: 2000,
  })

  const ruleAgents = useMemo(() => {
    if (!selectedRule) return []
    return (activeAgentsData?.agents || []).filter((agent: AgentInfo) => agent.rule_id === selectedRule)
  }, [activeAgentsData, selectedRule])

  const isAgentRunning = ruleAgents.length > 0

  const formatDuration = (seconds: number) => {
    if (!seconds || seconds < 1) return '<1s'
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    if (mins > 0) {
      return `${mins}m ${secs}s`
    }
    return `${secs}s`
  }

  const formatDurationMs = (milliseconds: number) => {
    if (!milliseconds || milliseconds < 1) {
      return '<1ms'
    }
    if (milliseconds < 1000) {
      return `${Math.round(milliseconds)}ms`
    }

    const totalSeconds = milliseconds / 1000
    if (totalSeconds < 60) {
      if (totalSeconds >= 10) {
        return `${totalSeconds.toFixed(1)}s`
      }
      return `${totalSeconds.toFixed(2)}s`
    }

    const minutes = Math.floor(totalSeconds / 60)
    const seconds = Math.round(totalSeconds - minutes*60)
    if (minutes < 60) {
      return `${minutes}m ${seconds}s`
    }

    const hours = Math.floor(minutes / 60)
    const remainingMinutes = minutes % 60
    return `${hours}h ${remainingMinutes}m`
  }

  const describeAgentAction = (action: string) => {
    switch (action) {
      case 'add_rule_tests':
        return 'Add Test Cases'
      case 'fix_rule_tests':
        return 'Fix Test Cases'
      case 'create_rule':
        return 'Create Rule'
      default:
        return action
    }
  }

  const [showExecutionOutput, setShowExecutionOutput] = useState<{ [key: number]: boolean }>({})
  const [executionInfoExpanded, setExecutionInfoExpanded] = useState(false)
  const [isRefreshing, setIsRefreshing] = useState(false)

  const { data: rulesData, isLoading, refetch } = useQuery({
    queryKey: ['rules'],
    queryFn: () => apiService.getRules(),
  })

  const { data: ruleDetail, isLoading: ruleDetailLoading } = useQuery({
    queryKey: ['rule', selectedRule],
    queryFn: () => selectedRule ? apiService.getRule(selectedRule) : null,
    enabled: !!selectedRule,
  })

  const { data: scenarioOptions, isFetching: isFetchingScenarios } = useQuery<Scenario[]>({
    queryKey: ['scenarios', 'rule-tests'],
    queryFn: () => apiService.getScenarios(),
    enabled: isScenarioTestModalOpen,
  })

  const apiCategories = (rulesData?.categories || {}) as Record<string, { name?: string }>
  const executionInfo = ruleDetail?.execution_info

  useEffect(() => {
    setExecutionInfoExpanded(false)
  }, [selectedRule])

  useEffect(() => {
    setScenarioTestResults([])
    setScenarioTestError(null)
    setScenarioTestCompletedAt(null)
    setSelectedScenarios(new Set())
  }, [selectedRule])

  const ruleStatuses = (rulesData?.rule_statuses || {}) as Record<string, RuleTestStatus>
  const filteredScenarioOptions = useMemo(() => {
    const list = Array.isArray(scenarioOptions) ? scenarioOptions : []
    const term = scenarioSearchTerm.trim().toLowerCase()
    if (!term) {
      return list
    }
    return list.filter(option => option.name.toLowerCase().includes(term))
  }, [scenarioOptions, scenarioSearchTerm])
  const createCategoryEntries: Array<[string, { name?: string }]> = Object.keys(apiCategories).length > 0
    ? Object.entries(apiCategories)
    : [
      ['api', { name: 'API Standards' }],
      ['config', { name: 'Configuration' }],
      ['test', { name: 'Testing' }],
      ['ui', { name: 'User Interface' }],
    ]

  // Run tests when rule is selected and tests tab is active
  const runRuleTests = async () => {
    if (!selectedRule) return

    setIsRunningTests(true)
    try {
      const results = await apiService.testRule(selectedRule)
      setTestResults(results)
    } catch (error) {
      console.error('Failed to run tests:', error)
      setTestResults({ error: 'Failed to run tests' })
    } finally {
      setIsRunningTests(false)
    }
  }

  const handleRunScenarioTests = async () => {
    if (!selectedRule || selectedScenarios.size === 0) return

    setScenarioTestError(null)
    setIsRunningScenarioTest(true)
    setIsScenarioTestModalOpen(false)

    try {
      const scenariosArray = Array.from(selectedScenarios)
      const response = await apiService.testRuleOnScenarios(selectedRule, scenariosArray)

      // Handle both single and batch responses
      if ('results' in response) {
        // Batch response
        setScenarioTestResults(response.results)
      } else {
        // Single response (backward compatibility)
        setScenarioTestResults([response])
      }
      setScenarioTestCompletedAt(new Date())
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to run scenario tests'
      setScenarioTestError(message)
      setScenarioTestResults([])
      setScenarioTestCompletedAt(new Date())
    } finally {
      setIsRunningScenarioTest(false)
    }
  }

  // Run playground code validation
  const runPlaygroundTest = async () => {
    if (!selectedRule || !playgroundCode.trim()) return

    setIsRunningPlayground(true)
    try {
      const result = await apiService.validateRule(selectedRule, playgroundCode, 'go')
      setPlaygroundResult(result)
    } catch (error) {
      console.error('Failed to run playground test:', error)
      setPlaygroundResult({ error: 'Failed to run playground test' })
    } finally {
      setIsRunningPlayground(false)
    }
  }

  const handleSubmitReport = async (payload: ReportPayload) => {
    return await apiService.reportRuleIssue(payload)
  }

  const handleStopAgent = async (agentId: string) => {
    setAgentError(null)
    setIsLaunchingAgent(true)
    try {
      await apiService.stopAgent(agentId)
      await queryClient.invalidateQueries({ queryKey: ['activeAgents'] })
    } catch (error) {
      console.error('Failed to stop agent:', error)
      setAgentError((error as Error).message)
    } finally {
      setIsLaunchingAgent(false)
    }
  }

  const severityOptions = ['critical', 'high', 'medium', 'low']

  const openCreateRuleModal = () => {
    const availableCategories = createCategoryEntries.map(([key]) => key as string)
    const defaultCategory = availableCategories.includes('api')
      ? 'api'
      : (availableCategories[0] || 'api')

    setCreateForm({
      name: '',
      description: '',
      category: defaultCategory,
      severity: 'medium',
      motivation: '',
    })
    setCreateError(null)
    setCreateSuccess(null)
    setIsCreateModalOpen(true)
  }

  const handleCreateRule = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!createForm.name.trim() || !createForm.description.trim()) {
      setCreateError('Rule name and description are required')
      return
    }

    setCreateError(null)
    setIsCreatingRule(true)

    try {
      const payload = {
        name: createForm.name.trim(),
        description: createForm.description.trim(),
        category: createForm.category,
        severity: createForm.severity,
        motivation: createForm.motivation.trim(),
      }

      const response = await apiService.createRuleWithAI(payload)

      setIsCreateModalOpen(false)

      // Show success with clickable link to issue
      const successMessage = response.issueUrl
        ? `Rule creation issue created for "${payload.name}". View at: ${response.issueUrl}`
        : response.message || `Rule creation issue created for ${payload.name}`

      setCreateSuccess(successMessage)

      // No need to refresh agents - issue is in app-issue-tracker now
      await refetch()
    } catch (error) {
      setCreateError(error instanceof Error ? error.message : 'Failed to create rule creation issue')
    } finally {
      setIsCreatingRule(false)
    }
  }

  const handleRefreshRules = async () => {
    setIsRefreshing(true)
    try {
      await refetch()
    } finally {
      setIsRefreshing(false)
    }
  }

  // Reset tab and test data when rule changes
  React.useEffect(() => {
    setAgentError(null)
    if (selectedRule) {
      setActiveTab('implementation')
      setTestResults(null)
      setPlaygroundCode('')
      setPlaygroundResult(null)
    }
  }, [selectedRule])

  // Sync local rules state with fetched data
  React.useEffect(() => {
    if (rulesData?.rules) {
      setLocalRules(rulesData.rules)
    }
  }, [rulesData])

  const handleToggleRule = async (ruleId: string, enabled: boolean) => {
    // Optimistically update the UI
    setLocalRules(prev => ({
      ...prev,
      [ruleId]: {
        ...prev[ruleId],
        enabled
      }
    }))

    try {
      // Call the API to toggle the rule
      await apiService.toggleRule(ruleId, enabled)
      // Refetch rules to ensure consistency
      refetch()
    } catch (error) {
      console.error('Failed to toggle rule:', error)
      // Revert on error
      setLocalRules(prev => ({
        ...prev,
        [ruleId]: {
          ...prev[ruleId],
          enabled: !enabled
        }
      }))
    }
  }

  const rules = localRules
  const rulesArray = useMemo(() => Object.values(rules || {}), [rules])

  // Filter rules by search term
  const filteredRulesArray = useMemo(() => {
    const term = searchTerm.trim().toLowerCase()
    if (!term) return rulesArray

    return rulesArray.filter((rule: any) => {
      return (
        rule.name?.toLowerCase().includes(term) ||
        rule.description?.toLowerCase().includes(term) ||
        rule.id?.toLowerCase().includes(term) ||
        rule.category?.toLowerCase().includes(term)
      )
    })
  }, [rulesArray, searchTerm])

  const categorizedRules = useMemo(() => {
    const base: Record<string, any[]> = {}
    TARGET_CATEGORY_CONFIG.forEach(cfg => {
      base[cfg.id] = []
    })

    filteredRulesArray.forEach((rule: any) => {
      const rawTargets: string[] = Array.isArray(rule?.targets) ? (rule.targets as string[]) : []
      const normalizedTargets = Array.from(new Set<string>(
        rawTargets
          .map((target) => (typeof target === 'string' ? target.toLowerCase().trim() : ''))
          .filter(Boolean)
      ))

      const recognizedTargets = normalizedTargets.filter(target => target !== 'misc' && TARGET_CATEGORY_CONFIG.some(cfg => cfg.id === target && cfg.id !== 'misc'))
      const hasRecognizedTargets = recognizedTargets.length > 0
      const bucketTargets: string[] = hasRecognizedTargets ? recognizedTargets : ['misc']
      const displayTargets = hasRecognizedTargets ? recognizedTargets : []
      const missingTargets = !hasRecognizedTargets

      bucketTargets.forEach((targetId: string) => {
        const categoryKey = TARGET_CATEGORY_CONFIG.some(cfg => cfg.id === targetId) ? targetId : 'misc'
        if (!base[categoryKey]) {
          base[categoryKey] = []
        }
        base[categoryKey].push({
          ...rule,
          displayTargets,
          missingTargets,
        })
      })
    })

    Object.keys(base).forEach(categoryId => {
      base[categoryId].sort((a, b) => (a.name || '').localeCompare(b.name || ''))
    })

    return base
  }, [filteredRulesArray])

  const targetCategoryEntries = useMemo(() => (
    TARGET_CATEGORY_CONFIG.map(config => [config.id, {
      ...config,
      rules: categorizedRules[config.id] || [],
    }]) as [string, { label: string; description: string; rules: any[] }][]
  ), [categorizedRules])

  const selectedCategoryRules = selectedCategory ? (categorizedRules[selectedCategory] || []) : []

  if (isLoading) {
    return (
      <div className="px-6">
        <div className="animate-pulse">
          <div className="h-8 bg-gray-200 rounded w-1/4 mb-6"></div>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {[...Array(6)].map((_, i) => (
              <div key={i} className="bg-white p-6 rounded-lg shadow">
                <div className="h-4 bg-gray-200 rounded w-3/4 mb-2"></div>
                <div className="h-3 bg-gray-200 rounded w-1/2"></div>
              </div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="px-6">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900" data-testid="rules-title">
            Rules Manager
          </h1>
          <p className="mt-1 text-sm text-gray-500">
            Manage and configure auditing rules
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            type="button"
            className="inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
            onClick={handleRefreshRules}
            disabled={isRefreshing || isLoading}
            title="Refresh rules from disk (includes file changes and new rules)"
          >
            <RefreshCw className={`mr-2 h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
            {isRefreshing ? 'Refreshing...' : 'Refresh'}
          </button>
          <button
            type="button"
            className="inline-flex items-center rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
            data-testid="create-rule-btn"
            onClick={openCreateRuleModal}
          >
            <Plus className="mr-2 h-4 w-4" />
            Create Rule
          </button>
        </div>
      </div>

      {createSuccess && (
        <div className="mb-6 rounded-md border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-800">
          <div className="flex items-start">
            <CheckCircle className="mr-2 h-4 w-4 flex-shrink-0" />
            <span>{createSuccess}</span>
          </div>
        </div>
      )}

      {/* Filters */}
      <div className="mb-6 flex flex-col sm:flex-row items-stretch sm:items-center gap-4">
        <div className="relative flex-1">
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <Search className="h-5 w-5 text-gray-400" />
          </div>
          <input
            type="text"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder="Search rules by name, description, or category..."
            className="block w-full pl-10 pr-3 py-2 border border-gray-300 rounded-md leading-5 bg-white placeholder-gray-500 focus:outline-none focus:placeholder-gray-400 focus:ring-1 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
            data-testid="rule-search"
          />
        </div>
        <select
          value={selectedCategory}
          onChange={(e) => setSelectedCategory(e.target.value)}
          className="rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:w-48"
          data-testid="category-filter"
        >
          <option value="">All Targets</option>
          {targetCategoryEntries.map(([key, category]) => (
            <option key={key} value={key}>
              {category.label}
            </option>
          ))}
        </select>
      </div>

      {/* Rules Display */}
      {selectedCategory ? (
        /* Single Category Grid */
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {selectedCategoryRules.length > 0 ? (
            selectedCategoryRules.map((rule: any) => (
              <RuleCard
                key={`${rule.id}-${selectedCategory}`}
                rule={rule}
                status={ruleStatuses[rule.id]}
                onViewRule={setSelectedRule}
                onToggleRule={handleToggleRule}
              />
            ))
          ) : (
            <div className="col-span-full rounded-md border border-gray-200 bg-gray-50 p-6 text-center text-sm text-gray-500">
              No rules currently run for this target. Add a <code className="font-mono text-xs">Targets:</code> line to a rule to include it.
            </div>
          )}
        </div>
      ) : (
        /* Grouped by Category */
        <div className="space-y-8">
          {targetCategoryEntries.map(([key, category]) => {
            const categoryRules = category.rules
            if (!categoryRules || categoryRules.length === 0) {
              return null
            }

            return (
              <div key={key} className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h2 className="text-xl font-semibold text-gray-900">{category.label}</h2>
                    <p className="text-sm text-gray-500">{category.description}</p>
                  </div>
                  <span className="text-sm text-gray-400">{categoryRules.length} rule{categoryRules.length !== 1 ? 's' : ''}</span>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {categoryRules.map((rule: any) => (
                    <RuleCard
                      key={`${rule.id}-${key}`}
                      rule={rule}
                      status={ruleStatuses[rule.id]}
                      onViewRule={setSelectedRule}
                      onToggleRule={handleToggleRule}
                    />
                  ))}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {Object.keys(rules).length === 0 && (
        <div className="text-center py-12">
          <Shield className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-sm font-medium text-gray-900">No rules found</h3>
          <p className="mt-1 text-sm text-gray-500">
            {selectedCategory ? 'No rules in this category.' : 'Get started by creating your first rule.'}
          </p>
        </div>
      )}

      {isScenarioTestModalOpen && (
        <div className="fixed inset-0 z-[70] overflow-y-auto">
          <div className="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
            <div
              className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"
              onClick={() => setIsScenarioTestModalOpen(false)}
            />
            <div className="relative transform overflow-hidden rounded-lg bg-white px-4 pb-4 pt-5 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg sm:p-6">
              <div className="absolute right-0 top-0 hidden pr-4 pt-4 sm:block">
                <button
                  type="button"
                  className="rounded-md bg-white text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
                  onClick={() => setIsScenarioTestModalOpen(false)}
                >
                  <span className="sr-only">Close</span>
                  <X className="h-6 w-6" />
                </button>
              </div>

              <div>
                <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-slate-100">
                  <Target className="h-6 w-6 text-slate-700" />
                </div>
                <div className="mt-3 text-center sm:mt-4">
                  <h3 className="text-base font-semibold leading-6 text-gray-900">Select scenarios to evaluate</h3>
                  <p className="mt-1 text-sm text-gray-500">
                    Run this rule against multiple scenarios without toggling global standards scans.
                  </p>
                </div>
              </div>

              <div className="mt-5 space-y-4">
                <div>
                  <label htmlFor="scenario-search" className="sr-only">Search scenarios</label>
                  <input
                    id="scenario-search"
                    type="text"
                    value={scenarioSearchTerm}
                    onChange={(event) => setScenarioSearchTerm(event.target.value)}
                    placeholder="Search scenarios"
                    className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                <div className="max-h-64 overflow-y-auto rounded-md border border-gray-200">
                  {isFetchingScenarios ? (
                    <div className="flex items-center justify-center gap-2 px-3 py-6 text-sm text-gray-600">
                      <Clock className="h-4 w-4 animate-spin" />
                      <span>Loading scenarios...</span>
                    </div>
                  ) : (
                    <>
                      {filteredScenarioOptions.length > 0 && (
                        <div className="sticky top-0 z-10 border-b border-gray-200 bg-white px-3 py-2">
                          <label className="flex items-center gap-2 text-sm font-medium text-gray-700 cursor-pointer">
                            <input
                              type="checkbox"
                              checked={selectedScenarios.size === filteredScenarioOptions.length && filteredScenarioOptions.length > 0}
                              onChange={(e) => {
                                if (e.target.checked) {
                                  setSelectedScenarios(new Set(filteredScenarioOptions.map(s => s.name)))
                                } else {
                                  setSelectedScenarios(new Set())
                                }
                              }}
                              className="h-4 w-4 rounded border-gray-300 text-slate-700 focus:ring-slate-500"
                            />
                            <span>Select All ({filteredScenarioOptions.length})</span>
                          </label>
                        </div>
                      )}
                      <ul className="divide-y divide-gray-200 text-left text-sm">
                        {filteredScenarioOptions.length === 0 ? (
                          <li className="px-3 py-4 text-center text-gray-500">
                            No scenarios match your search.
                          </li>
                        ) : (
                          filteredScenarioOptions.map(option => (
                            <li key={option.name} className="px-3 py-3 hover:bg-gray-50">
                              <label className="flex items-center gap-3 cursor-pointer">
                                <input
                                  type="checkbox"
                                  checked={selectedScenarios.has(option.name)}
                                  onChange={(e) => {
                                    const newSelection = new Set(selectedScenarios)
                                    if (e.target.checked) {
                                      newSelection.add(option.name)
                                    } else {
                                      newSelection.delete(option.name)
                                    }
                                    setSelectedScenarios(newSelection)
                                  }}
                                  className="h-4 w-4 rounded border-gray-300 text-slate-700 focus:ring-slate-500"
                                />
                                <div className="flex-1">
                                  <p className="text-sm font-medium text-gray-900">{option.name}</p>
                                  {option.description && (
                                    <p className="text-xs text-gray-500">{option.description}</p>
                                  )}
                                </div>
                              </label>
                            </li>
                          ))
                        )}
                      </ul>
                    </>
                  )}
                </div>

                <div className="flex items-center justify-between pt-2">
                  <span className="text-sm text-gray-600">
                    {selectedScenarios.size} scenario{selectedScenarios.size !== 1 ? 's' : ''} selected
                  </span>
                  <button
                    type="button"
                    onClick={handleRunScenarioTests}
                    disabled={selectedScenarios.size === 0 || isRunningScenarioTest}
                    className="inline-flex items-center rounded-md border border-transparent bg-slate-700 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-slate-800 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-slate-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {isRunningScenarioTest ? (
                      <>
                        <Clock className="h-4 w-4 mr-2 animate-spin" />
                        Testing...
                      </>
                    ) : (
                      <>
                        <Target className="h-4 w-4 mr-2" />
                        Run Tests
                      </>
                    )}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Create Rule Modal */}
      {isCreateModalOpen && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
            <div className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity" onClick={() => setIsCreateModalOpen(false)} />

            <div className="relative transform overflow-hidden rounded-lg bg-white px-4 pb-4 pt-5 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-xl sm:p-6">
              <div className="absolute right-0 top-0 hidden pr-4 pt-4 sm:block">
                <button
                  type="button"
                  className="rounded-md bg-white text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
                  onClick={() => setIsCreateModalOpen(false)}
                >
                  <span className="sr-only">Close</span>
                  <X className="h-6 w-6" />
                </button>
              </div>

              <div className="sm:flex sm:items-start">
                <div className="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-blue-100 sm:mx-0 sm:h-10 sm:w-10">
                  <Plus className="h-6 w-6 text-blue-600" />
                </div>
                <div className="mt-3 text-center sm:ml-4 sm:mt-0 sm:text-left">
                  <h3 className="text-base font-semibold leading-6 text-gray-900">Create New Rule</h3>
                  <p className="mt-1 text-sm text-gray-500">Provide the details for the rule you want an agent to implement.</p>
                </div>
              </div>

              <form className="mt-5 space-y-4" onSubmit={handleCreateRule}>
                <div>
                  <label className="block text-sm font-medium text-gray-700" htmlFor="rule-name">Rule name</label>
                  <input
                    id="rule-name"
                    type="text"
                    className="mt-1 w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                    value={createForm.name}
                    onChange={(e) => setCreateForm(prev => ({ ...prev, name: e.target.value }))}
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700" htmlFor="rule-description">What should this rule enforce?</label>
                  <textarea
                    id="rule-description"
                    className="mt-1 w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                    rows={4}
                    value={createForm.description}
                    onChange={(e) => setCreateForm(prev => ({ ...prev, description: e.target.value }))}
                    required
                  />
                </div>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <div>
                    <label className="block text-sm font-medium text-gray-700" htmlFor="rule-category">Category</label>
                    <select
                      id="rule-category"
                      className="mt-1 w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                      value={createForm.category}
                      onChange={(e) => setCreateForm(prev => ({ ...prev, category: e.target.value }))}
                    >
                      {createCategoryEntries.map(([key, value]: [string, any]) => (
                        <option key={key} value={key}>{value.name || key}</option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700" htmlFor="rule-severity">Severity</label>
                    <select
                      id="rule-severity"
                      className="mt-1 w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                      value={createForm.severity}
                      onChange={(e) => setCreateForm(prev => ({ ...prev, severity: e.target.value }))}
                    >
                      {severityOptions.map(option => (
                        <option key={option} value={option}>{option}</option>
                      ))}
                    </select>
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700" htmlFor="rule-motivation">Additional context (optional)</label>
                  <textarea
                    id="rule-motivation"
                    className="mt-1 w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                    rows={3}
                    value={createForm.motivation}
                    onChange={(e) => setCreateForm(prev => ({ ...prev, motivation: e.target.value }))}
                    placeholder="Share scenarios, edge cases, or references that will help the agent design the rule."
                  />
                </div>

                {createError && (
                  <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                    {createError}
                  </div>
                )}

                <div className="flex items-center justify-end gap-3 pt-2">
                  <button
                    type="button"
                    className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
                    onClick={() => setIsCreateModalOpen(false)}
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="inline-flex items-center rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:bg-blue-400"
                    disabled={isCreatingRule}
                  >
                    {isCreatingRule ? 'Starting agent…' : 'Start Agent'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}

      {/* Report Issue Dialog */}
      <ReportIssueDialog
        isOpen={isReportDialogOpen}
        onClose={() => setIsReportDialogOpen(false)}
        ruleId={selectedRule}
        ruleName={ruleDetail?.rule?.name || null}
        scenarioTestResults={scenarioTestResults}
        onSubmitReport={handleSubmitReport}
      />

      {/* Rule Detail Modal */}
      {selectedRule && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
            <div className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity" onClick={() => setSelectedRule(null)} />

            <div className="relative transform overflow-hidden rounded-lg bg-white px-4 pb-4 pt-5 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-4xl sm:p-6">
              <div className="absolute right-0 top-0 hidden pr-4 pt-4 sm:block">
                <button
                  type="button"
                  className="rounded-md bg-white text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
                  onClick={() => setSelectedRule(null)}
                >
                  <span className="sr-only">Close</span>
                  <X className="h-6 w-6" />
                </button>
              </div>

              <div className="sm:flex sm:items-start">
                <div className="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-blue-100 sm:mx-0 sm:h-10 sm:w-10">
                  <Terminal className="h-6 w-6 text-blue-600" />
                </div>
                <div className="mt-3 text-center sm:ml-4 sm:mt-0 sm:text-left flex-1">
                  <h3 className="text-base font-semibold leading-6 text-gray-900">
                    {ruleDetail?.rule?.name || selectedRule}
                  </h3>
                  <div className="mt-2">
                    <p className="text-sm text-gray-500">
                      {ruleDetail?.rule?.description || 'Loading rule details...'}
                    </p>
                    {ruleDetail?.file_path && (
                      <p className="text-xs text-gray-400 mt-1">
                        {ruleDetail.file_path}
                      </p>
                    )}
                    {ruleDetail?.rule?.implementation && ruleDetail.rule.implementation.valid === false && (
                      <div className="mt-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700">
                        <p className="font-medium">Implementation error</p>
                        <p>{ruleDetail.rule.implementation.error || 'Rule implementation failed to load.'}</p>
                        {ruleDetail.rule.implementation.details && (
                          <p className="mt-1 text-red-600/80">{ruleDetail.rule.implementation.details}</p>
                        )}
                      </div>
                    )}
                  </div>
                </div>
              </div>

              {/* Tab Navigation */}
              <div className="mt-5 border-b border-gray-200">
                <div className="flex space-x-8">
                  <button
                    onClick={() => setActiveTab('implementation')}
                    className={`py-2 px-1 border-b-2 font-medium text-sm ${activeTab === 'implementation'
                        ? 'border-blue-500 text-blue-600'
                        : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                      }`}
                  >
                    <Code className="h-4 w-4 inline-block mr-2" />
                    Implementation
                  </button>
                  <button
                    onClick={() => {
                      setActiveTab('tests')
                      if (!testResults && !isRunningTests) runRuleTests()
                    }}
                    className={`py-2 px-1 border-b-2 font-medium text-sm ${activeTab === 'tests'
                        ? 'border-blue-500 text-blue-600'
                        : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                      }`}
                  >
                    <TestTube className="h-4 w-4 inline-block mr-2" />
                    Test Cases
                    {testResults && (
                      <span className={`ml-2 inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${testResults.error ? 'bg-red-100 text-red-800' :
                          testResults.failed > 0 ? 'bg-yellow-100 text-yellow-800' :
                            'bg-green-100 text-green-800'
                        }`}>
                        {testResults.error ? 'Error' : `${testResults.passed}/${testResults.total_tests} correct`}
                      </span>
                    )}
                  </button>
                  <button
                    onClick={() => setActiveTab('playground')}
                    className={`py-2 px-1 border-b-2 font-medium text-sm ${activeTab === 'playground'
                        ? 'border-blue-500 text-blue-600'
                        : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                      }`}
                  >
                    <Play className="h-4 w-4 inline-block mr-2" />
                    Playground
                  </button>
                </div>
              </div>

              {/* Tab Content */}
              <div className="mt-5">
                {activeTab === 'implementation' && (
                  <div className="space-y-4">
                    {ruleDetail?.rule?.implementation && ruleDetail.rule.implementation.valid === false && (
                      <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                        <p className="font-semibold">Rule implementation could not be loaded.</p>
                        <p className="mt-1">{ruleDetail.rule.implementation.error || 'An unknown error occurred while loading this rule.'}</p>
                        {ruleDetail.rule.implementation.details && (
                          <p className="mt-1 text-red-600/80">{ruleDetail.rule.implementation.details}</p>
                        )}
                      </div>
                    )}

                    {executionInfo && (
                      <div className="rounded-lg border border-blue-200 bg-blue-50/80 text-blue-900 shadow-sm">
                        <button
                          type="button"
                          onClick={() => setExecutionInfoExpanded(prev => !prev)}
                          className="flex w-full items-center justify-between gap-4 px-4 py-3 text-left"
                        >
                          <div className="flex items-center gap-3">
                            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-blue-100 text-blue-600">
                              <Info className="h-4 w-4" aria-hidden="true" />
                            </div>
                            <div>
                              <p className="text-sm font-semibold">How this rule is executed</p>
                              <p className="text-xs text-blue-700">Expand to review signature, arguments, and call flow.</p>
                            </div>
                          </div>
                          <ChevronDown
                            className={'h-4 w-4 text-blue-600 transition-transform ' + (executionInfoExpanded ? 'rotate-180' : '')}
                            aria-hidden="true"
                          />
                        </button>
                        {executionInfoExpanded && (
                          <div className="space-y-3 border-t border-blue-100 px-4 py-3 text-sm">
                            <div className="flex flex-wrap items-center gap-2">
                              <span className="font-medium">Signature:</span>
                              <code className="rounded bg-blue-100 px-1.5 py-0.5 font-mono text-xs text-blue-900">
                                {executionInfo.signature}
                              </code>
                            </div>
                            {executionInfo.arguments && executionInfo.arguments.length > 0 && (
                              <div>
                                <span className="font-medium">Arguments</span>
                                <ul className="mt-1 space-y-1 text-blue-800">
                                  {executionInfo.arguments.map((arg: any) => (
                                    <li key={arg.name} className="leading-5">
                                      <span className="font-semibold capitalize">{arg.name}</span>
                                      <span className="text-blue-800"> — {arg.description}</span>
                                    </li>
                                  ))}
                                </ul>
                              </div>
                            )}
                            {executionInfo.call_flow && executionInfo.call_flow.length > 0 && (
                              <div>
                                <span className="font-medium">Call flow</span>
                                <ul className="mt-1 space-y-1 text-blue-800">
                                  {executionInfo.call_flow.map((step: any, index: number) => (
                                    <li key={`${step.source}-${index}`} className="leading-5">
                                      <span className="font-semibold">{step.source}:</span>
                                      <span className="text-blue-800"> {step.description}</span>
                                      {step.reference && (
                                        <span className="ml-1 text-blue-700">({step.reference})</span>
                                      )}
                                    </li>
                                  ))}
                                </ul>
                              </div>
                            )}
                            {executionInfo.notes && executionInfo.notes.length > 0 && (
                              <div>
                                <span className="font-medium">Key points</span>
                                <ul className="mt-1 list-disc pl-5 space-y-1 text-blue-800">
                                  {executionInfo.notes.map((note: string, index: number) => (
                                    <li key={index}>{note}</li>
                                  ))}
                                </ul>
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    )}


                    <div className="overflow-hidden rounded-lg bg-gray-900">
                      <div className="border-b border-gray-700 bg-gray-800 px-4 py-2">
                        <h4 className="flex items-center justify-between text-sm font-medium text-gray-200">
                          <span>Rule Implementation</span>
                          {ruleDetail?.file_path && (
                            <span className="font-mono text-xs text-gray-400">
                              {ruleDetail.file_path.replace(/^.*\/rules\//, 'rules/')}
                            </span>
                          )}
                        </h4>
                      </div>
                      <div className="p-0">
                        {ruleDetailLoading ? (
                          <div className="p-4">
                            <div className="animate-pulse">
                              <div className="mb-2 h-4 w-3/4 rounded bg-gray-700"></div>
                              <div className="mb-2 h-4 w-1/2 rounded bg-gray-700"></div>
                              <div className="h-4 w-2/3 rounded bg-gray-700"></div>
                            </div>
                          </div>
                        ) : (
                          <CodeBlock code={ruleDetail?.file_content || ''} />
                        )}
                      </div>
                    </div>
                  </div>
                )}

                {activeTab === 'tests' && (
                  <div className="space-y-4">
                    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                      <h4 className="text-lg font-medium text-gray-900">Test Cases</h4>
                      <div className="flex flex-wrap items-center gap-2">
                        <button
                          onClick={runRuleTests}
                          disabled={isRunningTests}
                          className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                        >
                          {isRunningTests ? (
                            <Clock className="h-4 w-4 mr-2 animate-spin" />
                          ) : (
                            <Play className="h-4 w-4 mr-2" />
                          )}
                          {isRunningTests ? 'Running...' : 'Run All Tests'}
                        </button>
                        <button
                          onClick={() => {
                            setScenarioSearchTerm('')
                            setIsScenarioTestModalOpen(true)
                          }}
                          disabled={isRunningScenarioTest || !selectedRule}
                          className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md text-white bg-slate-700 hover:bg-slate-800 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-slate-500 disabled:opacity-50"
                        >
                          {isRunningScenarioTest ? (
                            <Clock className="h-4 w-4 mr-2 animate-spin" />
                          ) : (
                            <Target className="h-4 w-4 mr-2" />
                          )}
                          {isRunningScenarioTest ? 'Testing...' : 'Test on Scenario'}
                        </button>
                        <button
                          onClick={() => setIsReportDialogOpen(true)}
                          disabled={!selectedRule}
                          className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                        >
                          <FileText className="h-4 w-4 mr-2" />
                          Report
                        </button>
                      </div>
                    </div>

                    {isRunningScenarioTest && (
                      <div className="flex items-center gap-2 rounded-md border border-blue-200 bg-blue-50 px-3 py-2 text-sm text-blue-700">
                        <Clock className="h-4 w-4 animate-spin" />
                        <span>
                          Testing rule on {selectedScenarios.size} scenario{selectedScenarios.size !== 1 ? 's' : ''}...
                        </span>
                      </div>
                    )}

                    {scenarioTestError && (
                      <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                        <p className="font-semibold">Failed to run scenario tests</p>
                        <p className="mt-1 text-red-600">{scenarioTestError}</p>
                      </div>
                    )}

                    {scenarioTestResults.length > 0 && (
                      <div className="space-y-4">
                        <div className="flex items-center justify-between">
                          <h5 className="text-sm font-semibold text-gray-900">
                            Scenario Test Results ({scenarioTestResults.length})
                          </h5>
                          {scenarioTestCompletedAt && (
                            <span className="text-xs text-slate-500">
                              Last run {scenarioTestCompletedAt.toLocaleTimeString()}
                            </span>
                          )}
                        </div>
                        {scenarioTestResults.map((result, resultIndex) => (
                          <div key={`scenario-result-${result.scenario}-${resultIndex}`} className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
                            <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                              <div>
                                <h6 className="text-sm font-semibold text-slate-900">
                                  {result.scenario}
                                </h6>
                                <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-slate-600">
                                  <span>{result.files_scanned} file{result.files_scanned === 1 ? '' : 's'} scanned</span>
                                  <span>•</span>
                                  <span>{result.violations.length} violation{result.violations.length === 1 ? '' : 's'}</span>
                                  <span>•</span>
                                  <span>{formatDurationMs(result.duration_ms)}</span>
                                </div>
                                {result.warning && (
                                  <p className="mt-2 inline-flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 px-2 py-1 text-xs text-amber-700">
                                    <AlertTriangle className="h-3.5 w-3.5 flex-shrink-0" />
                                    <span>{result.warning}</span>
                                  </p>
                                )}
                                {result.error && (
                                  <p className="mt-2 inline-flex items-start gap-2 rounded-md border border-red-200 bg-red-50 px-2 py-1 text-xs text-red-700">
                                    <XCircle className="h-3.5 w-3.5 flex-shrink-0" />
                                    <span>{result.error}</span>
                                  </p>
                                )}
                              </div>
                            </div>
                            <div className="mt-3 flex flex-wrap gap-2">
                              {result.targets.map((target, index) => {
                                const badgeClass = TARGET_BADGE_CLASSES[target] || 'bg-gray-100 text-gray-700'
                                const category = TARGET_CATEGORY_CONFIG.find(cfg => cfg.id === target)
                                return (
                                  <span
                                    key={`scenario-target-${result.scenario}-${target}-${index}`}
                                    className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${badgeClass}`}
                                  >
                                    {category?.label || target}
                                  </span>
                                )
                              })}
                            </div>
                            <div className="mt-4">
                              {result.violations.length === 0 ? (
                                <div className="rounded-md border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-700">
                                  No violations detected for this scenario.
                                </div>
                              ) : (
                                <div className="space-y-3">
                                  {result.violations.map((violation, index) => (
                                    <div key={violation.id || `${violation.file_path || 'violation'}-${index}`} className="rounded-md border border-slate-200 bg-slate-50 p-3">
                                      <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                                        <div className="space-y-1">
                                          <div className="flex flex-wrap items-center gap-2">
                                            <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${violation.severity === 'critical'
                                              ? 'bg-red-100 text-red-800'
                                              : violation.severity === 'high'
                                                ? 'bg-orange-100 text-orange-800'
                                                : violation.severity === 'medium'
                                                  ? 'bg-yellow-100 text-yellow-800'
                                                  : 'bg-green-100 text-green-800'
                                              }`}>
                                              {violation.severity}
                                            </span>
                                            {violation.file_path && (
                                              <span className="font-mono text-xs text-slate-600">{violation.file_path}</span>
                                            )}
                                            {violation.line_number ? (
                                              <span className="text-xs text-slate-500">Line {violation.line_number}</span>
                                            ) : null}
                                          </div>
                                          <p className="text-sm font-medium text-slate-900">
                                            {violation.title || violation.description || 'Rule violation detected'}
                                          </p>
                                          {violation.description && (
                                            <p className="text-sm text-slate-700">{violation.description}</p>
                                          )}
                                        </div>
                                        {violation.standard && (
                                          <span className="text-xs font-medium uppercase tracking-wide text-slate-500">
                                            {violation.standard}
                                          </span>
                                        )}
                                      </div>
                                      {violation.recommendation && (
                                        <p className="mt-2 text-xs text-slate-600">
                                          <span className="font-semibold text-slate-700">Recommended fix:</span> {violation.recommendation}
                                        </p>
                                      )}
                                      {violation.code_snippet && (
                                        <pre className="mt-2 max-h-40 overflow-auto rounded bg-slate-900 p-2 text-xs text-slate-100 whitespace-pre-wrap">
                                          {violation.code_snippet}
                                        </pre>
                                      )}
                                    </div>
                                  ))}
                                </div>
                              )}
                            </div>
                          </div>
                        ))}
                      </div>
                    )}

                    {agentError && (
                      <div className="bg-red-50 border border-red-200 rounded-md p-3 text-sm text-red-700">
                        {agentError}
                      </div>
                    )}

                    {isLaunchingAgent && !isAgentRunning && (
                      <div className="flex items-center gap-2 text-sm text-blue-700">
                        <Clock className="h-4 w-4 animate-spin" />
                        <span>Preparing AI agent…</span>
                      </div>
                    )}

                    {ruleAgents.map((agent: AgentInfo) => (
                      <div key={agent.id} className="border border-blue-200 bg-blue-50 rounded-lg p-4">
                        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                          <div className="flex items-center gap-3">
                            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-blue-600 text-white">
                              <Brain className="h-5 w-5" />
                            </div>
                            <div>
                              <p className="text-sm font-medium text-blue-900">{agent.label || agent.name}</p>
                              <p className="text-xs text-blue-700">{describeAgentAction(agent.action)} · {formatDuration(agent.duration_seconds)} elapsed</p>
                              {agent.metadata?.rule_file && (
                                <p className="text-xs text-blue-600">Rule file: {agent.metadata.rule_file}</p>
                              )}
                            </div>
                          </div>
                          <div className="flex items-center gap-3">
                            <button
                              onClick={() => handleStopAgent(agent.id)}
                              disabled={isLaunchingAgent}
                              className="inline-flex items-center px-3 py-2 border border-red-200 text-sm font-medium rounded-md text-red-600 hover:bg-red-50 disabled:opacity-50"
                            >
                              <CircleStop className="h-4 w-4 mr-2" />
                              Stop
                            </button>
                          </div>
                        </div>
                      </div>
                    ))}

                    {isRunningTests && (
                      <div className="flex items-center justify-center py-8">
                        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
                        <span className="ml-3 text-gray-600">Running tests...</span>
                      </div>
                    )}

                    {testResults?.tests && (
                      <div className="space-y-4">
                        <div className="bg-gray-50 rounded-lg p-4">
                          <div className="flex items-center justify-between text-sm">
                            <span>
                              <strong>Results:</strong> {testResults.passed} behaving correctly, {testResults.failed} not behaving as expected out of {testResults.total_tests} tests
                            </span>
                            {testResults.cached && (
                              <span className="text-gray-500">(Cached)</span>
                            )}
                          </div>
                        </div>

                        {testResults.tests.map((test: any, index: number) => (
                          <div key={index} className="border border-gray-200 rounded-lg overflow-hidden">
                            <div className={`px-4 py-3 border-b border-gray-200 ${test.passed ? 'bg-green-50 border-green-200' : 'bg-red-50 border-red-200'
                              }`}>
                              <div className="flex items-center justify-between">
                                <div className="flex items-center">
                                  {test.passed ? (
                                    <CheckCircle className="h-5 w-5 text-green-600 mr-2" />
                                  ) : (
                                    <XCircle className="h-5 w-5 text-red-600 mr-2" />
                                  )}
                                  <div>
                                    <h5 className="font-medium text-gray-900">{test.test_case.id}</h5>
                                    <p className="text-sm text-gray-600">{test.test_case.description}</p>
                                  </div>
                                </div>
                                <div className="flex items-center space-x-2">
                                  <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${test.test_case.should_fail ? 'bg-yellow-100 text-yellow-800' : 'bg-blue-100 text-blue-800'
                                    }`}>
                                    Expected: {test.test_case.should_fail ? 'Violations' : 'No Violations'}
                                  </span>
                                  <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${test.passed ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                                    }`}>
                                    {test.passed ? '✅ BEHAVED CORRECTLY' : '❌ UNEXPECTED BEHAVIOR'}
                                  </span>
                                </div>
                              </div>
                              {/* Show actual vs expected */}
                              <div className="mt-2 text-xs text-gray-600">
                                <span className="font-medium">Actual: </span>
                                {test.actual_violations && test.actual_violations.length > 0 ? (
                                  <span className="text-orange-600">{test.actual_violations.length} violation(s) found</span>
                                ) : (
                                  <span className="text-green-600">No violations found</span>
                                )}
                                {test.test_case.should_fail ? (
                                  test.actual_violations && test.actual_violations.length > 0 ? (
                                    <span className="ml-2 text-green-600">✓ As expected</span>
                                  ) : (
                                    <span className="ml-2 text-red-600">✗ Expected violations</span>
                                  )
                                ) : (
                                  test.actual_violations && test.actual_violations.length > 0 ? (
                                    <span className="ml-2 text-red-600">✗ Expected no violations</span>
                                  ) : (
                                    <span className="ml-2 text-green-600">✓ As expected</span>
                                  )
                                )}
                              </div>
                            </div>

                            <div className="p-4 space-y-4">
                              <div>
                                <h6 className="text-sm font-medium text-gray-900 mb-2">Test Input</h6>
                                <div className="bg-gray-900 rounded-lg overflow-hidden">
                                  <CodeBlock code={test.test_case.input} />
                                </div>
                              </div>

                              {test.actual_violations && test.actual_violations.length > 0 && (
                                <div>
                                  <h6 className="text-sm font-medium text-gray-900 mb-2">Violations Found</h6>
                                  <div className="space-y-2">
                                    {test.actual_violations.map((violation: any, vIndex: number) => (
                                      <div key={vIndex} className="bg-gray-50 rounded p-3 text-sm">
                                        <div className="flex items-start justify-between">
                                          <div>
                                            <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${violation.severity === 'critical' ? 'bg-red-100 text-red-800' :
                                                violation.severity === 'high' ? 'bg-orange-100 text-orange-800' :
                                                  violation.severity === 'medium' ? 'bg-yellow-100 text-yellow-800' :
                                                    'bg-blue-100 text-blue-800'
                                              }`}>
                                              {violation.severity}
                                            </span>
                                          </div>
                                        </div>
                                        <p className="mt-2 text-gray-800">{violation.message}</p>
                                      </div>
                                    ))}
                                  </div>
                                </div>
                              )}

                              {/* Execution Output */}
                              {test.execution_output && (
                                <div>
                                  <div className="flex items-center justify-between mb-2">
                                    <h6 className="text-sm font-medium text-gray-900">Execution Output</h6>
                                    <button
                                      onClick={() => setShowExecutionOutput(prev => ({
                                        ...prev,
                                        [index]: !prev[index]
                                      }))}
                                      className="inline-flex items-center text-xs text-gray-600 hover:text-gray-800"
                                    >
                                      {showExecutionOutput[index] ? (
                                        <>
                                          <EyeOff className="h-3 w-3 mr-1" />
                                          Hide
                                        </>
                                      ) : (
                                        <>
                                          <Eye className="h-3 w-3 mr-1" />
                                          Show
                                        </>
                                      )}
                                    </button>
                                  </div>

                                  {showExecutionOutput[index] && (
                                    <div className="bg-gray-50 rounded p-3 text-xs space-y-2">
                                      <div className="flex items-center gap-2 text-gray-600">
                                        <span className="font-medium">Method:</span>
                                        <span className={`px-2 py-0.5 rounded text-xs ${test.execution_output.method === 'judge0' ? 'bg-green-100 text-green-700' : 'bg-blue-100 text-blue-700'
                                          }`}>
                                          {test.execution_output.method}
                                        </span>
                                      </div>

                                      {test.execution_output.stdout && (
                                        <div>
                                          <span className="font-medium text-gray-700">Stdout:</span>
                                          <pre className="mt-1 bg-gray-900 text-gray-100 p-2 rounded text-xs whitespace-pre-wrap">{test.execution_output.stdout}</pre>
                                        </div>
                                      )}

                                      {test.execution_output.stderr && (
                                        <div>
                                          <span className="font-medium text-gray-700">Stderr:</span>
                                          <pre className="mt-1 bg-red-900 text-red-100 p-2 rounded text-xs whitespace-pre-wrap">{test.execution_output.stderr}</pre>
                                        </div>
                                      )}

                                      {test.execution_output.compile_output && (
                                        <div>
                                          <span className="font-medium text-gray-700">Compile Output:</span>
                                          <pre className="mt-1 bg-yellow-900 text-yellow-100 p-2 rounded text-xs whitespace-pre-wrap">{test.execution_output.compile_output}</pre>
                                        </div>
                                      )}

                                      {test.execution_output.exit_code !== undefined && (
                                        <div className="text-gray-600">
                                          <span className="font-medium">Exit Code:</span> {test.execution_output.exit_code}
                                        </div>
                                      )}
                                    </div>
                                  )}
                                </div>
                              )}
                            </div>
                          </div>
                        ))}
                      </div>
                    )}

                    {!isRunningTests && !testResults && (
                      <div className="text-center py-8 text-gray-500">
                        <TestTube className="h-12 w-12 mx-auto mb-3 text-gray-400" />
                        <p>Click "Run All Tests" to execute test cases for this rule</p>
                      </div>
                    )}
                  </div>
                )}

                {activeTab === 'playground' && (
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <h4 className="text-lg font-medium text-gray-900">Test Playground</h4>
                      <button
                        onClick={runPlaygroundTest}
                        disabled={isRunningPlayground || !playgroundCode.trim()}
                        className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                      >
                        {isRunningPlayground ? (
                          <Clock className="h-4 w-4 mr-2 animate-spin" />
                        ) : (
                          <Play className="h-4 w-4 mr-2" />
                        )}
                        {isRunningPlayground ? 'Testing...' : 'Test Code'}
                      </button>
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        Enter Go code to test against this rule:
                      </label>
                      <div className="h-64">
                        <CodeEditor
                          value={playgroundCode}
                          onChange={setPlaygroundCode}
                          language="go"
                          placeholder="func HandleRequest(w http.ResponseWriter, r *http.Request) {
    // Your code here
}"
                          className="h-full"
                        />
                      </div>
                    </div>

                    {playgroundResult && (
                      <div className="border border-gray-200 rounded-lg overflow-hidden">
                        <div className={`px-4 py-3 border-b border-gray-200 ${playgroundResult.error ? 'bg-red-50' :
                            playgroundResult.actual_violations?.length > 0 ? 'bg-yellow-50' : 'bg-green-50'
                          }`}>
                          <h5 className="font-medium text-gray-900">Test Result</h5>
                        </div>

                        <div className="p-4">
                          {playgroundResult.error ? (
                            <div className="text-red-800">
                              <strong>Error:</strong> {playgroundResult.error}
                            </div>
                          ) : playgroundResult.actual_violations?.length > 0 ? (
                            <div className="space-y-3">
                              <p className="text-gray-800">
                                <strong>Found {playgroundResult.actual_violations.length} violation(s):</strong>
                              </p>
                              {playgroundResult.actual_violations.map((violation: any, index: number) => (
                                <div key={index} className="bg-gray-50 rounded p-3">
                                  <div className="flex items-start justify-between">
                                    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${violation.severity === 'critical' ? 'bg-red-100 text-red-800' :
                                        violation.severity === 'high' ? 'bg-orange-100 text-orange-800' :
                                          violation.severity === 'medium' ? 'bg-yellow-100 text-yellow-800' :
                                            'bg-blue-100 text-blue-800'
                                      }`}>
                                      {violation.severity}
                                    </span>
                                  </div>
                                  <p className="mt-2 text-gray-800">{violation.message}</p>
                                </div>
                              ))}
                            </div>
                          ) : (
                            <div className="text-green-800">
                              <CheckCircle className="h-5 w-5 inline-block mr-2" />
                              <strong>No violations found!</strong> Your code follows this rule.
                            </div>
                          )}

                          {/* Execution Output for Playground */}
                          {playgroundResult.test_result?.execution_output && (
                            <div className="mt-4 pt-4 border-t border-gray-200">
                              <h6 className="text-sm font-medium text-gray-900 mb-2">Execution Output</h6>
                              <div className="bg-gray-50 rounded p-3 text-xs space-y-2">
                                <div className="flex items-center gap-2 text-gray-600">
                                  <span className="font-medium">Method:</span>
                                  <span className={`px-2 py-0.5 rounded text-xs ${playgroundResult.test_result.execution_output.method === 'judge0' ? 'bg-green-100 text-green-700' : 'bg-blue-100 text-blue-700'
                                    }`}>
                                    {playgroundResult.test_result.execution_output.method}
                                  </span>
                                </div>

                                {playgroundResult.test_result.execution_output.stdout && (
                                  <div>
                                    <span className="font-medium text-gray-700">Stdout:</span>
                                    <pre className="mt-1 bg-gray-900 text-gray-100 p-2 rounded text-xs whitespace-pre-wrap">{playgroundResult.test_result.execution_output.stdout}</pre>
                                  </div>
                                )}

                                {playgroundResult.test_result.execution_output.stderr && (
                                  <div>
                                    <span className="font-medium text-gray-700">Stderr:</span>
                                    <pre className="mt-1 bg-red-900 text-red-100 p-2 rounded text-xs whitespace-pre-wrap">{playgroundResult.test_result.execution_output.stderr}</pre>
                                  </div>
                                )}

                                {playgroundResult.test_result.execution_output.compile_output && (
                                  <div>
                                    <span className="font-medium text-gray-700">Compile Output:</span>
                                    <pre className="mt-1 bg-yellow-900 text-yellow-100 p-2 rounded text-xs whitespace-pre-wrap">{playgroundResult.test_result.execution_output.compile_output}</pre>
                                  </div>
                                )}

                                {playgroundResult.test_result.execution_output.exit_code !== undefined && (
                                  <div className="text-gray-600">
                                    <span className="font-medium">Exit Code:</span> {playgroundResult.test_result.execution_output.exit_code}
                                  </div>
                                )}
                              </div>
                            </div>
                          )}
                        </div>
                      </div>
                    )}

                    {!playgroundResult && (
                      <div className="text-center py-8 text-gray-500">
                        <Code className="h-12 w-12 mx-auto mb-3 text-gray-400" />
                        <p>Enter some Go code above and click "Test Code" to validate it against this rule</p>
                      </div>
                    )}
                  </div>
                )}
              </div>

              <div className="mt-5 sm:mt-4 sm:flex sm:flex-row-reverse">
                <button
                  type="button"
                  className="inline-flex w-full justify-center rounded-md bg-blue-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-blue-500 sm:ml-3 sm:w-auto"
                  onClick={() => setSelectedRule(null)}
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
