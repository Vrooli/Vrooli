import { X } from 'lucide-react';
import type {
  AgentSettings,
  DashboardStats,
  Issue,
  ProcessorSettings,
} from '../data/sampleData';
import { Modal } from './Modal';
import { Dashboard } from '../pages/Dashboard';

export interface MetricsDialogProps {
  stats: DashboardStats;
  issues: Issue[];
  processor: ProcessorSettings;
  agentSettings: AgentSettings;
  onClose: () => void;
  onOpenIssue: (issueId: string) => void;
}

export function MetricsDialog({
  stats,
  issues,
  processor,
  agentSettings,
  onClose,
  onOpenIssue,
}: MetricsDialogProps) {
  return (
    <Modal
      onClose={onClose}
      labelledBy="metrics-dialog-title"
      panelClassName="modal-panel--wide"
      size="xl"
    >
      <div className="modal-header">
        <div>
          <p className="modal-eyebrow">Dashboard</p>
          <h2 id="metrics-dialog-title" className="modal-title">
            Metrics
          </h2>
        </div>
        <button className="modal-close" type="button" aria-label="Close metrics" onClick={onClose}>
          <X size={18} />
        </button>
      </div>

      <div className="modal-body modal-body--dashboard">
        <Dashboard
          stats={stats}
          issues={issues}
          processor={processor}
          agentSettings={agentSettings}
          onOpenIssues={onClose}
          onOpenIssue={onOpenIssue}
          onOpenAutomationSettings={onClose}
        />
      </div>
    </Modal>
  );
}
