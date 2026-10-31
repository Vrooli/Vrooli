import { createContext, useCallback, useContext, useReducer, useRef, type ReactNode } from 'react';
import type { APIError, ErrorDetail } from '../../types/api';

export type ToastSeverity = 'info' | 'success' | 'warning' | 'error';

export interface Toast {
  id: string;
  severity: ToastSeverity;
  message: string;
  dedupKey?: string;
  retryFn?: () => void;
  retryAfterSeconds?: number;
  autoDismissMs?: number;
}

interface ToastState {
  toasts: Toast[];
}

type ToastAction =
  | { type: 'ADD'; toast: Toast }
  | { type: 'DISMISS'; id: string };

function toastReducer(state: ToastState, action: ToastAction): ToastState {
  switch (action.type) {
    case 'ADD': {
      // Deduplicate by dedupKey (falls back to message)
      const key = action.toast.dedupKey ?? action.toast.message;
      if (state.toasts.some(t => (t.dedupKey ?? t.message) === key)) {
        return state;
      }
      return { toasts: [...state.toasts, action.toast].slice(-5) };
    }
    case 'DISMISS':
      return { toasts: state.toasts.filter(t => t.id !== action.id) };
    default:
      return state;
  }
}

interface ToastContextValue {
  toasts: Toast[];
  showToast: (severity: ToastSeverity, message: string, options?: { retryFn?: () => void; autoDismissMs?: number }) => void;
  dismissToast: (id: string) => void;
  showApiError: (err: unknown, retryFn?: () => void) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(toastReducer, { toasts: [] });
  const nextId = useRef(0);

  const dismissToast = useCallback((id: string) => {
    dispatch({ type: 'DISMISS', id });
  }, []);

  const showToast = useCallback((severity: ToastSeverity, message: string, options?: { retryFn?: () => void; autoDismissMs?: number; dedupKey?: string }) => {
    const id = `toast-${++nextId.current}`;
    const autoDismissMs = options?.autoDismissMs ?? (severity === 'error' ? 8000 : 5000);
    dispatch({
      type: 'ADD',
      toast: { id, severity, message, dedupKey: options?.dedupKey, retryFn: options?.retryFn, autoDismissMs },
    });
    if (autoDismissMs > 0) {
      setTimeout(() => { dispatch({ type: 'DISMISS', id }); }, autoDismissMs);
    }
  }, []);

  const showApiError = useCallback((err: unknown, retryFn?: () => void) => {
    if (err && typeof err === 'object' && 'error' in err) {
      const apiErr = err as APIError;
      const detail: ErrorDetail | undefined = apiErr.detail;
      let message = apiErr.error || 'An error occurred';

      // Append recovery-specific suffix
      const recovery = detail?.recovery;
      if (recovery === 'authenticate') {
        message += '. Please sign in and try again';
      } else if (recovery === 'wait' && !detail?.retry_after_seconds) {
        message += '. Try again shortly';
      } else if (recovery === 'fix_input') {
        message += detail?.field ? `. Please correct the "${detail.field}" field` : '. Please check your input';
      } else if (recovery === 'check_config') {
        message += '. Check your configuration settings';
      } else if (recovery === 'contact_admin') {
        message += '. Contact your administrator';
      } else if (recovery === 'none') {
        message += '. If this persists, contact support';
      }

      if (detail?.request_id) {
        message += ` (ref: ${detail.request_id.slice(-8)})`;
      }

      // Only offer retry for recoveries where retrying makes sense
      const canRetry = recovery === 'wait' || recovery === undefined;
      const retry = detail?.retryable && canRetry ? retryFn : undefined;
      const dedupKey = apiErr.error + (detail?.code ?? '');

      showToast('error', message, {
        retryFn: retry,
        autoDismissMs: detail?.retry_after_seconds ? detail.retry_after_seconds * 1000 + 2000 : 8000,
        dedupKey,
      });
      return;
    }
    const message = err instanceof Error ? err.message : 'An unknown error occurred';
    showToast('error', message);
  }, [showToast]);

  return (
    <ToastContext.Provider value={{ toasts: state.toasts, showToast, dismissToast, showApiError }}>
      {children}
    </ToastContext.Provider>
  );
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return ctx;
}
