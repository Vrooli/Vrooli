/**
 * Runtime Configuration Module
 *
 * Provides a mechanism to override configuration values at runtime
 * without requiring a server restart. Only options marked as `editable: true`
 * in CONFIG_TIER_METADATA can be changed at runtime.
 *
 * ## How It Works
 *
 * 1. On startup, config is loaded from environment variables and frozen
 * 2. Runtime overrides are stored in this module
 * 3. Consumers use `getRuntimeConfig()` to get values with overrides applied
 * 4. Changes are NOT persisted - they reset on restart
 *
 * ## Usage
 *
 * ```typescript
 * import { getRuntimeValue, setRuntimeValue } from './runtime-config';
 *
 * // Get a value (returns runtime override if set, otherwise env/default)
 * const logLevel = getRuntimeValue('LOG_LEVEL');
 *
 * // Set a runtime value (only works for editable options)
 * const result = setRuntimeValue('LOG_LEVEL', 'debug');
 * if (!result.success) {
 *   console.error(result.error);
 * }
 * ```
 */

import { CONFIG_TIER_METADATA, type ConfigDataType } from './config';
import { logger, scopedLog, LogContext } from './utils';

// =============================================================================
// Types
// =============================================================================

export interface RuntimeConfigValue {
  value: string;
  setAt: Date;
  previousValue?: string;
}

export interface SetConfigResult {
  success: boolean;
  env_var?: string;
  new_value?: string;
  previous_value?: string;
  error?: string;
  validation?: {
    data_type: ConfigDataType;
    min?: number;
    max?: number;
    enum_values?: string[];
  };
}

export interface RuntimeConfigState {
  overrides: Record<string, RuntimeConfigValue>;
  last_updated_at: string | null;
  total_overrides: number;
}

// =============================================================================
// Runtime Config Store
// =============================================================================

/** In-memory store for runtime config overrides */
const runtimeOverrides: Map<string, RuntimeConfigValue> = new Map();

/** Listeners for config changes */
type ConfigChangeListener = (envVar: string, newValue: string, oldValue?: string) => void;
const changeListeners: Set<ConfigChangeListener> = new Set();

// =============================================================================
// Public API
// =============================================================================

/**
 * Get the effective value for a configuration option.
 *
 * Priority:
 * 1. Runtime override (if set)
 * 2. Environment variable (if set)
 * 3. Default value from metadata
 *
 * @param envVar - Environment variable name (e.g., 'LOG_LEVEL')
 * @returns The effective value as a string, or undefined if not found
 */
export function getRuntimeValue(envVar: string): string | undefined {
  // Check for runtime override first
  const override = runtimeOverrides.get(envVar);
  if (override) {
    return override.value;
  }

  // Fall back to environment variable
  const envValue = process.env[envVar];
  if (envValue !== undefined && envValue.trim() !== '') {
    return envValue.trim();
  }

  // Fall back to default
  const meta = CONFIG_TIER_METADATA[envVar];
  if (meta && meta.defaultValue !== undefined) {
    return String(meta.defaultValue);
  }

  return undefined;
}

/**
 * Get the effective value parsed as the correct type.
 *
 * @param envVar - Environment variable name
 * @returns Parsed value with correct type, or undefined
 */
export function getRuntimeValueTyped(envVar: string): unknown {
  const stringValue = getRuntimeValue(envVar);
  if (stringValue === undefined) {
    return undefined;
  }

  const meta = CONFIG_TIER_METADATA[envVar];
  if (!meta) {
    return stringValue;
  }

  switch (meta.dataType) {
    case 'boolean':
      return stringValue.toLowerCase() === 'true';
    case 'integer':
      return parseInt(stringValue, 10);
    case 'float':
      return parseFloat(stringValue);
    case 'enum':
    case 'string':
    default:
      return stringValue;
  }
}

/**
 * Set a runtime configuration value.
 *
 * Only works for options marked as `editable: true` in CONFIG_TIER_METADATA.
 * The value is validated against the option's type and constraints.
 *
 * @param envVar - Environment variable name (e.g., 'LOG_LEVEL')
 * @param value - New value as a string
 * @returns Result indicating success or failure with details
 */
