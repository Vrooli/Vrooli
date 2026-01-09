/**
 * DiagnosticResults Component
 *
 * Displays diagnostic scan results with improved UX.
 */

import { useState, useMemo } from 'react';
import {
  ChevronDown,
  ChevronRight,
  AlertTriangle,
  AlertCircle,
  CheckCircle2,
  Zap,
  ExternalLink,
  Code,
  Globe,
  Radio,
  BarChart3,
  MinusCircle,
} from 'lucide-react';
import { StatusBadge } from './StatusBadge';

// ============================================================================
// Types
// ============================================================================

type DiagnosticCheckStatus = 'passed' | 'failed' | 'warning' | 'skipped';

interface DiagnosticCheck {
  id: string;
  name: string;
  category: 'script' | 'context' | 'events' | 'telemetry';
  status: DiagnosticCheckStatus;
  description: string;
  value?: string;
  details?: string;
}

interface DiagnosticIssue {
  severity: 'error' | 'warning' | 'info';
  category: string;
  message: string;
  suggestion?: string;
  docs_link?: string;
}

interface DiagnosticProvider {
  name: string;
  evaluateIsolated: boolean;
  exposeBindingIsolated: boolean;
}

interface RecordingDiagnostic {
  ready: boolean;
  timestamp: string;
  durationMs: number;
  level: 'quick' | 'standard' | 'full';
  checks?: DiagnosticCheck[];
  issues: DiagnosticIssue[];
  provider?: DiagnosticProvider;
}

interface DiagnosticResult {
  started_at: string;
  completed_at: string;
  duration_ms: number;
  results: {
    recording?: RecordingDiagnostic;
  };
}

// ============================================================================
// Capability Indicator
// ============================================================================

interface CapabilityIndicatorProps {
  label: string;
  enabled: boolean;
  description: string;
}

function CapabilityIndicator({ label, enabled, description }: CapabilityIndicatorProps) {
  return (
    <div
      className={`flex items-center gap-2 p-2 rounded text-xs ${
        enabled ? 'bg-emerald-500/10' : 'bg-gray-700/50'
      }`}
      title={description}
    >
      {enabled ? (
        <CheckCircle2 size={12} className="text-emerald-400 flex-shrink-0" />
      ) : (
        <AlertCircle size={12} className="text-gray-500 flex-shrink-0" />
      )}
      <span className={enabled ? 'text-emerald-300' : 'text-gray-500'}>{label}</span>
    </div>
  );
}

// ============================================================================
// Issue Card
// ============================================================================

interface IssueCardProps {
  issue: DiagnosticIssue;
}

