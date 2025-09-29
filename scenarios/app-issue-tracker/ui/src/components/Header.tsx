import { useMemo } from 'react';
import { Bug, Plus, ToggleLeft, ToggleRight } from 'lucide-react';
import { ProcessorSettings } from '../data/sampleData';

interface HeaderProps {
  processor: ProcessorSettings;
  onToggleActive: () => void;
  onCreateIssue: () => void;
}

export function Header({
  processor,
  onToggleActive,
  onCreateIssue,
}: HeaderProps) {
  const activeLabel = useMemo(() => (processor.active ? 'Active' : 'Paused'), [processor.active]);

  return (
    <header className="app-header">
      <div className="header-left">
        <div className="scenario-badge">
          <Bug size={20} />
        </div>
        <div>
          <h1 className="scenario-title">App Issue Tracker</h1>
          <p className="scenario-subtitle">AI-assisted triage for Vrooli scenarios</p>
        </div>
      </div>
      <div className="header-right">
        <button className={`processor-toggle ${processor.active ? 'on' : 'off'}`} onClick={onToggleActive}>
          {processor.active ? <ToggleRight size={18} /> : <ToggleLeft size={18} />}
          <span>{activeLabel}</span>
        </button>
        <button className="primary-action" onClick={onCreateIssue}>
          <Plus size={18} />
          <span>New Issue</span>
        </button>
      </div>
    </header>
  );
}
