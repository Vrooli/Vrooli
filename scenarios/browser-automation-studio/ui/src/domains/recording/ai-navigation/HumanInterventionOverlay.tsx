/**
 * HumanInterventionOverlay Component
 *
 * Displays an overlay when AI navigation requires human intervention.
 * Shown for both programmatic detection (CAPTCHAs) and AI-requested pauses.
 */

import { useState, useEffect } from 'react';
import type { HumanInterventionState } from './types';

interface HumanInterventionOverlayProps {
  intervention: HumanInterventionState;
  onComplete: () => void;
  onAbort: () => void;
}

/** Get icon for intervention type */
function getInterventionIcon(type: HumanInterventionState['interventionType']) {
  switch (type) {
    case 'captcha':
      return (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
        </svg>
      );
    case 'verification':
      return (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
        </svg>
      );
    case 'login_required':
      return (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
        </svg>
      );
    case 'complex_interaction':
      return (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z" />
        </svg>
      );
    default:
      return (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      );
  }
}

/** Get title for intervention type */
function getInterventionTitle(type: HumanInterventionState['interventionType']): string {
  switch (type) {
    case 'captcha':
      return 'CAPTCHA Detected';
    case 'verification':
      return 'Verification Required';
    case 'login_required':
      return 'Login Required';
    case 'complex_interaction':
      return 'Complex Interaction';
    default:
      return 'Human Help Needed';
  }
}

/** Format elapsed time */
function formatElapsedTime(startedAt: Date): string {
  const elapsed = Math.floor((Date.now() - startedAt.getTime()) / 1000);
  const minutes = Math.floor(elapsed / 60);
  const seconds = elapsed % 60;
  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  }
  return `${seconds}s`;
}

export function HumanInterventionOverlay({
  intervention,
  onComplete,
  onAbort,
}: HumanInterventionOverlayProps) {
  const [elapsedTime, setElapsedTime] = useState('0s');

  // Update elapsed time every second
  useEffect(() => {
    const updateTime = () => setElapsedTime(formatElapsedTime(intervention.startedAt));
    updateTime();
    const interval = setInterval(updateTime, 1000);
    return () => clearInterval(interval);
  }, [intervention.startedAt]);

  return (
    <div className="absolute inset-x-0 bottom-0 z-50 animate-slide-up">
      {/* Semi-transparent backdrop */}
      <div className="absolute inset-0 bg-gradient-to-t from-black/60 to-transparent pointer-events-none" />

      {/* Overlay content */}
      <div className="relative bg-white dark:bg-gray-800 border-t border-purple-300 dark:border-purple-700 shadow-2xl">
        <div className="max-w-2xl mx-auto p-6">
          {/* Header */}
          <div className="flex items-start gap-4">
            {/* Icon */}
            <div className="flex-shrink-0 p-3 bg-purple-100 dark:bg-purple-900/50 text-purple-600 dark:text-purple-400 rounded-xl">
              {getInterventionIcon(intervention.interventionType)}
            </div>

            {/* Content */}
            <div className="flex-1 min-w-0">
              {/* Title and badges */}
              <div className="flex items-center gap-2 flex-wrap">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                  {getInterventionTitle(intervention.interventionType)}
                </h3>
                <span
                  className={`px-2 py-0.5 text-xs font-medium rounded-full ${
                    intervention.trigger === 'programmatic'
                      ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
                      : 'bg-purple-100 dark:bg-purple-900/50 text-purple-700 dark:text-purple-300'
                  }`}
                >
                  {intervention.trigger === 'programmatic' ? 'Auto-detected' : 'AI requested'}
                </span>
                <span className="text-xs text-gray-500 dark:text-gray-400 font-mono">
                  {elapsedTime}
                </span>
              </div>

              {/* Reason */}
              <p className="mt-2 text-sm text-gray-600 dark:text-gray-300">
                {intervention.reason}
              </p>

              {/* Instructions */}
              {intervention.instructions && (
                <div className="mt-3 p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
                  <div className="flex items-start gap-2">
                    <svg
                      className="w-5 h-5 text-blue-500 flex-shrink-0 mt-0.5"
                      fill="currentColor"
                      viewBox="0 0 20 20"
                    >
                      <path
                        fillRule="evenodd"
                        d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                        clipRule="evenodd"
                      />
                    </svg>
                    <p className="text-sm text-blue-800 dark:text-blue-200">
                      {intervention.instructions}
                    </p>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Actions */}
          <div className="mt-6 flex items-center justify-end gap-3">
            <button
              onClick={onAbort}
              className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg transition-colors"
            >
              Cancel Navigation
            </button>
            <button
              onClick={onComplete}
              className="px-6 py-2 text-sm font-medium text-white bg-purple-600 hover:bg-purple-700 rounded-lg shadow-lg shadow-purple-500/30 transition-colors flex items-center gap-2"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              I'm Done
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
