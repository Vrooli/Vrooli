import { useState, useCallback, useEffect, createContext, useContext, type ReactNode } from 'react';
import { CheckCircle, XCircle, AlertTriangle, AlertCircle, X } from 'lucide-react';
import { cn } from '../lib/utils';

/**
 * Toast notification types for visual feedback on completed operations.
 * [REQ:SIGNAL-FEEDBACK] Success feedback for completed operations
 */
export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface ToastItem {
  id: string;
  type: ToastType;
  message: string;
  title?: string;
  /** Duration in ms before auto-dismiss. 0 = manual dismiss only. Default 4000. */
  duration?: number;
}

interface ToastContextValue {
  toasts: ToastItem[];
  addToast: (toast: Omit<ToastItem, 'id'>) => string;
  removeToast: (id: string) => void;
  /** Convenience methods */
  success: (message: string, title?: string) => string;
  error: (message: string, title?: string) => string;
  warning: (message: string, title?: string) => string;
  info: (message: string, title?: string) => string;
}

const ToastContext = createContext<ToastContextValue | null>(null);

/**
 * Hook to access toast notifications.
 * Must be used within a ToastProvider.
 *
 * @example
 * const { success, error } = useToast();
 *
 * const handleSave = async () => {
 *   try {
 *     await saveData();
 *     success('Changes saved successfully');
 *   } catch (err) {
 *     error('Failed to save changes');
 *   }
 * };
 */
export function useToast(): ToastContextValue {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return context;
}

interface ToastProviderProps {
  children: ReactNode;
  /** Default duration for toasts in ms. Default 4000. */
  defaultDuration?: number;
  /** Maximum number of toasts to show at once. Default 5. */
  maxToasts?: number;
}

let toastIdCounter = 0;

/**
 * ToastProvider manages toast state and renders the toast container.
 * Wrap your app or admin layout to enable toast notifications.
 */
export function ToastProvider({ children, defaultDuration = 4000, maxToasts = 5 }: ToastProviderProps) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const addToast = useCallback(
    (toast: Omit<ToastItem, 'id'>): string => {
      const id = `toast-${++toastIdCounter}-${Date.now()}`;
      const newToast: ToastItem = { id, duration: defaultDuration, ...toast };

      setToasts((prev) => {
        const updated = [...prev, newToast];
        // Remove oldest if exceeding max
        return updated.slice(-maxToasts);
      });

      return id;
    },
    [defaultDuration, maxToasts]
  );

  // Convenience methods
  const success = useCallback(
    (message: string, title?: string) => addToast({ type: 'success', message, title }),
    [addToast]
  );

  const error = useCallback(
    (message: string, title?: string) => addToast({ type: 'error', message, title, duration: 6000 }),
    [addToast]
  );

  const warning = useCallback(
    (message: string, title?: string) => addToast({ type: 'warning', message, title }),
    [addToast]
  );

  const info = useCallback(
    (message: string, title?: string) => addToast({ type: 'info', message, title }),
    [addToast]
  );

  const value: ToastContextValue = {
    toasts,
    addToast,
    removeToast,
    success,
    error,
    warning,
    info,
  };

  return (
    <ToastContext.Provider value={value}>
      {children}
      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </ToastContext.Provider>
  );
}

// Visual config per toast type
const toastConfig: Record<
  ToastType,
  { icon: typeof CheckCircle; bg: string; border: string; text: string; iconColor: string }
> = {
  success: {
    icon: CheckCircle,
    bg: 'bg-emerald-500/15',
    border: 'border-emerald-500/30',
    text: 'text-emerald-100',
    iconColor: 'text-emerald-400',
  },
  error: {
    icon: XCircle,
    bg: 'bg-red-500/15',
    border: 'border-red-500/30',
    text: 'text-red-100',
    iconColor: 'text-red-400',
  },
  warning: {
    icon: AlertTriangle,
    bg: 'bg-amber-500/15',
    border: 'border-amber-500/30',
    text: 'text-amber-100',
    iconColor: 'text-amber-400',
  },
  info: {
    icon: AlertCircle,
    bg: 'bg-blue-500/15',
    border: 'border-blue-500/30',
    text: 'text-blue-100',
    iconColor: 'text-blue-400',
  },
};

interface ToastContainerProps {
  toasts: ToastItem[];
  onRemove: (id: string) => void;
}

function ToastContainer({ toasts, onRemove }: ToastContainerProps) {
  if (toasts.length === 0) return null;

  return (
    <div
      className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm"
      role="region"
      aria-label="Notifications"
      data-testid="toast-container"
    >
      {toasts.map((toast) => (
        <ToastNotification key={toast.id} toast={toast} onRemove={onRemove} />
      ))}
    </div>
  );
}

interface ToastNotificationProps {
  toast: ToastItem;
  onRemove: (id: string) => void;
}

function ToastNotification({ toast, onRemove }: ToastNotificationProps) {
  const config = toastConfig[toast.type];
  const Icon = config.icon;

  // Auto-dismiss timer
  useEffect(() => {
    if (toast.duration && toast.duration > 0) {
      const timer = setTimeout(() => onRemove(toast.id), toast.duration);
      return () => clearTimeout(timer);
    }
  }, [toast.id, toast.duration, onRemove]);

  return (
    <div
      role="alert"
      aria-live="polite"
      className={cn(
        'rounded-xl border px-4 py-3 shadow-lg backdrop-blur-sm',
        'animate-in slide-in-from-right-full fade-in duration-200',
        'flex items-start gap-3 min-w-[280px]',
        config.bg,
        config.border
      )}
      data-testid={`toast-${toast.id}`}
    >
      <Icon className={cn('h-5 w-5 flex-shrink-0 mt-0.5', config.iconColor)} aria-hidden="true" />

      <div className="flex-1 min-w-0">
        {toast.title && (
          <p className={cn('font-medium text-sm mb-0.5', config.text)}>{toast.title}</p>
        )}
        <p className={cn('text-sm', config.text)}>{toast.message}</p>
      </div>

      <button
        type="button"
        onClick={() => onRemove(toast.id)}
        className={cn(
          'p-1 rounded hover:bg-white/10 transition-colors flex-shrink-0',
          config.text
        )}
        aria-label="Dismiss notification"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}
