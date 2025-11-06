import { useCallback, useEffect, useMemo, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, AlertTriangle, Loader2, RefreshCcw, ShieldAlert, ShieldCheck } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import { buildApiUrl } from '../utils/apiClient'
import { usePrepareDraft } from '../utils/useDraft'
import type { PublishedPRDResponse } from '../types'
import { formatDate } from '../utils/formatters'

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

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function getSectionResult(section: unknown): Record<string, unknown> | null {
  if (!isRecord(section)) {
    return null
  }

  const status = section.status
  if (isRecord(status) && isRecord(status.result)) {
    return status.result
  }

  if (isRecord(section.result)) {
    return section.result
  }

  return section
}

function toNumber(value: unknown): number | undefined {
  return typeof value === 'number' ? value : undefined
}

function toString(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined
}

function DiagnosticsMetric({ label, value }: { label: string; value: number | undefined }) {
  return (
    <div className="diagnostics-metric">
      <span className="diagnostics-metric__label">{label}</span>
      <span className="diagnostics-metric__value">{value ?? 0}</span>
    </div>
  )
}

function DiagnosticsViolationList({ violations, limit }: { violations: unknown[]; limit?: number }) {
  const items = violations
    .filter((violation): violation is Record<string, unknown> => isRecord(violation))
    .slice(0, typeof limit === 'number' ? limit : violations.length)

  if (items.length === 0) {
    return <p className="diagnostics-empty">No violations reported in this category.</p>
  }

  return (
    <div className="diagnostics-list">
      {items.map((violation, index) => {
        const key = toString(violation.id) ?? `${index}`
        const title = toString(violation.title) ?? toString(violation.rule) ?? 'Unnamed rule'
        const message = toString(violation.description) ?? toString(violation.message)
        const filePath = toString(violation.file_path)
        const standard = toString(violation.standard)
        const severity = (toString(violation.severity) ?? toString(violation.level))?.toLowerCase()

        return (
          <div key={key} className="diagnostics-list__item">
            <div className="diagnostics-list__header">
              <h3 className="diagnostics-list__title">{title}</h3>
              {severity && (
                <span className={`severity-badge severity-badge--${severity}`}>{severity}</span>
              )}
            </div>
            {message && <p className="diagnostics-list__message">{message}</p>}
            {(filePath || standard) && (
              <div className="diagnostics-list__details">
                {filePath && (
                  <span className="diagnostics-list__path" aria-label="File path">
                    {filePath}
                  </span>
                )}
                {standard && (
                  <span className="diagnostics-list__standard">Standard: {standard}</span>
                )}
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}

function DiagnosticsSections({ data }: { data: Record<string, unknown> }) {
  const sections: Array<{
    id: 'security' | 'standards'
    section: Record<string, unknown>
  }> = []

  const securitySection = data['security']
  if (isRecord(securitySection)) {
    sections.push({ id: 'security', section: securitySection })
  }

  const standardsSection = data['standards']
  if (isRecord(standardsSection)) {
    sections.push({ id: 'standards', section: standardsSection })
  }

  if (sections.length === 0) {
    return <DiagnosticsJsonPreview value={data} />
  }

  return (
    <div className="diagnostics-cards">
      {sections.map(({ id, section }) => {
        const result = getSectionResult(section)
        const message = isRecord(section.status) ? toString(section.status.message) : toString(section.message)
        const icon = id === 'security' ? <ShieldCheck size={20} aria-hidden="true" /> : <ShieldAlert size={20} aria-hidden="true" />

        const statistics = result && isRecord(result.statistics) ? (result.statistics as Record<string, unknown>) : null
        const vulnerabilities = result && isRecord(result.vulnerabilities) ? (result.vulnerabilities as Record<string, unknown>) : null
        const violationsList = Array.isArray(result?.violations) ? (result?.violations as unknown[]) : []

        const metrics: Array<{ label: string; value: number | undefined }> = []

        if (id === 'security') {
          const total = vulnerabilities ? toNumber(vulnerabilities.total) : toNumber(statistics?.total_findings)
          metrics.push({ label: 'Total', value: total })

          const severitySource = vulnerabilities ?? (isRecord(statistics?.by_severity) ? (statistics?.by_severity as Record<string, unknown>) : null)
          const severities = ['critical', 'high', 'medium', 'low', 'info']
          severities.forEach((severity) => {
            const value = severitySource ? toNumber(severitySource[severity]) : undefined
            if (value !== undefined) {
              metrics.push({ label: severity.charAt(0).toUpperCase() + severity.slice(1), value })
            }
          })
        } else {
          const total = statistics ? toNumber(statistics.total) : undefined
          metrics.push({ label: 'Total', value: total })

          const severities = ['critical', 'high', 'medium', 'low']
          severities.forEach((severity) => {
            const value = statistics ? toNumber(statistics[severity]) : undefined
            if (value !== undefined) {
              metrics.push({ label: severity.charAt(0).toUpperCase() + severity.slice(1), value })
            }
          })
        }

        return (
          <div key={id} className="diagnostics-card">
            <div className="diagnostics-card__header">
              <span className={`diagnostics-card__icon diagnostics-card__icon--${id}`}>{icon}</span>
              <div>
                <h3 className="diagnostics-card__title">{id === 'security' ? 'Security Scan' : 'Standards Check'}</h3>
                {message && <p className="diagnostics-card__message">{message}</p>}
              </div>
            </div>

            {metrics.length > 0 && (
              <div className="diagnostics-card__metrics">
                {metrics.map((metric) => (
                  <DiagnosticsMetric key={`${id}-${metric.label}`} label={metric.label} value={metric.value} />
                ))}
              </div>
            )}

            {id === 'standards' && violationsList.length > 0 && (
              <div className="diagnostics-card__violations" aria-live="polite">
                <h4 className="diagnostics-card__subtitle">Most recent findings</h4>
                <DiagnosticsViolationList violations={violationsList} limit={5} />
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}

function DiagnosticsJsonPreview({ value }: { value: Record<string, unknown> }) {
  return (
    <pre className="diagnostics-pre" role="region" aria-label="Diagnostics output">
      {JSON.stringify(value, null, 2)}
    </pre>
  )
}

function DiagnosticsResults({ data }: { data: unknown }) {
  if (data === null || data === undefined) {
    return <p className="diagnostics-empty">Diagnostics completed without returning additional details.</p>
  }

  if (Array.isArray(data)) {
    return <DiagnosticsViolationList violations={data} />
  }

  if (isRecord(data)) {
    if (toString(data.error) || toString(data.message)) {
      return (
        <div className="diagnostics-error" role="alert">
          <AlertTriangle size={20} aria-hidden="true" />
          <div>
            <p className="diagnostics-error__title">Diagnostic tool reported an issue</p>
            <p>{toString(data.message) ?? toString(data.error)}</p>
          </div>
        </div>
      )
    }

    return <DiagnosticsSections data={data} />
  }

  return (
    <pre className="diagnostics-pre" role="region" aria-label="Diagnostics output">
      {String(data)}
    </pre>
  )
}

function DiagnosticsPanel({ diagnostics, loading, error, onRunDiagnostics, entityLabel }: DiagnosticsPanelProps) {
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
              {diagnostics.cacheUsed ? ' · served from cache' : ''}
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

export default function PRDViewer() {
  const { type, name } = useParams<{ type: string; name: string }>()
  const [prd, setPrd] = useState<PublishedPRDResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [diagnostics, setDiagnostics] = useState<DiagnosticsState | null>(null)
  const [diagnosticsLoading, setDiagnosticsLoading] = useState(false)
  const [diagnosticsError, setDiagnosticsError] = useState<string | null>(null)

  const { prepareDraft, preparing: preparingDraft } = usePrepareDraft()

  const hasRenderableContent = useMemo(() => {
    if (!prd?.content) {
      return false
    }
    return prd.content.trim().length > 0
  }, [prd])

  const fetchPRD = useCallback(async () => {
    if (!type || !name) return

    try {
      const response = await fetch(buildApiUrl(`/catalog/${type}/${name}`))
      if (!response.ok) {
        if (response.status === 404) {
          throw new Error(`PRD not found for ${type}/${name}`)
        }
        throw new Error(`Failed to fetch PRD: ${response.statusText}`)
      }
      const data: PublishedPRDResponse = await response.json()
      setPrd(data)
      setLoading(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      setLoading(false)
    }
  }, [type, name])

  useEffect(() => {
    fetchPRD()
  }, [fetchPRD])

  const handlePrepareDraft = () => {
    if (prd) {
      prepareDraft(prd.type, prd.name)
    }
  }

  const handleRunDiagnostics = useCallback(async () => {
    if (!prd) {
      return
    }

    setDiagnosticsLoading(true)
    setDiagnosticsError(null)

    try {
      const response = await fetch(buildApiUrl('/drafts/validate'), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          entity_type: prd.type,
          entity_name: prd.name,
        }),
      })

      if (!response.ok) {
        const body = await response.text()
        throw new Error(body || `Failed to run diagnostics: ${response.statusText}`)
      }

      const data: Record<string, unknown> = await response.json()

      setDiagnostics({
        entityType: typeof data.entity_type === 'string' ? (data.entity_type as string) : prd.type,
        entityName: typeof data.entity_name === 'string' ? (data.entity_name as string) : prd.name,
        validatedAt: typeof data.validated_at === 'string' ? (data.validated_at as string) : undefined,
        cacheUsed: data.cache_used === true,
        violations: data.violations,
      })
    } catch (err) {
      setDiagnosticsError(err instanceof Error ? err.message : 'Unknown error running diagnostics.')
    } finally {
      setDiagnosticsLoading(false)
    }
  }, [prd])

  if (loading) {
    return (
      <div className="container">
        <div className="loading">Loading PRD...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="container">
        <Link to="/" className="back-link">
          <ArrowLeft size={20} />
          Back to Catalog
        </Link>
        <div className="error">
          <AlertTriangle size={24} />
          <p>Error loading PRD: {error}</p>
        </div>
      </div>
    )
  }

  if (!prd) {
    return (
      <div className="container">
        <Link to="/" className="back-link">
          <ArrowLeft size={20} />
          Back to Catalog
        </Link>
        <div className="error">
          <AlertTriangle size={24} />
          <p>PRD not found</p>
        </div>
      </div>
    )
  }

  return (
    <div className="container">
      <Link to="/" className="back-link">
        <ArrowLeft size={20} />
        Back to Catalog
      </Link>

      <div className="prd-viewer">
        <div className="viewer-header">
          <div>
            <h1 className="viewer-title">{prd.name}</h1>
            <p className="viewer-subtitle">
              Type: <strong>{prd.type}</strong>
            </p>
          </div>
          <div className="viewer-meta">
            <span className={`type-badge ${prd.type}`}>
              {prd.type}
            </span>
            <button
              type="button"
              className="btn-link"
              onClick={handlePrepareDraft}
              disabled={preparingDraft}
            >
              {preparingDraft ? 'Preparing...' : 'Edit PRD'}
            </button>
          </div>
        </div>

        {hasRenderableContent ? (
          <div className="prd-content">
            <ReactMarkdown>{prd.content}</ReactMarkdown>
          </div>
        ) : (
          <div className="prd-content prd-content--empty">
            PRD content is not available.
          </div>
        )}

        <DiagnosticsPanel
          diagnostics={diagnostics}
          loading={diagnosticsLoading}
          error={diagnosticsError}
          onRunDiagnostics={handleRunDiagnostics}
          entityLabel={prd.name}
        />
      </div>
    </div>
  )
}
