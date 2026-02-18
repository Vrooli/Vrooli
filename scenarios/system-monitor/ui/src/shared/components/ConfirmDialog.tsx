import type { ReactNode } from 'react';
import { AlertTriangle } from 'lucide-react';
import { Modal } from './Modal';

interface ConfirmDialogProps {
  isOpen: boolean;
  title: string;
  message: ReactNode;
  confirmLabel?: string;
  variant?: 'danger' | 'warning';
  onConfirm: () => void;
  onCancel: () => void;
}

export const ConfirmDialog = ({
  isOpen,
  title,
  message,
  confirmLabel = 'Confirm',
  variant = 'danger',
  onConfirm,
  onCancel
}: ConfirmDialogProps) => (
  <Modal isOpen={isOpen} onClose={onCancel} ariaLabel={title}>
    <div className="confirm-dialog">
      <div className="icon-text" style={{ marginBottom: 'var(--spacing-lg)', color: variant === 'danger' ? 'var(--color-error)' : 'var(--color-warning)' }}>
        <AlertTriangle size={32} />
        <h3 style={{ margin: 0, color: 'var(--color-text-heading)', fontSize: 'var(--text-xl)' }}>
          {title}
        </h3>
      </div>

      <div style={{ marginBottom: 'var(--spacing-lg)', color: 'var(--color-text)', fontSize: 'var(--text-base)', lineHeight: '1.5' }}>
        {message}
      </div>

      <div style={{ display: 'flex', gap: 'var(--spacing-md)', justifyContent: 'flex-end' }}>
        <button
          className="btn btn-secondary"
          onClick={onCancel}
          style={{ padding: 'var(--spacing-sm) var(--spacing-lg)', fontSize: 'var(--text-sm)' }}
        >
          CANCEL
        </button>
        <button
          className="btn-terminate"
          onClick={onConfirm}
        >
          {confirmLabel}
        </button>
      </div>
    </div>
  </Modal>
);
