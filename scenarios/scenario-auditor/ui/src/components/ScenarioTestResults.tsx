import { RuleScenarioTestResult, StandardsViolation } from '@/types/api'
import { AlertTriangle, ChevronDown, ChevronUp, XCircle } from 'lucide-react'
import { useMemo, useState } from 'react'
import { TARGET_BADGE_CLASSES, TARGET_CATEGORY_CONFIG } from '@/constants/ruleCategories'

interface ScenarioTestResultsProps {
  results: RuleScenarioTestResult[]
  completedAt: Date | null
}

type FilterMode = 'all' | 'failed' | 'passed' | 'violations' | 'warnings'
type SortMode = 'scenario' | 'violations' | 'duration' | 'files'

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
  const seconds = Math.round(totalSeconds - minutes * 60)
  if (minutes < 60) {
    return `${minutes}m ${seconds}s`
  }

  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return `${hours}h ${remainingMinutes}m`
}

export default function ScenarioTestResults({ results, completedAt }: ScenarioTestResultsProps) {
  const [expandedResults, setExpandedResults] = useState<Set<number>>(new Set())
  const [filterMode, setFilterMode] = useState<FilterMode>('all')
  const [sortMode, setSortMode] = useState<SortMode>('scenario')

  // Calculate summary statistics
  const stats = useMemo(() => {
    const passed = results.filter(r => !r.error && r.violations.length === 0).length
    const failed = results.filter(r => r.error).length
    const withViolations = results.filter(r => !r.error && r.violations.length > 0).length
    const warnings = results.filter(r => r.warning).length
    const totalViolations = results.reduce((sum, r) => sum + r.violations.length, 0)
    const totalFiles = results.reduce((sum, r) => sum + r.files_scanned, 0)
    const totalDuration = results.reduce((sum, r) => sum + r.duration_ms, 0)

    return {
      passed,
      failed,
      withViolations,
      warnings,
      totalViolations,
      totalFiles,
      totalDuration,
      total: results.length,
    }
  }, [results])

  // Filter and sort results
  const filteredAndSortedResults = useMemo(() => {
    let filtered = [...results]

    // Apply filter
    switch (filterMode) {
      case 'failed':
        filtered = filtered.filter(r => r.error)
        break
      case 'passed':
        filtered = filtered.filter(r => !r.error && r.violations.length === 0)
        break
      case 'violations':
        filtered = filtered.filter(r => !r.error && r.violations.length > 0)
        break
      case 'warnings':
        filtered = filtered.filter(r => r.warning)
        break
    }

    // Apply sort
    switch (sortMode) {
      case 'scenario':
        filtered.sort((a, b) => a.scenario.localeCompare(b.scenario))
        break
      case 'violations':
        filtered.sort((a, b) => b.violations.length - a.violations.length)
        break
      case 'duration':
        filtered.sort((a, b) => b.duration_ms - a.duration_ms)
        break
      case 'files':
        filtered.sort((a, b) => b.files_scanned - a.files_scanned)
        break
    }

    return filtered
  }, [results, filterMode, sortMode])

  const toggleExpanded = (index: number) => {
    const newExpanded = new Set(expandedResults)
    if (newExpanded.has(index)) {
      newExpanded.delete(index)
    } else {
      newExpanded.add(index)
    }
    setExpandedResults(newExpanded)
  }

  const toggleExpandAll = () => {
    if (expandedResults.size === filteredAndSortedResults.length) {
      setExpandedResults(new Set())
    } else {
      setExpandedResults(new Set(filteredAndSortedResults.map((_, idx) => idx)))
    }
  }

  const allExpanded = expandedResults.size === filteredAndSortedResults.length && filteredAndSortedResults.length > 0

  return (
    <div className="space-y-4">
      {/* Summary Stats Bar */}
      <div className="rounded-md border border-slate-200 bg-slate-50 px-4 py-3">
        <div className="flex flex-wrap items-center gap-x-6 gap-y-2">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-slate-700">Test Results:</span>
            {completedAt && (
              <span className="text-xs text-slate-500">
                Last run {completedAt.toLocaleTimeString()}
              </span>
            )}
          </div>
          <div className="flex flex-wrap items-center gap-4">
            <span className="text-sm text-green-700 font-medium">✅ {stats.passed} passed</span>
            {stats.withViolations > 0 && (
              <span className="text-sm text-orange-600 font-medium">⚠️ {stats.withViolations} with violations</span>
            )}
            {stats.failed > 0 && (
              <span className="text-sm text-red-700 font-medium">❌ {stats.failed} failed</span>
            )}
            {stats.warnings > 0 && (
              <span className="text-sm text-amber-600 font-medium">⚡ {stats.warnings} warnings</span>
            )}
            <span className="text-sm text-slate-600">({stats.total} total)</span>
          </div>
        </div>
        <div className="mt-2 flex flex-wrap items-center gap-4 text-xs text-slate-600">
          <span>{stats.totalViolations} total violations</span>
          <span>•</span>
          <span>{stats.totalFiles} files scanned</span>
          <span>•</span>
          <span>{formatDurationMs(stats.totalDuration)} total duration</span>
        </div>
      </div>

      {/* Filter and Sort Controls */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-3">
          <select
            value={filterMode}
            onChange={(e) => setFilterMode(e.target.value as FilterMode)}
            className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">All Results ({results.length})</option>
            <option value="passed">Passed Only ({stats.passed})</option>
            <option value="violations">With Violations ({stats.withViolations})</option>
            <option value="failed">Failed Only ({stats.failed})</option>
            <option value="warnings">Warnings Only ({stats.warnings})</option>
          </select>
          <select
            value={sortMode}
            onChange={(e) => setSortMode(e.target.value as SortMode)}
            className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="scenario">Sort by Scenario Name</option>
            <option value="violations">Sort by Violations Count</option>
            <option value="duration">Sort by Duration</option>
            <option value="files">Sort by Files Scanned</option>
          </select>
        </div>
        <button
          onClick={toggleExpandAll}
          className="inline-flex items-center gap-1.5 rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
        >
          {allExpanded ? (
            <>
              <ChevronUp className="h-4 w-4" />
              Collapse All
            </>
          ) : (
            <>
              <ChevronDown className="h-4 w-4" />
              Expand All
            </>
          )}
        </button>
      </div>

      {/* Results List */}
      {filteredAndSortedResults.length === 0 ? (
        <div className="rounded-md border border-slate-200 bg-slate-50 px-4 py-8 text-center text-sm text-slate-500">
          No results match the selected filter.
        </div>
      ) : (
        <div className="space-y-3">
          {filteredAndSortedResults.map((result, resultIndex) => {
            const isExpanded = expandedResults.has(resultIndex)
            const hasViolations = result.violations.length > 0
            const hasFailed = !!result.error
            const hasWarning = !!result.warning

            return (
              <div
                key={`scenario-result-${result.scenario}-${resultIndex}`}
                className={`rounded-lg border shadow-sm transition-all ${
                  hasFailed
                    ? 'border-red-200 bg-red-50'
                    : hasViolations
                    ? 'border-orange-200 bg-orange-50'
                    : 'border-green-200 bg-green-50'
                }`}
              >
                {/* Collapsible Header */}
                <button
                  onClick={() => toggleExpanded(resultIndex)}
                  className="flex w-full items-center justify-between gap-4 px-4 py-3 text-left transition-colors hover:bg-white/50"
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-3">
                      <h6 className="text-sm font-semibold text-slate-900 truncate">
                        {result.scenario}
                      </h6>
                      {hasFailed && (
                        <span className="inline-flex items-center rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-800">
                          Failed
                        </span>
                      )}
                      {!hasFailed && hasViolations && (
                        <span className="inline-flex items-center rounded-full bg-orange-100 px-2 py-0.5 text-xs font-medium text-orange-800">
                          {result.violations.length} violation{result.violations.length === 1 ? '' : 's'}
                        </span>
                      )}
                      {!hasFailed && !hasViolations && (
                        <span className="inline-flex items-center rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-800">
                          Passed
                        </span>
                      )}
                    </div>
                    <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-slate-600">
                      <span>{result.files_scanned} file{result.files_scanned === 1 ? '' : 's'}</span>
                      <span>•</span>
                      <span>{formatDurationMs(result.duration_ms)}</span>
                      {result.targets.length > 0 && (
                        <>
                          <span>•</span>
                          <span>{result.targets.length} target{result.targets.length === 1 ? '' : 's'}</span>
                        </>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {isExpanded ? (
                      <ChevronUp className="h-5 w-5 text-slate-400" />
                    ) : (
                      <ChevronDown className="h-5 w-5 text-slate-400" />
                    )}
                  </div>
                </button>

                {/* Expanded Content */}
                {isExpanded && (
                  <div className="border-t border-slate-200 bg-white px-4 py-4">
                    {/* Targets */}
                    {result.targets.length > 0 && (
                      <div className="mb-3 flex flex-wrap gap-2">
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
                    )}

                    {/* Warning */}
                    {hasWarning && (
                      <div className="mb-3 flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700">
                        <AlertTriangle className="h-3.5 w-3.5 flex-shrink-0 mt-0.5" />
                        <span>{result.warning}</span>
                      </div>
                    )}

                    {/* Error */}
                    {hasFailed && (
                      <div className="mb-3 flex items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700">
                        <XCircle className="h-3.5 w-3.5 flex-shrink-0 mt-0.5" />
                        <span>{result.error}</span>
                      </div>
                    )}

                    {/* Violations */}
                    {hasViolations ? (
                      <div className="space-y-3">
                        <h6 className="text-sm font-medium text-slate-900">
                          Violations ({result.violations.length})
                        </h6>
                        {result.violations.map((violation: StandardsViolation, index: number) => (
                          <div
                            key={violation.id || `${violation.file_path || 'violation'}-${index}`}
                            className="rounded-md border border-slate-200 bg-slate-50 p-3"
                          >
                            <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                              <div className="space-y-1">
                                <div className="flex flex-wrap items-center gap-2">
                                  <span
                                    className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                                      violation.severity === 'critical'
                                        ? 'bg-red-100 text-red-800'
                                        : violation.severity === 'high'
                                        ? 'bg-orange-100 text-orange-800'
                                        : violation.severity === 'medium'
                                        ? 'bg-yellow-100 text-yellow-800'
                                        : 'bg-green-100 text-green-800'
                                    }`}
                                  >
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
                                <span className="font-semibold text-slate-700">Recommended fix:</span>{' '}
                                {violation.recommendation}
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
                    ) : !hasFailed ? (
                      <div className="rounded-md border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-700">
                        ✅ No violations detected for this scenario.
                      </div>
                    ) : null}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