function IssueCard({ issue }: IssueCardProps) {
  const [isExpanded, setIsExpanded] = useState(issue.severity === 'error');

  const colorClasses = {
    error: 'bg-red-500/10 border-red-500/30 text-red-300',
    warning: 'bg-amber-500/10 border-amber-500/30 text-amber-300',
    info: 'bg-blue-500/10 border-blue-500/30 text-blue-300',
  };

  const hasDetails = issue.suggestion || issue.docs_link;

  return (
    <div className={`rounded-lg border ${colorClasses[issue.severity]}`}>
      <button
        onClick={() => hasDetails && setIsExpanded(!isExpanded)}
        className={`w-full px-3 py-2.5 text-left ${hasDetails ? 'cursor-pointer' : 'cursor-default'}`}
        disabled={!hasDetails}
      >
        <div className="flex items-start gap-2">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-[10px] font-mono uppercase tracking-wider px-1.5 py-0.5 rounded bg-black/20">
                {issue.category}
              </span>
            </div>
            <p className="text-sm mt-1.5">{issue.message}</p>
          </div>
          {hasDetails && (
            <div className="flex-shrink-0 mt-0.5">
              {isExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
            </div>
          )}
        </div>
      </button>

      {isExpanded && hasDetails && (
        <div className="px-3 pb-3 pt-0 space-y-2">
          {issue.suggestion && (
            <div className="p-2.5 bg-black/20 rounded-lg">
              <div className="flex items-start gap-2">
                <Zap size={12} className="flex-shrink-0 mt-0.5 opacity-70" />
                <div>
                  <span className="text-[10px] uppercase tracking-wider opacity-70">Suggested Fix</span>
                  <p className="text-xs mt-1 opacity-90">{issue.suggestion}</p>
                </div>
              </div>
            </div>
          )}
          {issue.docs_link && (
            <a
              href={issue.docs_link}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1.5 text-xs hover:underline opacity-80 hover:opacity-100 transition-opacity"
            >
              <ExternalLink size={10} />
              View documentation
            </a>
          )}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Check Status Icon
// ============================================================================

interface CheckStatusIconProps {
  status: DiagnosticCheckStatus;
  size?: number;
}

function CheckStatusIcon({ status, size = 12 }: CheckStatusIconProps) {
  switch (status) {
    case 'passed':
      return <CheckCircle2 size={size} className="text-emerald-400" />;
    case 'failed':
      return <AlertCircle size={size} className="text-red-400" />;
    case 'warning':
      return <AlertTriangle size={size} className="text-amber-400" />;
    case 'skipped':
      return <MinusCircle size={size} className="text-gray-500" />;
  }
}

// ============================================================================
// Category Icon
// ============================================================================

interface CategoryIconProps {
  category: DiagnosticCheck['category'];
  size?: number;
  className?: string;
}

function CategoryIcon({ category, size = 14, className = '' }: CategoryIconProps) {
  switch (category) {
    case 'script':
      return <Code size={size} className={className} />;
    case 'context':
      return <Globe size={size} className={className} />;
    case 'events':
      return <Radio size={size} className={className} />;
    case 'telemetry':
      return <BarChart3 size={size} className={className} />;
  }
}

const CATEGORY_LABELS: Record<DiagnosticCheck['category'], string> = {
  script: 'Script Injection',
  context: 'Execution Context',
  events: 'Event Flow',
  telemetry: 'Browser Telemetry',
};

const CATEGORY_ORDER: DiagnosticCheck['category'][] = ['script', 'context', 'events', 'telemetry'];

// ============================================================================
// Check Row
// ============================================================================

interface CheckRowProps {
  check: DiagnosticCheck;
}

function CheckRow({ check }: CheckRowProps) {
  const statusColors = {
    passed: 'text-emerald-300',
    failed: 'text-red-300',
    warning: 'text-amber-300',
    skipped: 'text-gray-500',
  };

  return (
    <div className="flex items-start gap-2 py-1.5">
      <div className="flex-shrink-0 mt-0.5">
        <CheckStatusIcon status={check.status} />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className={`text-sm ${statusColors[check.status]}`}>{check.name}</span>
          {check.value && (
            <span className="text-xs text-gray-500 font-mono">{check.value}</span>
          )}
        </div>
        {check.details && (
          <p className="text-xs text-gray-500 mt-0.5">{check.details}</p>
        )}
      </div>
    </div>
  );
}

// ============================================================================
// Checks Breakdown
// ============================================================================

interface ChecksBreakdownProps {
  checks: DiagnosticCheck[];
}

function ChecksBreakdown({ checks }: ChecksBreakdownProps) {
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(() => {
    // Start with all categories expanded
    return new Set(CATEGORY_ORDER);
  });

  // Group checks by category
  const checksByCategory = useMemo(() => {
    const grouped: Record<string, DiagnosticCheck[]> = {};
    for (const check of checks) {
      if (!grouped[check.category]) {
        grouped[check.category] = [];
      }
      grouped[check.category].push(check);
    }
    return grouped;
  }, [checks]);

  // Calculate summary stats per category
  const categoryStats = useMemo(() => {
    const stats: Record<string, { passed: number; failed: number; warning: number; skipped: number; total: number }> = {};
    for (const [category, categoryChecks] of Object.entries(checksByCategory)) {
      stats[category] = {
        passed: categoryChecks.filter(c => c.status === 'passed').length,
        failed: categoryChecks.filter(c => c.status === 'failed').length,
        warning: categoryChecks.filter(c => c.status === 'warning').length,
        skipped: categoryChecks.filter(c => c.status === 'skipped').length,
        total: categoryChecks.length,
      };
    }
    return stats;
  }, [checksByCategory]);

  const toggleCategory = (category: string) => {
    setExpandedCategories(prev => {
      const next = new Set(prev);
      if (next.has(category)) {
        next.delete(category);
      } else {
        next.add(category);
      }
      return next;
    });
  };

  return (
    <div className="space-y-2">
      {CATEGORY_ORDER.map(category => {
        const categoryChecks = checksByCategory[category];
        if (!categoryChecks || categoryChecks.length === 0) return null;

        const stats = categoryStats[category];
        const isExpanded = expandedCategories.has(category);
        const hasIssues = stats.failed > 0 || stats.warning > 0;

        return (
          <div
            key={category}
            className={`rounded-lg border ${
              stats.failed > 0
                ? 'border-red-500/30 bg-red-500/5'
                : stats.warning > 0
                ? 'border-amber-500/30 bg-amber-500/5'
                : 'border-gray-700 bg-gray-800/30'
            }`}
          >
            <button
              onClick={() => toggleCategory(category)}
              className="w-full px-3 py-2.5 text-left"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <CategoryIcon
                    category={category}
                    className={
                      stats.failed > 0
                        ? 'text-red-400'
                        : stats.warning > 0
                        ? 'text-amber-400'
                        : 'text-emerald-400'
                    }
                  />
                  <span
                    className={`text-sm font-medium ${
                      stats.failed > 0
                        ? 'text-red-300'
                        : stats.warning > 0
                        ? 'text-amber-300'
                        : 'text-emerald-300'
                    }`}
                  >
                    {CATEGORY_LABELS[category]}
                  </span>
                  <span className="text-xs text-gray-500">
                    {stats.passed}/{stats.total} passed
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  {hasIssues && (
                    <div className="flex items-center gap-1">
                      {stats.failed > 0 && (
                        <span className="flex items-center gap-0.5 text-xs text-red-400">
                          <AlertCircle size={10} />
                          {stats.failed}
                        </span>
                      )}
                      {stats.warning > 0 && (
                        <span className="flex items-center gap-0.5 text-xs text-amber-400">
                          <AlertTriangle size={10} />
                          {stats.warning}
                        </span>
                      )}
                    </div>
                  )}
                  {isExpanded ? (
                    <ChevronDown size={14} className="text-gray-500" />
                  ) : (
                    <ChevronRight size={14} className="text-gray-500" />
                  )}
                </div>
              </div>
            </button>

            {isExpanded && (
              <div className="px-3 pb-3 pt-0 border-t border-gray-700/50">
                <div className="pt-2 space-y-0.5">
                  {categoryChecks.map(check => (
                    <CheckRow key={check.id} check={check} />
                  ))}
                </div>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

// ============================================================================
// Diagnostic Results Card
// ============================================================================

interface DiagnosticResultsCardProps {
  result: DiagnosticResult;
}

export function DiagnosticResultsCard({ result }: DiagnosticResultsCardProps) {
  const [showRawDetails, setShowRawDetails] = useState(false);
  const recording = result.results.recording;

  if (!recording) {
    return (
      <div className="mt-4 p-4 bg-gray-800/50 border border-gray-700 rounded-lg">
        <div className="flex items-center gap-2 text-gray-400">
          <AlertCircle size={16} />
          <span className="text-sm">No diagnostic data available</span>
        </div>
      </div>
    );
  }

  // Group issues by severity
  const errorCount = recording.issues.filter(i => i.severity === 'error').length;
  const warningCount = recording.issues.filter(i => i.severity === 'warning').length;
  const infoCount = recording.issues.filter(i => i.severity === 'info').length;
  const hasIssues = recording.issues.length > 0;
  const isHealthy = recording.ready && errorCount === 0;

  // Determine overall status
  const overallStatus = isHealthy ? 'healthy' : errorCount > 0 ? 'error' : 'degraded';
  const statusLabel = isHealthy ? 'All Tests Passed' : errorCount > 0 ? `${errorCount} Issue${errorCount !== 1 ? 's' : ''} Found` : `${warningCount} Warning${warningCount !== 1 ? 's' : ''}`;

  // Format timestamp
  const scanTime = new Date(result.completed_at);
  const timeString = scanTime.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

  return (
    <div className="mt-4 rounded-lg overflow-hidden border border-gray-700">
      {/* Header - Overall Status */}
      <div className={`px-4 py-3 ${
        isHealthy ? 'bg-emerald-500/10' : errorCount > 0 ? 'bg-red-500/10' : 'bg-amber-500/10'
      }`}>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {isHealthy ? (
              <div className="flex items-center justify-center w-8 h-8 rounded-full bg-emerald-500/20">
                <CheckCircle2 size={18} className="text-emerald-400" />
              </div>
            ) : errorCount > 0 ? (
              <div className="flex items-center justify-center w-8 h-8 rounded-full bg-red-500/20">
                <AlertCircle size={18} className="text-red-400" />
              </div>
            ) : (
              <div className="flex items-center justify-center w-8 h-8 rounded-full bg-amber-500/20">
                <AlertTriangle size={18} className="text-amber-400" />
              </div>
            )}
            <div>
              <div className="flex items-center gap-2">
                <span className={`font-medium ${
                  isHealthy ? 'text-emerald-300' : errorCount > 0 ? 'text-red-300' : 'text-amber-300'
                }`}>
                  {statusLabel}
                </span>
                <StatusBadge status={overallStatus} />
              </div>
              <div className="flex items-center gap-3 mt-0.5 text-xs text-gray-500">
                <span>Scanned at {timeString}</span>
                <span>•</span>
                <span>{result.duration_ms}ms</span>
                <span>•</span>
                <span className="capitalize">{recording.level} scan</span>
              </div>
            </div>
          </div>

          {/* Issue Summary Badges */}
          {hasIssues && (
            <div className="flex items-center gap-2">
              {errorCount > 0 && (
                <span className="flex items-center gap-1 px-2 py-1 text-xs rounded-full bg-red-500/20 text-red-300">
                  <AlertCircle size={12} />
                  {errorCount}
                </span>
              )}
              {warningCount > 0 && (
                <span className="flex items-center gap-1 px-2 py-1 text-xs rounded-full bg-amber-500/20 text-amber-300">
                  <AlertTriangle size={12} />
                  {warningCount}
                </span>
              )}
              {infoCount > 0 && (
                <span className="flex items-center gap-1 px-2 py-1 text-xs rounded-full bg-blue-500/20 text-blue-300">
                  <CheckCircle2 size={12} />
                  {infoCount}
                </span>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Body */}
      <div className="bg-gray-800/50 p-4 space-y-4">
        {/* Checks Breakdown - Show when we have checks data */}
        {recording.checks && recording.checks.length > 0 && (
          <ChecksBreakdown checks={recording.checks} />
        )}

        {/* Fallback Success State - Only show if no checks data */}
        {isHealthy && !hasIssues && (!recording.checks || recording.checks.length === 0) && (
          <div className="flex items-center gap-3 p-4 bg-emerald-500/5 border border-emerald-500/20 rounded-lg">
            <div className="flex-shrink-0">
              <CheckCircle2 size={24} className="text-emerald-400" />
            </div>
            <div>
              <p className="text-sm text-emerald-300 font-medium">Recording System Healthy</p>
              <p className="text-xs text-emerald-400/70 mt-1">
                All diagnostic checks completed successfully. The recording script is properly injected and events are flowing correctly.
              </p>
            </div>
          </div>
        )}

        {/* Issues List - Grouped by Severity */}
        {hasIssues && (
          <div className="space-y-3">
            {/* Errors */}
            {errorCount > 0 && (
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-xs text-red-400 font-medium uppercase tracking-wide">
                  <AlertCircle size={12} />
                  <span>Errors</span>
                </div>
                {recording.issues
                  .filter(i => i.severity === 'error')
                  .map((issue, idx) => (
                    <IssueCard key={`error-${idx}`} issue={issue} />
                  ))}
              </div>
            )}

            {/* Warnings */}
            {warningCount > 0 && (
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-xs text-amber-400 font-medium uppercase tracking-wide">
                  <AlertTriangle size={12} />
                  <span>Warnings</span>
                </div>
                {recording.issues
                  .filter(i => i.severity === 'warning')
                  .map((issue, idx) => (
                    <IssueCard key={`warning-${idx}`} issue={issue} />
                  ))}
              </div>
            )}

            {/* Info */}
            {infoCount > 0 && (
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-xs text-blue-400 font-medium uppercase tracking-wide">
                  <CheckCircle2 size={12} />
                  <span>Info</span>
                </div>
                {recording.issues
                  .filter(i => i.severity === 'info')
                  .map((issue, idx) => (
                    <IssueCard key={`info-${idx}`} issue={issue} />
                  ))}
              </div>
            )}
          </div>
        )}

        {/* Technical Details (Collapsible) */}
        <div className="pt-3 border-t border-gray-700">
          <button
            onClick={() => setShowRawDetails(!showRawDetails)}
            className="flex items-center gap-2 text-xs text-gray-500 hover:text-gray-400 transition-colors"
          >
            {showRawDetails ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
            <span>Technical Details</span>
          </button>

          {showRawDetails && (
            <div className="mt-3 space-y-3">
              {/* Provider Info */}
              {recording.provider && (
                <div className="p-3 bg-gray-900/50 rounded-lg space-y-2">
                  <div className="text-xs text-gray-400 font-medium">Browser Provider</div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-surface">{recording.provider.name}</span>
                  </div>
                  <div className="grid grid-cols-2 gap-2 mt-2">
                    <CapabilityIndicator
                      label="Isolated Evaluation"
                      enabled={recording.provider.evaluateIsolated}
                      description="Can run scripts in isolated context"
                    />
                    <CapabilityIndicator
                      label="Isolated Bindings"
                      enabled={recording.provider.exposeBindingIsolated}
                      description="Can expose bindings in isolated context"
                    />
                  </div>
                </div>
              )}

              {/* Timing Details */}
              <div className="p-3 bg-gray-900/50 rounded-lg">
                <div className="text-xs text-gray-400 font-medium mb-2">Timing</div>
                <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
                  <span className="text-gray-500">Started:</span>
                  <span className="text-gray-300 font-mono">{new Date(result.started_at).toLocaleTimeString()}</span>
                  <span className="text-gray-500">Completed:</span>
                  <span className="text-gray-300 font-mono">{new Date(result.completed_at).toLocaleTimeString()}</span>
                  <span className="text-gray-500">Duration:</span>
                  <span className="text-gray-300 font-mono">{result.duration_ms}ms</span>
                  <span className="text-gray-500">Scan Level:</span>
                  <span className="text-gray-300 capitalize">{recording.level}</span>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default DiagnosticResultsCard;
