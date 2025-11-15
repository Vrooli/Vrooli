import { useCallback, useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import type { NavigateFunction } from 'react-router-dom'

import { buildApiUrl } from '../utils/apiClient'
import type { Draft, DraftSaveStatus, OperationalTargetsResponse, RequirementGroup, ViewMode } from '../types'

import { useDrafts } from './useDrafts'
import { useDraftEditor } from './useDraftEditor'
import { useDraftTargets } from './useDraftTargets'
import { useDraftRequirements } from './useDraftRequirements'
import { useAutoValidation } from './useAutoValidation'
import type { DraftMetrics } from '../utils/formatters'
import type { Violation, PRDTemplateValidationResult } from '../types'

type ConfirmHandler = (options: {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  variant?: 'danger' | 'warning' | 'info'
}) => Promise<boolean>

interface UseDraftWorkspaceOptions {
  routeEntityType?: string
  routeEntityName?: string
  decodedRouteName: string
  navigate: NavigateFunction
  confirm: ConfirmHandler
}

interface DraftWorkspaceState {
  // Draft list management
  loading: boolean
  refreshing: boolean
  error: string | null
  drafts: Draft[]
  filter: string
  setFilter: (value: string) => void
  filteredDrafts: Draft[]
  selectedDraft: Draft | null
  handleSelectDraft: (draft: Draft) => void
  handleDelete: (draftId: string) => Promise<void>

  // Editor state
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

  // Metadata dialog
  metaDialogOpen: boolean
  openMetaDialog: () => void
  closeMetaDialog: () => void

  // Operational targets
  targetsData: OperationalTargetsResponse | null
  targetsLoading: boolean
  targetsError: string | null

  // Requirements
  requirementsData: RequirementGroup[] | null
  requirementsLoading: boolean
  requirementsError: string | null

  // Auto-validation
  validationResult: {
    violations: Violation[]
    template_compliance?: PRDTemplateValidationResult
    summary?: {
      total_violations: number
      errors: number
      warnings: number
      info: number
    }
  } | null
  validating: boolean
  validationError: string | null
  lastValidatedAt: Date | null
  triggerValidation: () => Promise<void>
}

export function useDraftWorkspace({
  routeEntityType,
  routeEntityName,
  decodedRouteName,
  navigate,
  confirm,
}: UseDraftWorkspaceOptions): DraftWorkspaceState {
  const normalizedRouteType = routeEntityType?.toLowerCase()
  const normalizedRouteName = decodedRouteName?.toLowerCase?.() ?? ''

  // Draft list management
  const {
    drafts,
    loading,
    refreshing,
    error,
    filter,
    setFilter,
    filteredDrafts,
    selectedDraft,
    fetchDrafts,
  } = useDrafts({ routeEntityType, routeEntityName, decodedRouteName })

  // Editor state management
  const {
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
  } = useDraftEditor({ selectedDraft, fetchDrafts, confirm })

  // Operational targets
  const { targetsData, targetsLoading, targetsError } = useDraftTargets({ selectedDraft })

  // Requirements
  const { requirementsData, requirementsLoading, requirementsError } = useDraftRequirements({ selectedDraft })

  // Auto-validation (debounced, triggers 3s after user stops typing)
  const {
    validationResult,
    validating,
    error: validationError,
    lastValidatedAt,
    triggerValidation,
  } = useAutoValidation({
    draftId: selectedDraft?.id ?? null,
    content: editorContent,
    enabled: true,
    debounceMs: 3000,
  })

  // Metadata dialog state
  const [metaDialogOpen, setMetaDialogOpen] = useState(false)

  // Keyboard listener for meta dialog
  useEffect(() => {
    if (!metaDialogOpen) {
      return
    }

    const handleKeyDown = (event: KeyboardEvent) => {
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

  // Draft selection handler
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

  // Draft deletion handler
  const handleDelete = useCallback(
    async (draftId: string) => {
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
    },
    [confirm, fetchDrafts],
  )

  const openMetaDialog = () => setMetaDialogOpen(true)
  const closeMetaDialog = () => setMetaDialogOpen(false)

  return {
    // Draft list
    loading,
    refreshing,
    error,
    drafts,
    filter,
    setFilter,
    filteredDrafts,
    selectedDraft,
    handleSelectDraft,
    handleDelete,

    // Editor
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

    // Metadata dialog
    metaDialogOpen,
    openMetaDialog,
    closeMetaDialog,

    // Operational targets
    targetsData,
    targetsLoading,
    targetsError,

    // Requirements
    requirementsData,
    requirementsLoading,
    requirementsError,

    // Auto-validation
    validationResult,
    validating,
    validationError,
    lastValidatedAt,
    triggerValidation,
  }
}
