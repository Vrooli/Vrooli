/**
 * ConfigurationPanel Component
 *
 * Displays and allows editing of configuration options organized by tier.
 */

import { useState, useCallback } from 'react';
import {
  ChevronDown,
  ChevronRight,
  AlertCircle,
  CheckCircle2,
  Loader2,
  Settings2,
  Pencil,
  RotateCcw,
  X,
  Save,
} from 'lucide-react';
import type { ConfigOption, ConfigTier, ConfigUpdateResult } from '@/domains/observability';
import { SettingSection } from '../shared';

// ============================================================================
// Tier Colors
// ============================================================================

const TIER_COLORS: Record<ConfigTier, { bg: string; text: string; label: string }> = {
  essential: { bg: 'bg-emerald-500/20', text: 'text-emerald-300', label: 'Essential' },
  advanced: { bg: 'bg-amber-500/20', text: 'text-amber-300', label: 'Advanced' },
  internal: { bg: 'bg-gray-500/20', text: 'text-gray-400', label: 'Internal' },
};

// ============================================================================
// Config Option Row
// ============================================================================

interface ConfigOptionRowProps {
  option: ConfigOption;
  onUpdate?: (envVar: string, value: string) => Promise<ConfigUpdateResult>;
  onReset?: (envVar: string) => Promise<void>;
  isUpdating?: boolean;
}

