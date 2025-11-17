import { useCallback, useEffect, useMemo, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, AlertTriangle, ListTree } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import { buildApiUrl } from '../utils/apiClient'
import { usePrepareDraft } from '../utils/useDraft'
import type { ScenarioQualityReport } from '../types'
import { DiagnosticsPanel, QualityInsightsPanel } from '../components/prd-viewer'
import { fetchQualityReport } from '../utils/quality'
import { usePublishedPRD } from '../hooks/usePublishedPRD'

interface DiagnosticsState {
  entityType: string
  entityName: string
  validatedAt?: string
  cacheUsed?: boolean
  violations: unknown
}

export default function PRDViewer() {
  const { type, name } = useParams<{ type: string; name: string }>()
  const [diagnostics, setDiagnostics] = useState<DiagnosticsState | null>(null)
  const [diagnosticsLoading, setDiagnosticsLoading] = useState(false)
  const [diagnosticsError, setDiagnosticsError] = useState<string | null>(null)
  const [qualityReport, setQualityReport] = useState<ScenarioQualityReport | null>(null)
  const [qualityLoading, setQualityLoading] = useState(true)
  const [qualityError, setQualityError] = useState<string | null>(null)

  const { prepareDraft, preparing: preparingDraft } = usePrepareDraft()
  const { data: prd, loading, error } = usePublishedPRD(type, name)

  const hasRenderableContent = useMemo(() => {
    if (!prd?.content) {
      return false
    }
    return prd.content.trim().length > 0
  }, [prd])

  const loadQualityReport = useCallback(async (force = false) => {
    if (!type || !name) {
      return
    }

    setQualityLoading(true)
    setQualityError(null)

    try {
      const data = await fetchQualityReport(type, name, { useCache: !force })
      setQualityReport(data)
    } catch (err) {
      setQualityError(err instanceof Error ? err.message : 'Failed to load quality insights')
    } finally {
      setQualityLoading(false)
    }
  }, [type, name])

  useEffect(() => {
    loadQualityReport(true)
  }, [loadQualityReport])

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
          <Link to="/catalog" className="back-link">
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
        <Link to="/catalog" className="back-link">
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
      <Link to="/catalog" className="back-link">
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
            <Link
              to={`/requirements/${prd.type}/${prd.name}`}
              className="btn-link"
              style={{ marginRight: '8px' }}
            >
              <ListTree size={16} style={{ marginRight: '4px', display: 'inline' }} />
              Requirements
            </Link>
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

        <QualityInsightsPanel
          report={qualityReport}
          loading={qualityLoading}
          error={qualityError}
          onRefresh={() => loadQualityReport(true)}
        />

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

        <div className="flex flex-wrap gap-4 mt-6 text-sm">
          <Link to={`/requirements/${type}/${name}`} className="text-primary hover:underline">
            → View Requirements
          </Link>
          <Link to={`/targets/${type}/${name}`} className="text-primary hover:underline">
            → View Operational Targets
          </Link>
          <Link to="/catalog" className="text-primary hover:underline">
            ← Back to Catalog
          </Link>
        </div>
      </div>
    </div>
  )
}
