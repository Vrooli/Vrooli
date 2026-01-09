/**
 * AISettingsModal Component
 *
 * Modal for configuring AI navigation settings:
 * - Vision model selection
 * - Max steps configuration
 * - Cost estimation display
 */

import { useState, useCallback, useEffect } from 'react';
import { type AISettings } from './types';
import { type VisionModelSpec } from '../ai-navigation/types';
import { formatCost, estimateNavigationCost } from './useAISettings';

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Get tier badge color classes.
 */
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

// ============================================================================
// Component Props
// ============================================================================

export interface AISettingsModalProps {
  /** Whether the modal is open */
  isOpen: boolean;
  /** Close modal callback */
  onClose: () => void;
  /** Current settings */
  currentSettings: AISettings;
  /** Save settings callback */
  onSaveSettings: (settings: AISettings) => void;
  /** Available vision models */
  availableModels: VisionModelSpec[];
}

// ============================================================================
// Component
// ============================================================================

export function AISettingsModal({
  isOpen,
  onClose,
  currentSettings,
  onSaveSettings,
  availableModels,
}: AISettingsModalProps) {
  // Local state for editing
  const [selectedModelId, setSelectedModelId] = useState(currentSettings.model);
  const [maxSteps, setMaxSteps] = useState(currentSettings.maxSteps);

  // Sync with props when modal opens
  useEffect(() => {
    if (isOpen) {
      setSelectedModelId(currentSettings.model);
      setMaxSteps(currentSettings.maxSteps);
    }
  }, [isOpen, currentSettings]);

  // Get selected model spec for cost calculation
  const selectedModel = availableModels.find((m) => m.id === selectedModelId) ?? availableModels[0];

  // Calculate estimated cost
  const estimatedCost = selectedModel ? estimateNavigationCost(selectedModel, maxSteps) : 0;

  // Handle save
  const handleSave = useCallback(() => {
    onSaveSettings({
      model: selectedModelId,
      maxSteps,
    });
    onClose();
  }, [maxSteps, onClose, onSaveSettings, selectedModelId]);

  // Handle escape key
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
      role="dialog"
      aria-modal="true"
      aria-labelledby="ai-settings-title"
    >
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md mx-4 max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
          <h2 id="ai-settings-title" className="text-lg font-semibold text-gray-900 dark:text-white">
            AI Navigation Settings
          </h2>
          <button
            onClick={onClose}
            className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            aria-label="Close"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-6 py-4 space-y-6">
          {/* Model Selection */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Vision Model
            </label>
            <div className="space-y-2">
              {availableModels.map((model) => (
                <button
                  key={model.id}
                  type="button"
                  onClick={() => setSelectedModelId(model.id)}
                  className={`w-full flex items-center justify-between p-3 rounded-lg border transition-colors ${
                    selectedModelId === model.id
                      ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
                      : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    {/* Radio indicator */}
                    <div
                      className={`w-4 h-4 rounded-full border-2 flex-shrink-0 flex items-center justify-center ${
                        selectedModelId === model.id
                          ? 'border-purple-500 bg-purple-500'
                          : 'border-gray-300 dark:border-gray-600'
                      }`}
                    >
                      {selectedModelId === model.id && (
                        <svg className="w-2.5 h-2.5 text-white" fill="currentColor" viewBox="0 0 20 20">
                          <circle cx="10" cy="10" r="6" />
                        </svg>
                      )}
                    </div>

                    {/* Model info */}
                    <span className="font-medium text-gray-900 dark:text-white text-sm">
                      {model.displayName}
                    </span>
                    <span className={`px-1.5 py-0.5 text-[10px] font-medium rounded ${getTierBadgeClass(model.tier)}`}>
                      {model.tier.toUpperCase()}
                    </span>
                    {model.recommended && (
                      <span className="px-1.5 py-0.5 text-[10px] font-medium bg-green-100 dark:bg-green-900/50 text-green-700 dark:text-green-300 rounded">
                        Recommended
                      </span>
                    )}
                  </div>

                  {/* Cost */}
                  <div className="text-xs text-gray-500 text-right">
                    ~{formatCost(model.inputCostPer1MTokens)}/M in
                  </div>
                </button>
              ))}
            </div>
          </div>

          {/* Max Steps */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                Maximum Steps
              </label>
              <span className="text-sm text-gray-500">{maxSteps} steps</span>
            </div>
            <input
              type="range"
              min={5}
              max={50}
              value={maxSteps}
              onChange={(e) => setMaxSteps(Number(e.target.value))}
              className="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer accent-purple-500"
            />
            <div className="flex justify-between text-xs text-gray-400 mt-1">
              <span>5 (Quick)</span>
              <span>50 (Thorough)</span>
            </div>
            <p className="text-xs text-gray-500 mt-2">
              More steps allow AI to complete complex tasks but increase cost.
            </p>
          </div>

          {/* Cost Estimate */}
          <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600 dark:text-gray-400">Estimated cost per navigation:</span>
              <span className="text-lg font-semibold text-gray-900 dark:text-white">
                {formatCost(estimatedCost)}
              </span>
            </div>
            <p className="text-xs text-gray-500 mt-2">
              Actual cost depends on page complexity and number of steps taken. The AI may complete tasks in fewer steps.
            </p>
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 bg-gray-50 dark:bg-gray-800/50 rounded-b-lg flex justify-end gap-3">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            className="px-4 py-2 text-sm font-medium text-white bg-purple-600 rounded-lg hover:bg-purple-700 transition-colors"
          >
            Save Settings
          </button>
        </div>
      </div>
    </div>
  );
}
