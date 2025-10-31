import { useEffect, useMemo, useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { ArrowLeft, AlertTriangle } from 'lucide-react'
import { buildApiUrl } from '../utils/apiClient'

interface PRDResponse {
  type: string
  name: string
  content: string
  path: string
  content_html?: string
}

const escapeHtml = (value: string) =>
  value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')

const fallbackPlaintextToHtml = (value: string) => {
  if (!value.trim()) {
    return '<p></p>'
  }

  const paragraphs = value
    .split(/\n{2,}/)
    .map((paragraph) => escapeHtml(paragraph).replace(/\n/g, '<br />'))
    .filter(Boolean)

  return `<p>${paragraphs.join('</p><p>')}</p>`
}

export default function PRDViewer() {
  const { type, name } = useParams<{ type: string; name: string }>()
  const [prd, setPrd] = useState<PRDResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [preparingDraft, setPreparingDraft] = useState(false)
  const navigate = useNavigate()

  const renderedContent = useMemo(() => {
    if (!prd) {
      return ''
    }
    return prd.content_html ?? fallbackPlaintextToHtml(prd.content)
  }, [prd])

  const hasRenderableContent = useMemo(() => {
    if (!renderedContent) {
      return false
    }
    const textContent = renderedContent.replace(/<[^>]*>/g, '').trim()
    return textContent.length > 0
  }, [renderedContent])

  useEffect(() => {
    if (type && name) {
      fetchPRD()
    }
  }, [type, name])

  const fetchPRD = async () => {
    try {
      const response = await fetch(buildApiUrl(`/catalog/${type}/${name}`))
      if (!response.ok) {
        if (response.status === 404) {
          throw new Error(`PRD not found for ${type}/${name}`)
        }
        throw new Error(`Failed to fetch PRD: ${response.statusText}`)
      }
      const data: PRDResponse = await response.json()
      setPrd(data)
      setLoading(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      setLoading(false)
    }
  }

  const prepareDraft = async () => {
    if (!prd) {
      return
    }

    setPreparingDraft(true)

    try {
      const encodedName = encodeURIComponent(prd.name)
      const response = await fetch(buildApiUrl(`/catalog/${prd.type}/${encodedName}/draft`), {
        method: 'POST',
      })

      if (!response.ok) {
        const message = await response.text()
        throw new Error(message || 'Failed to prepare draft')
      }

      await response.json()
      navigate(`/draft/${prd.type}/${encodeURIComponent(prd.name)}`)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error'
      console.error('[PRDViewer] Failed to prepare draft', err)
      alert(`Failed to prepare draft: ${message}`)
    } finally {
      setPreparingDraft(false)
    }
  }

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
            <p style={{ color: '#718096', marginTop: '0.25rem' }}>
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
              onClick={prepareDraft}
              disabled={preparingDraft}
            >
              {preparingDraft ? 'Preparing...' : 'Edit PRD'}
            </button>
          </div>
        </div>

        {hasRenderableContent ? (
          <div
            className="prd-content"
            dangerouslySetInnerHTML={{ __html: renderedContent }}
          />
        ) : (
          <div className="prd-content prd-content--empty">
            PRD content is available but could not be rendered.
          </div>
        )}
      </div>
    </div>
  )
}
