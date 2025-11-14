import { useCallback, useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import type { NavigateFunction } from 'react-router-dom'

import { buildApiUrl } from '../utils/apiClient'
import { calculateDraftMetrics, type DraftMetrics } from '../utils/formatters'
import type {
  Draft,
  DraftSaveStatus,
  OperationalTargetsResponse,
  RequirementGroup,
  RequirementsResponse,
  ViewMode,
} from '../types'
import { ViewModes } from '../types'

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
  loading: boolean
  refreshing: boolean
  error: string | null
  drafts: Draft[]
  filter: string
  setFilter: (value: string) => void
  filteredDrafts: Draft[]
  selectedDraft: Draft | null
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
  handleSelectDraft: (draft: Draft) => void
  handleDelete: (draftId: string) => Promise<void>
  metaDialogOpen: boolean
  openMetaDialog: () => void
  closeMetaDialog: () => void
  targetsData: OperationalTargetsResponse | null
  targetsLoading: boolean
  targetsError: string | null
  requirementsData: RequirementGroup[] | null
  requirementsLoading: boolean
  requirementsError: string | null
  draftMetrics: DraftMetrics
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

  const [drafts, setDrafts] = useState<Draft[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState(decodedRouteName)
  const [selectedDraftId, setSelectedDraftId] = useState<string | null>(null)
  const [editorContent, setEditorContent] = useState('')
  const [lastLoadedContent, setLastLoadedContent] = useState('')
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveStatus, setSaveStatus] = useState<DraftSaveStatus | null>(null)
  const [viewMode, setViewMode] = useState<ViewMode>(ViewModes.SPLIT)
  const [metaDialogOpen, setMetaDialogOpen] = useState(false)
  const [targetsData, setTargetsData] = useState<OperationalTargetsResponse | null>(null)
  const [targetsLoading, setTargetsLoading] = useState(false)
  const [targetsError, setTargetsError] = useState<string | null>(null)
  const [requirementsData, setRequirementsData] = useState<RequirementGroup[] | null>(null)
  const [requirementsLoading, setRequirementsLoading] = useState(false)
  const [requirementsError, setRequirementsError] = useState<string | null>(null)

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
      const data: { drafts: Draft[] } = await response.json()
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
      setTargetsData(null)
      setTargetsError(null)
      setTargetsLoading(false)
      setRequirementsData(null)
      setRequirementsError(null)
      setRequirementsLoading(false)
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
      setTargetsData(null)
      setTargetsError(null)
      setTargetsLoading(false)
      setRequirementsData(null)
      setRequirementsError(null)
      setRequirementsLoading(false)
      return
    }

    if (!hasUnsavedChanges && selectedDraft.content !== lastLoadedContent) {
      setEditorContent(selectedDraft.content)
      setLastLoadedContent(selectedDraft.content)
    }
  }, [selectedDraft, selectedDraftId, hasUnsavedChanges, lastLoadedContent])

  useEffect(() => {
    if (!selectedDraft) {
      return
    }
    const controller = new AbortController()
    const loadTargets = async () => {
      setTargetsLoading(true)
      setTargetsError(null)
      try {
        const encodedName = encodeURIComponent(selectedDraft.entity_name)
        const response = await fetch(
          buildApiUrl(`/catalog/${selectedDraft.entity_type}/${encodedName}/targets`),
          { signal: controller.signal },
        )
        if (response.status === 404) {
          setTargetsData({
            entity_type: selectedDraft.entity_type,
            entity_name: selectedDraft.entity_name,
            targets: [],
            unmatched_requirements: [],
          })
          return
        }
        if (!response.ok) {
          throw new Error(await response.text())
        }
        const data: OperationalTargetsResponse = await response.json()
        setTargetsData(data)
      } catch (err) {
        if (controller.signal.aborted) {
          return
        }
        setTargetsError(err instanceof Error ? err.message : 'Failed to load operational targets')
      } finally {
        if (!controller.signal.aborted) {
          setTargetsLoading(false)
        }
      }
    }

    loadTargets()
    return () => controller.abort()
  }, [selectedDraft])

  useEffect(() => {
    if (!selectedDraft) {
      return
    }
    const controller = new AbortController()
    const loadRequirements = async () => {
      setRequirementsLoading(true)
      setRequirementsError(null)
      try {
        const encodedName = encodeURIComponent(selectedDraft.entity_name)
        const response = await fetch(
          buildApiUrl(`/catalog/${selectedDraft.entity_type}/${encodedName}/requirements`),
          { signal: controller.signal },
        )
        if (response.status === 404) {
          setRequirementsData([])
          return
        }
        if (!response.ok) {
          throw new Error(await response.text())
        }
        const data: RequirementsResponse = await response.json()
        setRequirementsData(data.groups)
      } catch (err) {
        if (controller.signal.aborted) {
          return
        }
        setRequirementsError(err instanceof Error ? err.message : 'Failed to load requirements')
      } finally {
        if (!controller.signal.aborted) {
          setRequirementsLoading(false)
        }
      }
    }

    loadRequirements()
    return () => controller.abort()
  }, [selectedDraft])

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

  const filteredDrafts = useMemo(() => {
    const searchLower = filter.toLowerCase()
    const routeFilterActive = Boolean(routeEntityName && decodedRouteName)

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
  }, [drafts, filter, normalizedRouteName, normalizedRouteType, routeEntityName, decodedRouteName])

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

  const draftMetrics = useMemo(() => {
    return calculateDraftMetrics(selectedDraft?.content ?? '')
  }, [selectedDraft?.content])

  return {
    loading,
    refreshing,
    error,
    drafts,
    filter,
    setFilter,
    filteredDrafts,
    selectedDraft,
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
    handleSelectDraft,
    handleDelete,
    metaDialogOpen,
    openMetaDialog,
    closeMetaDialog,
    targetsData,
    targetsLoading,
    targetsError,
    requirementsData,
    requirementsLoading,
    requirementsError,
    draftMetrics,
  }
}
