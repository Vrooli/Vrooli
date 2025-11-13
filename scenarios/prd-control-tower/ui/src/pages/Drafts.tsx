import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ChangeEvent, KeyboardEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import toast from 'react-hot-toast'
import ReactMarkdown from 'react-markdown'
import {
  FileEdit,
  Clock,
  User,
  Trash2,
  Save,
  RotateCcw,
  RefreshCcw,
  CheckCircle2,
  AlertTriangle,
  Loader2,
  Search,
  X,
  Info,
} from 'lucide-react'
import { buildApiUrl } from '../utils/apiClient'
import { useConfirm } from '../utils/confirmDialog'
import { formatDate, formatFileSize, decodeRouteSegment, calculateDraftMetrics } from '../utils/formatters'
import type { Draft, DraftListResponse, ViewMode } from '../types'
import { ViewModes } from '../types'

interface SaveStatus {
  type: 'success' | 'error'
  message: string
}

export default function Drafts() {
  const { entityType: routeEntityType, entityName: routeEntityName } = useParams<{
    entityType?: string
    entityName?: string
  }>()
  const navigate = useNavigate()
  const confirm = useConfirm()

  const decodedRouteName = useMemo(() => decodeRouteSegment(routeEntityName), [routeEntityName])
  const normalizedRouteType = routeEntityType?.toLowerCase()
  const normalizedRouteName = decodedRouteName.toLowerCase()

  const [drafts, setDrafts] = useState<Draft[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState(() => decodedRouteName)
  const [selectedDraftId, setSelectedDraftId] = useState<string | null>(null)
  const [editorContent, setEditorContent] = useState('')
  const [lastLoadedContent, setLastLoadedContent] = useState('')
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveStatus, setSaveStatus] = useState<SaveStatus | null>(null)
  const [viewMode, setViewMode] = useState<ViewMode>(ViewModes.SPLIT)
  const [metaDialogOpen, setMetaDialogOpen] = useState(false)

  const openMetaDialog = useCallback(() => {
    setMetaDialogOpen(true)
  }, [])

  const closeMetaDialog = useCallback(() => {
    setMetaDialogOpen(false)
  }, [])

  const fetchDrafts = useCallback(async (options?: { silent?: boolean }) => {
    if (options?.silent) {
      setRefreshing(true)
    } else {
      setLoading(true)
    }
    setError(null)

    try {
      const response = await fetch(buildApiUrl('/drafts'))
      if (!response.ok) {
        throw new Error(`Failed to fetch drafts: ${response.statusText}`)
      }
      const data: DraftListResponse = await response.json()
      setDrafts(data.drafts || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      if (options?.silent) {
        setRefreshing(false)
      } else {
        setLoading(false)
      }
    }
  }, [])

  const handleSelectDraft = useCallback(
    (draft: Draft) => {
      const encodedName = encodeURIComponent(draft.entity_name)
      const normalizedDraftType = draft.entity_type.toLowerCase()
      const normalizedDraftName = draft.entity_name.toLowerCase()
      const nextPath = `/draft/${draft.entity_type}/${encodedName}`
      const shouldReplace =
        normalizedRouteType === normalizedDraftType && normalizedRouteName === normalizedDraftName

      navigate(nextPath, shouldReplace ? { replace: true } : undefined)
    },
    [navigate, normalizedRouteName, normalizedRouteType],
  )

  const handleCardKeyDown = useCallback(
    (event: KeyboardEvent<HTMLDivElement>, draft: Draft) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault()
        handleSelectDraft(draft)
      }
    },
    [handleSelectDraft],
  )

  useEffect(() => {
    fetchDrafts()
  }, [fetchDrafts])

  useEffect(() => {
    if (decodedRouteName) {
      setFilter(decodedRouteName)
      return
    }

    if (routeEntityName === undefined) {
      setFilter('')
    }
  }, [decodedRouteName, routeEntityName])

  const selectedDraft = useMemo(() => {
    if (!normalizedRouteName) {
      return null
    }

    return (
      drafts.find((draft) => {
        const sameName = draft.entity_name.toLowerCase() === normalizedRouteName
        if (normalizedRouteType) {
          return sameName && draft.entity_type.toLowerCase() === normalizedRouteType
        }
        return sameName
      }) || null
    )
  }, [drafts, normalizedRouteName, normalizedRouteType])

  useEffect(() => {
    if (!selectedDraft) {
      setSelectedDraftId(null)
      setEditorContent('')
      setLastLoadedContent('')
      setHasUnsavedChanges(false)
      setViewMode(ViewModes.SPLIT)
      setMetaDialogOpen(false)
      return
    }

    const selectionChanged = selectedDraft.id !== selectedDraftId
    if (selectionChanged) {
      setSelectedDraftId(selectedDraft.id)
      setEditorContent(selectedDraft.content)
      setLastLoadedContent(selectedDraft.content)
      setHasUnsavedChanges(false)
      setSaveStatus(null)
      setViewMode(ViewModes.SPLIT)
      setMetaDialogOpen(false)
      return
    }

    if (!hasUnsavedChanges && selectedDraft.content !== lastLoadedContent) {
      setEditorContent(selectedDraft.content)
      setLastLoadedContent(selectedDraft.content)
    }
  }, [selectedDraft, selectedDraftId, hasUnsavedChanges, lastLoadedContent])

  useEffect(() => {
    if (!saveStatus) {
      return
    }

    const timeout = window.setTimeout(() => {
      setSaveStatus(null)
    }, saveStatus.type === 'success' ? 4000 : 6000)

    return () => {
      window.clearTimeout(timeout)
    }
  }, [saveStatus])

  useEffect(() => {
    if (!metaDialogOpen) {
      return
    }

    const handleKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        setMetaDialogOpen(false)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'

    return () => {
      window.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = previousOverflow
    }
  }, [metaDialogOpen])

  const handleEditorChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    const value = event.target.value
    setEditorContent(value)
    if (selectedDraft) {
      setHasUnsavedChanges(value !== lastLoadedContent)
    }
  }

  const handleDiscardChanges = async () => {
    if (!selectedDraft) {
      return
    }
    if (hasUnsavedChanges) {
      const shouldDiscard = await confirm({
        title: 'Discard Changes?',
        message: 'Are you sure you want to discard your unsaved changes? This action cannot be undone.',
        confirmText: 'Discard',
        cancelText: 'Keep Editing',
        variant: 'warning',
      })
      if (!shouldDiscard) {
        return
      }
    }

    setEditorContent(lastLoadedContent)
    setHasUnsavedChanges(false)
  }

  const handleRefreshDraft = async () => {
    if (!selectedDraft) {
      return
    }
    if (hasUnsavedChanges) {
      const shouldContinue = await confirm({
        title: 'Refresh Draft?',
        message: 'Refreshing will discard unsaved changes. Continue?',
        confirmText: 'Refresh',
        cancelText: 'Cancel',
        variant: 'warning',
      })
      if (!shouldContinue) {
        return
      }
    }

    await fetchDrafts({ silent: true })
  }

  const handleSaveDraft = async () => {
    if (!selectedDraft) {
      return
    }

    setSaving(true)
    setSaveStatus(null)

    try {
      const response = await fetch(buildApiUrl(`/drafts/${selectedDraft.id}`), {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ content: editorContent }),
      })

      if (!response.ok) {
        throw new Error(`Failed to save draft: ${response.statusText}`)
      }

      await fetchDrafts({ silent: true })
      setLastLoadedContent(editorContent)
      setHasUnsavedChanges(false)
      setSaveStatus({ type: 'success', message: 'Draft saved successfully.' })
    } catch (err) {
      setSaveStatus({
        type: 'error',
        message: err instanceof Error ? err.message : 'Unexpected error saving draft.',
      })
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (draftId: string) => {
    const shouldDelete = await confirm({
      title: 'Delete Draft?',
      message: 'Are you sure you want to delete this draft? This action cannot be undone.',
      confirmText: 'Delete',
      cancelText: 'Cancel',
      variant: 'danger',
    })

    if (!shouldDelete) {
      return
    }

    try {
      const response = await fetch(buildApiUrl(`/drafts/${draftId}`), {
        method: 'DELETE',
      })
      if (!response.ok) {
        throw new Error(`Failed to delete draft: ${response.statusText}`)
      }
      await fetchDrafts()
      toast.success('Draft deleted successfully')
    } catch (err) {
      toast.error(`Error deleting draft: ${err instanceof Error ? err.message : 'Unknown error'}`)
    }
  }

  const routeFilterActive = normalizedRouteName !== '' && filter.toLowerCase() === normalizedRouteName

  const filteredDrafts = useMemo(() => {
    const searchLower = filter.toLowerCase()
    return drafts.filter((draft) => {
      const matchesSearch =
        draft.entity_name.toLowerCase().includes(searchLower) ||
        draft.entity_type.toLowerCase().includes(searchLower) ||
        (draft.owner && draft.owner.toLowerCase().includes(searchLower))
      const matchesRouteType =
        !normalizedRouteType || !routeFilterActive || draft.entity_type.toLowerCase() === normalizedRouteType
      const matchesRouteName =
        !normalizedRouteName || !routeFilterActive || draft.entity_name.toLowerCase() === normalizedRouteName

      return matchesSearch && matchesRouteType && matchesRouteName
    })
  }, [drafts, filter, normalizedRouteName, normalizedRouteType, routeFilterActive])

  const draftMetrics = useMemo(() => {
    return calculateDraftMetrics(selectedDraft?.content ?? '')
  }, [selectedDraft?.content])

  if (loading && !refreshing) {
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

  const shouldShowDraftEditor = Boolean(selectedDraft)
  const shouldShowDraftNotFound =
    Boolean(routeEntityName && decodedRouteName) && !selectedDraft && !loading && !refreshing
  const isDetailRoute = routeEntityName !== undefined

  const listContent = filteredDrafts.length === 0 ? (
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
      {filteredDrafts.map((draft) => {
        const encodedName = encodeURIComponent(draft.entity_name)
        const draftPath = `/draft/${draft.entity_type}/${encodedName}`
        const isSelected = selectedDraft?.id === draft.id
        const cardClassNames = ['catalog-card', 'catalog-card--clickable']
        if (isSelected) {
          cardClassNames.push('catalog-card--selected')
        }

        return (
          <div
            key={draft.id}
            className={cardClassNames.join(' ')}
            role="button"
            tabIndex={0}
            aria-pressed={isSelected}
            onClick={() => handleSelectDraft(draft)}
            onKeyDown={(event) => handleCardKeyDown(event, draft)}
          >
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
                  <span>{formatFileSize(draft.content.length)} KB</span>
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
                  to={draftPath}
                  className="btn-link"
                  onClick={(event) => event.stopPropagation()}
                >
                  Edit
                </Link>
                <button
                  className="btn-link btn-danger"
                  onClick={(event) => {
                    event.stopPropagation()
                    handleDelete(draft.id)
                  }}
                  title="Delete draft"
                >
                  <Trash2 size={16} />
                </button>
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )

  const detailContent = shouldShowDraftNotFound ? (
    <section className="draft-editor draft-editor--empty">
      <div className="draft-editor__breadcrumbs">
        <Link to="/drafts">Drafts</Link>
        <span aria-hidden="true">/</span>
        <span>{routeEntityType}</span>
        <span aria-hidden="true">/</span>
        <span>{decodedRouteName}</span>
      </div>
      <div className="draft-editor__empty-message">
        <AlertTriangle size={24} />
        <div>
          <h2>Draft not found</h2>
          <p>
            We could not locate a draft for <strong>{decodedRouteName}</strong>. Double-check the draft URL or return to
            the draft list.
          </p>
        </div>
      </div>
      <Link to="/drafts" className="draft-editor__button draft-editor__button--secondary">
        Back to drafts
      </Link>
    </section>
  ) : shouldShowDraftEditor && selectedDraft ? (
    <>
      <section className="draft-editor" aria-labelledby="draft-editor-heading">
        <div className="draft-editor__breadcrumbs">
          <Link to="/drafts">Drafts</Link>
          <span aria-hidden="true">/</span>
          <span>{selectedDraft.entity_type}</span>
          <span aria-hidden="true">/</span>
          <span>{selectedDraft.entity_name}</span>
        </div>

        <header className="draft-editor__header">
          <div className="draft-editor__title">
            <div className="draft-editor__title-group">
              <h2 id="draft-editor-heading">{selectedDraft.entity_name}</h2>
              <button
                type="button"
                className="draft-editor__info-button"
                onClick={openMetaDialog}
                aria-label="View supplemental draft details"
                aria-haspopup="dialog"
                aria-expanded={metaDialogOpen}
                aria-controls="draft-info-dialog"
              >
                <Info size={18} aria-hidden="true" />
              </button>
            </div>
            <p className="draft-editor__subtitle">
              {selectedDraft.status === 'draft' ? 'Draft in progress' : selectedDraft.status}
            </p>
          </div>
          <div className="draft-editor__badges">
            <span className={`type-badge ${selectedDraft.entity_type}`}>
              {selectedDraft.entity_type}
            </span>
          </div>
        </header>

        <div className="draft-editor__toolbar">
          <button
            type="button"
            className="draft-editor__button draft-editor__button--primary"
            onClick={handleSaveDraft}
            disabled={saving || !hasUnsavedChanges}
          >
            {saving ? <Loader2 size={16} className="icon-spin" /> : <Save size={16} />}
            <span>Save draft</span>
          </button>
          <button
            type="button"
            className="draft-editor__button"
            onClick={handleDiscardChanges}
            disabled={!hasUnsavedChanges}
          >
            <RotateCcw size={16} />
            <span>Discard changes</span>
          </button>
          <button
            type="button"
            className="draft-editor__button"
            onClick={handleRefreshDraft}
            disabled={refreshing}
          >
            {refreshing ? <Loader2 size={16} className="icon-spin" /> : <RefreshCcw size={16} />}
            <span>Refresh</span>
          </button>
          <div className="draft-editor__spacer" aria-hidden="true" />
          <div className="draft-editor__view-toggle" role="group" aria-label="Editor view mode">
            <button
              type="button"
              className={`draft-editor__toggle ${viewMode === ViewModes.EDIT ? 'draft-editor__toggle--active' : ''}`}
              onClick={() => setViewMode(ViewModes.EDIT)}
            >
              Edit
            </button>
            <button
              type="button"
              className={`draft-editor__toggle ${viewMode === ViewModes.SPLIT ? 'draft-editor__toggle--active' : ''}`}
              onClick={() => setViewMode(ViewModes.SPLIT)}
            >
              Split
            </button>
            <button
              type="button"
              className={`draft-editor__toggle ${viewMode === ViewModes.PREVIEW ? 'draft-editor__toggle--active' : ''}`}
              onClick={() => setViewMode(ViewModes.PREVIEW)}
            >
              Preview
            </button>
          </div>
        </div>

        {saveStatus && (
          <div className={`draft-editor__status draft-editor__status--${saveStatus.type}`}>
            {saveStatus.type === 'success' ? <CheckCircle2 size={18} /> : <AlertTriangle size={18} />}
            <span>{saveStatus.message}</span>
          </div>
        )}

        <div className={`draft-editor__panes draft-editor__panes--${viewMode}`}>
          <div className="draft-editor__pane draft-editor__pane--editor">
            <label className="draft-editor__label" htmlFor="draft-content">
              Markdown source
            </label>
            <textarea
              id="draft-content"
              className="draft-editor__textarea"
              value={editorContent}
              onChange={handleEditorChange}
              spellCheck={false}
            />
            <p className="draft-editor__hint">
              Tip: Use markdown headings (e.g. # Overview) to match the PRD structure.
            </p>
          </div>
          <div className="draft-editor__pane draft-editor__pane--preview">
            <span className="draft-editor__label">Live preview</span>
            <div className="draft-preview">
              {editorContent.trim().length === 0 ? (
                <p className="draft-preview__empty">Start editing to see a formatted preview.</p>
              ) : (
                <div className="draft-preview__content">
                  <ReactMarkdown>{editorContent}</ReactMarkdown>
                </div>
              )}
            </div>
          </div>
        </div>
      </section>
      {metaDialogOpen && (
        <div className="draft-info-overlay" onClick={closeMetaDialog}>
          <div
            className="draft-info-dialog"
            role="dialog"
            aria-modal="true"
            aria-labelledby="draft-info-dialog-title"
            aria-describedby="draft-info-dialog-description"
            onClick={(event) => event.stopPropagation()}
            id="draft-info-dialog"
          >
            <div className="draft-info-dialog__header">
              <div>
                <h3 id="draft-info-dialog-title">Draft details</h3>
                <p id="draft-info-dialog-description" className="draft-info-dialog__subtitle">
                  Supplemental metadata and helpful stats for <strong>{selectedDraft.entity_name}</strong>.
                </p>
              </div>
              <button
                type="button"
                className="draft-info-dialog__close"
                onClick={closeMetaDialog}
                aria-label="Close draft details"
              >
                <X size={18} aria-hidden="true" />
              </button>
            </div>

            <dl className="draft-info-dialog__grid">
              <div className="draft-info-dialog__entry">
                <dt>Last updated</dt>
                <dd>{formatDate(selectedDraft.updated_at)}</dd>
              </div>
              <div className="draft-info-dialog__entry">
                <dt>Created</dt>
                <dd>{formatDate(selectedDraft.created_at)}</dd>
              </div>
              <div className="draft-info-dialog__entry">
                <dt>Status</dt>
                <dd>{selectedDraft.status}</dd>
              </div>
              <div className="draft-info-dialog__entry">
                <dt>Type</dt>
                <dd>{selectedDraft.entity_type}</dd>
              </div>
              {selectedDraft.owner && (
                <div className="draft-info-dialog__entry">
                  <dt>Owner</dt>
                  <dd>{selectedDraft.owner}</dd>
                </div>
              )}
              <div className="draft-info-dialog__entry">
                <dt>File size</dt>
                <dd>{draftMetrics.sizeKb.toFixed(1)} KB</dd>
              </div>
              <div className="draft-info-dialog__entry">
                <dt>Word count</dt>
                <dd>{draftMetrics.wordCount.toLocaleString()}</dd>
              </div>
              <div className="draft-info-dialog__entry">
                <dt>Character count</dt>
                <dd>{draftMetrics.characterCount.toLocaleString()}</dd>
              </div>
              {draftMetrics.estimatedReadMinutes > 0 && (
                <div className="draft-info-dialog__entry">
                  <dt>Estimated read time</dt>
                  <dd>
                    {draftMetrics.estimatedReadMinutes}{' '}
                    {draftMetrics.estimatedReadMinutes === 1 ? 'minute' : 'minutes'}
                  </dd>
                </div>
              )}
            </dl>
          </div>
        </div>
      )}
    </>
  ) : null

  if (isDetailRoute) {
    return (
      <div className="container">
        {detailContent}

        <div className="back-link-container">
          <Link to="/drafts" className="back-link">
            ← Back to Drafts
          </Link>
          <Link to="/" className="back-link">
            ← Back to Catalog
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="container">
      <header className="page-header">
        <h1 className="page-title">
          <span className="page-title__icon" aria-hidden="true">
            <FileEdit size={28} strokeWidth={2.5} />
          </span>
          <span>Draft PRDs</span>
        </h1>
        <p className="subtitle">
          Manage and edit PRD drafts before publishing
        </p>
      </header>

      <div className="controls">
        <div className="search-field">
          <label htmlFor="drafts-search" className="sr-only">
            Search drafts
          </label>
          <Search className="search-field__icon" size={18} aria-hidden="true" />
          <input
            id="drafts-search"
            type="text"
            className="search-input"
            placeholder="Search drafts by name, type, or owner..."
            value={filter}
            onChange={(event) => setFilter(event.target.value)}
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
      </div>

      <div className="stats">
        <div className="stat-item">
          <span className="stat-label">Total Drafts</span>
          <span className="stat-value">{drafts.length}</span>
        </div>
        <div className="stat-item">
          <span className="stat-label">Scenarios</span>
          <span className="stat-value">
            {drafts.filter((draft) => draft.entity_type === 'scenario').length}
          </span>
        </div>
        <div className="stat-item">
          <span className="stat-label">Resources</span>
          <span className="stat-value">
            {drafts.filter((draft) => draft.entity_type === 'resource').length}
          </span>
        </div>
      </div>

      <div className="drafts-layout">
        <div className="drafts-layout__list">{listContent}</div>
      </div>

      <div className="back-link-container">
        <Link to="/backlog" className="back-link">
          ← Go to Backlog
        </Link>
        <Link to="/" className="back-link">
          ← Back to Catalog
        </Link>
      </div>
    </div>
  )
}
