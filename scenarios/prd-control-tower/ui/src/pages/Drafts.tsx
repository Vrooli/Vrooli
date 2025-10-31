import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ChangeEvent, KeyboardEvent, ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
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
} from 'lucide-react'
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

type ViewMode = 'split' | 'edit' | 'preview'

interface SaveStatus {
  type: 'success' | 'error'
  message: string
}

const decodeRouteSegment = (value?: string) => {
  if (!value) {
    return ''
  }
  try {
    return decodeURIComponent(value)
  } catch (error) {
    console.warn('[Drafts] Failed to decode route segment', { value, error })
    return value
  }
}

export default function Drafts() {
  const { entityType: routeEntityType, entityName: routeEntityName } = useParams<{
    entityType?: string
    entityName?: string
  }>()
  const navigate = useNavigate()

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
  const [viewMode, setViewMode] = useState<ViewMode>('split')

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
      setViewMode('split')
      return
    }

    const selectionChanged = selectedDraft.id !== selectedDraftId
    if (selectionChanged) {
      setSelectedDraftId(selectedDraft.id)
      setEditorContent(selectedDraft.content)
      setLastLoadedContent(selectedDraft.content)
      setHasUnsavedChanges(false)
      setSaveStatus(null)
      setViewMode('split')
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

  const handleEditorChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    const value = event.target.value
    setEditorContent(value)
    if (selectedDraft) {
      setHasUnsavedChanges(value !== lastLoadedContent)
    }
  }

  const handleDiscardChanges = () => {
    if (!selectedDraft) {
      return
    }
    if (hasUnsavedChanges && !confirm('Discard unsaved changes?')) {
      return
    }

    setEditorContent(lastLoadedContent)
    setHasUnsavedChanges(false)
  }

  const handleRefreshDraft = async () => {
    if (!selectedDraft) {
      return
    }
    if (hasUnsavedChanges) {
      const shouldContinue = confirm('Refreshing will discard unsaved changes. Continue?')
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
      await fetchDrafts()
    } catch (err) {
      alert(`Error deleting draft: ${err instanceof Error ? err.message : 'Unknown error'}`)
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

  const previewElements = useMemo(() => buildMarkdownElements(editorContent), [editorContent])

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
    <section className="draft-editor" aria-labelledby="draft-editor-heading">
      <div className="draft-editor__breadcrumbs">
        <Link to="/drafts">Drafts</Link>
        <span aria-hidden="true">/</span>
        <span>{selectedDraft.entity_type}</span>
        <span aria-hidden="true">/</span>
        <span>{selectedDraft.entity_name}</span>
      </div>

      <header className="draft-editor__header">
        <div>
          <h2 id="draft-editor-heading">{selectedDraft.entity_name}</h2>
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

      <div className="draft-editor__meta-grid">
        <div className="draft-editor__meta-item">
          <span className="draft-editor__meta-label">Last updated</span>
          <span className="draft-editor__meta-value">{formatDate(selectedDraft.updated_at)}</span>
        </div>
        <div className="draft-editor__meta-item">
          <span className="draft-editor__meta-label">Created</span>
          <span className="draft-editor__meta-value">{formatDate(selectedDraft.created_at)}</span>
        </div>
        {selectedDraft.owner && (
          <div className="draft-editor__meta-item">
            <span className="draft-editor__meta-label">Owner</span>
            <span className="draft-editor__meta-value">{selectedDraft.owner}</span>
          </div>
        )}
        <div className="draft-editor__meta-item">
          <span className="draft-editor__meta-label">Size</span>
          <span className="draft-editor__meta-value">{(selectedDraft.content.length / 1024).toFixed(1)} KB</span>
        </div>
      </div>

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
            className={`draft-editor__toggle ${viewMode === 'edit' ? 'draft-editor__toggle--active' : ''}`}
            onClick={() => setViewMode('edit')}
          >
            Edit
          </button>
          <button
            type="button"
            className={`draft-editor__toggle ${viewMode === 'split' ? 'draft-editor__toggle--active' : ''}`}
            onClick={() => setViewMode('split')}
          >
            Split
          </button>
          <button
            type="button"
            className={`draft-editor__toggle ${viewMode === 'preview' ? 'draft-editor__toggle--active' : ''}`}
            onClick={() => setViewMode('preview')}
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
              <div className="draft-preview__content">{previewElements}</div>
            )}
          </div>
        </div>
      </div>
    </section>
  ) : null

  if (isDetailRoute) {
    return (
      <div className="container">
        <header className="page-header">
          <h1 className="page-title">
            <span className="page-title__icon" aria-hidden="true">
              <FileEdit size={28} strokeWidth={2.5} />
            </span>
            <span>Draft PRDs</span>
          </h1>
          <p className="subtitle">Focused view for a single draft</p>
        </header>

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
        <Link to="/" className="back-link">
          ← Back to Catalog
        </Link>
      </div>
    </div>
  )
}

