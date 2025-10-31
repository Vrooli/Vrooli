import { KeyboardEvent, useEffect, useMemo, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  CheckCircle,
  AlertTriangle,
  XCircle,
  FileEdit,
  ClipboardList,
  Layers,
  Search,
  X,
  ChevronDown,
  Filter as FilterIcon
} from 'lucide-react'
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
  const [preparingDraftId, setPreparingDraftId] = useState<string | null>(null)
  const [filterMenuOpen, setFilterMenuOpen] = useState(false)
  const filterMenuRef = useRef<HTMLDivElement | null>(null)
  const navigate = useNavigate()

  useEffect(() => {
    fetchCatalog()
  }, [])

  useEffect(() => {
    if (!filterMenuOpen) {
      return
    }

    const handleClickOutside = (event: MouseEvent) => {
      if (filterMenuRef.current && !filterMenuRef.current.contains(event.target as Node)) {
        setFilterMenuOpen(false)
      }
    }

    const handleEscape = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') {
        setFilterMenuOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    window.addEventListener('keydown', handleEscape)

    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      window.removeEventListener('keydown', handleEscape)
    }
  }, [filterMenuOpen])

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

  const catalogCounts = useMemo(() => {
    let scenarios = 0
    let resources = 0
    let withPrd = 0
    let drafts = 0

    entries.forEach((entry) => {
      if (entry.type === 'scenario') {
        scenarios += 1
      }
      if (entry.type === 'resource') {
        resources += 1
      }
      if (entry.has_prd) {
        withPrd += 1
      }
      if (entry.has_draft) {
        drafts += 1
      }
    })

    return {
      total: entries.length,
      scenarios,
      resources,
      withPrd,
      drafts,
      missing: entries.length - withPrd,
    }
  }, [entries])

  const filterOptions = useMemo(() => ([
    { value: 'all' as const, label: `All (${catalogCounts.total})` },
    { value: 'scenario' as const, label: `Scenarios (${catalogCounts.scenarios})` },
    { value: 'resource' as const, label: `Resources (${catalogCounts.resources})` },
  ]), [catalogCounts])

  const activeFilterOption = filterOptions.find(option => option.value === typeFilter) ?? filterOptions[0]

  const searchLower = filter.toLowerCase()

  const filteredEntries = entries.filter(entry => {
    const matchesSearch = entry.name.toLowerCase().includes(searchLower) ||
                         entry.description.toLowerCase().includes(searchLower)
    const matchesType = typeFilter === 'all' || entry.type === typeFilter
    return matchesSearch && matchesType
  })

  const handleFilterChange = (value: 'all' | 'scenario' | 'resource') => {
    setTypeFilter(value)
    setFilterMenuOpen(false)
  }

  const prepareDraft = async (entry: CatalogEntry) => {
    const draftKey = `${entry.type}:${entry.name}`
    setPreparingDraftId(draftKey)

    try {
      const encodedName = encodeURIComponent(entry.name)
      const response = await fetch(buildApiUrl(`/catalog/${entry.type}/${encodedName}/draft`), {
        method: 'POST',
      })

      if (!response.ok) {
        const errorMessage = await response.text()
        throw new Error(errorMessage || 'Failed to prepare draft')
      }

      await response.json()

      setEntries(prevEntries =>
        prevEntries.map(item =>
          item.type === entry.type && item.name === entry.name
            ? { ...item, has_draft: true }
            : item,
        ),
      )

      navigate(`/draft/${entry.type}/${encodeURIComponent(entry.name)}`)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error'
      console.error('[Catalog] Failed to prepare draft', err)
      alert(`Failed to prepare draft for ${entry.name}: ${message}`)
    } finally {
      setPreparingDraftId(null)
    }
  }

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
        <h1 className="page-title">
          <span className="page-title__icon" aria-hidden="true">
            <ClipboardList size={28} strokeWidth={2.5} />
          </span>
          <span>PRD Control Tower</span>
        </h1>
        <p className="subtitle">
          Centralized management, validation, and publishing of Product Requirements Documents
        </p>
      </header>

      <div className="nav-links">
        <Link to="/drafts" className="nav-link">
          <Layers size={16} aria-hidden="true" />
          <span>View Drafts ({catalogCounts.drafts})</span>
        </Link>
      </div>

      <div className="controls">
        <div className="search-field">
          <label htmlFor="catalog-search" className="sr-only">Search catalog</label>
          <Search className="search-field__icon" size={18} aria-hidden="true" />
          <input
            id="catalog-search"
            type="text"
            className="search-input"
            placeholder="Search by name or description..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          />
          {filter && (
            <button
              type="button"
              className="search-field__clear"
              onClick={() => setFilter('')}
              aria-label="Clear search"
            >
              <X size={16} aria-hidden="true" />
            </button>
          )}
        </div>
        <div className="filter-menu" ref={filterMenuRef}>
          <span className="filter-menu__label">Show</span>
          <button
            type="button"
            className="filter-menu__trigger"
            aria-haspopup="listbox"
            aria-expanded={filterMenuOpen}
            onClick={() => setFilterMenuOpen(open => !open)}
          >
            <span className="filter-menu__trigger-text">
              <FilterIcon size={16} aria-hidden="true" />
              <span>{activeFilterOption.label}</span>
            </span>
            <ChevronDown size={16} aria-hidden="true" className="filter-menu__trigger-icon" />
          </button>
          {filterMenuOpen && (
            <div className="filter-menu__dropdown" role="listbox" aria-label="Filter catalog entries">
              {filterOptions.map(option => (
                <button
                  key={option.value}
                  type="button"
                  role="option"
                  aria-selected={typeFilter === option.value}
                  className={`filter-menu__item${typeFilter === option.value ? ' filter-menu__item--active' : ''}`}
                  onClick={() => handleFilterChange(option.value)}
                >
                  {option.label}
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      <div className="stats">
        <div className="stat-item">
          <span className="stat-label">Total Entities</span>
          <span className="stat-value">{catalogCounts.total}</span>
        </div>
        <div className="stat-item">
          <span className="stat-label">With PRD</span>
          <span className="stat-value has-prd-color">
            {catalogCounts.withPrd}
          </span>
        </div>
        <div className="stat-item">
          <span className="stat-label">Drafts</span>
          <span className="stat-value draft-color">
            {catalogCounts.drafts}
          </span>
        </div>
        <div className="stat-item">
          <span className="stat-label">Missing</span>
          <span className="stat-value missing-color">
            {catalogCounts.missing}
          </span>
        </div>
      </div>

      <div className="catalog-grid">
        {filteredEntries.length === 0 ? (
          <div className="no-results">
            <p>No entities found matching your filters.</p>
          </div>
        ) : (
          filteredEntries.map(entry => {
            const encodedName = encodeURIComponent(entry.name)
            const prdPath = `/prd/${entry.type}/${encodedName}`
            const draftPath = `/draft/${entry.type}/${encodedName}`
            const primaryPath = entry.has_prd ? prdPath : entry.has_draft ? draftPath : null
            const isNavigable = Boolean(primaryPath)
            const cardKey = `${entry.type}:${entry.name}`

            const navigateToPrimary = () => {
              if (!primaryPath) {
                return
              }
              navigate(primaryPath)
            }

            const handleCardKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
              if (!primaryPath) {
                return
              }
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault()
                navigate(primaryPath)
              }
            }

            return (
              <div
                key={cardKey}
                className={`catalog-card${isNavigable ? ' catalog-card--clickable' : ''}`}
                role={isNavigable ? 'link' : undefined}
                tabIndex={isNavigable ? 0 : undefined}
                aria-label={isNavigable ? `View ${entry.has_prd ? 'PRD' : 'draft'} for ${entry.name}` : undefined}
                onClick={isNavigable ? navigateToPrimary : undefined}
                onKeyDown={isNavigable ? handleCardKeyDown : undefined}
              >
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
                  <div
                    className="card-actions"
                    onClick={(event) => event.stopPropagation()}
                    onKeyDown={(event) => event.stopPropagation()}
                  >
                    {entry.has_prd && (
                      <Link
                        to={prdPath}
                        className="btn-link"
                      >
                        View PRD
                      </Link>
                    )}
                    {entry.has_prd && (
                      <button
                        type="button"
                        className="btn-link"
                        onClick={() => prepareDraft(entry)}
                        disabled={preparingDraftId === cardKey}
                      >
                        {preparingDraftId === cardKey ? 'Preparing...' : 'Edit PRD'}
                      </button>
                    )}
                    {entry.has_draft && (
                      <Link
                        to={draftPath}
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
            )
          })
        )}
      </div>
    </div>
  )
}
