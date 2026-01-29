import { useState, useCallback } from 'react';
import type { ApiError } from '../api/common';
import type { AlertSeverity } from './InlineAlert';

interface InlineAlertPropsBase {
  message: string;
  severity?: AlertSeverity;
  title?: string;
  dismissible?: boolean;
  retryable?: boolean;
  onRetry?: () => void | Promise<void>;
  className?: string;
  'data-testid'?: string;
}

/**
 * Hook for managing inline alert state with auto-dismiss and retry support.
 * Use this to replace `alert()` patterns with proper error handling.
 *
 * @example
 * const { alert, showError, clearAlert } = useInlineAlert();
 *
 * try {
 *   await someAction();
 * } catch (err) {
 *   showError(err, () => someAction());
 * }
 *
 * return alert ? <InlineAlert {...alert} onDismiss={clearAlert} /> : null;
 */
export function useInlineAlert(options: { autoDismissMs?: number } = {}) {
  const { autoDismissMs } = options;
  const [alert, setAlert] = useState<InlineAlertPropsBase | null>(null);

  const clearAlert = useCallback(() => {
    setAlert(null);
  }, []);

  const showAlert = useCallback(
    (props: InlineAlertPropsBase) => {
      setAlert(props);
      if (autoDismissMs && autoDismissMs > 0) {
        setTimeout(clearAlert, autoDismissMs);
      }
    },
    [autoDismissMs, clearAlert]
  );

  /**
   * Show an error alert from an Error or ApiError.
   * Automatically sets severity, retryable based on error type.
   */
  const showError = useCallback(
    (error: unknown, retryFn?: () => void | Promise<void>) => {
      let message = 'An unexpected error occurred';
      let retryable = false;

      if (error && typeof error === 'object' && 'userMessage' in error) {
        // ApiError
        const apiErr = error as ApiError;
        message = apiErr.userMessage || apiErr.message;
        retryable = apiErr.retryable;
      } else if (error instanceof Error) {
        message = error.message;
      } else if (typeof error === 'string') {
        message = error;
      }

      showAlert({
        message,
        severity: 'error',
        retryable: retryable && !!retryFn,
        onRetry: retryFn,
      });
    },
    [showAlert]
  );

  /**
   * Show a warning alert.
   */
  const showWarning = useCallback(
    (message: string, title?: string) => {
      showAlert({ message, title, severity: 'warning' });
    },
    [showAlert]
  );

  /**
   * Show a success alert.
   */
  const showSuccess = useCallback(
    (message: string, title?: string) => {
      showAlert({ message, title, severity: 'success' });
    },
    [showAlert]
  );

  return {
    alert,
    showAlert,
    showError,
    showWarning,
    showSuccess,
    clearAlert,
  };
}