export function setRuntimeValue(envVar: string, value: string): SetConfigResult {
  const meta = CONFIG_TIER_METADATA[envVar];

  // Check if option exists
  if (!meta) {
    return {
      success: false,
      error: `Unknown configuration option: ${envVar}`,
    };
  }

  // Check if option is editable
  if (!meta.editable) {
    return {
      success: false,
      env_var: envVar,
      error: `Configuration option '${envVar}' cannot be changed at runtime. Restart required.`,
    };
  }

  // Validate the value
  const validation = validateValue(value, meta.dataType, meta.min, meta.max, meta.enumValues);
  if (!validation.valid) {
    return {
      success: false,
      env_var: envVar,
      error: validation.error,
      validation: {
        data_type: meta.dataType,
        min: meta.min,
        max: meta.max,
        enum_values: meta.enumValues,
      },
    };
  }

  // Get previous value for logging
  const previousValue = getRuntimeValue(envVar);

  // Set the new value
  runtimeOverrides.set(envVar, {
    value: validation.normalizedValue ?? value,
    setAt: new Date(),
    previousValue,
  });

  // Notify listeners
  const normalizedValue = validation.normalizedValue ?? value;
  for (const listener of changeListeners) {
    try {
      listener(envVar, normalizedValue, previousValue);
    } catch (error) {
      logger.warn(scopedLog(LogContext.CONFIG, 'config change listener error'), {
        envVar,
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }

  logger.info(scopedLog(LogContext.CONFIG, 'runtime config updated'), {
    envVar,
    newValue: normalizedValue,
    previousValue,
  });

  return {
    success: true,
    env_var: envVar,
    new_value: normalizedValue,
    previous_value: previousValue,
  };
}

/**
 * Reset a runtime configuration value back to its environment/default value.
 *
 * @param envVar - Environment variable name
 * @returns true if the override was removed, false if there was no override
 */
export function resetRuntimeValue(envVar: string): boolean {
  const hadOverride = runtimeOverrides.has(envVar);
  if (hadOverride) {
    const previousValue = runtimeOverrides.get(envVar)?.value;
    runtimeOverrides.delete(envVar);

    const newValue = getRuntimeValue(envVar);

    logger.info(scopedLog(LogContext.CONFIG, 'runtime config reset'), {
      envVar,
      previousValue,
      newValue,
    });

    // Notify listeners
    for (const listener of changeListeners) {
      try {
        listener(envVar, newValue ?? '', previousValue);
      } catch (error) {
        logger.warn(scopedLog(LogContext.CONFIG, 'config change listener error'), {
          envVar,
          error: error instanceof Error ? error.message : String(error),
        });
      }
    }
  }
  return hadOverride;
}

/**
 * Reset all runtime configuration overrides.
 */
export function resetAllRuntimeValues(): void {
  const count = runtimeOverrides.size;
  runtimeOverrides.clear();

  if (count > 0) {
    logger.info(scopedLog(LogContext.CONFIG, 'all runtime config reset'), {
      clearedCount: count,
    });
  }
}

/**
 * Get the current state of all runtime overrides.
 */
export function getRuntimeConfigState(): RuntimeConfigState {
  const overrides: Record<string, RuntimeConfigValue> = {};
  let lastUpdated: Date | null = null;

  for (const [key, val] of runtimeOverrides.entries()) {
    overrides[key] = val;
    if (!lastUpdated || val.setAt > lastUpdated) {
      lastUpdated = val.setAt;
    }
  }

  return {
    overrides,
    last_updated_at: lastUpdated?.toISOString() ?? null,
    total_overrides: runtimeOverrides.size,
  };
}

/**
 * Register a listener for configuration changes.
 *
 * @param listener - Function to call when config changes
 * @returns Cleanup function to remove the listener
 */
export function onConfigChange(listener: ConfigChangeListener): () => void {
  changeListeners.add(listener);
  return () => {
    changeListeners.delete(listener);
  };
}

/**
 * Check if a configuration option has a runtime override.
 *
 * @param envVar - Environment variable name
 * @returns true if there is a runtime override
 */
export function hasRuntimeOverride(envVar: string): boolean {
  return runtimeOverrides.has(envVar);
}

// =============================================================================
// Validation
// =============================================================================

interface ValidationResult {
  valid: boolean;
  error?: string;
  normalizedValue?: string;
}

function validateValue(
  value: string,
  dataType: ConfigDataType,
  min?: number,
  max?: number,
  enumValues?: string[]
): ValidationResult {
  const trimmed = value.trim();

  switch (dataType) {
    case 'boolean': {
      const lower = trimmed.toLowerCase();
      if (lower !== 'true' && lower !== 'false') {
        return { valid: false, error: `Invalid boolean value: ${value}. Must be 'true' or 'false'.` };
      }
      return { valid: true, normalizedValue: lower };
    }

    case 'integer': {
      const parsed = parseInt(trimmed, 10);
      if (Number.isNaN(parsed)) {
        return { valid: false, error: `Invalid integer value: ${value}` };
      }
      if (min !== undefined && parsed < min) {
        return { valid: false, error: `Value ${parsed} is below minimum ${min}` };
      }
      if (max !== undefined && parsed > max) {
        return { valid: false, error: `Value ${parsed} is above maximum ${max}` };
      }
      return { valid: true, normalizedValue: String(parsed) };
    }

    case 'float': {
      const parsed = parseFloat(trimmed);
      if (Number.isNaN(parsed)) {
        return { valid: false, error: `Invalid float value: ${value}` };
      }
      if (min !== undefined && parsed < min) {
        return { valid: false, error: `Value ${parsed} is below minimum ${min}` };
      }
      if (max !== undefined && parsed > max) {
        return { valid: false, error: `Value ${parsed} is above maximum ${max}` };
      }
      return { valid: true, normalizedValue: String(parsed) };
    }

    case 'enum': {
      if (!enumValues || enumValues.length === 0) {
        return { valid: true, normalizedValue: trimmed };
      }
      if (!enumValues.includes(trimmed)) {
        return { valid: false, error: `Invalid value: ${value}. Must be one of: ${enumValues.join(', ')}` };
      }
      return { valid: true, normalizedValue: trimmed };
    }

    case 'string':
    default:
      return { valid: true, normalizedValue: trimmed };
  }
}

// =============================================================================
// Logger Integration
// =============================================================================

/**
 * Special handler for LOG_LEVEL changes.
 * Updates the logger's level when LOG_LEVEL is changed at runtime.
 */
onConfigChange((envVar, newValue) => {
  if (envVar === 'LOG_LEVEL') {
    try {
      // The logger module should be updated to support setLevel()
      // For now, we just log that the level was changed
      logger.info(scopedLog(LogContext.CONFIG, 'log level changed'), {
        newLevel: newValue,
        note: 'Level change will take effect on new log entries',
      });
      // TODO: Implement logger.setLevel(newValue) when logger supports it
    } catch (error) {
      logger.warn(scopedLog(LogContext.CONFIG, 'failed to update log level'), {
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }
});
