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
    <div style={{
      position: 'fixed',
      bottom: 'var(--spacing-lg)',
      right: 'var(--spacing-lg)',
      zIndex: 9999,
      display: 'flex',
      flexDirection: 'column',
      gap: 'var(--spacing-sm)',
      maxWidth: '420px',
      width: '100%',
    }}>
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
            <div style={{ display: 'flex', gap: 'var(--spacing-xs)', flexShrink: 0 }}>
              {toast.retryFn && (
                <button
                  onClick={() => { dismissToast(toast.id); toast.retryFn?.(); }}
                  style={{
                    background: 'transparent',
                    border: 'none',
                    color: 'var(--color-primary)',
                    cursor: 'pointer',
                    padding: '2px',
                  }}
                  title="Retry"
                >
                  <RotateCcw size={14} />
                </button>
              )}
              <button
                onClick={() => dismissToast(toast.id)}
                style={{
                  background: 'transparent',
                  border: 'none',
                  color: 'var(--color-text-secondary)',
                  cursor: 'pointer',
                  padding: '2px',
                }}
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
