import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { CheckCircle, AlertTriangle, XCircle, FileEdit } from 'lucide-react'
import { buildApiUrl } from '../utils/apiClient'

interface CatalogEntry {
  type: string
  name: string
  has_prd: boolean
  prd_path: string
  has_draft: boolean
  description: string
}

interface CatalogResponse {
  entries: CatalogEntry[]
  total: number
}

export default function Catalog() {
  const [entries, setEntries] = useState<CatalogEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState<'all' | 'scenario' | 'resource'>('all')

  useEffect(() => {
    fetchCatalog()
  }, [])

  const fetchCatalog = async () => {
    try {
      const response = await fetch(buildApiUrl('/catalog'))
      if (!response.ok) {
        throw new Error(`Failed to fetch catalog: ${response.statusText}`)
      }
      const data: CatalogResponse = await response.json()
      setEntries(data.entries)
      setLoading(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      setLoading(false)
    }
  }

  const filteredEntries = entries.filter(entry => {
    const matchesSearch = entry.name.toLowerCase().includes(filter.toLowerCase()) ||
                         entry.description.toLowerCase().includes(filter.toLowerCase())
    const matchesType = typeFilter === 'all' || entry.type === typeFilter
    return matchesSearch && matchesType
  })

  const getStatusBadge = (entry: CatalogEntry) => {
    if (entry.has_draft) {
      return (
        <span className="status-badge draft" title="Draft pending">
          <FileEdit size={16} />
          Draft Pending
        </span>
      )
    }
    if (entry.has_prd) {
      return (
        <span className="status-badge has-prd" title="Has PRD">
          <CheckCircle size={16} />
          Has PRD
        </span>
      )
    }
    return (
      <span className="status-badge missing" title="Missing PRD">
        <XCircle size={16} />
        Missing
      </span>
    )
  }

  if (loading) {
    return (
      <div className="container">
        <div className="loading">Loading catalog...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="container">
        <div className="error">
          <AlertTriangle size={24} />
          <p>Error loading catalog: {error}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="container">
      <header className="page-header">
        <h1>📋 PRD Control Tower</h1>
        <p className="subtitle">
          Centralized management, validation, and publishing of Product Requirements Documents
        </p>
      </header>

      <div className="nav-links">
        <Link to="/drafts" className="nav-link">
          View Drafts ({entries.filter(e => e.has_draft).length})
        </Link>
      </div>

      <div className="controls">
        <input
          type="text"
          className="search-input"
          placeholder="Search by name or description..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        />
        <div className="filter-buttons">
          <button
            className={`filter-btn ${typeFilter === 'all' ? 'active' : ''}`}
            onClick={() => setTypeFilter('all')}
          >
            All ({entries.length})
          </button>
          <button
            className={`filter-btn ${typeFilter === 'scenario' ? 'active' : ''}`}
            onClick={() => setTypeFilter('scenario')}
          >
            Scenarios ({entries.filter(e => e.type === 'scenario').length})
          </button>
          <button
            className={`filter-btn ${typeFilter === 'resource' ? 'active' : ''}`}
            onClick={() => setTypeFilter('resource')}
          >
            Resources ({entries.filter(e => e.type === 'resource').length})
          </button>
        </div>
      </div>

      <div className="stats">
        <div className="stat-item">
          <span className="stat-label">Total Entities</span>
          <span className="stat-value">{entries.length}</span>
        </div>
        <div className="stat-item">
          <span className="stat-label">With PRD</span>
          <span className="stat-value has-prd-color">
            {entries.filter(e => e.has_prd).length}
          </span>
        </div>
        <div className="stat-item">
          <span className="stat-label">Drafts</span>
          <span className="stat-value draft-color">
            {entries.filter(e => e.has_draft).length}
          </span>
        </div>
        <div className="stat-item">
          <span className="stat-label">Missing</span>
          <span className="stat-value missing-color">
            {entries.filter(e => !e.has_prd).length}
          </span>
        </div>
      </div>

      <div className="catalog-grid">
        {filteredEntries.length === 0 ? (
          <div className="no-results">
            <p>No entities found matching your filters.</p>
          </div>
        ) : (
          filteredEntries.map(entry => (
            <div key={`${entry.type}-${entry.name}`} className="catalog-card">
              <div className="card-header">
                <h3 className="card-title">{entry.name}</h3>
                <span className={`type-badge ${entry.type}`}>
                  {entry.type}
                </span>
              </div>
              <div className="card-body">
                <p className="card-description">
                  {entry.description || 'No description available'}
                </p>
              </div>
              <div className="card-footer">
                {getStatusBadge(entry)}
                <div className="card-actions">
                  {entry.has_prd && (
                    <Link
                      to={`/prd/${entry.type}/${entry.name}`}
                      className="btn-link"
                    >
                      View PRD
                    </Link>
                  )}
                  {entry.has_draft && (
                    <Link
                      to={`/draft/${entry.type}/${entry.name}`}
                      className="btn-link"
                    >
                      View Draft
                    </Link>
                  )}
                  {!entry.has_prd && !entry.has_draft && (
                    <button className="btn-link" disabled>
                      No PRD
                    </button>
                  )}
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
