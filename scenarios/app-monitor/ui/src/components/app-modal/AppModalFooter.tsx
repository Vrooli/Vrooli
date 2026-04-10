import clsx from 'clsx';
import { Play, Square, RotateCcw, ScrollText, ExternalLink } from 'lucide-react';

interface AppModalFooterProps {
  isRunning: boolean;
  isStopped: boolean;
  actionLoading: string | null;
  currentUrl: string | null;
  onAction: (action: 'start' | 'stop' | 'restart') => void;
  onViewLogs: () => void;
  onOpenPreview: () => void;
}

/** Action bar with start/stop/restart, logs, and open-preview buttons. */
export default function AppModalFooter({
  isRunning,
  isStopped,
  actionLoading,
  currentUrl,
  onAction,
  onViewLogs,
  onOpenPreview,
}: AppModalFooterProps) {
  return (
    <div className="modal-footer">
      {!isRunning && (
        <button
          type="button"
          className={clsx('modal-btn', 'modal-btn--accent', { loading: actionLoading === 'start' })}
          onClick={() => onAction('start')}
          disabled={actionLoading !== null}
        >
          <Play aria-hidden size={16} />
          {actionLoading === 'start' ? 'Starting\u2026' : 'Start'}
        </button>
      )}
      {!isStopped && (
        <button
          type="button"
          className={clsx('modal-btn', 'modal-btn--danger', { loading: actionLoading === 'stop' })}
          onClick={() => onAction('stop')}
          disabled={actionLoading !== null}
        >
          <Square aria-hidden size={16} />
          {actionLoading === 'stop' ? 'Stopping\u2026' : 'Stop'}
        </button>
      )}
      <button
        type="button"
        className={clsx('modal-btn', 'modal-btn--neutral', { loading: actionLoading === 'restart' })}
        onClick={() => onAction('restart')}
        disabled={actionLoading !== null}
      >
        <RotateCcw aria-hidden size={16} />
        {actionLoading === 'restart' ? 'Restarting\u2026' : 'Restart'}
      </button>
      <button
        type="button"
        className={clsx('modal-btn', 'modal-btn--ghost')}
        onClick={onViewLogs}
        aria-label="View application logs"
      >
        <ScrollText aria-hidden size={16} />
        Logs
      </button>
      {currentUrl && (
        <button
          type="button"
          className={clsx('modal-btn', 'modal-btn--ghost')}
          onClick={onOpenPreview}
          aria-label="Open application preview"
        >
          <ExternalLink aria-hidden size={16} />
          Open
        </button>
      )}
    </div>
  );
}
