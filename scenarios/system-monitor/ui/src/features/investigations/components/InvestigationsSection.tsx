import { useNavigate } from 'react-router-dom';
import { AlertTriangle, Bot, Play, MessageCircle } from 'lucide-react';
import { ToggleSwitch } from '../../../shared/components/ToggleSwitch';
import { InvestigationsPanel } from './InvestigationsPanel';
import { InvestigationScriptsPanel } from './InvestigationScriptsPanel';
import { CollapsibleSection } from '../../../shared/components/CollapsibleSection';
import { useInvestigationsSectionState } from '../hooks/useInvestigationsSectionState';
import type { Investigation, InvestigationScript, InvestigationAgentState } from '../../../types';
import type { UnknownSpawnOutcome } from '../hooks/useInvestigationAgents';

interface InvestigationsSectionProps {
  investigations: Investigation[];
  onOpenScriptEditor: (script?: InvestigationScript, content?: string, mode?: 'create' | 'edit' | 'view') => void;
  onSpawnAgent: (options: { autoFix: boolean; note?: string }) => Promise<unknown>;
  agents: InvestigationAgentState[];
  isSpawningAgent: boolean;
  spawnAgentError?: string | null;
  incidentNote?: string | null;
  unknownSpawnOutcome?: UnknownSpawnOutcome | null;
  onRefreshSpawnStatus?: () => void;
  onRetryUnknownSpawn?: () => void;
}

export const InvestigationsSection = ({
  investigations,
  onOpenScriptEditor,
  onSpawnAgent,
  agents,
  isSpawningAgent,
  spawnAgentError,
  incidentNote,
  unknownSpawnOutcome,
  onRefreshSpawnStatus,
  onRetryUnknownSpawn,
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
    unknownSpawnOutcome: spawnOutcome,
    activeAgentSummary,
    handleSpawnAgent,
  } = useInvestigationsSectionState({
    investigations,
    agents,
    spawnAgentError,
    onSpawnAgent,
    incidentNote,
    unknownSpawnOutcome,
  });

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
                  disabled={Boolean(spawnOutcome) || isSpawningAgent}
                />
                Auto-fix
              </label>

              <button
                className="btn-icon"
                onClick={() => { setShowNoteField(!showNoteField); }}
                title="Add note for agent context"
                disabled={Boolean(spawnOutcome) || isSpawningAgent}
              >
                <MessageCircle size={16} />
              </button>

              <button
                className="btn btn-primary text-xs"
                onClick={() => { void handleSpawnAgent(); }}
                disabled={isSpawningAgent || Boolean(spawnOutcome)}
              >
                <Play size={14} className={isSpawningAgent ? 'animate-spin' : ''} />
                {isSpawningAgent ? 'Sending request…' : spawnOutcome ? 'Request needs receipt' : 'Spawn Agent'}
              </button>
            </div>
          </div>

          {combinedSpawnError && (
            <div className="error-banner text-sm" data-sm-style="sm-style-9ed1054ccd">
              {combinedSpawnError}
            </div>
          )}

          {spawnOutcome && (
            <div className="card" role="alert" aria-label="Investigation request needs review">
              <h4 className="section-heading">Investigation request needs a receipt</h4>
              <p>The response was not received, so the request may or may not have been accepted. Saved request key: <code>{spawnOutcome.requestId}</code>.</p>
              {spawnOutcome.message && <p role="status">Latest request result: {spawnOutcome.message}</p>}
              <p>Retrying uses this same key and the original settings. The System Monitor service returns the original investigation if it already accepted the request, instead of starting a second run.</p>
              {spawnOutcome.readback === 'checking' && <p role="status">Checking current agent status…</p>}
              {spawnOutcome.readback === 'failed' && <p>Current agent status could not be read: {spawnOutcome.readbackError}</p>}
              {spawnOutcome.readback === 'ready' && (spawnOutcome.currentAgent
                ? <p>Current active agent status: {spawnOutcome.currentAgent.status || 'active'}. Its presence does not prove it is this request.</p>
                : <p>No active agent was reported at this check. This does not prove whether this request was accepted or is waiting.</p>)}
              <div className="flex-row-center gap-sm">
                <button type="button" className="btn btn-action" onClick={onRefreshSpawnStatus} disabled={isSpawningAgent || !onRefreshSpawnStatus}>
                  Refresh current agent status
                </button>
                <button type="button" className="btn btn-primary" onClick={onRetryUnknownSpawn} disabled={isSpawningAgent || !onRetryUnknownSpawn}>
                  Retry same request safely
                </button>
              </div>
            </div>
          )}

          {autoFixEnabled && <p className="text-sm text-warning" role="status">Auto-fix allows the agent to apply changes. Leave it off for a read-only investigation.</p>}

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
                disabled={Boolean(spawnOutcome) || isSpawningAgent}
              />
            </div>
          )}
        </div>

        <section className="card" aria-labelledby="automatic-investigations-heading">
          <h3 id="automatic-investigations-heading">Automatic investigations</h3>
          <p className="text-sm text-muted-foreground">
            Configure alert thresholds, notification delivery, and which sustained critical alerts
            start a native investigation in System Monitor Settings and Notification Hub.
          </p>
        </section>

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
