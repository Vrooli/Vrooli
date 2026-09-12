import { X, RotateCcw } from 'lucide-react';
import { useToast, type ToastSeverity } from './ToastProvider';

const severityColors: Record<ToastSeverity, { bg: string; border: string; text: string }> = {
  info: { bg: 'var(--color-primary-muted)', border: 'var(--color-primary)', text: 'var(--color-primary)' },
  success: { bg: 'var(--color-primary-muted)', border: 'var(--color-success)', text: 'var(--color-success)' },
  warning: { bg: 'var(--color-warning-muted)', border: 'var(--color-warning)', text: 'var(--color-warning)' },
  error: { bg: 'var(--color-primary-muted)', border: 'var(--color-error)', text: 'var(--color-error)' },
};

export function ToastContainer() {
  const { toasts, dismissToast } = useToast();

  if (toasts.length === 0) return null;

  return (
    <div data-sm-style="sm-style-81adfbc78d">
      {toasts.map(toast => {
        const colors = severityColors[toast.severity];
        return (
          <div
            key={toast.id}
            style={{
              background: colors.bg,
              border: `1px solid ${colors.border}`,
              borderRadius: 'var(--radius-md)',
              padding: 'var(--spacing-md)',
              display: 'flex',
              alignItems: 'flex-start',
              gap: 'var(--spacing-sm)',
              animation: 'fadeIn 0.2s ease-out',
              backdropFilter: 'blur(8px)',
            }}
          >
            <span style={{ color: colors.text, flex: 1, fontSize: 'var(--text-sm)' }}>
              {toast.message}
            </span>
            <div data-sm-style="sm-style-3fd8adeb01">
              {toast.retryFn && (
                <button
                  onClick={() => { dismissToast(toast.id); toast.retryFn?.(); }}
                  data-sm-style="sm-style-f13784862f"
                  title="Retry"
                >
                  <RotateCcw size={14} />
                </button>
              )}
              <button
                onClick={() => { dismissToast(toast.id); }}
                data-sm-style="sm-style-847600675b"
                title="Dismiss"
              >
                <X size={14} />
              </button>
            </div>
          </div>
        );
      })}
    </div>
  );
}
