import { AlertCircle, CheckCircle2, Info, Loader2, TriangleAlert, X } from 'lucide-react';
import type { SnackDescriptor } from '@/notifications/snackBus';
import type { SnackState } from '@/notifications/SnackStackProvider';
import './snack-stack.css';

export type SnackStackAnchor = 'top-right' | 'top-left' | 'bottom-right' | 'bottom-left';

interface SnackStackProps {
  snacks: SnackState[];
  anchor?: SnackStackAnchor;
  onDismiss: (id: string) => void;
}

const ICONS: Record<SnackDescriptor['variant'], JSX.Element> = {
  info: <Info size={16} strokeWidth={2.4} />,
  success: <CheckCircle2 size={16} strokeWidth={2.4} />,
  warning: <TriangleAlert size={16} strokeWidth={2.4} />,
  error: <AlertCircle size={16} strokeWidth={2.4} />,
  loading: <Loader2 size={16} strokeWidth={2.4} className="snack-icon--spin" />,
};

const ARIA_ROLE: Record<SnackDescriptor['variant'], { role: 'status' | 'alert'; live: 'polite' | 'assertive' }> = {
  info: { role: 'status', live: 'polite' },
  success: { role: 'status', live: 'polite' },
  warning: { role: 'status', live: 'assertive' },
  error: { role: 'alert', live: 'assertive' },
  loading: { role: 'status', live: 'polite' },
};

export function SnackStack({ snacks, anchor = 'top-right', onDismiss }: SnackStackProps) {
  if (snacks.length === 0) {
    return null;
  }

  return (
    <div className={`snack-stack snack-stack--${anchor}`} aria-live="polite" aria-relevant="additions text">
      {snacks.map((snack) => {
        const { role, live } = ARIA_ROLE[snack.variant];
        return (
          <div key={snack.id} className={`snack-card snack-card--${snack.variant}`} role={role} aria-live={live}>
            <div className="snack-card__icon" aria-hidden="true">
              {ICONS[snack.variant]}
            </div>
            <div className="snack-card__body">
              {snack.title && <div className="snack-card__title">{snack.title}</div>}
              <div className="snack-card__message">{snack.message}</div>
            </div>
            {snack.action && (
              <button
                type="button"
                className="snack-card__action"
                onClick={() => {
                  snack.action?.handler();
                  if (snack.action?.dismissOnAction !== false) {
                    onDismiss(snack.id);
                  }
                }}
              >
                {snack.action.label}
              </button>
            )}
            {snack.dismissible && (
              <button
                type="button"
                className="snack-card__close"
                onClick={() => onDismiss(snack.id)}
                aria-label="Dismiss notification"
              >
                <X size={14} strokeWidth={2.4} />
              </button>
            )}
          </div>
        );
      })}
    </div>
  );
}
