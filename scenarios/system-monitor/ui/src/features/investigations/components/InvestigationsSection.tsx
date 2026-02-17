import { useNavigate } from 'react-router-dom';
import { AlertTriangle, Shield, Bot, Play, MessageCircle, ChevronUp, ChevronDown, Square, Clock } from 'lucide-react';
import { InvestigationsPanel } from './InvestigationsPanel';
import { InvestigationScriptsPanel } from './InvestigationScriptsPanel';
import { AutomaticTriggersSection } from './AutomaticTriggersSection';
import { CollapsibleSection } from '../../../shared/components/CollapsibleSection';
import { useInvestigationsSectionState } from '../hooks/useInvestigationsSectionState';
import type { Investigation, InvestigationScript, InvestigationAgentState } from '../../../types';

interface InvestigationsSectionProps {
  investigations: Investigation[];
  onOpenScriptEditor: (script?: InvestigationScript, content?: string, mode?: 'create' | 'edit' | 'view') => void;
  onSpawnAgent: (options: { autoFix: boolean; note?: string }) => Promise<unknown>;
  agents: InvestigationAgentState[];
  isSpawningAgent: boolean;
  spawnAgentError?: string | null;
}

export const InvestigationsSection = ({
  investigations,
  onOpenScriptEditor,
  onSpawnAgent,
  agents,
  isSpawningAgent,
  spawnAgentError
}: InvestigationsSectionProps) => {
  const navigate = useNavigate();
  const {
    reportsSearch,
    setReportsSearch,
    scriptsSearch,
    setScriptsSearch,
    filteredInvestigations,
    autoFixEnabled,
    setAutoFixEnabled,
    agentNote,
    setAgentNote,
    showNoteField,
    setShowNoteField,
    combinedSpawnError,
    activeAgentSummary,
    summaryAccentColor,
    handleSpawnAgent,
  } = useInvestigationsSectionState({
    investigations,
    agents,
    spawnAgentError,
    onSpawnAgent,
  });

  const handleUpdateTrigger = (triggerId: string, config: unknown) => {
    // TODO: Implement API call to update trigger configuration
    console.log('Updating trigger:', triggerId, config);
  };

  return (
    <div className="panel-accent">
      <div className="panel-accent-header">
        <h2 className="investigations-heading">
          <AlertTriangle size={24} style={{ color: 'var(--color-warning)' }} />
          INVESTIGATIONS
        </h2>
      </div>

      <div className="panel-accent-body">

        {/* Agent Spawn Card */}
        <div className="agent-spawn-card">
          <div className="flex-col-gap-lg">
            <div className="agent-spawn-layout">
              <Bot
                size={48}
                className="agent-spawn-icon"
                style={{ color: 'var(--color-success)' }}
              />

              <div className="flex-col-gap-md" style={{ flex: 1, minWidth: '260px' }}>
                <div>
                  <h3 className="agent-spawn-title">
                    System Anomaly Investigation Agent
                  </h3>

                  <p className="agent-spawn-description">
                    Launch an autonomous agent to analyze real-time metrics, investigate anomalies, and propose or execute remediation steps. Results and live status now surface in the header for quick access.
                  </p>
                </div>

                <div className="flex-col-gap-sm">
                  <div className="icon-text text-dim-xs">
                    <Square size={10} />
                    <span>Active agents are managed from the header control with a full dropdown of progress and stop actions.</span>
                  </div>

                  <div className="icon-text" style={{
                    color: summaryAccentColor,
                    fontSize: 'var(--font-size-sm)',
                    fontWeight: 600,
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em'
                  }}>
                    <Clock size={14} />
                    <span>{activeAgentSummary.text}</span>
                  </div>
                </div>
              </div>
            </div>

            {combinedSpawnError && (
              <div className="error-banner text-sm">
                {combinedSpawnError}
              </div>
            )}

            <div className="agent-spawn-controls">
              <div className="flex-col-gap-sm">
                <label className="agent-checkbox-label">
                  <input
                    type="checkbox"
                    checked={autoFixEnabled}
                    onChange={(event) => setAutoFixEnabled(event.target.checked)}
                    className="agent-checkbox-input"
                  />
                  <Shield size={16} style={{ color: autoFixEnabled ? 'var(--color-success)' : 'var(--color-text-dim)' }} />
                  <span style={{
                    fontWeight: autoFixEnabled ? 'bold' : 'normal',
                    color: autoFixEnabled ? 'var(--color-text-bright)' : 'inherit'
                  }}>
                    Adaptive auto-fix & recovery
                  </span>
                </label>
                <span className="text-dim-xs">
                  Allows the agent to apply safe remediation steps automatically; otherwise it stays report-only.
                </span>
              </div>

              <button
                className="btn btn-secondary icon-text icon-text-xs text-xs"
                onClick={() => setShowNoteField(!showNoteField)}
              >
                <MessageCircle size={14} />
                NOTE
                {showNoteField ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
              </button>

              <button
                className="btn btn-primary icon-text icon-text-xs"
                onClick={handleSpawnAgent}
                disabled={isSpawningAgent}
                style={{
                  padding: 'var(--spacing-sm) var(--spacing-lg)',
                  fontSize: 'var(--font-size-sm)',
                  fontWeight: 'bold',
                  textTransform: 'uppercase',
                  letterSpacing: '0.5px'
                }}
              >
                <Play size={16} className={isSpawningAgent ? 'animate-spin' : ''} />
                {isSpawningAgent ? 'SPAWNING…' : 'SPAWN AGENT'}
              </button>
            </div>

            {showNoteField && (
              <div className="flex-col-gap-sm">
                <label
                  htmlFor="agent-note"
                  className="text-dim-xs agent-note-label"
                >
                  OPTIONAL NOTE FOR AGENT CONTEXT
                </label>
                <textarea
                  id="agent-note"
                  value={agentNote}
                  onChange={(event) => setAgentNote(event.target.value)}
                  rows={3}
                  className="agent-note-textarea"
                />
                <span className="text-dim-xs">
                  Provide guardrails or extra context. The agent attaches this note to its reasoning and audit trail.
                </span>
              </div>
            )}
          </div>
        </div>

        {/* Automatic Triggers Section */}
        <AutomaticTriggersSection onUpdateTrigger={handleUpdateTrigger} />

        {/* Reports Subsection */}
        <CollapsibleSection
          title="Reports"
          count={filteredInvestigations.length}
          search={{
            placeholder: 'Search reports...',
            value: reportsSearch,
            onChange: setReportsSearch,
          }}
        >
          <InvestigationsPanel investigations={filteredInvestigations} embedded={true} />
        </CollapsibleSection>

        {/* Scripts Subsection */}
        <CollapsibleSection
          title="Scripts"
          count="Tools"
          search={{
            placeholder: 'Search scripts...',
            value: scriptsSearch,
            onChange: setScriptsSearch,
          }}
        >
          <div style={{ marginTop: 'var(--spacing-md)' }}>
            <InvestigationScriptsPanel
              onOpenScriptEditor={onOpenScriptEditor}
              embedded={true}
              searchFilter={scriptsSearch}
              maxVisible={4}
              onShowAll={() => navigate('/scripts')}
            />
          </div>
        </CollapsibleSection>
      </div>
    </div>
  );
};
