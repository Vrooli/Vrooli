import { useCallback, useEffect, useMemo, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, AlertTriangle } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import { buildApiUrl } from '../utils/apiClient'
import { usePrepareDraft } from '../utils/useDraft'
import type { PublishedPRDResponse } from '../types'
import { DiagnosticsPanel } from '../components/prd-viewer'

interface DiagnosticsState {
  entityType: string
  entityName: string
  validatedAt?: string
  cacheUsed?: boolean
  violations: unknown
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
