import React, { useState, useMemo, useEffect } from 'react'
import { X, AlertTriangle, CheckCircle, Clock, ExternalLink, FileText, Bug, Wrench, XCircle } from 'lucide-react'
import type { RuleScenarioTestResult } from '@/types/api'

export type ReportType = 'add_tests' | 'fix_tests' | 'fix_violations'

interface ReportIssueDialogProps {
  isOpen: boolean
  onClose: () => void
  ruleId: string | null
  ruleName: string | null
  scenarioTestResults: RuleScenarioTestResult[]
  onSubmitReport: (payload: ReportPayload) => Promise<{ issueId: string; issueUrl?: string; message: string }>
}

export interface ReportPayload {
  reportType: ReportType
  ruleId: string
  customInstructions: string
  selectedScenarios: string[]
}

interface BatchResult {
  batchIndex: number
  success: boolean
  issueId?: string
  issueUrl?: string
  message?: string
  error?: string
  scenarios: string[]
}

const REPORT_TYPE_OPTIONS: Array<{
  id: ReportType
  label: string
  description: string
  icon: React.ReactNode
}> = [
  {
    id: 'add_tests',
    label: 'Add Tests',
    description: 'Request AI to generate new test cases for this rule',
    icon: <FileText className="h-5 w-5" />,
  },
  {
    id: 'fix_tests',
    label: 'Fix Tests',
    description: 'Request AI to repair failing or broken tests for this rule',
    icon: <Wrench className="h-5 w-5" />,
  },
  {
    id: 'fix_violations',
    label: 'Fix Violations',
    description: 'Request AI to resolve rule violations found during scenario audits',
    icon: <Bug className="h-5 w-5" />,
  },
]

const MAX_SCENARIOS_WARNING_THRESHOLD = 20
const MAX_SCENARIOS_PER_ISSUE = 20
const DEFAULT_BATCH_SIZE = 10