function ConfigOptionRow({ option, onUpdate, onReset, isUpdating }: ConfigOptionRowProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState(option.current_value);
  const [error, setError] = useState<string | null>(null);

  const handleStartEdit = useCallback(() => {
    setIsEditing(true);
    setEditValue(option.current_value);
    setError(null);
  }, [option.current_value]);

  const handleCancel = useCallback(() => {
    setIsEditing(false);
    setEditValue(option.current_value);
    setError(null);
  }, [option.current_value]);

  const handleSave = useCallback(async () => {
    if (!onUpdate) return;
    setError(null);
    try {
      const result = await onUpdate(option.env_var, editValue);
      if (result.success) {
        setIsEditing(false);
      } else {
        setError(result.error || 'Update failed');
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Update failed');
    }
  }, [onUpdate, option.env_var, editValue]);

  const handleReset = useCallback(async () => {
    if (!onReset) return;
    try {
      await onReset(option.env_var);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Reset failed');
    }
  }, [onReset, option.env_var]);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      void handleSave();
    } else if (e.key === 'Escape') {
      handleCancel();
    }
  }, [handleSave, handleCancel]);

  // Render input based on data type
  const renderInput = () => {
    if (option.data_type === 'boolean') {
      return (
        <select
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          className="bg-gray-700 border border-gray-600 rounded px-2 py-1 text-xs text-surface focus:ring-2 focus:ring-flow-accent focus:border-transparent"
        >
          <option value="true">true</option>
          <option value="false">false</option>
        </select>
      );
    }

    if (option.data_type === 'enum' && option.enum_values) {
      return (
        <select
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          className="bg-gray-700 border border-gray-600 rounded px-2 py-1 text-xs text-surface focus:ring-2 focus:ring-flow-accent focus:border-transparent"
        >
          {option.enum_values.map((val) => (
            <option key={val} value={val}>{val}</option>
          ))}
        </select>
      );
    }

    // Default: text input with type hint
    return (
      <input
        type={option.data_type === 'integer' || option.data_type === 'float' ? 'number' : 'text'}
        value={editValue}
        onChange={(e) => setEditValue(e.target.value)}
        onKeyDown={handleKeyDown}
        min={option.min}
        max={option.max}
        step={option.data_type === 'float' ? 0.01 : 1}
        className="w-32 bg-gray-700 border border-gray-600 rounded px-2 py-1 text-xs font-mono text-surface focus:ring-2 focus:ring-flow-accent focus:border-transparent"
        autoFocus
      />
    );
  };

  const hasRuntimeOverride = option.is_modified && option.current_value !== option.default_value;

  return (
    <div className={`flex flex-col gap-2 text-sm rounded px-3 py-2 ${option.is_modified ? 'bg-gray-800 border border-amber-500/30' : 'bg-gray-800/50'}`}>
      {/* Main row */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-mono text-xs text-gray-300">{option.env_var}</span>
            {option.is_modified && (
              <span className="text-xs px-1.5 py-0.5 rounded bg-amber-500/20 text-amber-300">modified</span>
            )}
            {!option.editable && (
              <span className="text-xs px-1.5 py-0.5 rounded bg-gray-600 text-gray-400" title="Requires restart to change">
                restart required
              </span>
            )}
          </div>
          <p className="text-xs text-gray-500 mt-0.5">{option.description}</p>
          {option.min !== undefined && option.max !== undefined && (
            <p className="text-xs text-gray-600 mt-0.5">Range: {option.min} - {option.max}</p>
          )}
        </div>

        <div className="flex items-center gap-2 flex-shrink-0">
          {isEditing ? (
            <>
              {renderInput()}
              <button
                onClick={() => void handleSave()}
                disabled={isUpdating}
                className="p-1 text-emerald-400 hover:text-emerald-300 disabled:opacity-50"
                title="Save"
              >
                {isUpdating ? <Loader2 size={14} className="animate-spin" /> : <Save size={14} />}
              </button>
              <button
                onClick={handleCancel}
                className="p-1 text-gray-400 hover:text-gray-300"
                title="Cancel"
              >
                <X size={14} />
              </button>
            </>
          ) : (
            <>
              <span className="text-xs text-gray-400 font-mono">{option.current_value || '(empty)'}</span>
              {option.is_modified && option.default_value !== option.current_value && (
                <span className="text-xs text-gray-600" title={`Default: ${option.default_value}`}>
                  ← {option.default_value || '(empty)'}
                </span>
              )}
              {option.editable && onUpdate && (
                <button
                  onClick={handleStartEdit}
                  className="p-1 text-gray-400 hover:text-flow-accent transition-colors"
                  title="Edit"
                >
                  <Pencil size={14} />
                </button>
              )}
              {hasRuntimeOverride && onReset && (
                <button
                  onClick={() => void handleReset()}
                  className="p-1 text-gray-400 hover:text-amber-400 transition-colors"
                  title="Reset to default"
                >
                  <RotateCcw size={14} />
                </button>
              )}
            </>
          )}
        </div>
      </div>

      {/* Error message */}
      {error && (
        <div className="flex items-center gap-2 text-xs text-red-400 bg-red-500/10 rounded px-2 py-1">
          <AlertCircle size={12} />
          {error}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Config Tier Section
// ============================================================================

interface ConfigTierSectionProps {
  tier: ConfigTier;
  options: ConfigOption[];
  defaultOpen?: boolean;
  onUpdate?: (envVar: string, value: string) => Promise<ConfigUpdateResult>;
  onReset?: (envVar: string) => Promise<void>;
  isUpdating?: boolean;
}

function ConfigTierSection({ tier, options, defaultOpen = false, onUpdate, onReset, isUpdating }: ConfigTierSectionProps) {
  const [isExpanded, setIsExpanded] = useState(defaultOpen);
  const modifiedCount = options.filter(o => o.is_modified).length;
  const editableCount = options.filter(o => o.editable).length;
  const tierStyle = TIER_COLORS[tier];

  return (
    <div className="border border-gray-700 rounded-lg overflow-hidden">
      <button
        type="button"
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full flex items-center justify-between px-4 py-3 bg-gray-800/50 hover:bg-gray-800 transition-colors text-left"
      >
        <div className="flex items-center gap-3">
          <span className={`text-xs px-2 py-0.5 rounded ${tierStyle.bg} ${tierStyle.text}`}>
            {tierStyle.label}
          </span>
          <span className="text-sm text-surface">{options.length} options</span>
          {editableCount > 0 && (
            <span className="text-xs text-emerald-400">({editableCount} editable)</span>
          )}
          {modifiedCount > 0 && (
            <span className="text-xs text-amber-400">({modifiedCount} modified)</span>
          )}
        </div>
        {isExpanded ? <ChevronDown size={16} className="text-gray-400" /> : <ChevronRight size={16} className="text-gray-400" />}
      </button>
      {isExpanded && (
        <div className="p-3 bg-gray-900/50 border-t border-gray-700 space-y-2">
          {options.map((option) => (
            <ConfigOptionRow
              key={option.env_var}
              option={option}
              onUpdate={onUpdate}
              onReset={onReset}
              isUpdating={isUpdating}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Configuration Panel
// ============================================================================

interface ConfigurationPanelProps {
  config: {
    summary: string;
    modified_count: number;
    total_count?: number;
    by_tier?: { essential: number; advanced: number; internal: number };
    modified_options?: Array<{ env_var: string; tier: ConfigTier; description?: string; current_value: string; default_value?: string }>;
    all_options?: { essential: ConfigOption[]; advanced: ConfigOption[]; internal: ConfigOption[] };
  };
  onUpdate?: (envVar: string, value: string) => Promise<ConfigUpdateResult>;
  onReset?: (envVar: string) => Promise<void>;
  isUpdating?: boolean;
}

export function ConfigurationPanel({ config, onUpdate, onReset, isUpdating }: ConfigurationPanelProps) {
  const [showOnlyModified, setShowOnlyModified] = useState(true);

  // Count total editable options
  const totalEditableCount = config.all_options
    ? config.all_options.essential.filter(o => o.editable).length +
      config.all_options.advanced.filter(o => o.editable).length +
      config.all_options.internal.filter(o => o.editable).length
    : 0;

  return (
    <SettingSection title="Configuration" tooltip="Current configuration status and all options" defaultOpen={false}>
      <div className="space-y-4">
        {/* Summary */}
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-2 text-sm">
            <Settings2 size={16} className="text-gray-400" />
            <span className="text-surface">{config.summary}</span>
            {totalEditableCount > 0 && (
              <span className="text-xs text-emerald-400">
                ({totalEditableCount} editable at runtime)
              </span>
            )}
          </div>
          {config.all_options && (
            <button
              onClick={() => setShowOnlyModified(!showOnlyModified)}
              className="text-xs px-2 py-1 rounded bg-gray-700 hover:bg-gray-600 text-gray-300 transition-colors"
            >
              {showOnlyModified ? 'Show All Options' : 'Show Modified Only'}
            </button>
          )}
        </div>

        {/* Modified-only view */}
        {showOnlyModified && (
          <div className="space-y-2">
            {config.modified_count === 0 ? (
              <div className="p-3 bg-gray-800/50 border border-gray-700 rounded text-gray-400 text-center text-sm">
                <CheckCircle2 size={20} className="mx-auto mb-2 text-emerald-500" />
                <p>All options at defaults</p>
                <p className="text-xs text-gray-500 mt-1">Click "Show All Options" to browse available settings</p>
              </div>
            ) : config.modified_options ? (
              config.modified_options.map((opt) => (
                <div key={opt.env_var} className="flex items-center justify-between text-sm bg-gray-800 border border-amber-500/30 rounded px-3 py-2">
                  <div>
                    <span className="font-mono text-xs text-gray-300">{opt.env_var}</span>
                    {opt.description && (
                      <p className="text-xs text-gray-500 mt-0.5">{opt.description}</p>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={`text-xs px-1.5 py-0.5 rounded ${TIER_COLORS[opt.tier].bg} ${TIER_COLORS[opt.tier].text}`}>
                      {opt.tier}
                    </span>
                    <span className="text-gray-400 font-mono text-xs">{opt.current_value}</span>
                    {opt.default_value && opt.default_value !== opt.current_value && (
                      <span className="text-xs text-gray-600">← {opt.default_value}</span>
                    )}
                  </div>
                </div>
              ))
            ) : null}
          </div>
        )}

        {/* All options view */}
        {!showOnlyModified && config.all_options && (
          <div className="space-y-3">
            <ConfigTierSection
              tier="essential"
              options={config.all_options.essential}
              defaultOpen={true}
              onUpdate={onUpdate}
              onReset={onReset}
              isUpdating={isUpdating}
            />
            <ConfigTierSection
              tier="advanced"
              options={config.all_options.advanced}
              onUpdate={onUpdate}
              onReset={onReset}
              isUpdating={isUpdating}
            />
            <ConfigTierSection
              tier="internal"
              options={config.all_options.internal}
              onUpdate={onUpdate}
              onReset={onReset}
              isUpdating={isUpdating}
            />
          </div>
        )}
      </div>
    </SettingSection>
  );
}

export default ConfigurationPanel;
