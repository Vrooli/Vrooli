import { useNavigate } from 'react-router-dom';
import { AlertTriangle, Bot, Play, MessageCircle } from 'lucide-react';
import { ToggleSwitch } from '../../../shared/components/ToggleSwitch';
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

  const statusBadgeClass = activeAgentSummary.tone === 'error'
    ? 'badge-error'
    : activeAgentSummary.tone === 'success'
    ? 'badge-success'
    : activeAgentSummary.tone === 'active'
    ? 'badge-info'
    : 'badge-warning';

  return (
    <div className="panel-accent">
      <div className="panel-accent-header">
        <h2 className="investigations-heading">
          <AlertTriangle size={24} data-sm-style="sm-style-38c5f4e767" />
          INVESTIGATIONS
        </h2>
      </div>

      <div className="panel-accent-body">

        {/* Agent Spawn Card */}
        <div className="agent-spawn-card">
          <div className="agent-spawn-header">
            <div className="agent-spawn-header-left">
              <Bot size={16} data-sm-style="sm-style-eab9fc4afc" />
              <h3 className="agent-spawn-title">Investigation Agent</h3>
              <span className={`badge ${statusBadgeClass}`}>{activeAgentSummary.text}</span>
            </div>

            <div className="agent-spawn-controls">
              <label className="agent-checkbox-label">
                <ToggleSwitch
                  checked={autoFixEnabled}
                  onChange={() => { setAutoFixEnabled(!autoFixEnabled); }}
                  size="sm"
                />
                Auto-fix
              </label>

              <button
                className="btn-icon"
                onClick={() => { setShowNoteField(!showNoteField); }}
                title="Add note for agent context"
              >
                <MessageCircle size={16} />
              </button>

              <button
                className="btn btn-primary text-xs"
                onClick={() => { void handleSpawnAgent(); }}
                disabled={isSpawningAgent}
              >
                <Play size={14} className={isSpawningAgent ? 'animate-spin' : ''} />
                {isSpawningAgent ? 'Spawning...' : 'Spawn Agent'}
              </button>
            </div>
          </div>

          {combinedSpawnError && (
            <div className="error-banner text-sm" data-sm-style="sm-style-9ed1054ccd">
              {combinedSpawnError}
            </div>
          )}

          {showNoteField && (
            <div className="flex-col-gap-sm" data-sm-style="sm-style-9ed1054ccd">
              <label
                htmlFor="agent-note"
                className="text-xs text-muted"
              >
                Optional note for agent context
              </label>
              <textarea
                id="agent-note"
                value={agentNote}
                onChange={(event) => { setAgentNote(event.target.value); }}
                rows={3}
                className="agent-note-textarea"
              />
            </div>
          )}
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
          <div data-sm-style="sm-style-323fdcc1e0">
            <InvestigationScriptsPanel
              onOpenScriptEditor={onOpenScriptEditor}
              embedded={true}
              searchFilter={scriptsSearch}
              maxVisible={4}
              onShowAll={() => { void navigate('/scripts'); }}
            />
          </div>
        </CollapsibleSection>
      </div>
    </div>
  );
};
