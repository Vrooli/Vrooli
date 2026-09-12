import { X } from 'lucide-react';

interface ProcessAlertItemProps {
  pid: number;
  name: string;
  variant: 'zombie' | 'high_thread' | 'leak_candidate';
  /** Extra info displayed before the kill button (e.g. thread count, memory) */
  detail?: string;
  onKill: (pid: number, name: string, variant: 'zombie' | 'high_thread' | 'leak_candidate') => void;
}

const variantClass: Record<ProcessAlertItemProps['variant'], string> = {
  zombie: 'pool-item-zombie',
  high_thread: 'pool-item-high-thread',
  leak_candidate: 'pool-item-leak',
};

const killButtonClass: Record<ProcessAlertItemProps['variant'], string> = {
  zombie: 'btn-kill-error',
  high_thread: 'btn-kill-warning',
  leak_candidate: 'btn-kill-warning',
};

export const ProcessAlertItem = ({ pid, name, variant, detail, onKill }: ProcessAlertItemProps) => (
  <div className={`pool-item ${variantClass[variant]}`}>
    <span>{name} (PID: {pid})</span>
    <div data-sm-style="sm-style-67bc078b69">
      {detail && (
        <span data-sm-style="sm-style-38c5f4e767">
          {detail}
        </span>
      )}
      <button
        className={`btn-kill ${killButtonClass[variant]}`}
        onClick={() => { onKill(pid, name, variant); }}
        title={`Kill ${variant.replace('_', ' ')} process`}
      >
        <X size={16} />
      </button>
    </div>
  </div>
);
