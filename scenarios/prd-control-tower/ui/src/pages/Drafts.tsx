import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { FileEdit, Clock, User, Trash2 } from 'lucide-react'
import { buildApiUrl } from '../utils/apiClient'

interface Draft {
  id: string
  entity_type: string
  entity_name: string
  content: string
  owner?: string
  created_at: string
  updated_at: string
  status: string
}

interface DraftListResponse {
  drafts: Draft[]
  total: number
}

export default function Drafts() {
  const [drafts, setDrafts] = useState<Draft[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState('')

  useEffect(() => {
    fetchDrafts()
  }, [])

  const fetchDrafts = async () => {
    try {
      const response = await fetch(buildApiUrl('/drafts'))
      if (!response.ok) {
        throw new Error(`Failed to fetch drafts: ${response.statusText}`)
      }
      const data: DraftListResponse = await response.json()
      setDrafts(data.drafts || [])
      setLoading(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      setLoading(false)
    }
  }

  const handleDelete = async (draftId: string) => {
    if (!confirm('Are you sure you want to delete this draft?')) {
      return
    }

    try {
      const response = await fetch(buildApiUrl(`/drafts/${draftId}`), {
        method: 'DELETE',
      })
      if (!response.ok) {
        throw new Error(`Failed to delete draft: ${response.statusText}`)
      }
      // Refresh drafts list
      fetchDrafts()
    } catch (err) {
      alert(`Error deleting draft: ${err instanceof Error ? err.message : 'Unknown error'}`)
    }
  }

  const filteredDrafts = drafts.filter(draft => {
    const searchLower = filter.toLowerCase()
    return (
      draft.entity_name.toLowerCase().includes(searchLower) ||
      draft.entity_type.toLowerCase().includes(searchLower) ||
      (draft.owner && draft.owner.toLowerCase().includes(searchLower))
    )
  })

  const formatDate = (dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  if (loading) {
    return (
      <div className="container">
        <div className="loading">Loading drafts...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="container">
        <div className="error">
          <p>Error loading drafts: {error}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="container">
      <header className="page-header">
        <h1>✏️ Draft PRDs</h1>
        <p className="subtitle">
          Manage and edit PRD drafts before publishing
        </p>
      </header>

      <div className="controls">
        <input
          type="text"
          className="search-input"
          placeholder="Search drafts by name, type, or owner..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        />
      </div>

      <div className="stats">
        <div className="stat-item">
          <span className="stat-label">Total Drafts</span>
          <span className="stat-value">{drafts.length}</span>
        </div>
        <div className="stat-item">
          <span className="stat-label">Scenarios</span>
          <span className="stat-value">
            {drafts.filter(d => d.entity_type === 'scenario').length}
          </span>
        </div>
        <div className="stat-item">
          <span className="stat-label">Resources</span>
          <span className="stat-value">
            {drafts.filter(d => d.entity_type === 'resource').length}
          </span>
        </div>
      </div>

      {filteredDrafts.length === 0 ? (
        <div className="no-results">
          {drafts.length === 0 ? (
            <>
              <FileEdit size={64} style={{ margin: '0 auto 1rem', color: '#cbd5e0' }} />
              <p>No drafts found. Create a new draft to get started.</p>
            </>
          ) : (
            <p>No drafts found matching your search.</p>
          )}
        </div>
      ) : (
        <div className="catalog-grid">
          {filteredDrafts.map(draft => (
            <div key={draft.id} className="catalog-card">
              <div className="card-header">
                <h3 className="card-title">{draft.entity_name}</h3>
                <span className={`type-badge ${draft.entity_type}`}>
                  {draft.entity_type}
                </span>
              </div>
              <div className="card-body">
                <div className="draft-meta">
                  {draft.owner && (
                    <div className="draft-meta-item">
                      <User size={14} />
                      <span>{draft.owner}</span>
                    </div>
                  )}
                  <div className="draft-meta-item">
                    <Clock size={14} />
                    <span>Updated {formatDate(draft.updated_at)}</span>
                  </div>
                  <div className="draft-meta-item">
                    <FileEdit size={14} />
                    <span>{(draft.content.length / 1024).toFixed(1)} KB</span>
                  </div>
                </div>
              </div>
              <div className="card-footer">
                <div className="status-badge draft">
                  <FileEdit size={16} />
                  {draft.status}
                </div>
                <div className="card-actions">
                  <Link
                    to={`/draft/${draft.entity_type}/${draft.entity_name}`}
                    className="btn-link"
                  >
                    Edit
                  </Link>
                  <button
                    className="btn-link btn-danger"
                    onClick={() => handleDelete(draft.id)}
                    title="Delete draft"
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="back-link-container">
        <Link to="/" className="back-link">
          ← Back to Catalog
        </Link>
      </div>
    </div>
  )
}
