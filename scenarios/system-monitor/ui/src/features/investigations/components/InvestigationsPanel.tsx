import type { Investigation } from '../../../types';
import { InvestigationStatus } from '../../../types';
import { LoadingSkeleton } from '../../../shared/components/LoadingSkeleton';
import { apiFetch } from '../../../shared/api/apiFetch';
import { useToast } from '../../../shared/components/ToastProvider';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import { str, bool } from '../../../shared/utils/typeGuards';
import { getRiskLevelColor } from '../../../shared/utils/colors';

interface InvestigationsPanelProps {
  investigations: Investigation[];
  embedded?: boolean;
}

export const InvestigationsPanel = ({ investigations, embedded = false }: InvestigationsPanelProps) => {
  const { showApiError } = useToast();

  const triggerInvestigation = async () => {
    try {
      await apiFetch('/investigations/trigger', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        }
      });
    } catch (error) {
      showApiError(error);
    }
  };

  const renderInvestigationCard = (investigation: Investigation, options?: { compact?: boolean }) => {
    const compact = options?.compact ?? false;
    const details = (investigation.details ?? {}) as Record<string, unknown>;
    const formattedStart = investigation.startTime
      ? timestampDate(investigation.startTime).toLocaleString()
      : 'Unknown';
    const operationMode = str(details['operation_mode']) ?? 'report-only';
    const riskLevel = str(details['risk_level']);
    const agentModel = str(details['agent_model']);
    const agentResource = str(details['agent_resource']);
    const userNote = str(details['user_note']);
    const autoFix = bool(details['auto_fix']) ?? false;
    const progress = typeof investigation.progress === 'number' ? investigation.progress : undefined;
    const confidenceScore = typeof investigation.confidenceScore === 'number' ? investigation.confidenceScore : undefined;

    const riskColor = getRiskLevelColor(riskLevel);

    return (
      <div
        key={investigation.id}
        className="investigation-item hover-bg-dark"
        style={{
          padding: 'var(--spacing-md)',
          borderBottom: '1px solid var(--color-primary)',
          background: 'var(--overlay-light)',
          transition: 'background 0.2s'
        }}
      >
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 'var(--spacing-xs)'
        }}>
          <span style={{ color: 'var(--color-text-heading)', fontWeight: 'bold' }}>
            Investigation {investigation.id}
          </span>
          <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
            {autoFix && (
              <span style={{
                background: 'var(--color-primary-muted)',
                border: '1px solid var(--color-success)',
                color: 'var(--color-success)',
                padding: '2px 6px',
                borderRadius: '999px',
                fontSize: 'var(--text-xs)'
              }}>
                Auto-Fix
              </span>
            )}
            <span style={{
              color: investigation.status === InvestigationStatus.COMPLETED
                ? 'var(--color-success)'
                : investigation.status === InvestigationStatus.IN_PROGRESS
                ? 'var(--color-warning)'
                : investigation.status === InvestigationStatus.FAILED
                ? 'var(--color-error)'
                : 'var(--color-text-secondary)',
              fontSize: 'var(--text-sm)',
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
          <span style={{ color: 'var(--color-text-secondary)', fontSize: 'var(--text-sm)' }}>
            Started: {formattedStart}
          </span>
          <span style={{ color: 'var(--color-text-secondary)', fontSize: 'var(--text-sm)' }}>
            Mode: {operationMode}
          </span>
          {agentModel && (
            <span style={{ color: 'var(--color-text-secondary)', fontSize: 'var(--text-sm)' }}>
              Model: {agentModel}
            </span>
          )}
          {agentResource && (
            <span style={{ color: 'var(--color-text-secondary)', fontSize: 'var(--text-sm)' }}>
              Resource: {agentResource}
            </span>
          )}
          {riskLevel && (
            <span style={{ color: riskColor, fontSize: 'var(--text-sm)', fontWeight: 600 }}>
              Risk: {riskLevel}
            </span>
          )}
        </div>

        {typeof progress === 'number' && !compact && (
          <div style={{ marginBottom: 'var(--spacing-sm)' }}>
            <div style={{
              height: '6px',
              borderRadius: '999px',
              background: 'var(--color-primary-muted)',
              overflow: 'hidden'
            }}>
              <div style={{
                width: `${Math.max(0, Math.min(100, progress))}%`,
                height: '6px',
                background: 'var(--color-success)',
                transition: 'width var(--transition-normal)'
              }} />
            </div>
            <span style={{ color: 'var(--color-text-secondary)', fontSize: 'var(--text-xs)' }}>
              Progress: {Math.round(progress)}%
            </span>
          </div>
        )}

        {investigation.findings && (
          <div
            style={{
              color: 'var(--color-text)',
              fontSize: 'var(--text-sm)',
              marginBottom: 'var(--spacing-sm)'
            }}
          >
            {investigation.findings}
          </div>
        )}

        {userNote && (
          <div style={{
            color: 'var(--color-text-secondary)',
            fontSize: 'var(--text-xs)',
            fontStyle: 'italic',
            marginBottom: 'var(--spacing-sm)'
          }}>
            User Note: {userNote}
          </div>
        )}

        {typeof confidenceScore === 'number' && !Number.isNaN(confidenceScore) && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
            <span style={{ color: 'var(--color-text-secondary)', fontSize: 'var(--text-sm)' }}>
              Confidence:
            </span>
            <div style={{
              width: '100px',
              height: '4px',
              background: 'var(--color-primary-muted)',
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
            <span style={{ color: 'var(--color-primary)', fontSize: 'var(--text-sm)' }}>
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
        <h2 className="section-heading">
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
