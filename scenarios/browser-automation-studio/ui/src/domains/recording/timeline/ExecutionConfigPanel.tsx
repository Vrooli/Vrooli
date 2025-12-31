/**
 * ExecutionConfigPanel Component
 *
 * A collapsible panel for configuring execution settings before running a workflow.
 * Extracted from WorkflowCreationForm's advanced settings for reuse in WorkflowInfoCard.
 *
 * Features:
 * - Navigation wait strategy
 * - Action timeout
 * - Viewport size
 * - Continue on error toggle
 * - Artifact collection profile (optional)
 */

import { useState, useCallback, useEffect } from 'react';
import { ChevronDown, ChevronRight, Settings2 } from 'lucide-react';
import type { NavigationWaitUntil } from '@/types/workflow';
import type { ArtifactProfile } from '@/domains/executions/store';
import { ArtifactProfileSelector } from '@/domains/executions/components/ArtifactProfileSelector';

// ============================================================================
// Types
// ============================================================================

export interface ExecutionConfigSettings {
  navigationWaitUntil: NavigationWaitUntil;
  actionTimeoutSeconds: number;
  viewportWidth: number;
  viewportHeight: number;
  continueOnError: boolean;
  artifactProfile?: ArtifactProfile;
}

export interface ExecutionConfigPanelProps {
  /** Initial values from workflow settings */
  defaults: ExecutionConfigSettings;
  /** Current values (controlled mode) */
  value?: ExecutionConfigSettings;
  /** Callback when settings change */
  onChange: (settings: ExecutionConfigSettings) => void;
  /** Whether panel is collapsed */
  collapsed?: boolean;
  /** Callback when collapse state changes */
  onCollapseChange?: (collapsed: boolean) => void;
  /** Show artifact profile selector */
  showArtifactProfile?: boolean;
  /** Additional CSS classes */
  className?: string;
}

// ============================================================================
// Constants (exported for reuse)
// ============================================================================

export const DEFAULT_EXECUTION_SETTINGS: ExecutionConfigSettings = {
  navigationWaitUntil: 'domcontentloaded',
  actionTimeoutSeconds: 30,
  viewportWidth: 1920,
  viewportHeight: 1080,
  continueOnError: false,
  artifactProfile: 'standard',
};

export const NAVIGATION_WAIT_OPTIONS: {
  value: NavigationWaitUntil;
  label: string;
  description: string;
}[] = [
  { value: 'domcontentloaded', label: 'DOM Ready', description: 'Wait for HTML to parse (fast, recommended)' },
  { value: 'load', label: 'Page Load', description: 'Wait for all resources to load' },
  { value: 'networkidle', label: 'Network Idle', description: 'Wait for no network activity (slow on heavy sites)' },
];

// ============================================================================
// Component
// ============================================================================

