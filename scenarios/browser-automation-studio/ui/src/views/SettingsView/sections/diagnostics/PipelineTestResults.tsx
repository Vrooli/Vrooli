/**
 * PipelineTestResults Component
 *
 * Displays the results of the automated recording pipeline test.
 * Shows step-by-step results with detailed diagnostics for failures.
 */

import { useState } from 'react';
import {
  CheckCircle2,
  XCircle,
  ChevronDown,
  ChevronRight,
  Clock,
  Lightbulb,
  Activity,
} from 'lucide-react';
import type { PipelineTestResponse, PipelineStepResult } from '@/domains/observability';

interface PipelineTestResultsProps {
  result: PipelineTestResponse;
}

export function PipelineTestResults({ result }: PipelineTestResultsProps) {
  const [showDetails, setShowDetails] = useState(!result.success);
  const [showDiagnostics, setShowDiagnostics] = useState(false);

  return (
    <div className="space-y-4">
      {/* Overall Result */}
      <div
        className={`p-4 rounded-lg border ${
          result.success
            ? 'bg-emerald-500/10 border-emerald-500/30'
            : 'bg-red-500/10 border-red-500/30'
        }`}
      >
        <div className="flex items-start gap-3">
          {result.success ? (
            <CheckCircle2 size={24} className="text-emerald-400 flex-shrink-0 mt-0.5" />
          ) : (
            <XCircle size={24} className="text-red-400 flex-shrink-0 mt-0.5" />
          )}
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <span
                className={`font-semibold ${result.success ? 'text-emerald-300' : 'text-red-300'}`}
              >
                {result.success ? 'Pipeline Test Passed' : 'Pipeline Test Failed'}
              </span>
              <span className="text-xs text-gray-500 flex items-center gap-1">
                <Clock size={12} />
                {result.duration_ms}ms
              </span>
            </div>

            {!result.success && result.failure_message && (
              <p className="text-sm text-red-200/80 mt-2">{result.failure_message}</p>
            )}

            {!result.success && result.failure_point && (
              <div className="mt-2 flex items-center gap-2">
                <span className="text-xs px-2 py-0.5 bg-red-500/20 text-red-300 rounded">
                  Failed at: {result.failure_point.replace(/_/g, ' ')}
                </span>
              </div>
            )}
          </div>
        </div>

        {/* Suggestions */}
        {!result.success && result.suggestions && result.suggestions.length > 0 && (
          <div className="mt-4 p-3 bg-amber-500/10 border border-amber-500/20 rounded-lg">
            <div className="flex items-center gap-2 mb-2">
              <Lightbulb size={16} className="text-amber-400" />
              <span className="text-sm font-medium text-amber-300">Suggestions</span>
            </div>
            <ul className="text-sm text-amber-200/80 space-y-1">
              {result.suggestions.map((suggestion, i) => (
                <li key={i} className="flex items-start gap-2">
                  <span className="text-amber-400">•</span>
                  <span>{suggestion}</span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>

      {/* Steps */}
      <div className="bg-gray-800/50 border border-gray-700 rounded-lg">
        <button
          onClick={() => setShowDetails(!showDetails)}
          className="w-full flex items-center justify-between p-3 hover:bg-gray-700/50 transition-colors"
        >
          <div className="flex items-center gap-2">
            <Activity size={16} className="text-gray-400" />
            <span className="text-sm font-medium text-surface">Test Steps</span>
            <span className="text-xs text-gray-500">
              ({result.steps.filter((s) => s.passed).length}/{result.steps.length} passed)
            </span>
          </div>
          {showDetails ? (
            <ChevronDown size={16} className="text-gray-400" />
          ) : (
            <ChevronRight size={16} className="text-gray-400" />
          )}
        </button>

        {showDetails && (
          <div className="border-t border-gray-700 p-3 space-y-2">
            {result.steps.map((step, i) => (
              <StepResult key={i} step={step} />
            ))}
          </div>
        )}
      </div>

      {/* Diagnostics */}
      <div className="bg-gray-800/50 border border-gray-700 rounded-lg">
        <button
          onClick={() => setShowDiagnostics(!showDiagnostics)}
          className="w-full flex items-center justify-between p-3 hover:bg-gray-700/50 transition-colors"
        >
          <div className="flex items-center gap-2">
            <Activity size={16} className="text-gray-400" />
            <span className="text-sm font-medium text-surface">Raw Diagnostics</span>
          </div>
          {showDiagnostics ? (
            <ChevronDown size={16} className="text-gray-400" />
          ) : (
            <ChevronRight size={16} className="text-gray-400" />
          )}
        </button>

        {showDiagnostics && (
          <div className="border-t border-gray-700 p-3">
            <pre className="text-xs text-gray-400 overflow-auto max-h-64">
              {JSON.stringify(result.diagnostics, null, 2)}
            </pre>
          </div>
        )}
      </div>
    </div>
  );
}

function StepResult({ step }: { step: PipelineStepResult }) {
  const [showDetails, setShowDetails] = useState(!step.passed && !!step.details);

  return (
    <div
      className={`p-2 rounded ${
        step.passed ? 'bg-emerald-500/5' : 'bg-red-500/10 border border-red-500/20'
      }`}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          {step.passed ? (
            <CheckCircle2 size={14} className="text-emerald-400" />
          ) : (
            <XCircle size={14} className="text-red-400" />
          )}
          <span className="text-sm text-surface">{formatStepName(step.name)}</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-gray-500">{step.duration_ms}ms</span>
          {step.details && (
            <button
              onClick={() => setShowDetails(!showDetails)}
              className="text-xs text-gray-400 hover:text-surface"
            >
              {showDetails ? 'Hide' : 'Details'}
            </button>
          )}
        </div>
      </div>

      {step.error && <p className="text-xs text-red-300 mt-1 ml-6">{step.error}</p>}

      {showDetails && step.details && (
        <pre className="text-xs text-gray-400 mt-2 ml-6 overflow-auto max-h-32">
          {JSON.stringify(step.details, null, 2)}
        </pre>
      )}
    </div>
  );
}

function formatStepName(name: string): string {
  return name
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

export default PipelineTestResults;
