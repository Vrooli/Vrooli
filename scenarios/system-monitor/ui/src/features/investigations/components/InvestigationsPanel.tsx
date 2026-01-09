import type { Investigation } from '../../../types';
import { LoadingSkeleton } from '../../../shared/components/LoadingSkeleton';
import { buildApiUrl } from '../../../shared/api/apiBase';

interface InvestigationsPanelProps {
  investigations: Investigation[];
  embedded?: boolean;
}

export const InvestigationsPanel = ({ investigations, embedded = false }: InvestigationsPanelProps) => {
  const triggerInvestigation = async () => {
    try {
      const response = await fetch(buildApiUrl('/investigations/trigger'), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        }
      });
      
      if (response.ok) {
        // TODO: Show success message or refresh investigations
        console.log('Investigation triggered successfully');
      }
    } catch (error) {
      console.error('Failed to trigger investigation:', error);
    }
  };

  const renderInvestigationCard = (investigation: Investigation, options?: { compact?: boolean }) => {
    const compact = options?.compact ?? false;
    const details = (investigation.details ?? {}) as Record<string, unknown>;
    const startTimeSource = investigation.start_time ?? investigation.timestamp ?? details['start_time'];
    const formattedStart = typeof startTimeSource === 'string' && startTimeSource
      ? new Date(startTimeSource).toLocaleString()
      : 'Unknown';
    const operationMode = typeof details['operation_mode'] === 'string' ? details['operation_mode'] : 'report-only';
    const riskLevel = typeof details['risk_level'] === 'string' ? details['risk_level'] : undefined;
    const agentModel = typeof details['agent_model'] === 'string' ? details['agent_model'] : undefined;
    const agentResource = typeof details['agent_resource'] === 'string' ? details['agent_resource'] : undefined;
    const userNote = typeof details['user_note'] === 'string' ? details['user_note'] : undefined;
    const autoFix = Boolean(details['auto_fix']);
    const progress = typeof investigation.progress === 'number' ? investigation.progress : undefined;
    const confidenceScore = typeof investigation.confidence_score === 'number' ? investigation.confidence_score : undefined;

    const riskColor = riskLevel === 'high'
      ? 'var(--color-error)'
      : riskLevel === 'medium'
      ? 'var(--color-warning)'
      : 'var(--color-success)';

    return (
      <div
        key={investigation.id}
        className="investigation-item"
        style={{
          padding: 'var(--spacing-md)',
          borderBottom: '1px solid var(--color-accent)',
          background: 'rgba(0, 0, 0, 0.2)',
          transition: 'background 0.2s'
        }}
        onMouseEnter={(e) => { e.currentTarget.style.background = 'rgba(0, 0, 0, 0.4)'; }}
        onMouseLeave={(e) => { e.currentTarget.style.background = 'rgba(0, 0, 0, 0.2)'; }}
      >
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 'var(--spacing-xs)'
        }}>
          <span style={{ color: 'var(--color-text-bright)', fontWeight: 'bold' }}>
            Investigation {investigation.id}
          </span>
          <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
            {autoFix && (
              <span style={{
                background: 'var(--alpha-accent-15)',
                border: '1px solid var(--color-success)',
                color: 'var(--color-success)',
                padding: '2px 6px',
                borderRadius: '999px',
                fontSize: 'var(--font-size-xs)'
              }}>
                Auto-Fix
              </span>
            )}
            <span style={{
              color: investigation.status === 'completed'
                ? 'var(--color-success)'
                : investigation.status === 'in_progress'
                ? 'var(--color-warning)'
                : investigation.status === 'failed'
                ? 'var(--color-error)'
                : 'var(--color-text-dim)',
              fontSize: 'var(--font-size-sm)',
              textTransform: 'uppercase'
            }}>
              {investigation.status}
            </span>
          </div>
        </div>

        <div style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: 'var(--spacing-md)',
          marginBottom: 'var(--spacing-sm)'
        }}>
          <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
            Started: {formattedStart}
          </span>
          <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
            Mode: {operationMode}
          </span>
          {agentModel && (
            <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
              Model: {agentModel}
            </span>
          )}
          {agentResource && (
            <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
              Resource: {agentResource}
            </span>
          )}
          {riskLevel && (
            <span style={{ color: riskColor, fontSize: 'var(--font-size-sm)', fontWeight: 600 }}>
              Risk: {riskLevel}
            </span>
          )}
        </div>

        {typeof progress === 'number' && !compact && (
          <div style={{ marginBottom: 'var(--spacing-sm)' }}>
            <div style={{
              height: '6px',
              borderRadius: '999px',
              background: 'var(--alpha-accent-15)',
              overflow: 'hidden'
            }}>
              <div style={{
                width: `${Math.max(0, Math.min(100, progress))}%`,
                height: '6px',
                background: 'var(--color-success)',
                transition: 'width var(--transition-normal)'
              }} />
            </div>
            <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)' }}>
              Progress: {Math.round(progress)}%
            </span>
          </div>
        )}

        {investigation.findings && (
          <div
            style={{
              color: 'var(--color-text)',
              fontSize: 'var(--font-size-sm)',
              marginBottom: 'var(--spacing-sm)'
            }}
          >
            {investigation.findings}
          </div>
        )}

        {userNote && (
          <div style={{
            color: 'var(--color-text-dim)',
            fontSize: 'var(--font-size-xs)',
            fontStyle: 'italic',
            marginBottom: 'var(--spacing-sm)'
          }}>
            User Note: {userNote}
          </div>
        )}

        {typeof confidenceScore === 'number' && !Number.isNaN(confidenceScore) && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
            <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
              Confidence:
            </span>
            <div style={{
              width: '100px',
              height: '4px',
              background: 'var(--alpha-accent-20)',
              borderRadius: '2px',
              overflow: 'hidden'
            }}>
              <div style={{
                width: `${Math.max(0, Math.min(10, confidenceScore)) * 10}%`,
                height: '100%',
                background: confidenceScore >= 8
                  ? 'var(--color-success)'
                  : confidenceScore >= 6
                  ? 'var(--color-warning)'
                  : 'var(--color-error)',
                transition: 'width var(--transition-normal)'
              }} />
            </div>
            <span style={{ color: 'var(--color-accent)', fontSize: 'var(--font-size-sm)' }}>
              {confidenceScore}/10
            </span>
          </div>
        )}
      </div>
    );
  };

  // If embedded, render without header
  if (embedded) {
    return (
      <div className="investigation-list">
        {investigations.length === 0 ? (
          <LoadingSkeleton variant="list" count={3} />
        ) : (
          investigations.map(investigation => renderInvestigationCard(investigation, { compact: true }))
        )}
      </div>
    );
  }

  return (
    <section className="investigations-panel card">
      <div className="panel-header" style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 'var(--spacing-md)'
      }}>
        <h2 style={{ margin: 0, color: 'var(--color-text-bright)' }}>
          RECENT INVESTIGATIONS
        </h2>
        <button 
          className="btn btn-action"
          onClick={triggerInvestigation}
        >
          RUN ANOMALY CHECK
        </button>
      </div>
      
      <div className="investigation-list">
        {investigations.length === 0 ? (
          <LoadingSkeleton variant="list" count={3} />
        ) : (
          investigations.map(investigation => renderInvestigationCard(investigation))
        )}
      </div>
    </section>
  );
};
