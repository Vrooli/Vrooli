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
// Advanced Specialized Variants
// ============================================================================

export interface UseSyncedArrayAsTextOptions {
  /** Separator regex for parsing (default: /[\n,]+/) */
  separator?: RegExp;
  /** Join string for display (default: '\n') */
  joinWith?: string;
  /** Called when the field should be committed */
  onCommit?: (value: string[]) => void;
}

/**
 * Specialized hook for array fields edited as text (comma/newline separated).
 * Local state is a string for editing, but commits as string[].
 *
 * @example
 * ```tsx
 * const selectors = useSyncedArrayAsText(params?.selectors ?? [], {
 *   onCommit: (v) => updateParams({ selectors: v.length ? v : undefined }),
 * });
 *
 * <textarea
 *   value={selectors.value}
 *   onChange={(e) => selectors.setValue(e.target.value)}
 *   onBlur={selectors.commit}
 *   placeholder="selector1, selector2"
 * />
 * ```
 */
export function useSyncedArrayAsText(
  externalValue: string[],
  options: UseSyncedArrayAsTextOptions = {},
): UseSyncedFieldResult<string> & { parsedValue: string[] } {
  const { separator = /[\n,]+/, joinWith = '\n', onCommit } = options;

  // Convert array to display string
  const displayValue = Array.isArray(externalValue) ? externalValue.join(joinWith) : '';

  // Parse string to array
  const parseToArray = useCallback(
    (text: string): string[] =>
      text
        .split(separator)
        .map((entry) => entry.trim())
        .filter((entry) => entry.length > 0),
    [separator],
  );

  const field = useSyncedField(displayValue, {
    onCommit: (text) => {
      const parsed = parseToArray(text);
      onCommit?.(parsed);
    },
  });

  return {
    ...field,
    parsedValue: parseToArray(field.value),
  };
}

export interface UseSyncedOptionalNumberOptions {
  /** Minimum allowed value */
  min?: number;
  /** Maximum allowed value */
  max?: number;
  /** Whether to allow floating point (default: false, rounds to int) */
  float?: boolean;
  /** Called when the field should be committed */
  onCommit?: (value: number | undefined) => void;
}

/**
 * Specialized hook for optional number fields.
 * Local state is a string for editing, commits as number | undefined.
 * Empty string commits as undefined.
 *
 * @example
 * ```tsx
 * const padding = useSyncedOptionalNumber(params?.padding, {
 *   min: 0,
 *   onCommit: (v) => updateParams({ padding: v }),
 * });
 *
 * <input
 *   type="number"
 *   value={padding.value}
 *   onChange={(e) => padding.setValue(e.target.value)}
 *   onBlur={padding.commit}
 * />
 * ```
 */
export function useSyncedOptionalNumber(
  externalValue: number | undefined,
  options: UseSyncedOptionalNumberOptions = {},
): UseSyncedFieldResult<string> & { numericValue: number | undefined } {
  const { min, max, float = false, onCommit } = options;

  // Convert number to display string
  const displayValue = externalValue !== undefined ? String(externalValue) : '';

  // Parse and normalize
  const parseToNumber = useCallback(
    (text: string): number | undefined => {
      const trimmed = text.trim();
      if (trimmed === '') return undefined;

      const parsed = float ? Number.parseFloat(trimmed) : Number.parseInt(trimmed, 10);
      if (Number.isNaN(parsed)) return undefined;

      let result = parsed;
      if (min !== undefined) result = Math.max(min, result);
      if (max !== undefined) result = Math.min(max, result);
      return result;
    },
    [min, max, float],
  );

  const field = useSyncedField(displayValue, {
    onCommit: (text) => {
      const parsed = parseToNumber(text);
      onCommit?.(parsed);
    },
  });

  return {
    ...field,
    numericValue: parseToNumber(field.value),
  };
}

export interface UseSyncedJsonOptions<T> {
  /** Fallback value for invalid JSON (default: {}) */
  fallback?: T;
  /** Indentation for display (default: 2) */
  indent?: number;
  /** Called when the field should be committed */
  onCommit?: (value: T) => void;
}

/**
 * Specialized hook for JSON object fields.
 * Local state is a string for editing, commits as parsed object.
 * Invalid JSON is not committed (keeps previous value).
 *
 * @example
 * ```tsx
 * const params = useSyncedJson<Record<string, unknown>>(data?.parameters ?? {}, {
 *   onCommit: (v) => updateData({ parameters: v }),
 * });
 *
 * <textarea
 *   value={params.value}
 *   onChange={(e) => params.setValue(e.target.value)}
 *   onBlur={params.commit}
 * />
 * {params.parseError && <span className="text-red-400">{params.parseError}</span>}
 * ```
 */
export function useSyncedJson<T extends object>(
  externalValue: T,
  options: UseSyncedJsonOptions<T> = {},
): UseSyncedFieldResult<string> & { parsedValue: T | undefined; parseError: string | undefined } {
  const { fallback = {} as T, indent = 2, onCommit } = options;

  // Convert object to display string
  const displayValue = JSON.stringify(externalValue ?? fallback, null, indent);

  const [parseError, setParseError] = useState<string | undefined>(undefined);

  const field = useSyncedField(displayValue, {
    onCommit: (text) => {
      try {
        const parsed = JSON.parse(text) as T;
        setParseError(undefined);
        onCommit?.(parsed);
      } catch (err) {
        setParseError(err instanceof Error ? err.message : 'Invalid JSON');
        // Don't commit invalid JSON
      }
    },
  });

  // Try to parse current value for live feedback
  let parsedValue: T | undefined;
  try {
    parsedValue = JSON.parse(field.value) as T;
  } catch {
    parsedValue = undefined;
  }

  return {
    ...field,
    parsedValue,
    parseError,
  };
}

export interface UseSyncedObjectOptions<T extends object> {
  /** Called when the field should be committed */
  onCommit?: (value: T) => void;
}

/**
 * Specialized hook for simple object fields (like modifier checkboxes).
 * Provides toggle helpers for boolean fields within the object.
 *
 * @example
 * ```tsx
 * const modifiers = useSyncedObject({ ctrl: false, shift: false, alt: false }, {
 *   onCommit: (v) => updateData({ modifiers: v }),
 * });
 *
 * <input
 *   type="checkbox"
 *   checked={modifiers.value.ctrl}
 *   onChange={() => modifiers.toggle('ctrl')}
 * />
 * ```
 */
export function useSyncedObject<T extends object>(
  externalValue: T,
  options: UseSyncedObjectOptions<T> = {},
): UseSyncedFieldResult<T> & {
  toggle: (key: keyof T) => void;
  setField: <K extends keyof T>(key: K, value: T[K]) => void;
} {
  const { onCommit } = options;

  // Deep equality check for objects
  const isEqual = useCallback((a: T, b: T): boolean => {
    return JSON.stringify(a) === JSON.stringify(b);
  }, []);

  const field = useSyncedField(externalValue, { onCommit, isEqual });

  // Toggle a boolean field and commit immediately
  const toggle = useCallback(
    (key: keyof T) => {
      field.setValue((prev) => {
        const next = { ...prev, [key]: !prev[key] } as T;
        // Commit immediately for checkboxes
        onCommit?.(next);
        return next;
      });
    },
    [field, onCommit],
  );

  // Set a specific field and commit immediately
  const setField = useCallback(
    <K extends keyof T>(key: K, value: T[K]) => {
      field.setValue((prev) => {
        const next = { ...prev, [key]: value } as T;
        onCommit?.(next);
        return next;
      });
    },
    [field, onCommit],
  );

  return {
    ...field,
    toggle,
    setField,
  };
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
