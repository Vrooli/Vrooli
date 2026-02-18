import { FileText } from 'lucide-react';
import type { Investigation } from '../../../types';
import { InvestigationStatus } from '../../../types';
import { EmptyState } from '../../../shared/components/EmptyState';
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

  const getStatusBadgeClass = (status: InvestigationStatus | string) => {
    if (status === InvestigationStatus.COMPLETED) return 'badge-success';
    if (status === InvestigationStatus.IN_PROGRESS) return 'badge-warning';
    if (status === InvestigationStatus.FAILED) return 'badge-error';
    return '';
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
      >
        <div className="investigation-header">
          <span className="text-sm font-bold text-bright">
            Investigation {investigation.id}
          </span>
          <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
            {autoFix && (
              <span className="badge badge-success">Auto-Fix</span>
            )}
            <span className={`badge ${getStatusBadgeClass(investigation.status)}`}>
              {investigation.status}
            </span>
          </div>
        </div>

        <div className="investigation-meta">
          <span className="text-xs text-muted">Started: {formattedStart}</span>
          <span className="text-xs text-muted">Mode: {operationMode}</span>
          {agentModel && <span className="text-xs text-muted">Model: {agentModel}</span>}
          {agentResource && <span className="text-xs text-muted">Resource: {agentResource}</span>}
          {riskLevel && (
            <span className="text-xs font-semibold" style={{ color: riskColor }}>
              Risk: {riskLevel}
            </span>
          )}
        </div>

        {typeof progress === 'number' && !compact && (
          <div style={{ marginBottom: 'var(--spacing-sm)' }}>
            <div className="progress-bar">
              <div
                className="progress-fill"
                style={{ width: `${Math.max(0, Math.min(100, progress))}%` }}
              />
            </div>
            <span className="text-xs text-muted">Progress: {Math.round(progress)}%</span>
          </div>
        )}

        {investigation.findings && (
          <div className="text-sm mb-sm">{investigation.findings}</div>
        )}

        {userNote && (
          <div className="text-xs text-muted" style={{ fontStyle: 'italic', marginBottom: 'var(--spacing-sm)' }}>
            User Note: {userNote}
          </div>
        )}

        {typeof confidenceScore === 'number' && !Number.isNaN(confidenceScore) && (
          <div className="investigation-confidence">
            <span className="text-xs text-muted">Confidence:</span>
            <div className="investigation-confidence-bar">
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
            <span className="text-xs text-accent">{confidenceScore}/10</span>
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
          <EmptyState
            icon={FileText}
            message="No investigations yet"
            description="Run an anomaly check to generate reports."
          />
        ) : (
          investigations.map(investigation => renderInvestigationCard(investigation, { compact: true }))
        )}
      </div>
    );
  }

  return (
    <section className="investigations-panel card">
      <div className="flex-row-between mb-md">
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
          <EmptyState
            icon={FileText}
            message="No investigations yet"
            description="Run an anomaly check to generate reports."
          />
        ) : (
          investigations.map(investigation => renderInvestigationCard(investigation))
        )}
      </div>
    </section>
  );
};
