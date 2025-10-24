import { X } from 'lucide-react';
import type {
  AgentSettings,
  DisplaySettings,
  ProcessorSettings,
} from '../data/sampleData';
import { Modal } from './Modal';
import { SettingsPage } from '../pages/Settings';

export interface SettingsDialogProps {
  apiBaseUrl: string;
  processor: ProcessorSettings;
  agent: AgentSettings;
  display: DisplaySettings;
  onProcessorChange: (settings: ProcessorSettings) => void;
  onAgentChange: (settings: AgentSettings) => void;
  onDisplayChange: (settings: DisplaySettings) => void;
  onClose: () => void;
  issuesProcessed?: number;
  issuesRemaining?: number | string;
}

export function SettingsDialog({
  apiBaseUrl,
  processor,
  agent,
  display,
  onProcessorChange,
  onAgentChange,
  onDisplayChange,
  onClose,
  issuesProcessed,
  issuesRemaining,
}: SettingsDialogProps) {
  return (
    <Modal
      onClose={onClose}
      labelledBy="settings-dialog-title"
      panelClassName="modal-panel--wide"
      size="xl"
    >
      <div className="modal-header">
        <div>
          <p className="modal-eyebrow">Configuration</p>
          <h2 id="settings-dialog-title" className="modal-title">
            Settings
          </h2>
        </div>
        <button className="modal-close" type="button" aria-label="Close settings" onClick={onClose}>
          <X size={18} />
        </button>
      </div>

      <div className="modal-body modal-body--settings">
        <SettingsPage
          apiBaseUrl={apiBaseUrl}
          processor={processor}
          agent={agent}
          display={display}
          onProcessorChange={onProcessorChange}
          onAgentChange={onAgentChange}
          onDisplayChange={onDisplayChange}
          issuesProcessed={issuesProcessed}
          issuesRemaining={issuesRemaining}
        />
      </div>
    </Modal>
  );
}
