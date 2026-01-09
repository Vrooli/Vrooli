/**
 * DriverStatusIndicator
 *
 * A compact indicator for the header bar that shows playwright-driver health status.
 * Click to expand and see detailed status information.
 */

import React, { useState } from 'react';
import {
  useDriverStatus,
  getStatusLabel,
  getStatusColor,
  formatDuration,
  DriverHealthStatus,
} from '../../hooks/useDriverStatus';

/** Props for the DriverStatusIndicator */
interface DriverStatusIndicatorProps {
  /** Additional class names */
  className?: string;
}

/** Get the indicator dot classes based on status */
function getIndicatorClasses(status: DriverHealthStatus): {
  dot: string;
  ping: string;
  animate: boolean;
} {
  switch (status) {
    case 'healthy':
      return {
        dot: 'bg-green-400',
        ping: 'bg-green-400',
        animate: false,
      };
    case 'degraded':
      return {
        dot: 'bg-yellow-400',
        ping: 'bg-yellow-400',
        animate: true,
      };
    case 'unhealthy':
      return {
        dot: 'bg-red-500',
        ping: 'bg-red-500',
        animate: false,
      };
    case 'restarting':
      return {
        dot: 'bg-yellow-400',
        ping: 'bg-yellow-400',
        animate: true,
      };
    case 'unrecoverable':
      return {
        dot: 'bg-red-600',
        ping: 'bg-red-600',
        animate: false,
      };
    default:
      return {
        dot: 'bg-gray-400',
        ping: 'bg-gray-400',
        animate: false,
      };
  }
}

export const DriverStatusIndicator: React.FC<DriverStatusIndicatorProps> = ({ className = '' }) => {
  const { health, isConnected, isSubscribed } = useDriverStatus();
  const [isExpanded, setIsExpanded] = useState(false);

  // Determine display status
  const displayStatus: DriverHealthStatus = !isConnected
    ? 'unknown'
    : !health
      ? 'unknown'
      : health.status;

  const indicator = getIndicatorClasses(displayStatus);
  const label = getStatusLabel(displayStatus);
  const colorClass = getStatusColor(displayStatus);

  return (
    <div className={`relative ${className}`}>
      {/* Compact indicator button */}
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex items-center gap-2 px-3 py-1.5 bg-gray-800/50 hover:bg-gray-700/50 border border-gray-700 rounded-lg transition-colors"
        title={`Driver: ${label}`}
      >
        <div className="relative flex items-center justify-center w-4 h-4">
          {indicator.animate && (
            <div
              className={`absolute inset-0 ${indicator.ping} rounded-full animate-ping opacity-50`}
            />
          )}
          <div className={`w-2.5 h-2.5 ${indicator.dot} rounded-full`} />
        </div>
        <span className={`text-sm font-medium ${colorClass}`}>{label}</span>
        <svg
          className={`w-4 h-4 text-gray-400 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {/* Expanded dropdown */}
      {isExpanded && (
        <>
          {/* Backdrop */}
          <div className="fixed inset-0 z-40" onClick={() => setIsExpanded(false)} />

          {/* Dropdown panel */}
          <div className="absolute right-0 top-full mt-2 w-80 bg-gray-900 border border-gray-700 rounded-lg shadow-xl z-50 overflow-hidden">
            {/* Header */}
            <div className="px-4 py-3 border-b border-gray-700 bg-gray-800/50">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-gray-200">Driver Status</span>
                <div className="flex items-center gap-2">
                  <div className={`w-2 h-2 ${indicator.dot} rounded-full`} />
                  <span className={`text-sm ${colorClass}`}>{label}</span>
                </div>
              </div>
            </div>

            {/* Content */}
            <div className="p-4 space-y-3">
              {!health ? (
                <div className="text-sm text-gray-400">
                  {isConnected
                    ? 'Waiting for driver status...'
                    : 'WebSocket disconnected. Reconnecting...'}
                </div>
              ) : (
                <>
                  {/* Status details */}
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <div className="text-xs text-gray-500 uppercase tracking-wide">
                        Circuit Breaker
                      </div>
                      <div className="text-sm text-gray-300 mt-0.5">
                        {health.circuitBreaker === 'closed'
                          ? 'Closed (OK)'
                          : health.circuitBreaker === 'open'
                            ? 'Open (Blocking)'
                            : health.circuitBreaker === 'half-open'
                              ? 'Half-open (Testing)'
                              : 'Disabled'}
                      </div>
                    </div>
                    <div>
                      <div className="text-xs text-gray-500 uppercase tracking-wide">
                        Active Sessions
                      </div>
                      <div className="text-sm text-gray-300 mt-0.5">{health.activeSessions}</div>
                    </div>
                    <div>
                      <div className="text-xs text-gray-500 uppercase tracking-wide">
                        Restart Count
                      </div>
                      <div className="text-sm text-gray-300 mt-0.5">{health.restartCount}</div>
                    </div>
                    <div>
                      <div className="text-xs text-gray-500 uppercase tracking-wide">Uptime</div>
                      <div className="text-sm text-gray-300 mt-0.5">
                        {formatDuration(health.uptimeMs)}
                      </div>
                    </div>
                  </div>

                  {/* Estimated recovery time */}
                  {health.estimatedRecoveryMs && health.estimatedRecoveryMs > 0 && (
                    <div className="mt-3 p-2 bg-yellow-900/20 border border-yellow-700/30 rounded">
                      <div className="flex items-center gap-2">
                        <svg
                          className="w-4 h-4 text-yellow-500"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                          />
                        </svg>
                        <span className="text-sm text-yellow-400">
                          Estimated recovery: {formatDuration(health.estimatedRecoveryMs)}
                        </span>
                      </div>
                    </div>
                  )}

                  {/* Last error */}
                  {health.lastError && (
                    <div className="mt-3 p-2 bg-red-900/20 border border-red-700/30 rounded">
                      <div className="text-xs text-gray-500 uppercase tracking-wide mb-1">
                        Last Error
                      </div>
                      <div className="text-sm text-red-400 break-words">{health.lastError}</div>
                    </div>
                  )}

                  {/* Updated timestamp */}
                  <div className="text-xs text-gray-500 mt-2">
                    Updated: {new Date(health.updatedAt).toLocaleTimeString()}
                  </div>
                </>
              )}
            </div>

            {/* Footer with subscription status */}
            <div className="px-4 py-2 border-t border-gray-700 bg-gray-800/30">
              <div className="flex items-center justify-between text-xs text-gray-500">
                <span>
                  WebSocket: {isConnected ? 'Connected' : 'Disconnected'}
                  {isSubscribed && ' (subscribed)'}
                </span>
                {!isConnected && (
                  <button
                    className="text-blue-400 hover:text-blue-300"
                    onClick={() => window.location.reload()}
                  >
                    Refresh
                  </button>
                )}
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
};