export default function ReportIssueDialog({
  isOpen,
  onClose,
  ruleId,
  ruleName,
  scenarioTestResults,
  onSubmitReport,
}: ReportIssueDialogProps) {
  const [reportType, setReportType] = useState<ReportType>('fix_violations')
  const [customInstructions, setCustomInstructions] = useState('')
  const [selectedScenarios, setSelectedScenarios] = useState<Set<string>>(new Set())
  const [batchSize, setBatchSize] = useState<number>(DEFAULT_BATCH_SIZE)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [currentBatch, setCurrentBatch] = useState<number>(0)
  const [error, setError] = useState<string | null>(null)
  const [batchResults, setBatchResults] = useState<BatchResult[]>([])

  // Reset state when dialog opens/closes
  useEffect(() => {
    if (isOpen) {
      setReportType('fix_violations')
      setCustomInstructions('')
      setBatchSize(DEFAULT_BATCH_SIZE)
      setError(null)
      setBatchResults([])
      setCurrentBatch(0)

      // Pre-select all scenarios with violations for fix_violations
      if (scenarioTestResults.length > 0) {
        const scenariosWithViolations = scenarioTestResults
          .filter(result => result.violations && result.violations.length > 0)
          .map(result => result.scenario)
        setSelectedScenarios(new Set(scenariosWithViolations))
      } else {
        setSelectedScenarios(new Set())
      }
    }
  }, [isOpen, scenarioTestResults])

  // Get scenarios with violations for the checklist
  const scenariosWithViolations = useMemo(() => {
    return scenarioTestResults
      .filter(result => result.violations && result.violations.length > 0)
      .map(result => ({
        name: result.scenario,
        violationCount: result.violations.length,
      }))
  }, [scenarioTestResults])

  const hasViolations = scenariosWithViolations.length > 0

  const showScenarioSelection = reportType === 'fix_violations' && hasViolations

  const showBatchSizeSlider = selectedScenarios.size > MAX_SCENARIOS_WARNING_THRESHOLD

  const batchCount = useMemo(() => {
    if (!showBatchSizeSlider) return 1
    return Math.ceil(selectedScenarios.size / batchSize)
  }, [selectedScenarios.size, batchSize, showBatchSizeSlider])

  const canSubmit = useMemo(() => {
    if (!ruleId) return false
    if (reportType === 'fix_violations') {
      return selectedScenarios.size > 0
    }
    return true
  }, [ruleId, reportType, selectedScenarios.size])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!ruleId || !canSubmit) return

    setError(null)
    setIsSubmitting(true)
    setBatchResults([])

    try {
      const scenariosArray = Array.from(selectedScenarios)

      // Split scenarios into batches
      const batches: string[][] = []
      if (showBatchSizeSlider) {
        for (let i = 0; i < scenariosArray.length; i += batchSize) {
          batches.push(scenariosArray.slice(i, i + batchSize))
        }
      } else {
        batches.push(scenariosArray)
      }

      const results: BatchResult[] = []

      // Create issues sequentially
      for (let i = 0; i < batches.length; i++) {
        setCurrentBatch(i + 1)

        try {
          const payload: ReportPayload = {
            reportType,
            ruleId,
            customInstructions: customInstructions.trim(),
            selectedScenarios: batches[i],
          }

          const response = await onSubmitReport(payload)
          results.push({
            batchIndex: i + 1,
            success: true,
            issueId: response.issueId,
            issueUrl: response.issueUrl,
            message: response.message,
            scenarios: batches[i],
          })
        } catch (err) {
          results.push({
            batchIndex: i + 1,
            success: false,
            error: err instanceof Error ? err.message : 'Failed to create issue',
            scenarios: batches[i],
          })
        }
      }

      setBatchResults(results)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit report')
    } finally {
      setIsSubmitting(false)
      setCurrentBatch(0)
    }
  }

  const handleSelectAll = () => {
    if (selectedScenarios.size === scenariosWithViolations.length) {
      setSelectedScenarios(new Set())
    } else {
      setSelectedScenarios(new Set(scenariosWithViolations.map(s => s.name)))
    }
  }

  const handleToggleScenario = (scenarioName: string) => {
    const newSelection = new Set(selectedScenarios)
    if (newSelection.has(scenarioName)) {
      newSelection.delete(scenarioName)
    } else {
      newSelection.add(scenarioName)
    }
    setSelectedScenarios(newSelection)
  }

  if (!isOpen) return null

  // Success/Results state
  if (batchResults.length > 0) {
    const successCount = batchResults.filter(r => r.success).length
    const failureCount = batchResults.filter(r => !r.success).length
    const allSuccess = failureCount === 0

    return (
      <div className="fixed inset-0 z-[60] overflow-y-auto">
        <div className="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <div
            className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"
            onClick={onClose}
          />

          <div className="relative transform overflow-hidden rounded-lg bg-white px-4 pb-4 pt-5 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-2xl sm:p-6">
            <div>
              <div className={`mx-auto flex h-12 w-12 items-center justify-center rounded-full ${allSuccess ? 'bg-green-100' : 'bg-amber-100'}`}>
                {allSuccess ? (
                  <CheckCircle className="h-6 w-6 text-green-600" />
                ) : (
                  <AlertTriangle className="h-6 w-6 text-amber-600" />
                )}
              </div>
              <div className="mt-3 text-center sm:mt-5">
                <h3 className="text-lg font-semibold leading-6 text-gray-900">
                  {allSuccess ? 'All Issues Created Successfully' : 'Issues Created with Some Failures'}
                </h3>
                <div className="mt-2">
                  <p className="text-sm text-gray-500">
                    {allSuccess
                      ? `Successfully created ${successCount} issue${successCount !== 1 ? 's' : ''}`
                      : `Created ${successCount} of ${batchResults.length} issue${batchResults.length !== 1 ? 's' : ''} successfully`
                    }
                  </p>
                </div>
              </div>
            </div>

            {/* List of results */}
            <div className="mt-5 max-h-96 overflow-y-auto space-y-3">
              {batchResults.map((result) => (
                <div
                  key={result.batchIndex}
                  className={`rounded-lg border p-4 ${
                    result.success
                      ? 'border-green-200 bg-green-50'
                      : 'border-red-200 bg-red-50'
                  }`}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex items-start gap-3 flex-1 min-w-0">
                      {result.success ? (
                        <CheckCircle className="h-5 w-5 text-green-600 flex-shrink-0 mt-0.5" />
                      ) : (
                        <XCircle className="h-5 w-5 text-red-600 flex-shrink-0 mt-0.5" />
                      )}
                      <div className="flex-1 min-w-0">
                        <p className={`text-sm font-medium ${result.success ? 'text-green-900' : 'text-red-900'}`}>
                          Batch {result.batchIndex} ({result.scenarios.length} scenario{result.scenarios.length !== 1 ? 's' : ''})
                        </p>
                        {result.success && result.message && (
                          <p className="mt-1 text-xs text-green-700">{result.message}</p>
                        )}
                        {result.success && result.issueId && (
                          <p className="mt-1 text-xs text-green-700">
                            Issue ID: <span className="font-mono">{result.issueId}</span>
                          </p>
                        )}
                        {!result.success && result.error && (
                          <p className="mt-1 text-xs text-red-700">{result.error}</p>
                        )}
                      </div>
                    </div>
                    {result.success && result.issueUrl && (
                      <a
                        href={result.issueUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1.5 rounded-md bg-green-600 px-2.5 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-green-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-green-600 flex-shrink-0"
                      >
                        View
                        <ExternalLink className="h-3 w-3" />
                      </a>
                    )}
                  </div>
                </div>
              ))}
            </div>

            <div className="mt-5 sm:mt-6">
              <button
                type="button"
                className="inline-flex w-full justify-center rounded-md bg-white px-3 py-2 text-sm font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50"
                onClick={onClose}
              >
                Close
              </button>
            </div>
          </div>
        </div>
      </div>
    )
  }

  // Main form
  return (
    <div className="fixed inset-0 z-[60] overflow-y-auto">
      <div className="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
        <div
          className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"
          onClick={onClose}
        />

        <div className="relative transform overflow-hidden rounded-lg bg-white px-4 pb-4 pt-5 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-2xl sm:p-6">
          <div className="absolute right-0 top-0 hidden pr-4 pt-4 sm:block">
            <button
              type="button"
              className="rounded-md bg-white text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
              onClick={onClose}
            >
              <span className="sr-only">Close</span>
              <X className="h-6 w-6" />
            </button>
          </div>

          <div className="sm:flex sm:items-start">
            <div className="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-blue-100 sm:mx-0 sm:h-10 sm:w-10">
              <FileText className="h-6 w-6 text-blue-600" />
            </div>
            <div className="mt-3 text-center sm:ml-4 sm:mt-0 sm:text-left flex-1">
              <h3 className="text-base font-semibold leading-6 text-gray-900">
                Report Issue to Tracker
              </h3>
              <p className="mt-1 text-sm text-gray-500">
                Create an issue in app-issue-tracker for AI investigation and resolution
              </p>
              {ruleName && (
                <p className="mt-2 text-sm font-medium text-gray-700">
                  <span className="text-gray-500">Rule:</span> {ruleName}
                </p>
              )}
            </div>
          </div>

          <form className="mt-6 space-y-5" onSubmit={handleSubmit}>
            {/* Report Type Dropdown */}
            <div>
              <label htmlFor="report-type" className="block text-sm font-medium text-gray-700 mb-1">
                Report Type
              </label>
              <select
                id="report-type"
                value={reportType}
                onChange={(e) => setReportType(e.target.value as ReportType)}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm py-2"
              >
                {REPORT_TYPE_OPTIONS.map(option => (
                  <option key={option.id} value={option.id} disabled={option.id === 'fix_violations' && !hasViolations}>
                    {option.label}
                  </option>
                ))}
              </select>
              {REPORT_TYPE_OPTIONS.find(opt => opt.id === reportType) && (
                <p className="mt-2 flex items-start gap-2 text-sm text-gray-600">
                  <span className="text-blue-500 flex-shrink-0 mt-0.5">ⓘ</span>
                  <span>{REPORT_TYPE_OPTIONS.find(opt => opt.id === reportType)!.description}</span>
                </p>
              )}
              {reportType === 'fix_violations' && !hasViolations && (
                <div className="mt-2 flex items-start gap-2 rounded-md bg-amber-50 border border-amber-200 p-3">
                  <AlertTriangle className="h-4 w-4 text-amber-500 flex-shrink-0 mt-0.5" />
                  <p className="text-sm text-amber-700">
                    No violations found. Run "Test on Scenario" first to detect violations.
                  </p>
                </div>
              )}
            </div>

            {/* Custom Instructions */}
            <div>
              <label htmlFor="custom-instructions" className="block text-sm font-medium text-gray-700 mb-1">
                Custom Instructions <span className="text-gray-400 font-normal">(optional)</span>
              </label>
              <textarea
                id="custom-instructions"
                rows={4}
                value={customInstructions}
                onChange={(e) => setCustomInstructions(e.target.value)}
                placeholder="Add any special instructions, context, or requirements for the AI agent..."
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm placeholder:text-gray-400"
              />
            </div>

            {/* Scenario Selection (only for fix_violations) */}
            {showScenarioSelection && (
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="block text-sm font-medium text-gray-700">
                    Scenarios with Violations
                  </label>
                  <button
                    type="button"
                    onClick={handleSelectAll}
                    className="text-xs font-medium text-blue-600 hover:text-blue-700 hover:underline"
                  >
                    {selectedScenarios.size === scenariosWithViolations.length ? 'Deselect All' : 'Select All'}
                  </button>
                </div>
                <div className="max-h-48 overflow-y-auto rounded-md border border-gray-200 bg-white">
                  {scenariosWithViolations.map(scenario => (
                    <label
                      key={scenario.name}
                      className="flex items-center gap-3 py-2.5 px-3 cursor-pointer hover:bg-gray-50 border-b border-gray-100 last:border-b-0"
                    >
                      <input
                        type="checkbox"
                        checked={selectedScenarios.has(scenario.name)}
                        onChange={() => handleToggleScenario(scenario.name)}
                        className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                      />
                      <div className="flex-1 min-w-0">
                        <span className="text-sm font-medium text-gray-900 block truncate">{scenario.name}</span>
                        <span className="text-xs text-gray-500">
                          {scenario.violationCount} violation{scenario.violationCount !== 1 ? 's' : ''}
                        </span>
                      </div>
                    </label>
                  ))}
                </div>
                <p className="mt-2 text-xs text-gray-600 font-medium">
                  {selectedScenarios.size} of {scenariosWithViolations.length} scenario{scenariosWithViolations.length !== 1 ? 's' : ''} selected
                </p>
              </div>
            )}

            {/* Batch Size Slider (only when >20 scenarios selected) */}
            {showBatchSizeSlider && (
              <div className="rounded-md bg-blue-50 border border-blue-200 p-4">
                <div className="flex items-start gap-3">
                  <AlertTriangle className="h-5 w-5 text-blue-600 flex-shrink-0 mt-0.5" />
                  <div className="flex-1">
                    <h3 className="text-sm font-medium text-blue-900 mb-3">
                      Large Selection - Batching Required
                    </h3>
                    <div className="space-y-3">
                      <div>
                        <label htmlFor="batch-size" className="block text-xs font-medium text-blue-900 mb-2">
                          Scenarios per Issue: <span className="font-bold">{batchSize}</span>
                        </label>
                        <input
                          id="batch-size"
                          type="range"
                          min="1"
                          max={MAX_SCENARIOS_PER_ISSUE}
                          value={batchSize}
                          onChange={(e) => setBatchSize(Number(e.target.value))}
                          className="w-full h-2 bg-blue-200 rounded-lg appearance-none cursor-pointer accent-blue-600"
                        />
                        <div className="flex justify-between text-xs text-blue-700 mt-1">
                          <span>1</span>
                          <span>{MAX_SCENARIOS_PER_ISSUE}</span>
                        </div>
                      </div>
                      <div className="text-xs text-blue-800 bg-blue-100 rounded px-2 py-1.5">
                        <span className="font-medium">Will create {batchCount} issue{batchCount !== 1 ? 's' : ''}</span> to handle {selectedScenarios.size} scenario{selectedScenarios.size !== 1 ? 's' : ''}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Error Display */}
            {error && (
              <div className="rounded-md bg-red-50 border border-red-200 p-3">
                <div className="flex">
                  <AlertTriangle className="h-5 w-5 text-red-400 flex-shrink-0" />
                  <div className="ml-3">
                    <h3 className="text-sm font-medium text-red-800">Error</h3>
                    <p className="mt-1 text-xs text-red-700">{error}</p>
                  </div>
                </div>
              </div>
            )}

            {/* Submit Buttons */}
            <div className="mt-6 pt-4 border-t border-gray-200 sm:flex sm:flex-row-reverse gap-3">
              <button
                type="submit"
                disabled={!canSubmit || isSubmitting}
                className="inline-flex w-full justify-center items-center gap-2 rounded-md bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-blue-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600 disabled:opacity-50 disabled:cursor-not-allowed sm:w-auto"
              >
                {isSubmitting ? (
                  <>
                    <Clock className="h-4 w-4 animate-spin" />
                    {showBatchSizeSlider && currentBatch > 0
                      ? `Creating issue ${currentBatch} of ${batchCount}...`
                      : 'Submitting...'
                    }
                  </>
                ) : (
                  showBatchSizeSlider ? `Create ${batchCount} Issues` : 'Create Issue'
                )}
              </button>
              <button
                type="button"
                onClick={onClose}
                disabled={isSubmitting}
                className="mt-3 inline-flex w-full justify-center rounded-md bg-white px-4 py-2.5 text-sm font-semibold text-gray-700 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50 disabled:opacity-50 sm:mt-0 sm:w-auto"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
