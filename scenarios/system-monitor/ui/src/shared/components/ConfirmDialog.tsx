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
      <div className={`icon-text dialog-variant-${variant}`}>
        <AlertTriangle size={32} />
        <h3 data-sm-style="sm-style-6d057eff06">
          {title}
        </h3>
      </div>

      <div data-sm-style="sm-style-6c952c3a20">
        {message}
      </div>

      <div data-sm-style="sm-style-8104be9250">
        <button
          className="btn btn-secondary"
          onClick={onCancel}
          data-sm-style="sm-style-817bf65bc0"
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
