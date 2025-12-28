/**
 * AINavigationPanel Component
 *
 * Panel for initiating AI-driven browser navigation.
 * Features:
 * - Prompt input for describing the navigation goal
 * - Model selector with cost display
 * - Max steps configuration
 * - Start/Abort controls
 * - Live step display during navigation
 */

import { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import { AIProcessingIndicator } from './AINavigationSkeleton';
import type { VisionModelSpec, AINavigationStep } from './types';

// LocalStorage keys for persisting user preferences
const STORAGE_KEY_MODEL = 'ai-navigation-model';
const STORAGE_KEY_MAX_STEPS = 'ai-navigation-max-steps';

/** Get persisted model from localStorage */
function getPersistedModel(defaultModel: string): string {
  if (typeof window === 'undefined') return defaultModel;
  try {
    const stored = localStorage.getItem(STORAGE_KEY_MODEL);
    return stored ?? defaultModel;
  } catch {
    return defaultModel;
  }
}

/** Get persisted max steps from localStorage */
function getPersistedMaxSteps(defaultSteps: number): number {
  if (typeof window === 'undefined') return defaultSteps;
  try {
    const stored = localStorage.getItem(STORAGE_KEY_MAX_STEPS);
    if (stored) {
      const parsed = parseInt(stored, 10);
      if (!isNaN(parsed) && parsed >= 5 && parsed <= 50) {
        return parsed;
      }
    }
    return defaultSteps;
  } catch {
    return defaultSteps;
  }
}

/** Persist model to localStorage */
function persistModel(model: string): void {
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem(STORAGE_KEY_MODEL, model);
  } catch {
    // Ignore storage errors
  }
}

/** Persist max steps to localStorage */
function persistMaxSteps(maxSteps: number): void {
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem(STORAGE_KEY_MAX_STEPS, String(maxSteps));
  } catch {
    // Ignore storage errors
  }
}

interface AINavigationPanelProps {
  /** Available vision models */
  models: VisionModelSpec[];
  /** Whether navigation is in progress */
  isNavigating: boolean;
  /** Current steps in the navigation */
  steps: AINavigationStep[];
  /** Total tokens used */
  totalTokens: number;
  /** Current status */
  status: 'idle' | 'navigating' | 'completed' | 'failed' | 'aborted' | 'max_steps_reached' | 'loop_detected';
  /** Error message if any */
  error: string | null;
  /** Callback to start navigation */
  onStart: (prompt: string, model: string, maxSteps: number) => void;
  /** Callback to abort navigation */
  onAbort: () => void;
  /** Callback to reset state */
  onReset: () => void;
  /** Whether the panel is disabled (e.g., no session) */
  disabled?: boolean;
  /** Initial prompt to pre-fill (from template) */
  initialPrompt?: string;
}

/** Get tier badge color */
function getTierBadgeClass(tier: 'budget' | 'standard' | 'premium'): string {
  switch (tier) {
    case 'budget':
      return 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300';
    case 'standard':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300';
    case 'premium':
      return 'bg-purple-100 text-purple-700 dark:bg-purple-900/50 dark:text-purple-300';
  }
}

/** Format cost for display */
function formatCost(cost: number): string {
  if (cost < 0.01) {
    return `$${cost.toFixed(4)}`;
  }
  return `$${cost.toFixed(2)}`;
}

/** Estimate cost for a navigation session */
function estimateCost(model: VisionModelSpec, maxSteps: number): number {
  // Rough estimate: ~2000 input tokens per step (screenshot + prompt), ~100 output tokens
  const avgInputTokensPerStep = 2000;
  const avgOutputTokensPerStep = 100;
  const totalInputTokens = avgInputTokensPerStep * maxSteps;
  const totalOutputTokens = avgOutputTokensPerStep * maxSteps;

  const inputCost = (totalInputTokens / 1_000_000) * model.inputCostPer1MTokens;
  const outputCost = (totalOutputTokens / 1_000_000) * model.outputCostPer1MTokens;

  return inputCost + outputCost;
}

