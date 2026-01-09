/**
 * SelectorEditor Component
 *
 * Advanced selector editing with:
 * - Confidence indicators and warnings
 * - Alternative selector suggestions
 * - Live validation
 * - Recovery guidance for unstable selectors
 */

import { useState, useCallback, useEffect } from 'react';
import type { SelectorSet, SelectorCandidate, SelectorValidation } from '../types/types';

interface SelectorEditorProps {
  /** Current selector set with candidates */
  selectorSet: SelectorSet;
  /** Current confidence score (0-1) */
  confidence: number;
  /** Callback to validate a selector */
  onValidate: (selector: string) => Promise<SelectorValidation>;
  /** Callback when selector is saved */
  onSave: (newSelector: string) => void;
  /** Callback to cancel editing */
  onCancel: () => void;
}

/** Confidence thresholds for warnings */
const CONFIDENCE = {
  HIGH: 0.8,
  MEDIUM: 0.5,
};

/** Selector type display names and descriptions */
const SELECTOR_TYPE_INFO: Record<string, { label: string; description: string }> = {
  'data-testid': {
    label: 'Test ID',
    description: 'Most stable - developer-defined test identifiers'
  },
  'id': {
    label: 'ID',
    description: 'Stable if not dynamically generated'
  },
  'aria': {
    label: 'ARIA',
    description: 'Accessibility attributes - usually stable'
  },
  'text': {
    label: 'Text',
    description: 'Based on visible text - may break if text changes'
  },
  'data-attr': {
    label: 'Data Attr',
    description: 'Custom data attributes'
  },
  'css': {
    label: 'CSS Path',
    description: 'Structural path - sensitive to layout changes'
  },
  'xpath': {
    label: 'XPath',
    description: 'Fallback - least stable'
  },
};

