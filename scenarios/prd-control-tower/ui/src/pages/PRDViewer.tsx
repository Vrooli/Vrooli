import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, AlertTriangle } from 'lucide-react'
import { buildApiUrl } from '../utils/apiClient'

interface PRDResponse {
  type: string
  name: string
  content: string
  path: string
}

export default function PRDViewer() {
  const { type, name } = useParams<{ type: string; name: string }>()
  const [prd, setPrd] = useState<PRDResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

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
          </div>
        </div>

        <div className="prd-content">
          <pre style={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit', fontSize: '1rem', lineHeight: '1.8', color: '#2d3748' }}>{prd.content}</pre>
        </div>
      </div>
    </div>
  )
}
