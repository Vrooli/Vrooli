/**
 * ValidationAccordion Component
 *
 * Displays validation checks in an expandable accordion.
 * - Collapsed: Shows status icon + summary text (e.g., "3 passed, 1 warning")
 * - Expanded: Shows all checks with icons, labels, and descriptions
 * - Auto-expands when there are warnings/errors, collapsed when all pass
 */

import { useState, useEffect, useMemo } from 'react';
import { ChevronDown, ChevronRight, Check, AlertTriangle, AlertCircle, Info } from 'lucide-react';
import type { ValidationSummary, ValidationCheck, ValidationCheckStatus } from '../types';

interface ValidationAccordionProps {
  validation: ValidationSummary;
  className?: string;
}

const statusConfig: Record<
  ValidationCheckStatus,
  {
    bg: string;
    text: string;
    icon: React.ReactNode;
    headerBg: string;
  }
> = {
  pass: {
    bg: 'bg-emerald-500/10 border-emerald-500/30',
    text: 'text-emerald-300',
    icon: <Check size={16} />,
    headerBg: 'bg-emerald-500/5',
  },
  warn: {
    bg: 'bg-amber-500/10 border-amber-500/30',
    text: 'text-amber-300',
    icon: <AlertTriangle size={16} />,
    headerBg: 'bg-amber-500/5',
  },
  error: {
    bg: 'bg-red-500/10 border-red-500/30',
    text: 'text-red-300',
    icon: <AlertCircle size={16} />,
    headerBg: 'bg-red-500/5',
  },
  info: {
    bg: 'bg-blue-500/10 border-blue-500/30',
    text: 'text-blue-300',
    icon: <Info size={16} />,
    headerBg: 'bg-blue-500/5',
  },
};

export function ValidationAccordion({ validation, className = '' }: ValidationAccordionProps) {
  // Auto-expand if there are warnings or errors
  const hasIssues = validation.error_count > 0 || validation.warn_count > 0;
  const [isExpanded, setIsExpanded] = useState(hasIssues);

  // Update expansion state when validation changes
  useEffect(() => {
    if (hasIssues) {
      setIsExpanded(true);
    }
  }, [hasIssues]);

  const summaryText = useMemo(() => {
    const parts: string[] = [];
    if (validation.pass_count > 0) {
      parts.push(`${validation.pass_count} passed`);
    }
    if (validation.warn_count > 0) {
      parts.push(`${validation.warn_count} warning${validation.warn_count !== 1 ? 's' : ''}`);
    }
    if (validation.error_count > 0) {
      parts.push(`${validation.error_count} error${validation.error_count !== 1 ? 's' : ''}`);
    }
    if (validation.info_count > 0) {
      parts.push(`${validation.info_count} info`);
    }
    return parts.join(', ') || 'No checks';
  }, [validation]);

  const headerText = useMemo(() => {
    if (validation.overall_status === 'error') {
      return 'Validation Failed';
    }
    if (validation.overall_status === 'warn') {
      return 'Validation Passed with Warnings';
    }
    return 'Validation Passed';
  }, [validation.overall_status]);

  const overallStyle = statusConfig[validation.overall_status];

  return (
    <div className={`rounded-xl border ${overallStyle.bg} overflow-hidden ${className}`}>
      {/* Header - always visible */}
      <button
        type="button"
        onClick={() => setIsExpanded(!isExpanded)}
        className={`w-full px-4 py-3 flex items-center justify-between hover:${overallStyle.headerBg} transition-colors`}
      >
        <div className="flex items-center gap-3">
          <div
            className={`flex items-center justify-center w-8 h-8 rounded-lg ${overallStyle.bg}`}
          >
            <span className={overallStyle.text}>{overallStyle.icon}</span>
          </div>
          <div className="text-left">
            <span className={`text-sm font-medium ${overallStyle.text}`}>{headerText}</span>
            <p className="text-xs text-gray-500">{summaryText}</p>
          </div>
        </div>
        <div className="text-gray-500">
          {isExpanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
        </div>
      </button>

      {/* Expanded content */}
      {isExpanded && (
        <div className="px-4 pb-4 border-t border-gray-700/50">
          <div className="pt-3 space-y-2">
            {validation.checks.map((check) => (
              <ValidationCheckRow key={check.id} check={check} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

interface ValidationCheckRowProps {
  check: ValidationCheck;
}

function ValidationCheckRow({ check }: ValidationCheckRowProps) {
  const checkStatusConfig: Record<
    ValidationCheckStatus,
    { icon: React.ReactNode; color: string }
  > = {
    pass: { icon: <Check size={14} />, color: 'text-emerald-400' },
    warn: { icon: <AlertTriangle size={14} />, color: 'text-amber-400' },
    error: { icon: <AlertCircle size={14} />, color: 'text-red-400' },
    info: { icon: <Info size={14} />, color: 'text-blue-400' },
  };

  const config = checkStatusConfig[check.status];

  return (
    <div className="flex items-start gap-2 py-1.5">
      <div className={`flex-shrink-0 mt-0.5 ${config.color}`}>{config.icon}</div>
      <div className="flex-1 min-w-0">
        <span className={`text-sm ${config.color}`}>{check.label}</span>
        <p className="text-xs text-gray-500 mt-0.5">{check.description}</p>
      </div>
    </div>
  );
}

export default ValidationAccordion;