export function SelectorEditor({
  selectorSet,
  confidence,
  onValidate,
  onSave,
  onCancel,
}: SelectorEditorProps) {
  const [editedSelector, setEditedSelector] = useState(selectorSet.primary);
  const [validation, setValidation] = useState<SelectorValidation | null>(null);
  const [isValidating, setIsValidating] = useState(false);
  const [selectedCandidateIdx, setSelectedCandidateIdx] = useState<number | null>(null);
  const [showAllCandidates, setShowAllCandidates] = useState(false);

  // Reset state when selector changes
  useEffect(() => {
    setEditedSelector(selectorSet.primary);
    setValidation(null);
    setSelectedCandidateIdx(null);
  }, [selectorSet.primary]);

  const handleValidate = useCallback(async () => {
    if (!editedSelector.trim()) return;

    setIsValidating(true);
    try {
      const result = await onValidate(editedSelector);
      setValidation(result);
    } catch (err) {
      setValidation({
        valid: false,
        match_count: 0,
        selector: editedSelector,
        error: err instanceof Error ? err.message : 'Validation failed',
      });
    } finally {
      setIsValidating(false);
    }
  }, [editedSelector, onValidate]);

  const handleSelectCandidate = useCallback((candidate: SelectorCandidate, idx: number) => {
    setEditedSelector(candidate.value);
    setSelectedCandidateIdx(idx);
    setValidation(null);
  }, []);

  const handleSave = useCallback(() => {
    onSave(editedSelector);
  }, [editedSelector, onSave]);

  const getConfidenceLevel = (conf: number): 'high' | 'medium' | 'low' => {
    if (conf >= CONFIDENCE.HIGH) return 'high';
    if (conf >= CONFIDENCE.MEDIUM) return 'medium';
    return 'low';
  };

  const getConfidenceColor = (level: 'high' | 'medium' | 'low') => {
    switch (level) {
      case 'high': return 'text-green-600 dark:text-green-400';
      case 'medium': return 'text-yellow-600 dark:text-yellow-400';
      case 'low': return 'text-red-600 dark:text-red-400';
    }
  };

  const getConfidenceBg = (level: 'high' | 'medium' | 'low') => {
    switch (level) {
      case 'high': return 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800';
      case 'medium': return 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800';
      case 'low': return 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800';
    }
  };

  const confidenceLevel = getConfidenceLevel(confidence);
  const displayedCandidates = showAllCandidates
    ? selectorSet.candidates
    : selectorSet.candidates.slice(0, 3);

  return (
    <div className="space-y-4 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg border border-gray-200 dark:border-gray-700">
      {/* Confidence Warning */}
      {confidenceLevel !== 'high' && (
        <div className={`p-3 rounded-lg border ${getConfidenceBg(confidenceLevel)}`}>
          <div className="flex items-start gap-2">
            <svg
              className={`w-5 h-5 mt-0.5 flex-shrink-0 ${getConfidenceColor(confidenceLevel)}`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
              />
            </svg>
            <div>
              <p className={`text-sm font-medium ${getConfidenceColor(confidenceLevel)}`}>
                {confidenceLevel === 'medium' ? 'Selector may be unstable' : 'Selector is likely unstable'}
              </p>
              <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                {confidenceLevel === 'medium'
                  ? 'This selector might break if the page structure changes. Consider using a more stable alternative below.'
                  : 'This selector is very likely to break. We strongly recommend selecting a more stable alternative or editing manually.'}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Current Selector Editor */}
      <div>
        <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1.5">
          Selector
        </label>
        <div className="flex gap-2">
          <input
            type="text"
            value={editedSelector}
            onChange={(e) => {
              setEditedSelector(e.target.value);
              setValidation(null);
              setSelectedCandidateIdx(null);
            }}
            className="flex-1 px-3 py-2 text-sm font-mono bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            placeholder="Enter CSS selector or XPath"
          />
          <button
            onClick={handleValidate}
            disabled={isValidating || !editedSelector.trim()}
            className="px-3 py-2 text-sm font-medium text-blue-600 dark:text-blue-400 border border-blue-300 dark:border-blue-700 rounded-lg hover:bg-blue-50 dark:hover:bg-blue-900/30 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isValidating ? (
              <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
            ) : (
              'Test'
            )}
          </button>
        </div>

        {/* Validation Result */}
        {validation && (
          <div className={`mt-2 p-2 rounded text-sm ${
            validation.valid
              ? 'bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-300'
              : 'bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-300'
          }`}>
            {validation.valid ? (
              <span className="flex items-center gap-1.5">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                Found exactly 1 element - perfect match!
              </span>
            ) : validation.match_count === 0 ? (
              <span className="flex items-center gap-1.5">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
                No elements found - selector doesn't match any element on the page
              </span>
            ) : (
              <span className="flex items-center gap-1.5">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                Found {validation.match_count} elements - selector is ambiguous (should match exactly 1)
              </span>
            )}
          </div>
        )}
      </div>

      {/* Alternative Selectors */}
      {selectorSet.candidates.length > 1 && (
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="text-xs font-medium text-gray-700 dark:text-gray-300">
              Alternative Selectors
            </label>
            <span className="text-xs text-gray-500">
              Click to use
            </span>
          </div>
          <div className="space-y-1.5">
            {displayedCandidates.map((candidate, idx) => {
              const typeInfo = SELECTOR_TYPE_INFO[candidate.type] || {
                label: candidate.type,
                description: 'Unknown selector type'
              };
              const candidateLevel = getConfidenceLevel(candidate.confidence);
              const isSelected = selectedCandidateIdx === idx ||
                (selectedCandidateIdx === null && candidate.value === selectorSet.primary);

              return (
                <button
                  key={idx}
                  onClick={() => handleSelectCandidate(candidate, idx)}
                  className={`w-full text-left p-2 rounded-lg border transition-colors ${
                    isSelected
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30'
                      : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-800'
                  }`}
                >
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex items-center gap-2 min-w-0">
                      <span className={`flex-shrink-0 px-1.5 py-0.5 text-xs font-medium rounded ${
                        candidateLevel === 'high'
                          ? 'bg-green-100 dark:bg-green-900/40 text-green-700 dark:text-green-300'
                          : candidateLevel === 'medium'
                          ? 'bg-yellow-100 dark:bg-yellow-900/40 text-yellow-700 dark:text-yellow-300'
                          : 'bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-300'
                      }`}>
                        {typeInfo.label}
                      </span>
                      <code className="text-xs font-mono text-gray-600 dark:text-gray-400 truncate">
                        {candidate.value}
                      </code>
                    </div>
                    <span className={`flex-shrink-0 text-xs ${getConfidenceColor(candidateLevel)}`}>
                      {Math.round(candidate.confidence * 100)}%
                    </span>
                  </div>
                  <p className="text-xs text-gray-500 mt-1">{typeInfo.description}</p>
                </button>
              );
            })}
          </div>

          {/* Show More/Less */}
          {selectorSet.candidates.length > 3 && (
            <button
              onClick={() => setShowAllCandidates(!showAllCandidates)}
              className="mt-2 w-full text-xs text-center text-blue-600 dark:text-blue-400 hover:underline"
            >
              {showAllCandidates
                ? `Show less`
                : `Show ${selectorSet.candidates.length - 3} more`}
            </button>
          )}
        </div>
      )}

      {/* Tips for Unstable Selectors */}
      {confidenceLevel === 'low' && (
        <div className="p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
          <p className="text-xs font-medium text-blue-700 dark:text-blue-300 mb-2">
            Tips for more stable selectors:
          </p>
          <ul className="text-xs text-blue-600 dark:text-blue-400 space-y-1 list-disc list-inside">
            <li>Ask developers to add <code className="bg-blue-100 dark:bg-blue-900/40 px-1 rounded">data-testid</code> attributes</li>
            <li>Use ARIA labels: <code className="bg-blue-100 dark:bg-blue-900/40 px-1 rounded">[aria-label="..."]</code></li>
            <li>Use text content: <code className="bg-blue-100 dark:bg-blue-900/40 px-1 rounded">button:has-text("Submit")</code></li>
            <li>Combine selectors: <code className="bg-blue-100 dark:bg-blue-900/40 px-1 rounded">form input[name="email"]</code></li>
          </ul>
        </div>
      )}

      {/* Action Buttons */}
      <div className="flex justify-end gap-2 pt-2 border-t border-gray-200 dark:border-gray-700">
        <button
          onClick={onCancel}
          className="px-3 py-1.5 text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
        >
          Cancel
        </button>
        <button
          onClick={handleSave}
          disabled={!editedSelector.trim()}
          className="px-4 py-1.5 text-sm font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          Save
        </button>
      </div>
    </div>
  );
}
