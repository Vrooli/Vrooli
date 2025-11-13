import { KeyboardEvent, ReactNode, useCallback, useEffect, useMemo, useRef, useState } from 'react'
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
  Filter as FilterIcon,
  Sparkles,
  StickyNote,
} from 'lucide-react'
import { buildApiUrl } from '../utils/apiClient'
import { usePrepareDraft } from '../utils/useDraft'
import type { CatalogEntry, CatalogResponse } from '../types'

export default function Catalog() {
  const [entries, setEntries] = useState<CatalogEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState<'all' | 'scenario' | 'resource'>('all')
  const [filterMenuOpen, setFilterMenuOpen] = useState(false)
  const filterMenuRef = useRef<HTMLDivElement | null>(null)
  const navigate = useNavigate()

  const { prepareDraft, preparingId } = usePrepareDraft({
    onSuccess: (entityType, entityName) => {
      setEntries(prevEntries =>
        prevEntries.map(item =>
          item.type === entityType && item.name === entityName
            ? { ...item, has_draft: true }
            : item,
        ),
      )
    },
  })

  const fetchCatalog = useCallback(async () => {
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
  }, [])

  useEffect(() => {
    fetchCatalog()
  }, [fetchCatalog])

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

  const coverage = useMemo(() => {
    if (!catalogCounts.total) {
      return 0
    }
    return Math.round((catalogCounts.withPrd / catalogCounts.total) * 100)
  }, [catalogCounts])

  const missingPercentage = useMemo(() => {
    if (!catalogCounts.total) {
      return 0
    }
    return Math.round((catalogCounts.missing / catalogCounts.total) * 100)
  }, [catalogCounts])

  const statHighlights = useMemo(() => {
    const highlights: Array<{
      key: string
      label: string
      value: string
      description: string
      icon: ReactNode
      tone: 'slate' | 'success' | 'info' | 'warning'
      chip?: string
    }> = [
      {
        key: 'total',
        label: 'Total entities',
        value: catalogCounts.total.toLocaleString(),
        description: 'Scenarios & resources monitored',
        icon: <Layers size={18} aria-hidden="true" />,
        tone: 'slate',
      },
      {
        key: 'with-prd',
        label: 'With PRD',
        value: catalogCounts.withPrd.toLocaleString(),
        description: `${coverage}% coverage`,
        icon: <CheckCircle size={18} aria-hidden="true" />,
        tone: 'success',
        chip: `${coverage}%`,
      },
      {
        key: 'drafts',
        label: 'Active drafts',
        value: catalogCounts.drafts.toLocaleString(),
        description: 'Awaiting review',
        icon: <FileEdit size={18} aria-hidden="true" />,
        tone: 'info',
      },
      {
        key: 'missing',
        label: 'Missing PRDs',
        value: catalogCounts.missing.toLocaleString(),
        description: `${missingPercentage}% need attention`,
        icon: <XCircle size={18} aria-hidden="true" />,
        tone: 'warning',
        chip: 'Action needed',
      },
    ]

    return highlights
  }, [catalogCounts, coverage, missingPercentage])

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

  const handleFilterChange = useCallback((value: 'all' | 'scenario' | 'resource') => {
    setTypeFilter(value)
    setFilterMenuOpen(false)
  }, [])

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

  const draftsLabel = catalogCounts.drafts === 1 ? 'Draft' : 'Drafts'

  return (
    <div className="container catalog-page">
      <header className="page-header page-header--catalog">
        <span className="page-eyebrow">Product operations</span>
        <div className="page-header__body">
          <div className="page-header__text">
            <h1 className="page-title">
              <span className="page-title__icon" aria-hidden="true">
                <ClipboardList size={28} strokeWidth={2.5} />
              </span>
              <span>PRD Control Tower</span>
            </h1>
            <p className="subtitle">
              Centralized management, validation, and publishing for {catalogCounts.total.toLocaleString()} tracked entities with {coverage}% PRD coverage so far.
            </p>
          </div>
          <div className="page-header__actions">
            <Link to="/backlog" className="nav-button nav-button--secondary" aria-label="Open idea backlog">
              <StickyNote size={18} aria-hidden="true" />
              <span className="nav-button__label">
                <strong>Backlog</strong>
                <small>Capture ideas</small>
              </span>
            </Link>
            <Link to="/drafts" className="nav-button" aria-label={`View ${catalogCounts.drafts} ${draftsLabel.toLowerCase()}`}>
              <Layers size={18} aria-hidden="true" />
              <span className="nav-button__label">
                <strong>View Drafts</strong>
                <small>Curate updates</small>
              </span>
              <span className="nav-button__pill">{catalogCounts.drafts}</span>
            </Link>
          </div>
        </div>
        <div className="insight-banner" role="status" aria-live="polite">
          <span className="insight-banner__icon" aria-hidden="true">
            <Sparkles size={18} />
          </span>
          <div>
            <p className="insight-banner__title">Quality pulse</p>
            <p className="insight-banner__body">
              {catalogCounts.withPrd.toLocaleString()} PRDs published · {catalogCounts.missing.toLocaleString()} gaps remain ({missingPercentage}% of catalog)
            </p>
          </div>
        </div>
      </header>

      <section className="surface-card catalog-toolbar" aria-label="Catalog controls">
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
      </section>

      <section className="stat-panel" aria-label="Catalog health overview">
        {statHighlights.map((stat) => (
          <article key={stat.key} className={`stat-card stat-card--${stat.tone}`}>
            <div className="stat-card__icon" aria-hidden="true">
              {stat.icon}
            </div>
            <div className="stat-card__content">
              <span className="stat-card__label">{stat.label}</span>
              <div className="stat-card__value-row">
                <span className="stat-card__value">{stat.value}</span>
                {stat.chip ? <span className="stat-card__chip">{stat.chip}</span> : null}
              </div>
              <p className="stat-card__description">{stat.description}</p>
            </div>
          </article>
        ))}
      </section>

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
                        onClick={() => prepareDraft(entry.type, entry.name)}
                        disabled={preparingId === cardKey}
                      >
                        {preparingId === cardKey ? 'Preparing...' : 'Edit PRD'}
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
