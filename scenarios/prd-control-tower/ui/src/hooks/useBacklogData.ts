import { useCallback, useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'

import { convertBacklogEntriesRequest, updateBacklogEntryRequest } from '../utils/backlog'
import type { BacklogEntry } from '../types'
import { buildApiUrl } from '../utils/apiClient'

type ConfirmHandler = (options: {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  variant?: 'danger' | 'warning' | 'info'
}) => Promise<boolean>

interface UseBacklogDataOptions {
  confirm: ConfirmHandler
}

interface UseBacklogDataResult {
  backlogEntries: BacklogEntry[]
  filteredBacklogEntries: BacklogEntry[]
  loadingBacklog: boolean
  listFilter: string
  setListFilter: (value: string) => void
  backlogSelection: Set<string>
  toggleBacklogSelection: (id: string) => void
  toggleBacklogSelectAll: () => void
  backlogBusy: boolean
  backlogBusyId: string | null
  convertExistingEntries: (ids?: string[]) => Promise<void>
  deleteEntries: (ids?: string[]) => Promise<void>
  refreshBacklog: () => Promise<void>
  updateBacklogEntryNotes: (id: string, notes: string) => Promise<void>
}

export function useBacklogData({ confirm }: UseBacklogDataOptions): UseBacklogDataResult {
  const [backlogEntries, setBacklogEntries] = useState<BacklogEntry[]>([])
  const [backlogSelection, setBacklogSelection] = useState<Set<string>>(() => new Set())
  const [listFilter, setListFilter] = useState('')
  const [loadingBacklog, setLoadingBacklog] = useState(true)
  const [backlogBusy, setBacklogBusy] = useState(false)
  const [backlogBusyId, setBacklogBusyId] = useState<string | null>(null)

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

  const toggleBacklogSelection = useCallback((id: string) => {
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
    setBacklogSelection((prev) => {
      if (prev.size === filteredBacklogEntries.length) {
        return new Set()
      }
      return new Set(filteredBacklogEntries.map((entry) => entry.id))
    })
  }, [filteredBacklogEntries])

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
        const results = await convertBacklogEntriesRequest(targets)
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
    [backlogSelection, fetchBacklog],
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

  return {
    backlogEntries,
    filteredBacklogEntries,
    loadingBacklog,
    listFilter,
    setListFilter,
    backlogSelection,
    toggleBacklogSelection,
    toggleBacklogSelectAll,
    backlogBusy,
    backlogBusyId,
    convertExistingEntries,
    deleteEntries,
    refreshBacklog: fetchBacklog,
    updateBacklogEntryNotes: async (id: string, notes: string) => {
      try {
        await updateBacklogEntryRequest(id, { notes })
        toast.success('Notes updated')
        await fetchBacklog()
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to update notes'
        toast.error(message)
      }
    },
  }
}
