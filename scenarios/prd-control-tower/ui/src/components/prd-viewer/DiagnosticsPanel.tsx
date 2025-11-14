import { AlertTriangle, Loader2, RefreshCcw } from 'lucide-react'
import { DiagnosticsResults } from './DiagnosticsResults'
import { formatDate } from '../../utils/formatters'

interface DiagnosticsState {
  entityType: string
  entityName: string
  validatedAt?: string
  cacheUsed?: boolean
  violations: unknown
}

interface DiagnosticsPanelProps {
  diagnostics: DiagnosticsState | null
  loading: boolean
  error: string | null
  onRunDiagnostics: () => void
  entityLabel: string
}

export function DiagnosticsPanel({ diagnostics, loading, error, onRunDiagnostics, entityLabel }: DiagnosticsPanelProps) {
  const lastRunLabel = diagnostics?.validatedAt && !Number.isNaN(new Date(diagnostics.validatedAt).getTime())
    ? formatDate(diagnostics.validatedAt)
    : null

  return (
    <section className="diagnostics" aria-labelledby="diagnostics-heading">
      <div className="diagnostics__header">
        <div className="diagnostics__heading-group">
          <h2 id="diagnostics-heading" className="diagnostics__title">
            Scenario Diagnostics
          </h2>
          <p className="diagnostics__description">
            Run scenario-auditor to check whether {entityLabel} still aligns with platform standards. Results update in-place when new rules are added.
          </p>
        </div>
        <div className="diagnostics__actions">
          <button
            type="button"
            className="diagnostics__run-button"
            onClick={onRunDiagnostics}
            disabled={loading}
          >
            {loading ? <Loader2 size={16} className="icon-spin" aria-hidden="true" /> : <RefreshCcw size={16} aria-hidden="true" />}
            <span>{diagnostics ? 'Re-run Diagnostics' : 'Run Diagnostics'}</span>
          </button>
          {lastRunLabel && (
            <span className="diagnostics__meta">
              Last run {lastRunLabel}
              {diagnostics?.cacheUsed ? ' · served from cache' : ''}
            </span>
          )}
        </div>
      </div>

      {error && (
        <div className="diagnostics-error" role="alert">
          <AlertTriangle size={20} aria-hidden="true" />
          <div>
            <p className="diagnostics-error__title">Unable to run diagnostics</p>
            <p>{error}</p>
          </div>
        </div>
      )}

      {loading ? (
        <div className="diagnostics-loading">
          <Loader2 size={20} className="icon-spin" aria-hidden="true" />
          <span>Running scenario-auditor…</span>
        </div>
      ) : diagnostics ? (
        <DiagnosticsResults data={diagnostics.violations} />
      ) : (
        <p className="diagnostics-empty" aria-live="polite">
          Run the auditor to generate a diagnostics report for this scenario.
        </p>
      )}
    </section>
  )
}
