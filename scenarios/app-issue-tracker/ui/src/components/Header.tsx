import { useMemo } from 'react';
import { Bug, ChevronDown, GaugeCircle, Plus, ToggleLeft, ToggleRight } from 'lucide-react';
import { ActiveAgentOption, ProcessorSettings } from '../data/sampleData';

interface HeaderProps {
  processor: ProcessorSettings;
  agents: ActiveAgentOption[];
  selectedAgentId: string;
  onToggleActive: () => void;
  onCreateIssue: () => void;
  onSelectAgent: (agentId: string) => void;
}

export function Header({
  processor,
  agents,
  selectedAgentId,
  onToggleActive,
  onCreateIssue,
  onSelectAgent,
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
        <div className="active-agent-selector">
          <GaugeCircle size={18} />
          <select
            aria-label="Active agents"
            value={selectedAgentId}
            onChange={(event) => onSelectAgent(event.target.value)}
          >
            {agents.map((agent) => (
              <option key={agent.id} value={agent.id}>
                {agent.label}
              </option>
            ))}
          </select>
          <ChevronDown size={16} />
        </div>
        <button className="primary-action" onClick={onCreateIssue}>
          <Plus size={18} />
          <span>New Issue</span>
        </button>
      </div>
    </header>
  );
}
