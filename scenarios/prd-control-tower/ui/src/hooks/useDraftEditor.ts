import { useCallback, useEffect, useMemo, useState } from 'react'
import { buildApiUrl } from '../utils/apiClient'
import { calculateDraftMetrics, type DraftMetrics } from '../utils/formatters'
import type { Draft, DraftSaveStatus, ViewMode } from '../types'
import { ViewModes } from '../types'

type ConfirmHandler = (options: {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  variant?: 'danger' | 'warning' | 'info'
}) => Promise<boolean>

interface UseDraftEditorOptions {
  selectedDraft: Draft | null
  fetchDrafts: (options?: { silent?: boolean }) => Promise<void>
  confirm: ConfirmHandler
}

interface UseDraftEditorReturn {
  viewMode: ViewMode
  setViewMode: (mode: ViewMode) => void
  editorContent: string
  handleEditorChange: (value: string) => void
  hasUnsavedChanges: boolean
  saving: boolean
  saveStatus: DraftSaveStatus | null
  handleSaveDraft: () => Promise<void>
  handleDiscardChanges: () => Promise<void>
  handleRefreshDraft: () => Promise<void>
  draftMetrics: DraftMetrics
}

export function useDraftEditor({
  selectedDraft,
  fetchDrafts,
  confirm,
}: UseDraftEditorOptions): UseDraftEditorReturn {
  const [editorContent, setEditorContent] = useState('')
  const [lastLoadedContent, setLastLoadedContent] = useState('')
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveStatus, setSaveStatus] = useState<DraftSaveStatus | null>(null)
  const [viewMode, setViewMode] = useState<ViewMode>(ViewModes.SPLIT)
  const [autosaveEnabled] = useState(true) // Could make this configurable
  const [lastAutosaveContent, setLastAutosaveContent] = useState('')

  // Sync editor content when draft changes
  useEffect(() => {
    if (!selectedDraft) {
      setEditorContent('')
      setLastLoadedContent('')
      setHasUnsavedChanges(false)
      setViewMode(ViewModes.SPLIT)
      return
    }

    setEditorContent(selectedDraft.content)
    setLastLoadedContent(selectedDraft.content)
    setHasUnsavedChanges(false)
    setSaveStatus(null)
    setViewMode(ViewModes.SPLIT)
  }, [selectedDraft?.id]) // Only re-run when draft ID changes

  // Update editor if draft content changes externally (but preserve unsaved changes)
  useEffect(() => {
    if (!selectedDraft || hasUnsavedChanges) {
      return
    }

    if (selectedDraft.content !== lastLoadedContent) {
      setEditorContent(selectedDraft.content)
      setLastLoadedContent(selectedDraft.content)
    }
  }, [selectedDraft?.content, lastLoadedContent, hasUnsavedChanges])

  // Auto-clear save status after delay
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

  // Autosave with debouncing (30 seconds after user stops typing)
  useEffect(() => {
    if (!selectedDraft || !autosaveEnabled || !hasUnsavedChanges || saving) {
      return
    }

    // Don't autosave if content hasn't changed since last autosave
    if (editorContent === lastAutosaveContent) {
      return
    }

    const autosaveTimeout = window.setTimeout(async () => {
      // Perform autosave
      setSaving(true)

      try {
        const response = await fetch(buildApiUrl(`/drafts/${selectedDraft.id}`), {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ content: editorContent }),
        })

        if (!response.ok) {
          throw new Error(`Autosave failed: ${response.statusText}`)
        }

        setLastLoadedContent(editorContent)
        setLastAutosaveContent(editorContent)
        setHasUnsavedChanges(false)
        setSaveStatus({ type: 'success', message: 'Draft autosaved.' })
      } catch (err) {
        // Don't show error for autosave failures - just log silently
        console.warn('Autosave failed:', err)
      } finally {
        setSaving(false)
      }
    }, 30000) // 30 second debounce

    return () => {
      window.clearTimeout(autosaveTimeout)
    }
  }, [selectedDraft, autosaveEnabled, hasUnsavedChanges, saving, editorContent, lastAutosaveContent])

  const handleEditorChange = (value: string) => {
    setEditorContent(value)
    if (selectedDraft) {
      setHasUnsavedChanges(value !== lastLoadedContent)
    }
  }

  const handleDiscardChanges = useCallback(async () => {
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
  }, [confirm, hasUnsavedChanges, lastLoadedContent, selectedDraft])

  const handleRefreshDraft = useCallback(async () => {
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
  }, [confirm, fetchDrafts, hasUnsavedChanges, selectedDraft])

  const handleSaveDraft = useCallback(async () => {
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
  }, [editorContent, fetchDrafts, selectedDraft])

  const draftMetrics = useMemo(() => {
    return calculateDraftMetrics(selectedDraft?.content ?? '')
  }, [selectedDraft?.content])

  return {
    viewMode,
    setViewMode,
    editorContent,
    handleEditorChange,
    hasUnsavedChanges,
    saving,
    saveStatus,
    handleSaveDraft,
    handleDiscardChanges,
    handleRefreshDraft,
    draftMetrics,
  }
}
