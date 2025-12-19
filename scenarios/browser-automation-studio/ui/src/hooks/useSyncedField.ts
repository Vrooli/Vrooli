/**
 * useSyncedField Hook
 *
 * Provides a pattern for managing local state that syncs with an external value.
 * This eliminates the common useState + useEffect boilerplate pattern found in node components.
 *
 * Features:
 * - Local state for responsive UI (no lag on keystroke)
 * - Auto-sync when external value changes
 * - Normalization on commit (trim strings, clamp numbers, etc.)
 * - Ready-to-spread inputProps for common input elements
 */

import { useState, useEffect, useCallback, useRef } from 'react';

// ============================================================================
// Core Hook
// ============================================================================

export interface UseSyncedFieldOptions<T> {
  /** Transform value before committing (e.g., trim strings, clamp numbers) */
  normalize?: (value: T) => T;
  /** Called when the field should be committed (on blur) */
  onCommit?: (value: T) => void;
  /** Compare function for detecting external changes (default: Object.is) */
  isEqual?: (a: T, b: T) => boolean;
}

export interface UseSyncedFieldResult<T> {
  /** Current local value */
  value: T;
  /** Update local value (doesn't commit) */
  setValue: React.Dispatch<React.SetStateAction<T>>;
  /** Commit the current value (calls normalize + onCommit) */
  commit: () => void;
  /** Whether local value differs from last committed value */
  isDirty: boolean;
}

/**
 * Hook for managing local state synced with an external value.
 *
 * @param externalValue - The value from the external source (e.g., params?.selector)
 * @param options - Configuration options
 * @returns Object with value, setValue, commit, and isDirty
 *
 * @example
 * ```tsx
 * const selector = useSyncedField(params?.selector ?? '', {
 *   normalize: (v) => v.trim(),
 *   onCommit: (v) => updateParams({ selector: v }),
 * });
 *
 * <input
 *   value={selector.value}
 *   onChange={(e) => selector.setValue(e.target.value)}
 *   onBlur={selector.commit}
 * />
 * ```
 */
export function useSyncedField<T>(
  externalValue: T,
  options: UseSyncedFieldOptions<T> = {},
): UseSyncedFieldResult<T> {
  const { normalize, onCommit, isEqual = Object.is } = options;

  const [localValue, setLocalValue] = useState<T>(externalValue);
  const lastExternalRef = useRef(externalValue);
  const lastCommittedRef = useRef(externalValue);

  // Sync local state when external value changes (from undo, external edit, etc.)
  useEffect(() => {
    if (!isEqual(externalValue, lastExternalRef.current)) {
      setLocalValue(externalValue);
      lastExternalRef.current = externalValue;
      lastCommittedRef.current = externalValue;
    }
  }, [externalValue, isEqual]);

  // Commit handler - normalize and notify parent
  const commit = useCallback(() => {
    const finalValue = normalize ? normalize(localValue) : localValue;

    // Update local to normalized value if different
    if (!isEqual(finalValue, localValue)) {
      setLocalValue(finalValue);
    }

    // Notify parent
    onCommit?.(finalValue);
    lastCommittedRef.current = finalValue;
    lastExternalRef.current = finalValue;
  }, [localValue, normalize, onCommit, isEqual]);

  // Track dirty state
  const isDirty = !isEqual(localValue, lastCommittedRef.current);

  return {
    value: localValue,
    setValue: setLocalValue,
    commit,
    isDirty,
  };
}

// ============================================================================
// Specialized Variants
// ============================================================================

export interface UseSyncedStringOptions {
  /** Whether to trim the string on commit (default: true) */
  trim?: boolean;
  /** Called when the field should be committed */
  onCommit?: (value: string) => void;
}

/**
 * Specialized hook for string fields with automatic trimming.
 *
 * @example
 * ```tsx
 * const name = useSyncedString(params?.name ?? '', {
 *   onCommit: (v) => updateParams({ name: v || undefined }),
 * });
 * ```
 */
export function useSyncedString(
  externalValue: string,
  options: UseSyncedStringOptions = {},
): UseSyncedFieldResult<string> {
  const { trim = true, onCommit } = options;

  return useSyncedField(externalValue, {
    normalize: trim ? (v) => v.trim() : undefined,
    onCommit,
  });
}

