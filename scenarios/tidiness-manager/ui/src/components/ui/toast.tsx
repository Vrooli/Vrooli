import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from "lucide-react";
import { createContext, useContext, useState, useCallback, ReactNode } from "react";
import { cn } from "../../lib/utils";

type ToastVariant = "success" | "error" | "info" | "warning";

interface Toast {
  id: string;
  message: string;
  variant: ToastVariant;
  duration?: number;
}

interface ToastContextType {
  showToast: (message: string, variant?: ToastVariant, duration?: number) => void;
}

const ToastContext = createContext<ToastContextType | null>(null);

export const useToast = () => {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error("useToast must be used within ToastProvider");
  }
  return context;
};

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const showToast = useCallback((message: string, variant: ToastVariant = "info", duration = 5000) => {
    const id = `toast-${Date.now()}-${Math.random()}`;
    const toast: Toast = { id, message, variant, duration };

    setToasts((prev) => [...prev, toast]);

    if (duration > 0) {
      setTimeout(() => {
        setToasts((prev) => prev.filter((t) => t.id !== id));
      }, duration);
    }
  }, []);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  return (
    <ToastContext.Provider value={{ showToast }}>
      {children}
      <div
        className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-md"
        role="region"
        aria-label="Notifications"
      >
        {toasts.map((toast) => (
          <ToastItem
            key={toast.id}
            toast={toast}
            onDismiss={() => removeToast(toast.id)}
          />
        ))}
      </div>
    </ToastContext.Provider>
  );
}

function ToastItem({ toast, onDismiss }: { toast: Toast; onDismiss: () => void }) {
  const icons = {
    success: CheckCircle,
    error: AlertCircle,
    info: Info,
    warning: AlertTriangle,
  };

  const styles = {
    success: "bg-green-500/10 border-green-500/20 text-green-200",
    error: "bg-red-500/10 border-red-500/20 text-red-200",
    info: "bg-blue-500/10 border-blue-500/20 text-blue-200",
    warning: "bg-yellow-500/10 border-yellow-500/20 text-yellow-200",
  };

  const iconStyles = {
    success: "text-green-400",
    error: "text-red-400",
    info: "text-blue-400",
    warning: "text-yellow-400",
  };

  const Icon = icons[toast.variant];

  return (
    <div
      className={cn(
        "flex items-start gap-3 p-4 rounded-lg border shadow-lg animate-slide-in-right",
        styles[toast.variant]
      )}
      role="alert"
      aria-live="polite"
    >
      <Icon className={cn("h-5 w-5 flex-shrink-0 mt-0.5", iconStyles[toast.variant])} aria-hidden="true" />
      <p className="flex-1 text-sm leading-relaxed pr-2 whitespace-pre-line font-mono">{toast.message}</p>
      <button
        onClick={onDismiss}
        className="flex-shrink-0 text-slate-400 hover:text-slate-200 transition-colors"
        aria-label="Dismiss notification"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}
