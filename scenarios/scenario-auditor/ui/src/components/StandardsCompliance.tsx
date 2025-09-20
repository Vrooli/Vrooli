import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { 
  Shield, 
  Search, 
  AlertCircle, 
  CheckCircle,
  XCircle,
  RefreshCw,
  FileCode,
  AlertTriangle,
  ChevronRight,
  Play,
  Info,
  Zap,
  Target,
  X,
  Award,
  Settings,
  TestTube,
  Globe,
  Code,
  Bot,
  Loader2
} from 'lucide-react'
import clsx from 'clsx'
import { format } from 'date-fns'
import { apiService } from '../services/api'
import { Card } from './common/Card'
import { Badge } from './common/Badge'
import { Dialog } from './common/Dialog'
import { AgentInfo, StandardsScanStatus, StandardsViolation } from '@/types/api'

interface CompletedAgentStatus {
  agentId: string
  scenario: string
  startTime: Date
  status: 'completed' | 'failed'
  message?: string
}

export default function StandardsCompliance() {
  const [selectedScenario, setSelectedScenario] = useState<string>('')
  const [checkType, setCheckType] = useState<'quick' | 'full' | 'targeted'>('full')
  const [selectedViolation, setSelectedViolation] = useState<StandardsViolation | null>(null)
  const [showCheckTypeInfo, setShowCheckTypeInfo] = useState(false)
  const [showDisabledTooltip, setShowDisabledTooltip] = useState(false)
  const [activeScan, setActiveScan] = useState<{ id: string; scenario: string } | null>(null)
  const [lastScanStatus, setLastScanStatus] = useState<StandardsScanStatus | null>(null)
  const [searchQuery, setSearchQuery] = useState<string>('')
  const [severityFilter, setSeverityFilter] = useState<string | null>(null)
  const [completedAgents, setCompletedAgents] = useState<Map<string, CompletedAgentStatus>>(new Map())
  const [launchingScenario, setLaunchingScenario] = useState<string | null>(null)
  const prevAgentsRef = useRef<Map<string, string>>(new Map())
  const queryClient = useQueryClient()
  const { data: activeAgentsData } = useQuery({
    queryKey: ['activeAgents'],
    queryFn: () => apiService.getActiveAgents(),
    refetchInterval: 2000,
  })

  const standardsAgents = useMemo(() => {
    const map = new Map<string, AgentInfo>()
    const agents = activeAgentsData?.agents ?? []
    for (const agent of agents as AgentInfo[]) {
      if (agent.action !== 'standards_fix') continue
      const scenarioId = agent.metadata?.scenario || agent.scenario || agent.rule_id
      if (!scenarioId) continue
      map.set(scenarioId, agent)
    }
    return map
  }, [activeAgentsData])

  const { data: scenarios } = useQuery({
    queryKey: ['scenarios'],
    queryFn: apiService.getScenarios,
  })

  const { data: violations, isLoading: loadingViolations, refetch } = useQuery({
    queryKey: ['standardsViolations', selectedScenario],
    queryFn: () => apiService.getStandardsViolations(selectedScenario || undefined),
  })

  const checkMutation = useMutation({
    mutationFn: (scenario: string) => apiService.checkStandards(scenario, { type: checkType }),
    onSuccess: (response, scenario) => {
      setActiveScan({ id: response.job_id, scenario })
      setLastScanStatus(null)
    },
  })

  const scanStatusQuery = useQuery({
    queryKey: ['standardsScanStatus', activeScan?.id],
    queryFn: () => apiService.getStandardsCheckStatus(activeScan!.id),
    enabled: !!activeScan,
    refetchInterval: activeScan ? 1000 : false,
  })

  const cancelScanMutation = useMutation({
    mutationFn: (jobId: string) => apiService.cancelStandardsScan(jobId),
  })

  const activeScanStatus = scanStatusQuery.data

  useEffect(() => {
    if (!activeScan || !activeScanStatus) {
      return
    }
    if (['completed', 'failed', 'cancelled'].includes(activeScanStatus.status)) {
      setLastScanStatus(activeScanStatus)
      setActiveScan(null)
      queryClient.invalidateQueries({ queryKey: ['standardsViolations'] })
      queryClient.invalidateQueries({ queryKey: ['scenarios'] })
      refetch()
    }
  }, [activeScan, activeScanStatus, queryClient, refetch])

  // Auto-refresh violations when agents complete successfully
  useEffect(() => {
    const completedSuccessfully = Array.from(completedAgents.values())
      .filter(agent => agent.status === 'completed')
    
    if (completedSuccessfully.length > 0) {
      const timer = setTimeout(() => {
        refetch()
        // Clear completed agents after refresh
        setCompletedAgents(new Map())
      }, 3000)
      
      return () => clearTimeout(timer)
    }
  }, [completedAgents, refetch])

  // Helper function to get elapsed time
  const getElapsedTime = (startTime: Date): string => {
    const now = new Date()
    const elapsed = Math.floor((now.getTime() - startTime.getTime()) / 1000)
    
    if (elapsed < 60) {
      return `${elapsed}s`
    } else if (elapsed < 3600) {
      const minutes = Math.floor(elapsed / 60)
      const seconds = elapsed % 60
      return `${minutes}m ${seconds}s`
    } else {
      const hours = Math.floor(elapsed / 3600)
      const minutes = Math.floor((elapsed % 3600) / 60)
      return `${hours}h ${minutes}m`
    }
  }

  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'critical':
        return <XCircle className="h-5 w-5 text-danger-600" />
      case 'high':
        return <AlertTriangle className="h-5 w-5 text-warning-600" />
      case 'medium':
        return <AlertCircle className="h-5 w-5 text-warning-500" />
      case 'low':
        return <AlertCircle className="h-5 w-5 text-blue-500" />
      default:
        return null
    }
  }

  const getStandardIcon = (standard: string) => {
    switch (standard.toLowerCase()) {
      case 'vrooli lifecycle management':
        return <Settings className="h-4 w-4" />
      case 'test coverage':
        return <TestTube className="h-4 w-4" />
      case 'code organization':
        return <Code className="h-4 w-4" />
      case 'configuration management':
        return <Globe className="h-4 w-4" />
      case 'api versioning':
        return <Shield className="h-4 w-4" />
      default:
        return <Award className="h-4 w-4" />
    }
  }

  // Filter violations based on search query and severity
  const allViolations = violations ?? []

  const filteredViolations = allViolations.filter(violation => {
    // Apply severity filter first
    if (severityFilter !== null && severityFilter !== undefined) {
      if (violation.severity !== severityFilter) {
        return false
      }
    }
    
    // Apply search filter
    if (!searchQuery.trim()) return true
    
    const searchLower = searchQuery.toLowerCase()
    return (
      violation.title?.toLowerCase().includes(searchLower) ||
      violation.description?.toLowerCase().includes(searchLower) ||
      violation.type?.toLowerCase().includes(searchLower) ||
      violation.severity?.toLowerCase().includes(searchLower) ||
      violation.file_path?.toLowerCase().includes(searchLower) ||
      violation.scenario_name?.toLowerCase().includes(searchLower) ||
      violation.standard?.toLowerCase().includes(searchLower) ||
      violation.recommendation?.toLowerCase().includes(searchLower)
    )
  })

  const groupedViolations = filteredViolations.reduce((acc, violation) => {
    const key = violation.scenario_name
    if (!acc[key]) acc[key] = []
    acc[key].push(violation)
    return acc
  }, {} as Record<string, StandardsViolation[]>)

  const stats = allViolations.reduce(
    (acc, violation) => {
      acc.total += 1
      if (violation.severity === 'critical') acc.critical += 1
      else if (violation.severity === 'high') acc.high += 1
      else if (violation.severity === 'medium') acc.medium += 1
      else if (violation.severity === 'low') acc.low += 1
      return acc
    },
    { total: 0, critical: 0, high: 0, medium: 0, low: 0 }
  )

  const formatDurationSeconds = (seconds?: number) => {
    if (!seconds || seconds <= 0) return '<1s'
    const totalSeconds = Math.max(1, Math.round(seconds))
    const mins = Math.floor(totalSeconds / 60)
    const secs = totalSeconds % 60
    if (mins > 0) {
      return `${mins}m ${secs}s`
    }
    return `${secs}s`
  }

  const handleAgentCompletion = useCallback(async (agentId: string, scenario: string) => {
    try {
      const status = await apiService.getClaudeFixStatus(agentId)
      const agent = status.agent as AgentInfo | undefined
      const finalStatus = (status.status || agent?.status || 'completed').toLowerCase() === 'failed' ? 'failed' : 'completed'
      const message = agent?.error || (finalStatus === 'failed' ? 'Fix failed - review agent logs' : 'Fix completed successfully')

      setCompletedAgents(prev => {
        const updated = new Map(prev)
        updated.set(scenario, {
          agentId,
          scenario,
          startTime: agent?.started_at ? new Date(agent.started_at) : new Date(),
          status: finalStatus,
          message,
        })
        return updated
      })

      if (launchingScenario === scenario) {
        setLaunchingScenario(null)
      }

      queryClient.invalidateQueries({ queryKey: ['standardsViolations'] })
      queryClient.invalidateQueries({ queryKey: ['scenarios'] })
      refetch()
    } catch (error) {
      setCompletedAgents(prev => {
        const updated = new Map(prev)
        updated.set(scenario, {
          agentId,
          scenario,
          startTime: new Date(),
          status: 'failed',
          message: error instanceof Error ? error.message : 'Unable to determine agent status',
        })
        return updated
      })
      if (launchingScenario === scenario) {
        setLaunchingScenario(null)
      }
    }
  }, [launchingScenario, queryClient, refetch])

  useEffect(() => {
    const prev = prevAgentsRef.current
    const currentScenarios = new Set<string>()
    standardsAgents.forEach((_, scenario) => {
      currentScenarios.add(scenario)
    })

    // Detect completed agents
    prev.forEach((agentId, scenario) => {
      if (!currentScenarios.has(scenario)) {
        void handleAgentCompletion(agentId, scenario)
        prev.delete(scenario)
      }
    })

    // Track currently running agents
    standardsAgents.forEach((agent, scenario) => {
      prev.set(scenario, agent.id)
    })
  }, [standardsAgents, handleAgentCompletion])

  useEffect(() => {
    if (completedAgents.size === 0) return
    const timer = setTimeout(() => {
      setCompletedAgents(new Map())
    }, 6000)
    return () => clearTimeout(timer)
  }, [completedAgents])

  return (
    <div className="space-y-6 animate-in">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-dark-900">Standards Compliance</h1>
        <p className="text-dark-500 mt-1">Check scenarios for compliance with Vrooli development standards</p>
        <div className="mt-2 flex items-start gap-2 rounded-lg bg-blue-50 border border-blue-200 p-3">
          <Award className="h-4 w-4 text-blue-600 flex-shrink-0 mt-0.5" />
          <div className="text-xs text-blue-800">
            <strong>Standards Checker:</strong> Validates lifecycle protection, test coverage, code organization, configuration management, and API versioning.
          </div>
        </div>
      </div>

      {/* Check Controls */}
      <Card>
        <div className="flex flex-col sm:flex-row gap-4">
          <div className="flex-1">
            <label className="block text-sm font-medium text-dark-700 mb-2">
              Select Scenario
            </label>
            <select
              value={selectedScenario}
              onChange={(e) => setSelectedScenario(e.target.value)}
              className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
            >
              <option value="">All Scenarios</option>
              {scenarios?.map((scenario) => (
                <option key={scenario.name} value={scenario.name}>
                  {scenario.name}
                </option>
              ))}
            </select>
          </div>
          
          <div>
            <label className="flex items-center gap-2 text-sm font-medium text-dark-700 mb-2">
              Check Type
              <button
                onClick={() => setShowCheckTypeInfo(true)}
                className="text-dark-400 hover:text-primary-600 transition-colors"
                aria-label="Learn about check types"
              >
                <Info className="h-4 w-4" />
              </button>
            </label>
            <div className="flex gap-2">
              {(['quick', 'full', 'targeted'] as const).map((type) => (
                <button
                  key={type}
                  onClick={() => setCheckType(type)}
                  className={clsx(
                    'px-4 py-2 rounded-lg text-sm font-medium capitalize transition-colors',
                    checkType === type
                      ? 'bg-primary-500 text-white'
                      : 'bg-dark-100 text-dark-700 hover:bg-dark-200'
                  )}
                >
                  {type}
                </button>
              ))}
            </div>
          </div>
          
          <div className="flex items-end gap-2">
            <div 
              className="relative"
              onMouseEnter={() => setShowDisabledTooltip(true)}
              onMouseLeave={() => setShowDisabledTooltip(false)}
            >
              <button
                onClick={() => {
                  // For targeted checks, require a specific scenario
                  // For quick/full checks, allow "all scenarios" (empty string)
                  const targetToCheck = selectedScenario || (checkType !== 'targeted' ? 'all' : '')
                  if (targetToCheck) {
                    checkMutation.mutate(targetToCheck)
                  }
                }}
                disabled={(checkType === 'targeted' && !selectedScenario) || checkMutation.isPending || !!activeScan}
                className="flex items-center gap-2 rounded-lg bg-primary-500 px-4 py-2 text-sm font-medium text-white hover:bg-primary-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {checkMutation.isPending ? (
                  <>
                    <RefreshCw className="h-4 w-4 animate-spin" />
                    Starting...
                  </>
                ) : activeScan ? (
                  <>
                    <RefreshCw className="h-4 w-4 animate-spin" />
                    Scanning...
                  </>
                ) : (
                  <>
                    <Play className="h-4 w-4" />
                    Start Check
                  </>
                )}
              </button>
              
              {/* Tooltip for disabled state (only for targeted checks without selection) */}
              {showDisabledTooltip && checkType === 'targeted' && !selectedScenario && !checkMutation.isPending && !activeScan && (
                <div className="absolute z-10 left-0 bottom-full mb-2 w-56 p-2 bg-dark-900 text-white text-xs rounded-lg shadow-lg">
                  <div className="absolute bottom-0 left-4 w-0 h-0 border-l-4 border-r-4 border-t-4 border-transparent border-t-dark-900 transform translate-y-full"></div>
                  Targeted checks require a specific scenario. Please select one from the dropdown.
                </div>
              )}
              
              {/* Tooltip for checking state */}
              {showDisabledTooltip && (checkMutation.isPending || !!activeScan) && (
                <div className="absolute z-10 left-0 bottom-full mb-2 w-40 p-2 bg-dark-900 text-white text-xs rounded-lg shadow-lg">
                  <div className="absolute bottom-0 left-4 w-0 h-0 border-l-4 border-r-4 border-t-4 border-transparent border-t-dark-900 transform translate-y-full"></div>
                  Standards check in progress...
                </div>
              )}
            </div>
            
            <button
              onClick={() => refetch()}
              className="flex items-center justify-center rounded-lg bg-dark-100 p-2 text-dark-700 hover:bg-dark-200 transition-colors"
              title="Refresh violations"
            >
              <RefreshCw className="h-4 w-4" />
            </button>
          </div>
        </div>

        {checkMutation.isError && !activeScan && (
          <div className="mt-4">
            <div className="rounded-lg bg-danger-50 p-3 border border-danger-200">
              <div className="flex items-start gap-2">
                <XCircle className="h-4 w-4 text-danger-600 mt-0.5" />
                <p className="text-sm text-danger-700">
                  {(checkMutation.error as Error)?.message || 'Failed to start standards check'}
                </p>
              </div>
            </div>
          </div>
        )}

        {activeScan && (
          <div className="mt-4">
            <div className="rounded-lg border border-primary-100 bg-primary-50/60 p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-semibold text-primary-700">
                    {activeScanStatus?.status === 'cancelling' ? 'Cancelling standards scan...' : 'Standards scan in progress'}
                  </p>
                  <p className="text-xs text-primary-600 mt-1">
                    {activeScanStatus?.message || 'Analyzing scenarios for standards compliance'}
                  </p>
                </div>
                <button
                  onClick={() => activeScan && cancelScanMutation.mutate(activeScan.id)}
                  disabled={cancelScanMutation.isPending || activeScanStatus?.status === 'cancelling'}
                  className="flex items-center gap-1 rounded-md border border-primary-300 px-3 py-1 text-xs font-medium text-primary-700 hover:bg-primary-100/70 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {cancelScanMutation.isPending ? (
                    <>
                      <Loader2 className="h-3 w-3 animate-spin" />
                      Canceling...
                    </>
                  ) : (
                    <>
                      <X className="h-3 w-3" />
                      Cancel Scan
                    </>
                  )}
                </button>
              </div>

              <div className="mt-4 grid gap-3 sm:grid-cols-3 text-sm text-primary-800">
                <div>
                  <p className="text-xs uppercase tracking-wide text-primary-600">Scenarios</p>
                  <p className="font-semibold">
                    {activeScanStatus?.processed_scenarios ?? 0}
                    {activeScanStatus?.total_scenarios ? ` / ${activeScanStatus.total_scenarios}` : ''}
                  </p>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-wide text-primary-600">Files Processed</p>
                  <p className="font-semibold">{activeScanStatus?.processed_files ?? 0}</p>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-wide text-primary-600">Elapsed Time</p>
                  <p className="font-semibold">{formatDurationSeconds(activeScanStatus?.elapsed_seconds)}</p>
                </div>
              </div>

              {activeScanStatus?.current_scenario && (
                <p className="mt-3 text-xs text-primary-600">
                  <span className="font-medium text-primary-700">Current:</span> {activeScanStatus.current_scenario}
                  {activeScanStatus.current_file ? ` / ${activeScanStatus.current_file}` : ''}
                </p>
              )}
            </div>
          </div>
        )}

        {lastScanStatus && (
          <div className="mt-4">
            <div
              className={clsx(
                'rounded-lg p-3 border',
                lastScanStatus.status === 'completed' && 'bg-success-50 border-success-200',
                lastScanStatus.status === 'failed' && 'bg-danger-50 border-danger-200',
                lastScanStatus.status === 'cancelled' && 'bg-dark-50 border-dark-200'
              )}
            >
              <div className="flex items-start gap-2">
                {lastScanStatus.status === 'completed' ? (
                  <CheckCircle className="h-4 w-4 text-success-700 mt-0.5" />
                ) : lastScanStatus.status === 'failed' ? (
                  <XCircle className="h-4 w-4 text-danger-600 mt-0.5" />
                ) : (
                  <Info className="h-4 w-4 text-dark-500 mt-0.5" />
                )}
                <div className="flex-1 text-sm">
                  <p
                    className={clsx(
                      lastScanStatus.status === 'completed' && 'text-success-700 font-medium',
                      lastScanStatus.status === 'failed' && 'text-danger-700 font-medium',
                      lastScanStatus.status === 'cancelled' && 'text-dark-700 font-medium'
                    )}
                  >
                    {lastScanStatus.message || lastScanStatus.result?.message || 'Standards scan finished'}
                  </p>
                  <div className="mt-2 text-xs text-dark-500 space-y-1">
                    <p>
                      <span className="font-medium">Scenarios processed:</span> {lastScanStatus.processed_scenarios}
                      {lastScanStatus.total_scenarios ? ` / ${lastScanStatus.total_scenarios}` : ''}
                    </p>
                    <p>
                      <span className="font-medium">Files scanned:</span> {lastScanStatus.processed_files}
                    </p>
                    <p>
                      <span className="font-medium">Duration:</span>{' '}
                      {formatDurationSeconds(lastScanStatus.result?.duration_seconds ?? lastScanStatus.elapsed_seconds)}
                    </p>
                    {lastScanStatus.status === 'failed' && lastScanStatus.error && (
                      <p className="text-danger-600">
                        <span className="font-medium">Error:</span> {lastScanStatus.error}
                      </p>
                    )}
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </Card>

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-5">
        <button
          onClick={() => setSeverityFilter(null)}
          className={clsx(
            "text-center p-4 rounded-lg border-2 transition-all",
            severityFilter === null 
              ? "border-primary-500 bg-primary-50 shadow-lg" 
              : "border-dark-200 bg-white hover:border-dark-300 hover:shadow-md"
          )}
        >
          <p className="text-2xl font-bold text-dark-900">{stats.total}</p>
          <p className="text-sm text-dark-500">Total</p>
        </button>
        
        <button
          onClick={() => setSeverityFilter(severityFilter === 'critical' ? null : 'critical')}
          className={clsx(
            "text-center p-4 rounded-lg border-2 transition-all",
            severityFilter === 'critical'
              ? "border-danger-500 bg-danger-100 shadow-lg"
              : "border-danger-200 bg-danger-50/50 hover:bg-danger-100 hover:shadow-md"
          )}
        >
          <p className="text-2xl font-bold text-danger-700">{stats.critical}</p>
          <p className="text-sm text-danger-600">Critical</p>
        </button>
        
        <button
          onClick={() => setSeverityFilter(severityFilter === 'high' ? null : 'high')}
          className={clsx(
            "text-center p-4 rounded-lg border-2 transition-all",
            severityFilter === 'high'
              ? "border-warning-500 bg-warning-100 shadow-lg"
              : "border-warning-200 bg-warning-50/50 hover:bg-warning-100 hover:shadow-md"
          )}
        >
          <p className="text-2xl font-bold text-warning-700">{stats.high}</p>
          <p className="text-sm text-warning-600">High</p>
        </button>
        
        <button
          onClick={() => setSeverityFilter(severityFilter === 'medium' ? null : 'medium')}
          className={clsx(
            "text-center p-4 rounded-lg border-2 transition-all",
            severityFilter === 'medium'
              ? "border-yellow-500 bg-yellow-100 shadow-lg"
              : "border-yellow-200 bg-yellow-50/50 hover:bg-yellow-100 hover:shadow-md"
          )}
        >
          <p className="text-2xl font-bold text-yellow-700">{stats.medium}</p>
          <p className="text-sm text-yellow-600">Medium</p>
        </button>
        
        <button
          onClick={() => setSeverityFilter(severityFilter === 'low' ? null : 'low')}
          className={clsx(
            "text-center p-4 rounded-lg border-2 transition-all",
            severityFilter === 'low'
              ? "border-blue-500 bg-blue-100 shadow-lg"
              : "border-blue-200 bg-blue-50/50 hover:bg-blue-100 hover:shadow-md"
          )}
        >
          <p className="text-2xl font-bold text-blue-700">{stats.low}</p>
          <p className="text-sm text-blue-600">Low</p>
        </button>
      </div>

      {/* Violations List */}
      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <div className="mb-4">
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-lg font-semibold text-dark-900">Standards Violations</h2>
              {severityFilter && (
                <button
                  onClick={() => setSeverityFilter(null)}
                  className="flex items-center gap-1 px-3 py-1 rounded-full bg-primary-100 text-primary-700 hover:bg-primary-200 transition-colors text-sm"
                >
                  <span>Filtering: {severityFilter}</span>
                  <X className="h-3 w-3" />
                </button>
              )}
            </div>
            
            {/* Search Bar */}
            {allViolations.length > 0 && (
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-dark-400" />
                <input
                  type="text"
                  placeholder="Search violations by title, type, severity, file path, standard..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full pl-10 pr-10 py-2 rounded-lg border border-dark-300 bg-white text-dark-900 placeholder-dark-400 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
                />
                {searchQuery && (
                  <button
                    onClick={() => setSearchQuery('')}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-dark-400 hover:text-dark-600 transition-colors"
                    aria-label="Clear search"
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
              </div>
            )}
            
            {/* Filter/Search Results Info */}
            {(searchQuery || severityFilter) && allViolations.length > 0 && (
              <div className="mt-2 text-sm text-dark-600">
                Showing {filteredViolations.length} of {allViolations.length} violations
                {filteredViolations.length === 0 && (
                  <span className="text-warning-600 ml-2">No matches found. Try different filters.</span>
                )}
              </div>
            )}
          </div>
          
          {loadingViolations ? (
            <div className="space-y-3">
              {[1, 2, 3].map(i => (
                <div key={i} className="h-24 bg-dark-100 rounded-lg animate-pulse" />
              ))}
            </div>
          ) : filteredViolations.length > 0 ? (
            <div className="space-y-4">
              {Object.entries(groupedViolations).map(([scenario, scenarioViolations]) => {
                const scenarioAgent = standardsAgents.get(scenario)
                const isLaunching = launchingScenario === scenario
                const hasActiveAgent = Boolean(scenarioAgent)
                return (
                <div key={scenario} className="border border-dark-200 rounded-lg">
                  <div className="bg-dark-50 px-4 py-2 border-b border-dark-200 flex items-center justify-between">
                    <h3 className="font-medium text-dark-900">{scenario}</h3>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-dark-500">
                        {scenarioViolations.length} violation{scenarioViolations.length !== 1 ? 's' : ''}
                      </span>
                      <button
                        onClick={async () => {
                          if (!confirm(`Trigger Claude agent to fix ${scenarioViolations.length} violation${scenarioViolations.length !== 1 ? 's' : ''} in ${scenario}?`)) {
                            return
                          }

                          setLaunchingScenario(scenario)
                          setCompletedAgents(prev => {
                            const updated = new Map(prev)
                            updated.delete(scenario)
                            return updated
                          })

                          try {
                            const result = await apiService.triggerClaudeFix(
                              scenario,
                              'standards',
                              scenarioViolations.map(v => v.id)
                            )

                            if (!result.success || !result.agent) {
                              setCompletedAgents(prev => {
                                const updated = new Map(prev)
                                updated.set(scenario, {
                                  agentId: result.fix_id || 'unknown',
                                  scenario,
                                  startTime: new Date(),
                                  status: 'failed',
                                  message: result.error || result.message || 'Failed to start agent',
                                })
                                return updated
                              })
                              setLaunchingScenario(null)
                              return
                            }

                            await queryClient.invalidateQueries({ queryKey: ['activeAgents'] })
                          } catch (error) {
                            setCompletedAgents(prev => {
                              const updated = new Map(prev)
                              updated.set(scenario, {
                                agentId: 'unknown',
                                scenario,
                                startTime: new Date(),
                                status: 'failed',
                                message: error instanceof Error ? error.message : 'Failed to trigger fix',
                              })
                              return updated
                            })
                            setLaunchingScenario(null)
                          }
                        }}
                        disabled={hasActiveAgent || isLaunching}
                        className="flex items-center gap-1 px-3 py-1 rounded-lg bg-primary-500 text-white hover:bg-primary-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors text-xs font-medium"
                        title="Fix violations with Claude agent"
                      >
                        {hasActiveAgent ? (
                          <>
                            <Loader2 className="h-3 w-3 animate-spin" />
                            Fixing... ({formatDurationSeconds(scenarioAgent?.duration_seconds)})
                          </>
                        ) : isLaunching ? (
                          <>
                            <Loader2 className="h-3 w-3 animate-spin" />
                            Starting...
                          </>
                        ) : (
                          <>
                            <Bot className="h-3 w-3" />
                            Fix with AI
                          </>
                        )}
                      </button>
                    </div>
                  </div>
                  {completedAgents.get(scenario) && (
                    <div className={clsx(
                      "px-4 py-2 text-xs",
                      completedAgents.get(scenario)?.status === 'completed' ? "bg-success-50 text-success-700" : "bg-danger-50 text-danger-700"
                    )}>
                      <div className="flex items-center justify-between">
                        <span>{completedAgents.get(scenario)?.message}</span>
                        <span className="text-xs opacity-75">
                          Completed in {getElapsedTime(completedAgents.get(scenario)!.startTime)}
                        </span>
                      </div>
                    </div>
                  )}
                  <div className="divide-y divide-dark-100">
                    {scenarioViolations.map((violation, idx) => (
                      <button
                        key={`${violation.id}-${violation.scenario_name}-${violation.line_number}-${idx}`}
                        onClick={() => setSelectedViolation(violation)}
                        className="w-full px-4 py-3 hover:bg-dark-50 transition-colors text-left"
                      >
                        <div className="flex items-start gap-3">
                          {getSeverityIcon(violation.severity)}
                          <div className="flex-1">
                            <div className="flex items-start justify-between">
                              <div>
                                <p className="font-medium text-dark-900">{violation.title}</p>
                                <p className="text-sm text-dark-600 mt-1 line-clamp-2">
                                  {violation.description}
                                </p>
                                <div className="flex items-center gap-4 mt-2">
                                  <div className="flex items-center gap-1 text-xs text-dark-500">
                                    <FileCode className="h-3 w-3" />
                                    {violation.file_path}:{violation.line_number}
                                  </div>
                                  <div className="flex items-center gap-1 text-xs text-dark-500">
                                    {getStandardIcon(violation.standard)}
                                    {violation.standard}
                                  </div>
                                  <Badge variant={violation.severity === 'critical' ? 'danger' : violation.severity === 'high' ? 'warning' : 'default'} size="sm">
                                    {violation.type}
                                  </Badge>
                                </div>
                              </div>
                              <ChevronRight className="h-4 w-4 text-dark-400 flex-shrink-0" />
                            </div>
                          </div>
                        </div>
                      </button>
                    ))}
                  </div>
                </div>
                )
              })}
            </div>
          ) : (
            <div className="text-center py-12">
              {searchQuery || severityFilter ? (
                <>
                  <Search className="h-12 w-12 text-dark-400 mx-auto mb-3" />
                  <p className="text-dark-700 font-medium">No matching violations</p>
                  <p className="text-sm text-dark-500 mt-1">
                    Try adjusting your {searchQuery ? 'search terms' : 'filters'} or clear them to see all violations
                  </p>
                  <button
                    onClick={() => {
                      setSearchQuery('')
                      setSeverityFilter(null)
                    }}
                    className="mt-3 px-4 py-2 text-sm bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition-colors"
                  >
                    Clear Filters
                  </button>
                </>
              ) : (
                <>
                  <Award className="h-12 w-12 text-success-500 mx-auto mb-3" />
                  <p className="text-dark-700 font-medium">No standards violations found</p>
                  <p className="text-sm text-dark-500 mt-1">
                    {selectedScenario ? 'This scenario appears to be compliant' : 'Select a scenario and run a standards check'}
                  </p>
                </>
              )}
            </div>
          )}
        </Card>

        {/* Violation Details */}
        <Card>
          <h2 className="text-lg font-semibold text-dark-900 mb-4">Violation Details</h2>
          
          {selectedViolation ? (
            <div className="space-y-4">
              <div>
                <div className="flex items-center gap-2 mb-2">
                  {getSeverityIcon(selectedViolation.severity)}
                  <Badge 
                    variant={
                      selectedViolation.severity === 'critical' ? 'danger' :
                      selectedViolation.severity === 'high' ? 'warning' : 
                      'default'
                    }
                  >
                    {selectedViolation.severity}
                  </Badge>
                </div>
                <h3 className="font-medium text-dark-900">{selectedViolation.title}</h3>
              </div>
              
              <div>
                <p className="text-sm font-medium text-dark-700 mb-1">Description</p>
                <p className="text-sm text-dark-600">{selectedViolation.description}</p>
              </div>

              <div>
                <p className="text-sm font-medium text-dark-700 mb-1">Standard</p>
                <div className="flex items-center gap-2 text-sm text-dark-600">
                  {getStandardIcon(selectedViolation.standard)}
                  {selectedViolation.standard}
                </div>
              </div>
              
              <div>
                <p className="text-sm font-medium text-dark-700 mb-1">Location</p>
                <div className="bg-dark-50 rounded-lg p-2">
                  <p className="text-xs font-mono text-dark-600">
                    {selectedViolation.file_path}:{selectedViolation.line_number}
                  </p>
                </div>
              </div>
              
              {selectedViolation.code_snippet && (
                <div>
                  <p className="text-sm font-medium text-dark-700 mb-1">Code</p>
                  <pre className="bg-dark-900 text-dark-100 rounded-lg p-3 text-xs overflow-x-auto">
                    <code>{selectedViolation.code_snippet}</code>
                  </pre>
                </div>
              )}
              
              <div>
                <p className="text-sm font-medium text-dark-700 mb-1">Recommendation</p>
                <p className="text-sm text-dark-600">{selectedViolation.recommendation}</p>
              </div>
              
              <div className="pt-4 border-t border-dark-200">
                {selectedViolation.discovered_at && !isNaN(new Date(selectedViolation.discovered_at).getTime()) && (
                  <p className="text-xs text-dark-500">
                    Discovered: {format(new Date(selectedViolation.discovered_at), 'PPp')}
                  </p>
                )}
              </div>
            </div>
          ) : (
            <div className="text-center py-8">
              <Search className="h-12 w-12 text-dark-300 mx-auto mb-3" />
              <p className="text-sm text-dark-600">Select a violation to view details</p>
            </div>
          )}
        </Card>
      </div>

      {/* Check Type Info Dialog */}
      <Dialog
        open={showCheckTypeInfo}
        onClose={() => setShowCheckTypeInfo(false)}
        title="Standards Check Types"
        size="lg"
      >
        <div className="space-y-6">
          <p className="text-dark-600">
            Choose the appropriate check type based on your compliance requirements and time constraints:
          </p>

          {/* Quick Check */}
          <div className="border-l-4 border-blue-500 pl-4">
            <div className="flex items-start gap-3">
              <Zap className="h-5 w-5 text-blue-600 mt-0.5" />
              <div>
                <h3 className="font-semibold text-dark-900 mb-2">Quick Check</h3>
                <p className="text-sm text-dark-600 mb-2">
                  A rapid compliance assessment focusing on the most critical standards violations.
                </p>
                <ul className="text-sm text-dark-600 space-y-1">
                  <li className="flex items-start gap-2">
                    <span className="text-dark-400">•</span>
                    <span><strong>Duration:</strong> 1-5 seconds</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-dark-400">•</span>
                    <span><strong>Coverage:</strong> Lifecycle protection and critical configuration issues</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-dark-400">•</span>
                    <span><strong>Best for:</strong> CI/CD validation and quick compliance checks</span>
                  </li>
                </ul>
              </div>
            </div>
          </div>

          {/* Full Check */}
          <div className="border-l-4 border-primary-500 pl-4">
            <div className="flex items-start gap-3">
              <Award className="h-5 w-5 text-primary-600 mt-0.5" />
              <div>
                <h3 className="font-semibold text-dark-900 mb-2">Full Check</h3>
                <p className="text-sm text-dark-600 mb-2">
                  Comprehensive standards compliance analysis covering all Vrooli development standards.
                </p>
                <ul className="text-sm text-dark-600 space-y-1">
                  <li className="flex items-start gap-2">
                    <span className="text-dark-400">•</span>
                    <span><strong>Duration:</strong> 5-30 seconds</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-dark-400">•</span>
                    <span><strong>Coverage:</strong> All standards checks and best practices</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-dark-400">•</span>
                    <span><strong>Best for:</strong> Pre-deployment validation and comprehensive audits</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-dark-400">•</span>
                    <span><strong>Checks:</strong> Lifecycle protection, test coverage, code organization, hardcoded values, API versioning</span>
                  </li>
                </ul>
              </div>
            </div>
          </div>

          {/* Targeted Check */}
          <div className="border-l-4 border-orange-500 pl-4">
            <div className="flex items-start gap-3">
              <Target className="h-5 w-5 text-orange-600 mt-0.5" />
              <div>
                <h3 className="font-semibold text-dark-900 mb-2">Targeted Check</h3>
                <p className="text-sm text-dark-600 mb-2">
                  Focused standards analysis on a specific scenario with deep inspection.
                </p>
                <ul className="text-sm text-dark-600 space-y-1">
                  <li className="flex items-start gap-2">
                    <span className="text-dark-400">•</span>
                    <span><strong>Duration:</strong> 3-20 seconds (varies by scenario size)</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-dark-400">•</span>
                    <span><strong>Coverage:</strong> Single scenario comprehensive standards analysis</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-dark-400">•</span>
                    <span><strong>Best for:</strong> Investigating specific compliance issues or validating fixes</span>
                  </li>
                </ul>
              </div>
            </div>
          </div>

          <div className="bg-primary-50 rounded-lg p-4">
            <div className="flex items-start gap-2">
              <Info className="h-5 w-5 text-primary-600 flex-shrink-0 mt-0.5" />
              <div className="text-sm text-primary-900">
                <strong>Standards Checked:</strong> Lifecycle protection (VROOLI_LIFECYCLE_MANAGED), test file coverage, 
                lightweight main.go structure, no hardcoded ports/URLs, and versioned API endpoints.
              </div>
            </div>
          </div>
        </div>
      </Dialog>
    </div>
  )
}