export function AINavigationPanel({
  models,
  isNavigating,
  steps,
  totalTokens,
  status,
  error,
  onStart,
  onAbort,
  onReset,
  disabled = false,
  initialPrompt,
}: AINavigationPanelProps) {
  const [prompt, setPrompt] = useState(initialPrompt || '');
  const [selectedModelId, setSelectedModelId] = useState(() => getPersistedModel('qwen3-vl-30b'));
  const [maxSteps, setMaxSteps] = useState(() => getPersistedMaxSteps(20));
  const [showAdvanced, setShowAdvanced] = useState(false);

  // Sync initialPrompt to prompt when it changes (e.g., from template)
  useEffect(() => {
    if (initialPrompt && !prompt) {
      setPrompt(initialPrompt);
    }
  }, [initialPrompt, prompt]);

  // Persist model selection when it changes
  const handleModelChange = useCallback((modelId: string) => {
    setSelectedModelId(modelId);
    persistModel(modelId);
  }, []);

  // Persist max steps when it changes
  const handleMaxStepsChange = useCallback((steps: number) => {
    setMaxSteps(steps);
    persistMaxSteps(steps);
  }, []);

  const selectedModel = useMemo(
    () => models.find((m) => m.id === selectedModelId) ?? models[0] ?? {
      id: 'unknown',
      displayName: 'No models available',
      tier: 'standard' as const,
      inputCostPer1MTokens: 0,
      outputCostPer1MTokens: 0,
      provider: 'openrouter' as const,
      recommended: false,
    },
    [models, selectedModelId]
  );

  const estimatedCost = useMemo(
    () => estimateCost(selectedModel, maxSteps),
    [selectedModel, maxSteps]
  );

  const canStart = !disabled && !isNavigating && prompt.trim().length > 0;
  const showResults = status !== 'idle' && steps.length > 0;

  // Show highlight animation when there's a pre-filled prompt ready to start (from template)
  const showStartHighlight = Boolean(initialPrompt) && status === 'idle' && canStart;

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (canStart) {
        onStart(prompt.trim(), selectedModelId, maxSteps);
      }
    },
    [canStart, onStart, prompt, selectedModelId, maxSteps]
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      // Cmd/Ctrl+Enter always starts (even without prompt focus)
      // Enter (without shift) starts when focused on textarea
      const isCmdEnter = e.key === 'Enter' && (e.metaKey || e.ctrlKey);
      const isEnter = e.key === 'Enter' && !e.shiftKey && !e.metaKey && !e.ctrlKey;

      if ((isCmdEnter || isEnter) && canStart) {
        e.preventDefault();
        onStart(prompt.trim(), selectedModelId, maxSteps);
      }
    },
    [canStart, onStart, prompt, selectedModelId, maxSteps]
  );

  // Ref to track the panel container for global keyboard shortcuts
  const panelRef = useRef<HTMLDivElement>(null);

  // Global keyboard shortcut: Cmd/Ctrl+Enter to start navigation
  useEffect(() => {
    const handleGlobalKeyDown = (e: KeyboardEvent) => {
      // Only handle if this panel is visible (has the ref)
      if (!panelRef.current) return;

      // Check if the event target is within this panel
      const isWithinPanel = panelRef.current.contains(e.target as Node);
      if (!isWithinPanel) return;

      // Cmd/Ctrl+Enter to start
      if (e.key === 'Enter' && (e.metaKey || e.ctrlKey) && canStart) {
        e.preventDefault();
        onStart(prompt.trim(), selectedModelId, maxSteps);
      }

      // Escape to abort
      if (e.key === 'Escape' && isNavigating) {
        e.preventDefault();
        onAbort();
      }
    };

    document.addEventListener('keydown', handleGlobalKeyDown);
    return () => document.removeEventListener('keydown', handleGlobalKeyDown);
  }, [canStart, isNavigating, onStart, onAbort, prompt, selectedModelId, maxSteps]);

  const getStatusMessage = () => {
    switch (status) {
      case 'navigating':
        return `Navigating... Step ${steps.length}`;
      case 'completed':
        return 'Navigation completed successfully';
      case 'failed':
        return 'Navigation failed';
      case 'aborted':
        return 'Navigation aborted';
      case 'max_steps_reached':
        return 'Maximum steps reached';
      default:
        return null;
    }
  };

  const statusMessage = getStatusMessage();

  return (
    <div
      ref={panelRef}
      className="flex flex-col h-full bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm"
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-2">
          <svg className="w-5 h-5 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
          </svg>
          <span className="font-medium text-gray-900 dark:text-white">AI Navigation</span>
        </div>
        {showResults && status !== 'navigating' && (
          <button
            onClick={onReset}
            className="text-xs text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
          >
            Clear
          </button>
        )}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Prompt Input */}
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
            What would you like the AI to do?
          </label>
          <textarea
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="e.g., Order chicken from the menu, Login with my saved credentials..."
            className="w-full px-3 py-2 text-sm bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent resize-none"
            rows={3}
            disabled={isNavigating}
          />
          <p className="mt-1 text-xs text-gray-500">
            <kbd className="px-1 py-0.5 text-[10px] font-mono bg-gray-100 dark:bg-gray-800 rounded">Enter</kbd> or{' '}
            <kbd className="px-1 py-0.5 text-[10px] font-mono bg-gray-100 dark:bg-gray-800 rounded">
              {typeof navigator !== 'undefined' && navigator.platform?.includes('Mac') ? '⌘' : 'Ctrl'}+Enter
            </kbd> to start
          </p>
        </div>

        {/* Model Selector - Collapsible accordion, collapsed by default */}
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
          <button
            type="button"
            onClick={() => setShowAdvanced(!showAdvanced)}
            disabled={isNavigating}
            className={`w-full flex items-center justify-between p-3 text-left transition-colors ${
              showAdvanced
                ? 'bg-gray-50 dark:bg-gray-800/50'
                : 'hover:bg-gray-50 dark:hover:bg-gray-800/50'
            } ${isNavigating ? 'opacity-50 cursor-not-allowed' : ''}`}
          >
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Vision Model</span>
              <span className={`px-1.5 py-0.5 text-[10px] font-medium rounded ${getTierBadgeClass(selectedModel.tier)}`}>
                {selectedModel.displayName}
              </span>
            </div>
            <svg
              className={`w-4 h-4 text-gray-400 transition-transform ${showAdvanced ? 'rotate-180' : ''}`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
            </svg>
          </button>

          {showAdvanced && (
            <div className="p-3 pt-0 space-y-2 border-t border-gray-200 dark:border-gray-700">
              {/* All models in a flat list */}
              {models.map((model) => (
                <button
                  key={model.id}
                  onClick={() => handleModelChange(model.id)}
                  disabled={isNavigating}
                  className={`w-full flex items-center justify-between p-2.5 rounded-lg border transition-colors ${
                    selectedModelId === model.id
                      ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
                      : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
                  } ${isNavigating ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}`}
                >
                  <div className="flex items-center gap-2">
                    <div className={`w-4 h-4 rounded-full border-2 flex-shrink-0 ${
                      selectedModelId === model.id
                        ? 'border-purple-500 bg-purple-500'
                        : 'border-gray-300 dark:border-gray-600'
                    }`}>
                      {selectedModelId === model.id && (
                        <svg className="w-full h-full text-white" fill="currentColor" viewBox="0 0 20 20">
                          <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                        </svg>
                      )}
                    </div>
                    <span className="font-medium text-gray-900 dark:text-white text-sm">{model.displayName}</span>
                    <span className={`px-1.5 py-0.5 text-[10px] font-medium rounded ${getTierBadgeClass(model.tier)}`}>
                      {model.tier.toUpperCase()}
                    </span>
                    {model.recommended && (
                      <span className="px-1.5 py-0.5 text-[10px] font-medium bg-green-100 dark:bg-green-900/50 text-green-700 dark:text-green-300 rounded">
                        Recommended
                      </span>
                    )}
                  </div>
                  <div className="text-xs text-gray-500 text-right">
                    ~{formatCost(model.inputCostPer1MTokens)}/M in
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Max Steps */}
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
              Max Steps
            </label>
            <span className="text-xs text-gray-500">{maxSteps} steps</span>
          </div>
          <input
            type="range"
            min={5}
            max={50}
            value={maxSteps}
            onChange={(e) => handleMaxStepsChange(Number(e.target.value))}
            disabled={isNavigating}
            className="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer accent-purple-500"
          />
          <div className="flex justify-between text-xs text-gray-400 mt-1">
            <span>5</span>
            <span>50</span>
          </div>
        </div>

        {/* Cost Estimate */}
        <div className="p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <div className="flex items-center justify-between text-sm">
            <span className="text-gray-600 dark:text-gray-400">Estimated cost:</span>
            <span className="font-medium text-gray-900 dark:text-white">
              {formatCost(estimatedCost)}
            </span>
          </div>
          <p className="text-xs text-gray-500 mt-1">
            Actual cost depends on page complexity and number of steps taken
          </p>
        </div>

        {/* Processing Indicator - shown when starting but no steps yet */}
        {isNavigating && steps.length === 0 && (
          <AIProcessingIndicator message="Starting AI navigation..." />
        )}

        {/* Status & Steps */}
        {showResults && (
          <div className="space-y-2">
            {/* Status Message */}
            {statusMessage && (
              <div className={`flex items-center gap-2 p-2 rounded-lg text-sm ${
                status === 'completed' ? 'bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-300' :
                status === 'failed' ? 'bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-300' :
                status === 'aborted' ? 'bg-yellow-50 dark:bg-yellow-900/20 text-yellow-700 dark:text-yellow-300' :
                status === 'max_steps_reached' ? 'bg-orange-50 dark:bg-orange-900/20 text-orange-700 dark:text-orange-300' :
                'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300'
              }`}>
                {status === 'navigating' && (
                  <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                )}
                {status === 'completed' && (
                  <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                  </svg>
                )}
                {(status === 'failed' || status === 'aborted' || status === 'max_steps_reached') && (
                  <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                  </svg>
                )}
                <span>{statusMessage}</span>
              </div>
            )}

            {/* Token Usage */}
            <div className="text-xs text-gray-500">
              Total tokens: {totalTokens.toLocaleString()}
            </div>
          </div>
        )}

        {/* Error Display */}
        {error && (
          <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
            <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
          </div>
        )}
      </div>

      {/* Footer Actions */}
      <div className="px-4 py-3 border-t border-gray-200 dark:border-gray-700">
        {isNavigating ? (
          <button
            onClick={onAbort}
            disabled={status === 'aborted'}
            className={`w-full px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
              status === 'aborted'
                ? 'text-yellow-700 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-300 cursor-not-allowed'
                : 'text-red-700 bg-red-100 hover:bg-red-200 dark:bg-red-900/30 dark:text-red-300 dark:hover:bg-red-900/50'
            }`}
          >
            <span className="flex items-center justify-center gap-2">
              {status === 'aborted' ? (
                <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
              ) : (
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              )}
              {status === 'aborted' ? 'Stopping...' : 'Abort Navigation'}
            </span>
          </button>
        ) : (
          <button
            onClick={handleSubmit as unknown as React.MouseEventHandler}
            disabled={!canStart}
            className={`w-full px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
              canStart
                ? 'text-white bg-purple-600 hover:bg-purple-700'
                : 'text-gray-400 bg-gray-100 dark:bg-gray-800 cursor-not-allowed'
            } ${showStartHighlight ? 'ring-2 ring-purple-400 animate-pulse' : ''}`}
          >
            <span className="flex items-center justify-center gap-2">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              {showStartHighlight ? 'Click to Start AI Navigation' : 'Start AI Navigation'}
            </span>
          </button>
        )}
      </div>
    </div>
  );
}
