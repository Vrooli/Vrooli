/**
 * ValidationStatus Component
 *
 * Displays validation feedback badges and status indicators.
 * Used across import modals to show validation results.
 */

import { Check, AlertTriangle, AlertCircle, Info, X } from 'lucide-react';
import type { ValidationResult } from '../types';

export interface ValidationStatusProps {
  /** Validation result to display */
  result?: ValidationResult;
  /** Custom class name */
  className?: string;
}

export function ValidationStatus({ result, className = '' }: ValidationStatusProps) {
  if (!result) return null;

  const { isValid, errors, warnings } = result;

  return (
    <div className={`space-y-2 ${className}`}>
      {/* Errors */}
      {errors.map((error, index) => (
        <ValidationBadge key={`error-${index}`} type="error" message={error.message} />
      ))}

      {/* Warnings */}
      {warnings.map((warning, index) => (
        <ValidationBadge key={`warning-${index}`} type="warning" message={warning.message} />
      ))}

      {/* Success indicator when valid with no warnings */}
      {isValid && errors.length === 0 && warnings.length === 0 && (
        <ValidationBadge type="success" message="Validation passed" />
      )}
    </div>
  );
}

/** Individual status badge */
export interface StatusBadgeProps {
  /** Whether the status is positive */
  success: boolean;
  /** Label when status is positive */
  label: string;
  /** Label when status is negative */
  warningLabel?: string;
  /** Custom class name */
  className?: string;
}

export function StatusBadge({ success, label, warningLabel, className = '' }: StatusBadgeProps) {
  return (
    <div
      className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm ${
        success
          ? 'bg-green-500/10 border border-green-500/30 text-green-300'
          : 'bg-yellow-500/10 border border-yellow-500/30 text-yellow-300'
      } ${className}`}
    >
      {success ? <Check size={14} /> : <AlertTriangle size={14} />}
      <span>{success ? label : warningLabel || label}</span>
    </div>
  );
}

/** Validation badge types */
type BadgeType = 'success' | 'error' | 'warning' | 'info';

interface ValidationBadgeProps {
  type: BadgeType;
  message: string;
  dismissible?: boolean;
  onDismiss?: () => void;
  className?: string;
}

export function ValidationBadge({
  type,
  message,
  dismissible = false,
  onDismiss,
  className = '',
}: ValidationBadgeProps) {
  const styles = {
    success: {
      bg: 'bg-green-500/10 border-green-500/30',
      text: 'text-green-300',
      icon: <Check size={14} />,
    },
    error: {
      bg: 'bg-red-500/10 border-red-500/30',
      text: 'text-red-300',
      icon: <AlertCircle size={14} />,
    },
    warning: {
      bg: 'bg-yellow-500/10 border-yellow-500/30',
      text: 'text-yellow-300',
      icon: <AlertTriangle size={14} />,
    },
    info: {
      bg: 'bg-blue-500/10 border-blue-500/30',
      text: 'text-blue-300',
      icon: <Info size={14} />,
    },
  };

  const style = styles[type];

  return (
    <div
      className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm border ${style.bg} ${style.text} ${className}`}
    >
      {style.icon}
      <span className="flex-1">{message}</span>
      {dismissible && onDismiss && (
        <button
          type="button"
          onClick={onDismiss}
          className="p-0.5 hover:bg-white/10 rounded transition-colors"
          aria-label="Dismiss"
        >
          <X size={12} />
        </button>
      )}
    </div>
  );
}

/** Alert box for larger messages */
export interface AlertBoxProps {
  type: BadgeType;
  title: string;
  message: string;
  dismissible?: boolean;
  onDismiss?: () => void;
  className?: string;
}

export function AlertBox({
  type,
  title,
  message,
  dismissible = false,
  onDismiss,
  className = '',
}: AlertBoxProps) {
  const styles = {
    success: {
      bg: 'bg-green-500/10 border-green-500/30',
      title: 'text-green-300',
      text: 'text-green-300/70',
      iconBg: 'bg-green-500/20',
      iconColor: 'text-green-400',
      icon: <Check size={16} />,
    },
    error: {
      bg: 'bg-red-500/10 border-red-500/30',
      title: 'text-red-300',
      text: 'text-red-300/70',
      iconBg: 'bg-red-500/20',
      iconColor: 'text-red-400',
      icon: <AlertCircle size={16} />,
    },
    warning: {
      bg: 'bg-yellow-500/10 border-yellow-500/30',
      title: 'text-yellow-300',
      text: 'text-yellow-300/70',
      iconBg: 'bg-yellow-500/20',
      iconColor: 'text-yellow-400',
      icon: <AlertTriangle size={16} />,
    },
    info: {
      bg: 'bg-blue-500/10 border-blue-500/30',
      title: 'text-blue-300',
      text: 'text-blue-300/70',
      iconBg: 'bg-blue-500/20',
      iconColor: 'text-blue-400',
      icon: <Info size={16} />,
    },
  };

  const style = styles[type];

  return (
    <div className={`p-3 rounded-xl border flex items-start gap-3 ${style.bg} ${className}`}>
      <div className={`w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 ${style.iconBg}`}>
        <span className={style.iconColor}>{style.icon}</span>
      </div>
      <div className="flex-1 min-w-0">
        <p className={`text-sm font-medium ${style.title}`}>{title}</p>
        <p className={`text-xs mt-1 ${style.text}`}>{message}</p>
      </div>
      {dismissible && onDismiss && (
        <button
          type="button"
          onClick={onDismiss}
          className={`p-1 hover:bg-white/10 rounded transition-colors ${style.iconColor}`}
          aria-label="Dismiss"
        >
          <X size={14} />
        </button>
      )}
    </div>
  );
}

export default ValidationStatus;
