import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import clsx from 'clsx'
import { format } from 'date-fns'
import {
  AlertTriangle,
  FileCode,
  Filter,
  GitBranch,
  History,
  Lock,
  RefreshCw,
  Settings,
  Shield,
  Unlock
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { apiService } from '../services/api'
import { AutomatedFix, AutomatedFixConfig, AutomatedFixJobSnapshot, AutomatedFixRescanResult } from '../types/api'
import { Badge } from './common/Badge'
import { Card } from './common/Card'

type FixStrategy = 'critical_first' | 'security_first' | 'standards_first' | 'low_first'

const STRATEGY_OPTIONS: Array<{ value: FixStrategy; label: string; description: string }> = [
  { value: 'critical_first', label: 'Critical first', description: 'Address the highest severity findings first each loop.' },
  { value: 'security_first', label: 'Security first', description: 'Fix security vulnerabilities before standards violations.' },
  { value: 'standards_first', label: 'Standards first', description: 'Fix standards violations before security vulnerabilities.' },
  { value: 'low_first', label: 'Low priority first', description: 'Work upward from low severity issues for gradual clean-up.' },
]

const selectDefaultModel = (options: Array<{ value: string }>, fallback: string) => {
  const freeOption = options.find(option => option.value.toLowerCase().endsWith(':free'))
  return freeOption?.value ?? fallback
}

const FALLBACK_AUTOMATION_MODEL = 'openrouter/x-ai/grok-code-fast-1'

const MODEL_OPTION_BLUEPRINT = [
  { value: FALLBACK_AUTOMATION_MODEL, label: 'x-ai/grok-code-fast-1' },
  { value: 'openrouter/google/gemini-2.5-flash', label: 'google/gemini-2.5-flash' },
  { value: 'openrouter/openai/gpt-5', label: 'openai/gpt-5' },
  { value: 'openrouter/x-ai/grok-4-fast:free', label: 'x-ai/grok-4-fast:free' },
]

const DEFAULT_AUTOMATION_MODEL = selectDefaultModel(MODEL_OPTION_BLUEPRINT, FALLBACK_AUTOMATION_MODEL)

const MODEL_OPTIONS = MODEL_OPTION_BLUEPRINT.map(option =>
  option.value === DEFAULT_AUTOMATION_MODEL
    ? { ...option, label: `${option.label} (default)` }
    : option,
)

export default function AutomatedFixPanel() {
  const [showEnableDialog, setShowEnableDialog] = useState(false)
  const [enableConfig, setEnableConfig] = useState<Pick<AutomatedFixConfig, 'violation_types' | 'severities' | 'strategy' | 'loop_delay_seconds' | 'timeout_seconds' | 'max_fixes' | 'model'>>({
    violation_types: ['security'] as Array<'security' | 'standards'>,
    severities: ['critical', 'high'] as Array<'low' | 'medium' | 'high' | 'critical'>,
    strategy: 'critical_first' as FixStrategy,
    loop_delay_seconds: 300,
    timeout_seconds: 3600,
    max_fixes: 0,
    model: DEFAULT_AUTOMATION_MODEL,
  })
  const [confirmationChecked, setConfirmationChecked] = useState(false)

  const queryClient = useQueryClient()

  const { data: fixConfig, isLoading, refetch } = useQuery({
    queryKey: ['fixConfig'],
    queryFn: apiService.getFixConfig,
    refetchInterval: 30000,
  })

  const { data: fixHistory } = useQuery({
    queryKey: ['fixHistory'],
    queryFn: apiService.getFixHistory,
  })

  const { data: automationJobs } = useQuery({
    queryKey: ['automationJobs'],
    queryFn: apiService.getFixJobs,
    refetchInterval: 5000,
  })

  const enableMutation = useMutation({
    mutationFn: () => apiService.enableAutomatedFixes({
      violation_types: enableConfig.violation_types,
      severities: enableConfig.severities,
      strategy: enableConfig.strategy,
      loop_delay_seconds: Math.max(0, enableConfig.loop_delay_seconds),
      timeout_seconds: Math.max(0, enableConfig.timeout_seconds),
      max_fixes: Math.max(0, enableConfig.max_fixes),
      model: enableConfig.model,
    }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fixConfig'] })
      queryClient.invalidateQueries({ queryKey: ['automationJobs'] })
      setShowEnableDialog(false)
      setConfirmationChecked(false)
    },
  })

  const disableMutation = useMutation({
    mutationFn: apiService.disableAutomatedFixes,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fixConfig'] })
    },
  })

  const rollbackMutation = useMutation({
    mutationFn: (fixId: string) => apiService.rollbackFix(fixId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fixHistory'] })
    },
  })

  const cancelJobMutation = useMutation({
    mutationFn: (jobId: string) => apiService.cancelFixJob(jobId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['automationJobs'] })
    },
  })

  const violationTypeOptions = [
    { value: 'security' as const, label: 'Security (vulnerabilities)' },
    { value: 'standards' as const, label: 'Standards (rule violations)' },
  ]

  const severityOptions = useMemo(() => [
    { value: 'critical' as const, label: 'Critical' },
    { value: 'high' as const, label: 'High' },
    { value: 'medium' as const, label: 'Medium' },
    { value: 'low' as const, label: 'Low' },
  ], [])

  const violationTypeLabels: Record<'security' | 'standards', string> = {
    security: 'Security',
    standards: 'Standards',
  }

  const severityBadgeVariant = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'danger'
      case 'high':
        return 'warning'
      case 'medium':
        return 'default'
      case 'low':
        return 'info'
      default:
        return 'default'
    }
  }

  const statusBadgeVariant = (status: AutomatedFix['status']) => {
    switch (status) {
      case 'applied':
        return 'success'
      case 'in_progress':
        return 'warning'
      case 'rolled_back':
        return 'info'
      case 'failed':
        return 'danger'
      default:
        return 'default'
    }
  }

  const safetyStatusVariant = (status?: string) => {
    switch (status) {
      case 'disabled':
        return 'success'
      case 'guarded':
        return 'default'
      case 'moderate':
        return 'warning'
      case 'high-risk':
        return 'danger'
      default:
        return 'default'
    }
  }

  const jobStatusVariant = (status: AutomatedFixJobSnapshot['status']) => {
    switch (status) {
      case 'running':
        return 'warning'
      case 'pending':
        return 'info'
      case 'completed':
        return 'success'
      case 'failed':
        return 'danger'
      case 'cancelled':
        return 'default'
      default:
        return 'default'
    }
  }

  const latestLoopSummary = (job: AutomatedFixJobSnapshot) => {
    if (!job.loops || job.loops.length === 0) {
      return job.message || 'No loops executed yet'
    }
    const latest = job.loops[job.loops.length - 1]
    const parts: string[] = []
    if (latest.message) {
      parts.push(latest.message)
    }
    if (latest.rescan_triggered && latest.rescan_results && latest.rescan_results.length > 0) {
      const rescanText = latest.rescan_results
        .map((result: AutomatedFixRescanResult) => `${result.type}: ${result.status}${result.message ? ` (${result.message})` : ''}`)
        .join('; ')
      parts.push(`Rescan — ${rescanText}`)
    }
    if (parts.length === 0) {
      parts.push('Loop completed')
    }
    return parts.join(' • ')
  }

  return (
    <div className="space-y-6 animate-in">
      {/* Header with Critical Safety Notice */}
      <div>
        <h1 className="text-2xl font-bold text-dark-900">Automated Fix Management</h1>
        <p className="text-dark-500 mt-1">Configure and monitor automated vulnerability fixes</p>
      </div>

      {/* Safety Status Card */}
      <Card className={clsx(
        'border-2',
        fixConfig?.enabled ? 'border-warning-500 bg-warning-50/30' : 'border-success-500 bg-success-50/30'
      )}>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            {fixConfig?.enabled ? (
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-warning-500 text-white">
                <Unlock className="h-6 w-6" />
              </div>
            ) : (
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-success-500 text-white">
                <Lock className="h-6 w-6" />
              </div>
            )}
            <div>
              <h2 className="text-lg font-bold text-dark-900">
                Automated Fixes are {fixConfig?.enabled ? 'ENABLED' : 'DISABLED'}
              </h2>
              <p className="text-sm text-dark-600">
                {fixConfig?.enabled
                  ? '⚠️ System can automatically apply code fixes based on configuration'
                  : '✅ System is in safe mode - no automatic fixes will be applied'}
              </p>
            </div>
          </div>

          <div className="flex gap-2">
            {fixConfig?.enabled ? (
              <button
                onClick={() => disableMutation.mutate()}
                disabled={disableMutation.isPending}
                className="flex items-center gap-2 rounded-lg bg-danger-500 px-4 py-2 text-sm font-medium text-white hover:bg-danger-600 transition-colors"
              >
                {disableMutation.isPending ? (
                  <RefreshCw className="h-4 w-4 animate-spin" />
                ) : (
                  <Lock className="h-4 w-4" />
                )}
                Disable Now
              </button>
            ) : (
              <button
                onClick={() => {
                  if (fixConfig) {
                    setEnableConfig({
                      violation_types: [...fixConfig.violation_types],
                      severities: [...fixConfig.severities],
                      strategy: fixConfig.strategy,
                      loop_delay_seconds: fixConfig.loop_delay_seconds,
                      timeout_seconds: fixConfig.timeout_seconds,
                      max_fixes: fixConfig.max_fixes,
                      model: fixConfig.model || DEFAULT_AUTOMATION_MODEL,
                    })
                  }
                  setShowEnableDialog(true)
                }}
                className="flex items-center gap-2 rounded-lg bg-dark-700 px-4 py-2 text-sm font-medium text-white hover:bg-dark-800 transition-colors"
              >
                <Settings className="h-4 w-4" />
                Configure & Enable
              </button>
            )}
            <button
              onClick={() => refetch()}
              className="flex items-center justify-center rounded-lg bg-dark-100 p-2 text-dark-700 hover:bg-dark-200 transition-colors"
            >
              <RefreshCw className="h-4 w-4" />
            </button>
          </div>
        </div>
      </Card>

      {/* Active Automation Jobs */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-dark-900">Active Automation Jobs</h3>
          <button
            onClick={() => queryClient.invalidateQueries({ queryKey: ['automationJobs'] })}
            className="flex items-center gap-2 rounded-lg bg-dark-100 px-3 py-1 text-sm text-dark-600 hover:bg-dark-200 transition-colors"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </button>
        </div>

        {automationJobs && automationJobs.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-dark-200">
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">Job ID</th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">Run</th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">Model</th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">Status</th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">Loops</th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">Issues</th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">Latest Loop</th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">Started</th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-dark-100">
                {automationJobs.map((job) => {
                  const latestLoop = job.loops && job.loops.length > 0 ? job.loops[job.loops.length - 1] : undefined
                  const latestDuration = latestLoop ? `${latestLoop.duration_seconds.toFixed(1)}s` : ''
                  return (
                    <tr key={job.id} className="align-top hover:bg-dark-50">
                      <td className="py-3 text-sm font-mono text-dark-600">{job.id.substring(0, 8)}...</td>
                      <td className="py-3 text-sm font-mono text-dark-600">{job.automation_run_id.substring(0, 8)}...</td>
                      <td className="py-3 text-sm text-dark-600">
                        {MODEL_OPTIONS.find(option => option.value === job.model)?.label || job.model || 'Default'}
                      </td>
                      <td className="py-3">
                        <Badge variant={jobStatusVariant(job.status)} size="sm" className="capitalize">
                          {job.status}
                        </Badge>
                      </td>
                      <td className="py-3 text-sm text-dark-600">
                        {job.loops_completed} loop{job.loops_completed === 1 ? '' : 's'}
                        {latestDuration ? ` • ${latestDuration}` : ''}
                      </td>
                      <td className="py-3 text-sm text-dark-600">
                        {job.issues_attempted}
                        {job.max_fixes ? ` / ${job.max_fixes}` : ''}
                      </td>
                      <td className="py-3 text-sm text-dark-500 whitespace-pre-line">
                        {latestLoopSummary(job)}
                      </td>
                      <td className="py-3 text-sm text-dark-500">
                        {format(new Date(job.started_at), 'MMM d, HH:mm')}
                      </td>
                      <td className="py-3 text-sm">
                        {(job.status === 'running' || job.status === 'pending') && (
                          <button
                            onClick={() => cancelJobMutation.mutate(job.id)}
                            disabled={cancelJobMutation.isPending}
                            className="text-sm text-danger-600 hover:text-danger-700 font-medium"
                          >
                            Cancel
                          </button>
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="text-center py-8">
            <History className="h-12 w-12 text-dark-300 mx-auto mb-3" />
            <p className="text-sm text-dark-600">No automation jobs are currently running</p>
            <p className="text-xs text-dark-500 mt-1">Launch an automated fix to see live telemetry</p>
          </div>
        )}
      </Card>

      {/* Current Configuration */}
      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <h3 className="text-lg font-semibold text-dark-900 mb-4">Current Configuration</h3>

          {isLoading ? (
            <div className="space-y-3">
              {[1, 2, 3, 4].map(i => (
                <div key={i} className="h-12 bg-dark-100 rounded animate-pulse" />
              ))}
            </div>
          ) : fixConfig ? (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-dark-700">Safety Posture</span>
                <Badge variant={safetyStatusVariant(fixConfig.safety_status)}>
                  {fixConfig.safety_status || 'unknown'}
                </Badge>
              </div>

              <div>
                <p className="text-sm font-medium text-dark-700 mb-2">Violation Types</p>
                <div className="flex flex-wrap gap-2">
                  {fixConfig.violation_types.map(type => (
                    <Badge key={type} variant="primary" size="sm">{violationTypeLabels[type]}</Badge>
                  ))}
                </div>
              </div>

              <div>
                <p className="text-sm font-medium text-dark-700 mb-2">Target Severities</p>
                <div className="flex flex-wrap gap-2">
                  {fixConfig.severities.map(severity => (
                    <Badge key={severity} variant={severityBadgeVariant(severity)} size="sm">
                      {severity}
                    </Badge>
                  ))}
                </div>
              </div>

              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <p className="text-sm font-medium text-dark-700">Strategy</p>
                  <p className="text-sm text-dark-500">
                    {STRATEGY_OPTIONS.find(option => option.value === fixConfig.strategy)?.label || fixConfig.strategy}
                  </p>
                </div>
                <div>
                  <p className="text-sm font-medium text-dark-700">Loop Delay</p>
                  <p className="text-sm text-dark-500">
                    {fixConfig.loop_delay_seconds ? `${fixConfig.loop_delay_seconds}s between loops` : 'No delay between loops'}
                  </p>
                </div>
                <div>
                  <p className="text-sm font-medium text-dark-700">Job Timeout</p>
                  <p className="text-sm text-dark-500">
                    {fixConfig.timeout_seconds ? `${fixConfig.timeout_seconds}s maximum runtime` : 'No timeout'}
                  </p>
                </div>
                <div>
                  <p className="text-sm font-medium text-dark-700">Auto-disable Threshold</p>
                  <p className="text-sm text-dark-500">
                    {fixConfig.max_fixes ? `${fixConfig.max_fixes} fixes before disabling` : 'Unlimited fixes'}
                  </p>
                </div>
                <div className="sm:col-span-2">
                  <p className="text-sm font-medium text-dark-700">Model</p>
                  <p className="text-sm text-dark-500">
                    {MODEL_OPTIONS.find(option => option.value === fixConfig.model)?.label || fixConfig.model || 'Default'}
                  </p>
                </div>
              </div>

              {fixConfig.updated_at && (
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium text-dark-700">Last Updated</span>
                  <span className="text-dark-500 text-sm">
                    {new Date(fixConfig.updated_at).toLocaleString()}
                  </span>
                </div>
              )}
            </div>
          ) : null}
        </Card>

        {/* Safety Controls */}
        <Card>
          <h3 className="text-lg font-semibold text-dark-900 mb-4">Safety Controls</h3>

          <div className="space-y-3">
            <div className="flex items-start gap-3 p-3 rounded-lg bg-dark-50">
              <Shield className="h-5 w-5 text-primary-500 mt-0.5" />
              <div className="flex-1">
                <p className="text-sm font-medium text-dark-900">Policy-Based Targeting</p>
                <p className="text-xs text-dark-600 mt-0.5">
                  Automation only runs for the violation types you explicitly select
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3 p-3 rounded-lg bg-dark-50">
              <Filter className="h-5 w-5 text-primary-500 mt-0.5" />
              <div className="flex-1">
                <p className="text-sm font-medium text-dark-900">Severity Guardrails</p>
                <p className="text-xs text-dark-600 mt-0.5">
                  Restrict fixes to high-impact issues or broaden coverage intentionally
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3 p-3 rounded-lg bg-dark-50">
              <History className="h-5 w-5 text-primary-500 mt-0.5" />
              <div className="flex-1">
                <p className="text-sm font-medium text-dark-900">Agent Telemetry</p>
                <p className="text-xs text-dark-600 mt-0.5">
                  Every automated run is tracked with agent IDs and timestamps
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3 p-3 rounded-lg bg-dark-50">
              <GitBranch className="h-5 w-5 text-primary-500 mt-0.5" />
              <div className="flex-1">
                <p className="text-sm font-medium text-dark-900">Git-Based Recovery</p>
                <p className="text-xs text-dark-600 mt-0.5">
                  Rely on Git history or dedicated backup scenarios for full rollbacks
                </p>
              </div>
            </div>
          </div>
        </Card>
      </div>

      {/* Fix History */}
      <Card>
        <h3 className="text-lg font-semibold text-dark-900 mb-4">Fix History</h3>

        {fixHistory && fixHistory.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-dark-200">
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Fix ID
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Run
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Scenario
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Type
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Severity
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Status
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Issues
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Applied
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-dark-100">
                {fixHistory.map((fix) => (
                  <tr key={fix.id} className="hover:bg-dark-50">
                    <td className="py-3 text-sm font-mono text-dark-600">
                      {fix.id.substring(0, 8)}...
                    </td>
                    <td className="py-3 text-sm font-mono text-dark-600">
                      {fix.automation_run_id ? `${fix.automation_run_id.substring(0, 8)}...` : '—'}
                    </td>
                    <td className="py-3 text-sm text-dark-900">
                      {fix.scenario_name}
                    </td>
                    <td className="py-3">
                      <Badge variant="default" size="sm">{violationTypeLabels[fix.violation_type]}</Badge>
                    </td>
                    <td className="py-3">
                      <Badge variant={severityBadgeVariant(fix.severity)} size="sm">
                        {fix.severity}
                      </Badge>
                    </td>
                    <td className="py-3">
                      <Badge variant={statusBadgeVariant(fix.status)} size="sm">
                        {fix.status.replace('_', ' ')}
                      </Badge>
                    </td>
                    <td className="py-3 text-sm text-dark-500">
                      {fix.issue_count}
                    </td>
                    <td className="py-3 text-sm text-dark-500">
                      {fix.applied_at && !isNaN(new Date(fix.applied_at).getTime())
                        ? format(new Date(fix.applied_at), 'MMM d, HH:mm')
                        : 'N/A'}
                    </td>
                    <td className="py-3">
                      {fix.status === 'applied' && (
                        <button
                          onClick={() => rollbackMutation.mutate(fix.id)}
                          disabled={rollbackMutation.isPending}
                          className="text-sm text-danger-600 hover:text-danger-700 font-medium"
                        >
                          Rollback
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="text-center py-8">
            <FileCode className="h-12 w-12 text-dark-300 mx-auto mb-3" />
            <p className="text-sm text-dark-600">No fixes have been applied yet</p>
            <p className="text-xs text-dark-500 mt-1">Automated fixes are currently disabled</p>
          </div>
        )}
      </Card>

      {/* Enable Dialog */}
      {showEnableDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
          <div className="w-full max-w-2xl bg-white rounded-xl shadow-xl p-6 m-4">
            <div className="flex items-center gap-3 mb-4">
              <AlertTriangle className="h-8 w-8 text-warning-500" />
              <h2 className="text-xl font-bold text-dark-900">Enable Automated Fixes</h2>
            </div>

            <div className="mb-6 p-4 rounded-lg bg-warning-50 border border-warning-300">
              <p className="text-sm font-medium text-warning-900 mb-2">⚠️ Critical Safety Warning</p>
              <p className="text-sm text-warning-800">
                Enabling automated fixes allows the system to modify your code automatically.
                While policy guardrails are in place, you should understand the risks:
              </p>
              <ul className="mt-2 space-y-1 text-sm text-warning-700 list-disc list-inside">
                <li>Fix agents run with full workspace permissions</li>
                <li>Changes land without human-in-the-loop review</li>
                <li>Always validate scenarios after automation runs</li>
                <li>Use Git or dedicated backup scenarios to recover</li>
              </ul>
            </div>

            <div className="space-y-4 mb-6">
              <div>
                <label className="block text-sm font-medium text-dark-700 mb-2">
                  Violation Types
                </label>
                <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                  {violationTypeOptions.map(option => (
                    <label key={option.value} className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={enableConfig.violation_types.includes(option.value)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setEnableConfig(prev => ({
                              ...prev,
                              violation_types: [...prev.violation_types, option.value]
                            }))
                          } else {
                            setEnableConfig(prev => ({
                              ...prev,
                              violation_types: prev.violation_types.filter(type => type !== option.value)
                            }))
                          }
                        }}
                        className="rounded text-primary-500 focus:ring-primary-500"
                      />
                      <span className="text-sm text-dark-700">{option.label}</span>
                    </label>
                  ))}
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-dark-700 mb-2">
                  Target Severities
                </label>
                <div className="grid grid-cols-2 gap-2">
                  {severityOptions.map(option => (
                    <label key={option.value} className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={enableConfig.severities.includes(option.value)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setEnableConfig(prev => ({
                              ...prev,
                              severities: [...prev.severities, option.value]
                            }))
                          } else {
                            setEnableConfig(prev => ({
                              ...prev,
                              severities: prev.severities.filter(sev => sev !== option.value)
                            }))
                          }
                        }}
                        className="rounded text-primary-500 focus:ring-primary-500"
                      />
                      <span className="text-sm text-dark-700">{option.label}</span>
                    </label>
                  ))}
                </div>
                <p className="text-xs text-dark-500 mt-2">
                  Tip: keep automation restricted to high / critical issues unless you need deeper coverage.
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-dark-700 mb-2">
                  Strategy
                </label>
                <select
                  value={enableConfig.strategy}
                  onChange={(e) => setEnableConfig(prev => ({ ...prev, strategy: e.target.value as FixStrategy }))}
                  className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
                >
                  {STRATEGY_OPTIONS.map(option => (
                    <option key={option.value} value={option.value}>{option.label}</option>
                  ))}
                </select>
                <p className="text-xs text-dark-500 mt-2">
                  {STRATEGY_OPTIONS.find(option => option.value === enableConfig.strategy)?.description}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-dark-700 mb-2">
                  Model
                </label>
                <select
                  value={enableConfig.model}
                  onChange={(e) => setEnableConfig(prev => ({ ...prev, model: e.target.value }))}
                  className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
                >
                  {MODEL_OPTIONS.map(option => (
                    <option key={option.value} value={option.value}>{option.label}</option>
                  ))}
                </select>
                <p className="text-xs text-dark-500 mt-2">Pick which model automated fix agents will use.</p>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <div>
                  <label className="block text-sm font-medium text-dark-700 mb-2">
                    Loop Delay (seconds)
                  </label>
                  <input
                    type="number"
                    min={0}
                    value={enableConfig.loop_delay_seconds}
                    onChange={(e) => setEnableConfig(prev => ({ ...prev, loop_delay_seconds: Number(e.target.value) }))}
                    className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
                  />
                  <p className="text-xs text-dark-500 mt-1">0 = no delay between automation loops.</p>
                </div>
                <div>
                  <label className="block text-sm font-medium text-dark-700 mb-2">
                    Timeout (seconds)
                  </label>
                  <input
                    type="number"
                    min={0}
                    value={enableConfig.timeout_seconds}
                    onChange={(e) => setEnableConfig(prev => ({ ...prev, timeout_seconds: Number(e.target.value) }))}
                    className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
                  />
                  <p className="text-xs text-dark-500 mt-1">0 = no maximum runtime.</p>
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-dark-700 mb-2">
                  Auto-disable after (fix count)
                </label>
                <input
                  type="number"
                  min={0}
                  value={enableConfig.max_fixes}
                  onChange={(e) => setEnableConfig(prev => ({ ...prev, max_fixes: Number(e.target.value) }))}
                  className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
                />
                <p className="text-xs text-dark-500 mt-1">0 = unlimited automated fixes before disabling.</p>
              </div>
            </div>

            <div className="mb-6 p-4 rounded-lg bg-danger-50 border border-danger-300">
              <label className="flex items-start gap-2">
                <input
                  type="checkbox"
                  checked={confirmationChecked}
                  onChange={(e) => setConfirmationChecked(e.target.checked)}
                  className="rounded text-danger-500 mt-1"
                />
                <span className="text-sm text-danger-900">
                  I understand the risks and confirm that I want to enable automated fixes
                  with the above configuration. I have backups of my code and will test
                  thoroughly after any fixes are applied.
                </span>
              </label>
            </div>

            <div className="flex gap-3 justify-end">
              <button
                onClick={() => {
                  setShowEnableDialog(false)
                  setConfirmationChecked(false)
                }}
                className="px-4 py-2 text-sm font-medium text-dark-700 hover:bg-dark-100 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => enableMutation.mutate()}
                disabled={!confirmationChecked || enableConfig.violation_types.length === 0 || enableConfig.severities.length === 0 || enableMutation.isPending}
                className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-danger-500 hover:bg-danger-600 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {enableMutation.isPending ? (
                  <>
                    <RefreshCw className="h-4 w-4 animate-spin" />
                    Enabling...
                  </>
                ) : (
                  <>
                    <Unlock className="h-4 w-4" />
                    Enable Automated Fixes
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
