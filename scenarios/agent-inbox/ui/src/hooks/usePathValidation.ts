import { useState, useEffect, useRef } from "react";
import { validatePath, type ValidatePathResult } from "../lib/api";

interface UsePathValidationOptions {
  /** Debounce delay in ms (default: 400) */
  debounceMs?: number;
}

interface PathValidationState {
  /** Whether validation is currently in-flight */
  isValidating: boolean;
  /** Result from the last completed validation (null if never validated) */
  result: ValidatePathResult | null;
}

/**
 * Debounced path validation hook.
 * Calls the server-side validate-path endpoint after a debounce delay.
 * Skips validation for empty paths.
 */
export function usePathValidation(
  path: string,
  { debounceMs = 400 }: UsePathValidationOptions = {},
): PathValidationState {
  const [isValidating, setIsValidating] = useState(false);
  const [result, setResult] = useState<ValidatePathResult | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout>>();
  const abortRef = useRef<AbortController>();

  useEffect(() => {
    // Clear previous timer
    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }

    // Reset when empty
    if (!path.trim()) {
      setResult(null);
      setIsValidating(false);
      return;
    }

    setIsValidating(true);

    timerRef.current = setTimeout(() => {
      // Cancel previous in-flight request
      if (abortRef.current) {
        abortRef.current.abort();
      }
      abortRef.current = new AbortController();

      validatePath(path.trim())
        .then((res) => {
          setResult(res);
        })
        .catch((err: unknown) => {
          // Ignore abort errors
          if (err instanceof Error && err.name === "AbortError") return;
          setResult({ valid: false, message: "Validation failed" });
        })
        .finally(() => {
          setIsValidating(false);
        });
    }, debounceMs);

    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, [path, debounceMs]);

  return { isValidating, result };
}