export function ExecutionConfigPanel({
  defaults,
  value,
  onChange,
  collapsed: controlledCollapsed,
  onCollapseChange,
  showArtifactProfile = true,
  className = '',
}: ExecutionConfigPanelProps) {
  // Use controlled collapsed state if provided, otherwise internal
  const [internalCollapsed, setInternalCollapsed] = useState(true);
  const collapsed = controlledCollapsed ?? internalCollapsed;

  // Initialize settings from defaults
  const [settings, setSettings] = useState<ExecutionConfigSettings>(() => ({
    ...DEFAULT_EXECUTION_SETTINGS,
    ...defaults,
  }));

  // Sync with controlled value if provided
  useEffect(() => {
    if (value) {
      setSettings(value);
    }
  }, [value]);

  // Update a single setting
  const updateSetting = useCallback(
    <K extends keyof ExecutionConfigSettings>(key: K, newValue: ExecutionConfigSettings[K]) => {
      setSettings((prev) => {
        const updated = { ...prev, [key]: newValue };
        onChange(updated);
        return updated;
      });
    },
    [onChange]
  );

  const handleToggleCollapse = useCallback(() => {
    const newCollapsed = !collapsed;
    if (onCollapseChange) {
      onCollapseChange(newCollapsed);
    } else {
      setInternalCollapsed(newCollapsed);
    }
  }, [collapsed, onCollapseChange]);

  return (
    <div className={`bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden ${className}`}>
      {/* Collapse Header */}
      <button
        type="button"
        onClick={handleToggleCollapse}
        className="w-full flex items-center justify-between p-4 text-left hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
      >
        <div className="flex items-center gap-2">
          <Settings2 size={16} className="text-gray-500 dark:text-gray-400" />
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
            Execution Settings
          </span>
        </div>
        {collapsed ? (
          <ChevronRight size={16} className="text-gray-400" />
        ) : (
          <ChevronDown size={16} className="text-gray-400" />
        )}
      </button>

      {/* Expanded Settings */}
      {!collapsed && (
        <div className="border-t border-gray-200 dark:border-gray-700 divide-y divide-gray-200 dark:divide-gray-700">
          {/* Navigation Wait */}
          <div className="p-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Navigation Wait
            </label>
            <select
              value={settings.navigationWaitUntil}
              onChange={(e) => updateSetting('navigationWaitUntil', e.target.value as NavigationWaitUntil)}
              className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              {NAVIGATION_WAIT_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1.5">
              {NAVIGATION_WAIT_OPTIONS.find((o) => o.value === settings.navigationWaitUntil)?.description}
            </p>
          </div>

          {/* Action Timeout */}
          <div className="p-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Action Timeout
            </label>
            <div className="flex items-center gap-2">
              <input
                type="number"
                min={1}
                max={300}
                value={settings.actionTimeoutSeconds}
                onChange={(e) =>
                  updateSetting(
                    'actionTimeoutSeconds',
                    Math.max(1, Math.min(300, parseInt(e.target.value) || 30))
                  )
                }
                className="w-24 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              <span className="text-sm text-gray-500 dark:text-gray-400">seconds</span>
            </div>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1.5">
              How long to wait for elements before timing out.
            </p>
          </div>

          {/* Viewport Size */}
          <div className="p-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Viewport Size
            </label>
            <div className="flex items-center gap-2">
              <input
                type="number"
                min={320}
                max={3840}
                value={settings.viewportWidth}
                onChange={(e) =>
                  updateSetting(
                    'viewportWidth',
                    Math.max(320, Math.min(3840, parseInt(e.target.value) || 1920))
                  )
                }
                className="w-24 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              <span className="text-sm text-gray-500 dark:text-gray-400">×</span>
              <input
                type="number"
                min={240}
                max={2160}
                value={settings.viewportHeight}
                onChange={(e) =>
                  updateSetting(
                    'viewportHeight',
                    Math.max(240, Math.min(2160, parseInt(e.target.value) || 1080))
                  )
                }
                className="w-24 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              <span className="text-sm text-gray-500 dark:text-gray-400">px</span>
            </div>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1.5">
              Browser window dimensions for workflow execution.
            </p>
          </div>

          {/* Continue on Error */}
          <div className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Continue on Error
                </label>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                  Keep running even if a step fails.
                </p>
              </div>
              <button
                type="button"
                role="switch"
                aria-checked={settings.continueOnError}
                onClick={() => updateSetting('continueOnError', !settings.continueOnError)}
                className={`relative w-11 h-6 rounded-full transition-colors ${
                  settings.continueOnError ? 'bg-blue-500' : 'bg-gray-300 dark:bg-gray-600'
                }`}
              >
                <span
                  className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform ${
                    settings.continueOnError ? 'translate-x-5' : 'translate-x-0'
                  }`}
                />
              </button>
            </div>
          </div>

          {/* Artifact Collection Profile */}
          {showArtifactProfile && (
            <div className="p-4">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Artifact Collection
              </label>
              <ArtifactProfileSelector className="w-full" />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1.5">
                What data to collect during execution (screenshots, logs, network, etc.).
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default ExecutionConfigPanel;