function buildMarkdownElements(markdown: string): ReactNode[] {
  const lines = markdown.split(/\r?\n/)
  const elements: ReactNode[] = []
  let keyCounter = 0
  const nextKey = () => `md-${keyCounter++}`

  let paragraphLines: string[] = []
  let unorderedItems: string[] = []
  let orderedItems: string[] = []
  let inCodeBlock = false
  let codeLines: string[] = []
  let codeLanguage = ''

  const flushParagraph = () => {
    if (!paragraphLines.length) {
      return
    }
    const text = paragraphLines.join(' ')
    elements.push(
      <p key={nextKey()}>{renderInlineMarkdown(text, nextKey)}</p>,
    )
    paragraphLines = []
  }

  const flushUnordered = () => {
    if (!unorderedItems.length) {
      return
    }
    elements.push(
      <ul key={nextKey()}>
        {unorderedItems.map((item) => (
          <li key={nextKey()}>{renderInlineMarkdown(item, nextKey)}</li>
        ))}
      </ul>,
    )
    unorderedItems = []
  }

  const flushOrdered = () => {
    if (!orderedItems.length) {
      return
    }
    elements.push(
      <ol key={nextKey()}>
        {orderedItems.map((item) => (
          <li key={nextKey()}>{renderInlineMarkdown(item, nextKey)}</li>
        ))}
      </ol>,
    )
    orderedItems = []
  }

  const flushLists = () => {
    flushUnordered()
    flushOrdered()
  }

  const flushCodeBlock = () => {
    if (!codeLines.length) {
      return
    }
    elements.push(
      <pre key={nextKey()}>
        <code>
          {codeLanguage ? `${codeLanguage}\n` : ''}
          {codeLines.join('\n')}
        </code>
      </pre>,
    )
    codeLines = []
    codeLanguage = ''
    inCodeBlock = false
  }

  lines.forEach((line) => {
    const trimmed = line.trim()

    if (trimmed.startsWith('```')) {
      if (inCodeBlock) {
        flushCodeBlock()
      } else {
        flushParagraph()
        flushLists()
        inCodeBlock = true
        codeLanguage = trimmed.slice(3).trim()
      }
      return
    }

    if (inCodeBlock) {
      codeLines.push(line)
      return
    }

    if (/^#{1,6} /.test(trimmed)) {
      flushParagraph()
      flushLists()
      const match = trimmed.match(/^#{1,6}/)
      const level = match ? match[0].length : 1
      const content = trimmed.slice(level).trim()
      const Tag = (`h${Math.min(level, 6)}`) as keyof JSX.IntrinsicElements
      elements.push(
        <Tag key={nextKey()}>{renderInlineMarkdown(content, nextKey)}</Tag>,
      )
      return
    }

    if (/^[-*] /.test(trimmed)) {
      flushParagraph()
      flushOrdered()
      unorderedItems.push(trimmed.slice(2).trim())
      return
    }

    if (/^\d+[.)] /.test(trimmed)) {
      flushParagraph()
      flushUnordered()
      orderedItems.push(trimmed.replace(/^\d+[.)]\s*/, '').trim())
      return
    }

    if (trimmed === '') {
      flushParagraph()
      flushLists()
      return
    }

    paragraphLines.push(line)
  })

  if (inCodeBlock) {
    flushCodeBlock()
  }

  flushParagraph()
  flushLists()

  return elements
}

function renderInlineMarkdown(text: string, nextKey: () => string): ReactNode[] {
  const nodes: ReactNode[] = []
  const pattern = /(\[[^\]]+\]\([^\)]+\)|\*\*[^*]+\*\*|__[^_]+__|`[^`]+`|\*[^*]+\*|_[^_]+_|~~[^~]+~~)/g
  let lastIndex = 0

  for (const match of text.matchAll(pattern)) {
    const fullMatch = match[0]
    const index = match.index ?? 0

    if (index > lastIndex) {
      nodes.push(text.slice(lastIndex, index))
    }

    if (fullMatch.startsWith('[')) {
      const labelMatch = fullMatch.match(/^\[([^\]]+)\]\(([^\)]+)\)$/)
      if (labelMatch) {
        nodes.push(
          <a key={nextKey()} href={labelMatch[2]} target="_blank" rel="noreferrer">
            {labelMatch[1]}
          </a>,
        )
      } else {
        nodes.push(fullMatch)
      }
    } else if ((fullMatch.startsWith('**') && fullMatch.endsWith('**')) || (fullMatch.startsWith('__') && fullMatch.endsWith('__'))) {
      nodes.push(<strong key={nextKey()}>{fullMatch.slice(2, -2)}</strong>)
    } else if ((fullMatch.startsWith('*') && fullMatch.endsWith('*')) || (fullMatch.startsWith('_') && fullMatch.endsWith('_'))) {
      nodes.push(<em key={nextKey()}>{fullMatch.slice(1, -1)}</em>)
    } else if (fullMatch.startsWith('~~') && fullMatch.endsWith('~~')) {
      nodes.push(<del key={nextKey()}>{fullMatch.slice(2, -2)}</del>)
    } else if (fullMatch.startsWith('`') && fullMatch.endsWith('`')) {
      nodes.push(<code key={nextKey()}>{fullMatch.slice(1, -1)}</code>)
    }

    lastIndex = index + fullMatch.length
  }

  if (lastIndex < text.length) {
    nodes.push(text.slice(lastIndex))
  }

  return nodes.length > 0 ? nodes : [text]
}
