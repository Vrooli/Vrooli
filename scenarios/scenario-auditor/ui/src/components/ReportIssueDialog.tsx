import React, { useState, useMemo, useEffect } from 'react'
import { X, AlertTriangle, CheckCircle, Clock, ExternalLink, FileText, Bug, Wrench } from 'lucide-react'
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
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<{ issueId: string; issueUrl?: string; message: string } | null>(null)

  // Reset state when dialog opens/closes
  useEffect(() => {
    if (isOpen) {
      setReportType('fix_violations')
      setCustomInstructions('')
      setError(null)
      setResult(null)

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

  const showWarning = selectedScenarios.size > MAX_SCENARIOS_WARNING_THRESHOLD

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

    try {
      const payload: ReportPayload = {
        reportType,
        ruleId,
        customInstructions: customInstructions.trim(),
        selectedScenarios: Array.from(selectedScenarios),
      }

      const response = await onSubmitReport(payload)
      setResult(response)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit report')
    } finally {
      setIsSubmitting(false)
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

  // Success state
  if (result) {
    return (
      <div className="fixed inset-0 z-[60] overflow-y-auto">
        <div className="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <div
            className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"
            onClick={onClose}
          />

          <div className="relative transform overflow-hidden rounded-lg bg-white px-4 pb-4 pt-5 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg sm:p-6">
            <div>
              <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-green-100">
                <CheckCircle className="h-6 w-6 text-green-600" />
              </div>
              <div className="mt-3 text-center sm:mt-5">
                <h3 className="text-lg font-semibold leading-6 text-gray-900">
                  Issue Created Successfully
                </h3>
                <div className="mt-2">
                  <p className="text-sm text-gray-500">{result.message}</p>
                  {result.issueId && (
                    <p className="mt-2 text-sm font-medium text-gray-700">
                      Issue ID: <span className="font-mono text-blue-600">{result.issueId}</span>
                    </p>
                  )}
                </div>
              </div>
            </div>

            <div className="mt-5 sm:mt-6 sm:grid sm:grid-flow-row-dense sm:grid-cols-2 sm:gap-3">
              {result.issueUrl && (
                <a
                  href={result.issueUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex w-full justify-center items-center gap-2 rounded-md bg-blue-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-blue-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600 sm:col-start-2"
                >
                  View Issue
                  <ExternalLink className="h-4 w-4" />
                </a>
              )}
              <button
                type="button"
                className="mt-3 inline-flex w-full justify-center rounded-md bg-white px-3 py-2 text-sm font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50 sm:col-start-1 sm:mt-0"
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
                {showWarning && (
                  <div className="mt-2 rounded-md bg-amber-50 border border-amber-200 p-3">
                    <div className="flex">
                      <AlertTriangle className="h-5 w-5 text-amber-400 flex-shrink-0" />
                      <div className="ml-3">
                        <h3 className="text-sm font-medium text-amber-800">Large selection</h3>
                        <p className="mt-1 text-xs text-amber-700">
                          You've selected more than {MAX_SCENARIOS_WARNING_THRESHOLD} scenarios.
                          This may be difficult to handle in a single issue. Consider splitting into multiple reports.
                        </p>
                      </div>
                    </div>
                  </div>
                )}
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
                    Submitting...
                  </>
                ) : (
                  'Create Issue'
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