export interface UseSyncedNumberOptions {
  /** Minimum allowed value */
  min?: number;
  /** Maximum allowed value */
  max?: number;
  /** Fallback value for NaN/invalid input */
  fallback?: number;
  /** Whether to round to integer (default: true) */
  round?: boolean;
  /** Called when the field should be committed */
  onCommit?: (value: number) => void;
}

/**
 * Specialized hook for number fields with clamping and validation.
 *
 * @example
 * ```tsx
 * const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 30000, {
 *   min: 100,
 *   max: 120000,
 *   onCommit: (v) => updateParams({ timeoutMs: v }),
 * });
 * ```
 */
export function useSyncedNumber(
  externalValue: number,
  options: UseSyncedNumberOptions = {},
): UseSyncedFieldResult<number> {
  const { min, max, fallback, round = true, onCommit } = options;

  const normalize = useCallback(
    (value: number): number => {
      // Handle NaN/Infinity
      if (!Number.isFinite(value)) {
        return fallback ?? min ?? 0;
      }

      let result = value;

      // Round if requested
      if (round) {
        result = Math.round(result);
      }

      // Apply min/max bounds
      if (min !== undefined) {
        result = Math.max(min, result);
      }
      if (max !== undefined) {
        result = Math.min(max, result);
      }

      return result;
    },
    [min, max, fallback, round],
  );

  return useSyncedField(externalValue, {
    normalize,
    onCommit,
  });
}

export interface UseSyncedBooleanOptions {
  /** Called when the field should be committed */
  onCommit?: (value: boolean) => void;
}

/**
 * Specialized hook for boolean fields.
 *
 * @example
 * ```tsx
 * const secure = useSyncedBoolean(params?.secure ?? false, {
 *   onCommit: (v) => updateParams({ secure: v || undefined }),
 * });
 * ```
 */
export function useSyncedBoolean(
  externalValue: boolean,
  options: UseSyncedBooleanOptions = {},
): UseSyncedFieldResult<boolean> {
  const { onCommit } = options;

  return useSyncedField(externalValue, { onCommit });
}

export interface UseSyncedSelectOptions<T extends string> {
  /** Called when the field should be committed */
  onCommit?: (value: T) => void;
}

/**
 * Specialized hook for select/dropdown fields.
 * Commits immediately on change (no blur needed for selects).
 *
 * @example
 * ```tsx
 * const sameSite = useSyncedSelect(params?.sameSite ?? '', {
 *   onCommit: (v) => updateParams({ sameSite: v || undefined }),
 * });
 *
 * <select
 *   value={sameSite.value}
 *   onChange={(e) => {
 *     sameSite.setValue(e.target.value);
 *     sameSite.commit(); // Immediate commit for selects
 *   }}
 * />
 * ```
 */
export function useSyncedSelect<T extends string>(
  externalValue: T,
  options: UseSyncedSelectOptions<T> = {},
): UseSyncedFieldResult<T> {
  const { onCommit } = options;

  return useSyncedField(externalValue, { onCommit });
}

// ============================================================================
// Input Props Helpers
// ============================================================================

/**
 * Helper to create onChange handler for text inputs.
 */
export function textInputHandler(
  setValue: React.Dispatch<React.SetStateAction<string>>,
): (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void {
  return (e) => setValue(e.target.value);
}

/**
 * Helper to create onChange handler for number inputs.
 */
export function numberInputHandler(
  setValue: React.Dispatch<React.SetStateAction<number>>,
): (e: React.ChangeEvent<HTMLInputElement>) => void {
  return (e) => setValue(Number(e.target.value));
}

/**
 * Helper to create onChange handler for checkbox inputs.
 */
export function checkboxInputHandler(
  setValue: React.Dispatch<React.SetStateAction<boolean>>,
  commitImmediately?: () => void,
): (e: React.ChangeEvent<HTMLInputElement>) => void {
  return (e) => {
    setValue(e.target.checked);
    // Checkboxes typically commit immediately (no blur)
    commitImmediately?.();
  };
}

/**
 * Helper to create onChange handler for select inputs.
 */
export function selectInputHandler<T extends string>(
  setValue: React.Dispatch<React.SetStateAction<T>>,
  commitImmediately?: () => void,
): (e: React.ChangeEvent<HTMLSelectElement>) => void {
  return (e) => {
    setValue(e.target.value as T);
    // Selects typically commit immediately (no blur)
    commitImmediately?.();
  };
}

export default useSyncedField;
