import { useCallback, useRef, useState } from "react";

export interface PromptDialogOptions {
  title: string;
  message?: string;
  label: string;
  defaultValue?: string;
  placeholder?: string;
  submitLabel?: string;
  cancelLabel?: string;
}

export interface PromptDialogConfig {
  /** Validate the input - return error message or null if valid */
  validate?: (value: string) => string | null;
  /** Normalize the input before returning */
  normalize?: (value: string) => string;
}

export interface PromptDialogState extends PromptDialogOptions {
  isOpen: boolean;
  value: string;
  error: string | null;
  validate: ((value: string) => string | null) | null;
  normalize: ((value: string) => string) | null;
}

export interface UsePromptDialogReturn {
  /** Current dialog state - pass to PromptDialog component */
  dialogState: PromptDialogState | null;
  /** Request input from user - returns promise that resolves to string or null if cancelled */
  prompt: (options: PromptDialogOptions, config?: PromptDialogConfig) => Promise<string | null>;
  /** Update the current input value */
  setValue: (value: string) => void;
  /** Set an error message */
  setError: (error: string | null) => void;
  /** Close the dialog - validates and normalizes before closing if result provided */
  close: (result: string | null) => void;
  /** Submit the current value (validates and normalizes) */
  submit: () => void;
}

/**
 * Hook for managing prompt dialogs with validation.
 *
 * @example
 * ```tsx
 * const { dialogState, prompt, setValue, setError, close, submit } = usePromptDialog();
 *
 * const handleRename = async () => {
 *   const newName = await prompt(
 *     {
 *       title: "Rename item",
 *       label: "New name",
 *       defaultValue: item.name,
 *       placeholder: "Enter new name...",
 *     },
 *     {
 *       validate: (value) => value.trim() ? null : "Name is required",
 *       normalize: (value) => value.trim(),
 *     }
 *   );
 *   if (newName) {
 *     // perform rename
 *   }
 * };
 *
 * return (
 *   <>
 *     <button onClick={handleRename}>Rename</button>
 *     <PromptDialog
 *       state={dialogState}
 *       onValueChange={setValue}
 *       onClose={close}
 *       onSubmit={submit}
 *     />
 *   </>
 * );
 * ```
 */
export function usePromptDialog(): UsePromptDialogReturn {
  const [dialogState, setDialogState] = useState<PromptDialogState | null>(null);
  const resolveRef = useRef<((value: string | null) => void) | null>(null);

  const prompt = useCallback(
    async (options: PromptDialogOptions, config?: PromptDialogConfig): Promise<string | null> => {
      // If there's a pending dialog, resolve it as cancelled
      if (resolveRef.current) {
        resolveRef.current(null);
        resolveRef.current = null;
      }

      return new Promise<string | null>((resolve) => {
        resolveRef.current = resolve;
        setDialogState({
          ...options,
          isOpen: true,
          value: options.defaultValue ?? "",
          error: null,
          validate: config?.validate ?? null,
          normalize: config?.normalize ?? null,
        });
      });
    },
    []
  );

  const setValue = useCallback((value: string) => {
    setDialogState((prev) => (prev ? { ...prev, value, error: null } : null));
  }, []);

  const setError = useCallback((error: string | null) => {
    setDialogState((prev) => (prev ? { ...prev, error } : null));
  }, []);

  const close = useCallback((result: string | null) => {
    const resolve = resolveRef.current;
    const state = dialogState;
    resolveRef.current = null;
    setDialogState(null);

    if (result !== null && state) {
      // Validate if validator provided
      if (state.validate) {
        const errorMessage = state.validate(result);
        if (errorMessage) {
          // Re-open with error
          setDialogState({ ...state, error: errorMessage });
          resolveRef.current = resolve;
          return;
        }
      }
      // Normalize if normalizer provided
      const finalValue = state.normalize ? state.normalize(result) : result.trim();
      resolve?.(finalValue);
    } else {
      resolve?.(null);
    }
  }, [dialogState]);

  const submit = useCallback(() => {
    if (!dialogState) return;

    const value = dialogState.value;

    // Validate if validator provided
    if (dialogState.validate) {
      const errorMessage = dialogState.validate(value);
      if (errorMessage) {
        setDialogState((prev) => (prev ? { ...prev, error: errorMessage } : null));
        return;
      }
    }

    // Normalize if normalizer provided
    const finalValue = dialogState.normalize ? dialogState.normalize(value) : value.trim();

    const resolve = resolveRef.current;
    resolveRef.current = null;
    setDialogState(null);
    resolve?.(finalValue);
  }, [dialogState]);

  return { dialogState, prompt, setValue, setError, close, submit };
}

export default usePromptDialog;
