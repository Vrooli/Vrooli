import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import toast from 'react-hot-toast'
import {
  StickyNote,
  ListChecks,
  Sparkles,
  Loader2,
  CheckCircle2,
  Trash2,
  Inbox,
  PlusCircle,
  ClipboardList,
} from 'lucide-react'
import { buildApiUrl } from '../utils/apiClient'
import { useConfirm } from '../utils/confirmDialog'
import { formatDate } from '../utils/formatters'
import { parseBacklogInput, PendingIdea, slugifyIdea } from '../utils/backlog'
import type { BacklogEntry, BacklogCreateResponse, BacklogConvertResponse, EntityType } from '../types'

const entityOptions: Array<{ value: EntityType; label: string }> = [
  { value: 'scenario', label: 'Scenario' },
  { value: 'resource', label: 'Resource' },
]

type PreviewOverride = Partial<Pick<PendingIdea, 'entityType' | 'suggestedName'>>

type PreviewActionMode = 'save' | 'convert'

function sanitizeSlugInput(value: string): string {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-+|-+$/g, '')
}

export default function Backlog() {
  const confirm = useConfirm()
  const [rawInput, setRawInput] = useState('')
  const [previewOverrides, setPreviewOverrides] = useState<Record<string, PreviewOverride>>({})
  const [previewSelection, setPreviewSelection] = useState<Set<string>>(() => new Set())
  const [backlogEntries, setBacklogEntries] = useState<BacklogEntry[]>([])
  const [backlogSelection, setBacklogSelection] = useState<Set<string>>(() => new Set())
  const [listFilter, setListFilter] = useState('')
  const [loadingBacklog, setLoadingBacklog] = useState(true)
  const [previewBusy, setPreviewBusy] = useState(false)
  const [previewBusyId, setPreviewBusyId] = useState<string | null>(null)
  const [backlogBusy, setBacklogBusy] = useState(false)
  const [backlogBusyId, setBacklogBusyId] = useState<string | null>(null)

  const parsedIdeas = useMemo(() => parseBacklogInput(rawInput, 'scenario'), [rawInput])

  useEffect(() => {
    const validIds = new Set(parsedIdeas.map((idea) => idea.id))
    setPreviewOverrides((prev) => {
      const next: Record<string, PreviewOverride> = {}
      parsedIdeas.forEach((idea) => {
        if (prev[idea.id]) {
          next[idea.id] = prev[idea.id]
        }
      })
      return next
    })
    setPreviewSelection((prev) => {
      const next = new Set<string>()
      prev.forEach((id) => {
        if (validIds.has(id)) {
          next.add(id)
        }
      })
      return next
    })
  }, [parsedIdeas])

  const previewIdeas = useMemo(() => {
    return parsedIdeas.map((idea) => {
      const override = previewOverrides[idea.id]
      if (!override) {
        return idea
      }
      return {
        ...idea,
        ...override,
      }
    })
  }, [parsedIdeas, previewOverrides])

  const fetchBacklog = useCallback(async () => {
    setLoadingBacklog(true)
    try {
      const response = await fetch(buildApiUrl('/backlog'))
      if (!response.ok) {
        throw new Error('Unable to load backlog items')
      }
      const data: { entries: BacklogEntry[] } = await response.json()
      setBacklogEntries(data.entries || [])
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to load backlog entries'
      toast.error(message)
    } finally {
      setLoadingBacklog(false)
    }
  }, [])

  useEffect(() => {
    fetchBacklog()
  }, [fetchBacklog])

  const filteredBacklogEntries = useMemo(() => {
    if (!listFilter.trim()) {
      return backlogEntries
    }
    const search = listFilter.toLowerCase()
    return backlogEntries.filter((entry) => {
      return (
        entry.idea_text.toLowerCase().includes(search) ||
        entry.suggested_name.toLowerCase().includes(search) ||
        entry.entity_type.toLowerCase().includes(search)
      )
    })
  }, [backlogEntries, listFilter])

  const handlePreviewSelection = useCallback((id: string) => {
    setPreviewSelection((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }, [])

  const toggleSelectAllPreview = useCallback(() => {
    if (previewSelection.size === previewIdeas.length) {
      setPreviewSelection(new Set())
      return
    }
    setPreviewSelection(new Set(previewIdeas.map((idea) => idea.id)))
  }, [previewSelection.size, previewIdeas])

  const updateOverride = useCallback((id: string, patch: PreviewOverride) => {
    setPreviewOverrides((prev) => ({
      ...prev,
      [id]: {
        ...prev[id],
        ...patch,
      },
    }))
  }, [])

  const createBacklogEntries = useCallback(async (ideas: PendingIdea[]) => {
    const payload = {
      entries: ideas.map((idea) => ({
        idea_text: idea.ideaText,
        entity_type: idea.entityType,
        suggested_name: idea.suggestedName || slugifyIdea(idea.ideaText),
      })),
    }

    const response = await fetch(buildApiUrl('/backlog'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })

    if (!response.ok) {
      const message = await response.text()
      throw new Error(message || 'Failed to add backlog items')
    }

    const data: BacklogCreateResponse = await response.json()
    return data.entries
  }, [])

  const convertEntries = useCallback(async (ids: string[]) => {
    const response = await fetch(buildApiUrl('/backlog/convert'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ entry_ids: ids }),
    })

    if (!response.ok) {
      const message = await response.text()
      throw new Error(message || 'Failed to convert backlog entries')
    }

    const data: BacklogConvertResponse = await response.json()
    return data.results || []
  }, [])

  const handlePreviewAction = useCallback(
    async (mode: PreviewActionMode, explicitIdea?: PendingIdea) => {
      const ideas = explicitIdea ? [explicitIdea] : previewIdeas.filter((idea) => previewSelection.has(idea.id))
      if (ideas.length === 0) {
        toast.error('Select at least one idea to continue')
        return
      }

      setPreviewBusy(true)
      setPreviewBusyId(explicitIdea ? explicitIdea.id : null)
      try {
        const created = await createBacklogEntries(ideas)
        if (mode === 'convert') {
          const results = await convertEntries(created.map((entry) => entry.id))
          const successes = results.filter((result) => !result.error)
          const failures = results.length - successes.length
          if (successes.length) {
            toast.success(`Created ${successes.length} draft${successes.length === 1 ? '' : 's'} from backlog ideas`)
          }
          if (failures > 0) {
            toast.error(`${failures} item${failures === 1 ? '' : 's'} failed to convert. Check logs for details.`)
          }
        } else {
          toast.success(`Added ${created.length} idea${created.length === 1 ? '' : 's'} to backlog`)
        }
        await fetchBacklog()
        if (!explicitIdea) {
          setPreviewSelection(new Set())
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Unexpected backlog error'
        toast.error(message)
      } finally {
        setPreviewBusy(false)
        setPreviewBusyId(null)
      }
    },
    [convertEntries, createBacklogEntries, fetchBacklog, previewIdeas, previewSelection],
  )

  const handleBacklogSelection = useCallback((id: string) => {
    setBacklogSelection((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }, [])

  const toggleBacklogSelectAll = useCallback(() => {
    if (backlogSelection.size === filteredBacklogEntries.length) {
      setBacklogSelection(new Set())
      return
    }
    setBacklogSelection(new Set(filteredBacklogEntries.map((entry) => entry.id)))
  }, [backlogSelection.size, filteredBacklogEntries])

  const convertExistingEntries = useCallback(
    async (ids?: string[]) => {
      const targets = ids ?? Array.from(backlogSelection)
      if (targets.length === 0) {
        toast.error('Select backlog entries to convert')
        return
      }

      setBacklogBusy(true)
      setBacklogBusyId(ids && ids.length === 1 ? ids[0] : null)
      try {
        const results = await convertEntries(targets)
        const successes = results.filter((result) => !result.error)
        const failures = results.filter((result) => result.error)
        if (successes.length) {
          toast.success(`Converted ${successes.length} backlog item${successes.length === 1 ? '' : 's'}`)
        }
        if (failures.length) {
          toast.error(`${failures.length} item${failures.length === 1 ? '' : 's'} failed to convert`)
        }
        await fetchBacklog()
        setBacklogSelection(new Set())
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to convert backlog entries'
        toast.error(message)
      } finally {
        setBacklogBusy(false)
        setBacklogBusyId(null)
      }
    },
    [backlogSelection, convertEntries, fetchBacklog],
  )

  const deleteEntries = useCallback(
    async (ids?: string[]) => {
      const targets = ids ?? Array.from(backlogSelection)
      if (targets.length === 0) {
        toast.error('Select backlog entries to delete')
        return
      }

      const confirmed = await confirm({
        title: targets.length === 1 ? 'Delete backlog idea?' : `Delete ${targets.length} backlog ideas?`,
        message: 'This will permanently remove the ideas from your backlog. Drafts that already exist will be untouched.',
        confirmText: 'Delete',
        variant: 'danger',
      })

      if (!confirmed) {
        return
      }

      setBacklogBusy(true)
      setBacklogBusyId(targets.length === 1 ? targets[0] : null)
      try {
        await Promise.all(
          targets.map((id) =>
            fetch(buildApiUrl(`/backlog/${id}`), {
              method: 'DELETE',
            }),
          ),
        )
        toast.success(`Removed ${targets.length} backlog item${targets.length === 1 ? '' : 's'}`)
        await fetchBacklog()
        setBacklogSelection(new Set())
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to delete backlog entries'
        toast.error(message)
      } finally {
        setBacklogBusy(false)
        setBacklogBusyId(null)
      }
    },
    [backlogSelection, confirm, fetchBacklog],
  )

  const renderStatusBadge = (entry: BacklogEntry) => {
    const status = entry.status
    if (status === 'converted') {
      return (
        <span className="status-badge status-badge--success">
          <CheckCircle2 size={14} />
          Converted
        </span>
      )
    }
    if (status === 'archived') {
      return (
        <span className="status-badge status-badge--muted">Archived</span>
      )
    }
    return (
      <span className="status-badge status-badge--pending">Pending</span>
    )
  }

  const previewSelectedCount = previewSelection.size
  const backlogSelectedCount = backlogSelection.size

  return (
    <div className="container backlog-page">
      <header className="page-header page-header--catalog">
        <span className="page-eyebrow">Idea backlog</span>
        <div className="page-header__body">
          <div className="page-header__text">
            <h1 className="page-title">
              <span className="page-title__icon" aria-hidden="true">
                <StickyNote size={28} strokeWidth={2.5} />
              </span>
              <span>Scenario Backlog</span>
            </h1>
            <p className="subtitle">Paste raw notes, triage ideas, and convert them into drafts without leaving PRD Control Tower.</p>
          </div>
          <div className="page-header__actions">
            <Link to="/" className="nav-button">
              <ClipboardList size={18} aria-hidden="true" />
              <span className="nav-button__label">
                <strong>Catalog</strong>
                <small>Browse coverage</small>
              </span>
            </Link>
            <Link to="/drafts" className="nav-button nav-button--secondary">
              <ListChecks size={18} aria-hidden="true" />
              <span className="nav-button__label">
                <strong>Drafts</strong>
                <small>Edit PRDs</small>
              </span>
            </Link>
          </div>
        </div>
        <div className="insight-banner" role="status" aria-live="polite">
          <span className="insight-banner__icon" aria-hidden="true">
            <Sparkles size={18} />
          </span>
          <div>
            <p className="insight-banner__title">Idea intake</p>
            <p className="insight-banner__body">
              Paste bullet lists or note dumps on the left. Preview, batch select, and spin up drafts from the right column.
            </p>
          </div>
        </div>
      </header>

      <section className="backlog-intake">
        <div className="backlog-intake__column">
          <div className="surface-card backlog-card">
            <header className="backlog-card__header">
              <div>
                <h2>Freeform intake</h2>
                <p>Drop in any list of scenario or resource ideas. We'll auto-split, clean bullets, and prep slugs.</p>
              </div>
              <PlusCircle size={24} aria-hidden="true" />
            </header>
            <textarea
              className="backlog-textarea"
              placeholder="Example:\n- Scenario blueprint generator\n- [resource] Shared vector cache\n1. Portfolio ops cockpit"
              value={rawInput}
              onChange={(event) => setRawInput(event.target.value)}
            />
            <div className="backlog-textarea__footer">
              <span>{previewIdeas.length} idea{previewIdeas.length === 1 ? '' : 's'} detected</span>
              {rawInput && (
                <button type="button" className="btn-link" onClick={() => setRawInput('')}>
                  Clear input
                </button>
              )}
            </div>
          </div>
        </div>
        <div className="backlog-intake__column">
          <div className="surface-card backlog-card">
            <header className="backlog-card__header">
              <div>
                <h2>Live preview</h2>
                <p>Select entries to save in backlog or instantly create drafts.</p>
              </div>
              <div className="preview-actions">
                <button type="button" className="preview-select" onClick={toggleSelectAllPreview}>
                  {previewSelection.size === previewIdeas.length ? 'Clear selection' : 'Select all'}
                </button>
                <span className="preview-count">{previewSelectedCount} selected</span>
              </div>
            </header>
            {previewIdeas.length === 0 ? (
              <div className="backlog-empty">
                <Inbox size={42} />
                <p>Paste ideas to see them parsed here.</p>
              </div>
            ) : (
              <div className="backlog-preview-list">
                {previewIdeas.map((idea) => {
                  const isSelected = previewSelection.has(idea.id)
                  const busy = previewBusy && previewBusyId === idea.id
                  return (
                    <div key={idea.id} className={`backlog-preview-item${isSelected ? ' backlog-preview-item--selected' : ''}`}>
                      <label className="backlog-preview-item__checkbox">
                        <input
                          type="checkbox"
                          checked={isSelected}
                          onChange={() => handlePreviewSelection(idea.id)}
                        />
                      </label>
                      <div className="backlog-preview-item__body">
                        <p className="backlog-preview-item__text">{idea.ideaText}</p>
                        <div className="backlog-preview-item__meta">
                          <select
                            value={idea.entityType}
                            onChange={(event) => updateOverride(idea.id, { entityType: event.target.value as EntityType })}
                          >
                            {entityOptions.map((option) => (
                              <option key={option.value} value={option.value}>
                                {option.label}
                              </option>
                            ))}
                          </select>
                          <input
                            type="text"
                            className="backlog-preview-item__slug"
                            value={idea.suggestedName}
                            onChange={(event) => updateOverride(idea.id, { suggestedName: sanitizeSlugInput(event.target.value) })}
                            placeholder="slug"
                          />
                          <button
                            type="button"
                            className="backlog-preview-item__action"
                            onClick={() => handlePreviewAction('convert', idea)}
                            disabled={previewBusy && !busy}
                          >
                            {busy ? <Loader2 size={16} className="icon-spin" /> : <CheckCircle2 size={16} />}
                            Create draft
                          </button>
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
            <footer className="backlog-preview-footer">
              <button
                type="button"
                className="backlog-preview-footer__button"
                disabled={previewIdeas.length === 0 || previewBusy}
                onClick={() => handlePreviewAction('save')}
              >
                {previewBusy && previewBusyId === null && <Loader2 size={16} className="icon-spin" />}
                Save selected to backlog
              </button>
              <button
                type="button"
                className="backlog-preview-footer__button backlog-preview-footer__button--primary"
                disabled={previewIdeas.length === 0 || previewBusy}
                onClick={() => handlePreviewAction('convert')}
              >
                {previewBusy && previewBusyId === null && <Loader2 size={16} className="icon-spin" />}
                Create drafts for selected
              </button>
            </footer>
          </div>
        </div>
      </section>

      <section className="surface-card backlog-table-card">
        <header className="backlog-table-card__header">
          <div>
            <h2>Backlog queue</h2>
            <p>{backlogEntries.length} captured idea{backlogEntries.length === 1 ? '' : 's'}</p>
          </div>
          <div className="backlog-table-actions">
            <div className="search-field search-field--compact">
              <input
                type="text"
                placeholder="Filter backlog..."
                value={listFilter}
                onChange={(event) => setListFilter(event.target.value)}
              />
            </div>
            <button type="button" className="backlog-action" disabled={backlogBusy} onClick={() => fetchBacklog()}>
              Refresh
            </button>
            <button
              type="button"
              className="backlog-action backlog-action--primary"
              disabled={backlogBusy}
              onClick={() => convertExistingEntries()}
            >
              {backlogBusy && backlogBusyId === null ? <Loader2 size={16} className="icon-spin" /> : <CheckCircle2 size={16} />}
              Convert selected
            </button>
            <button
              type="button"
              className="backlog-action backlog-action--danger"
              disabled={backlogBusy}
              onClick={() => deleteEntries()}
            >
              <Trash2 size={16} /> Delete selected
            </button>
          </div>
        </header>
        {loadingBacklog ? (
          <div className="backlog-loading">
            <Loader2 size={28} className="icon-spin" /> Loading backlog...
          </div>
        ) : filteredBacklogEntries.length === 0 ? (
          <div className="backlog-empty">
            <Inbox size={42} />
            <p>No backlog ideas yet. Save them from the intake panel to keep track.</p>
          </div>
        ) : (
          <div className="backlog-table-wrapper">
            <table className="backlog-table">
              <thead>
                <tr>
                  <th>
                    <input
                      type="checkbox"
                      aria-label="Select all backlog entries"
                      checked={backlogSelection.size === filteredBacklogEntries.length}
                      onChange={toggleBacklogSelectAll}
                    />
                  </th>
                  <th>Idea</th>
                  <th>Type</th>
                  <th>Slug / Draft</th>
                  <th>Status</th>
                  <th>Created</th>
                  <th>Updated</th>
                  <th aria-label="Actions" />
                </tr>
              </thead>
              <tbody>
                {filteredBacklogEntries.map((entry) => {
                  const isSelected = backlogSelection.has(entry.id)
                  const busy = backlogBusy && backlogBusyId === entry.id
                  const encodedSlug = encodeURIComponent(entry.suggested_name)
                  const draftLink = entry.status === 'converted' ? `/draft/${entry.entity_type}/${encodedSlug}` : null
                  return (
                    <tr key={entry.id} className={isSelected ? 'backlog-row--selected' : undefined}>
                      <td>
                        <input
                          type="checkbox"
                          checked={isSelected}
                          onChange={() => handleBacklogSelection(entry.id)}
                        />
                      </td>
                      <td>
                        <p className="backlog-idea-text">{entry.idea_text}</p>
                      </td>
                      <td>
                        <span className={`type-badge ${entry.entity_type}`}>{entry.entity_type}</span>
                      </td>
                      <td>
                        <div className="backlog-slug">
                          <code>{entry.suggested_name}</code>
                          {draftLink && (
                            <Link to={draftLink} className="btn-link">
                              Open draft
                            </Link>
                          )}
                        </div>
                      </td>
                      <td>{renderStatusBadge(entry)}</td>
                      <td>{formatDate(entry.created_at)}</td>
                      <td>{formatDate(entry.updated_at)}</td>
                      <td className="backlog-row-actions">
                        <button
                          type="button"
                          className="btn-link"
                          disabled={busy}
                          onClick={() => convertExistingEntries([entry.id])}
                        >
                          {busy ? <Loader2 size={16} className="icon-spin" /> : <CheckCircle2 size={16} />}
                          Convert
                        </button>
                        <button
                          type="button"
                          className="btn-link btn-link--danger"
                          disabled={busy}
                          onClick={() => deleteEntries([entry.id])}
                        >
                          <Trash2 size={16} />
                          Delete
                        </button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
        <footer className="backlog-table-footer">
          <span>{backlogSelectedCount} selected</span>
          <div className="backlog-table-footer__links">
            <Link to="/" className="back-link">
              ← Back to Catalog
            </Link>
            <Link to="/drafts" className="back-link">
              ← Back to Drafts
            </Link>
          </div>
        </footer>
      </section>
    </div>
  )
}
