import { X } from 'lucide-react';

export type SnackbarTone = 'info' | 'success' | 'error';

interface SnackbarProps {
  message: string;
  tone: SnackbarTone;
  onClose: () => void;
}

export function Snackbar({ message, tone, onClose }: SnackbarProps) {
  const role = tone === 'error' ? 'alert' : 'status';
  const live = tone === 'error' ? 'assertive' : 'polite';

  return (
    <div className="snackbar-container">
      <div className={`snackbar snackbar--${tone}`} role={role} aria-live={live}>
        <span className="snackbar-message">{message}</span>
        <button
          type="button"
          className="snackbar-close"
          onClick={onClose}
          aria-label="Dismiss notification"
        >
          <X size={14} />
        </button>
      </div>
    </div>
  );
}
