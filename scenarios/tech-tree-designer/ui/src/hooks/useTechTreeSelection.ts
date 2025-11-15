import { useCallback, useEffect, useMemo, useState } from 'react'
import type { TechTreeSummary } from '../types/techTree'
import { fetchTechTrees } from '../services/techTree'

const STORAGE_KEY = 'tech-tree-designer:selected-tree'

const readInitialTree = (): string | null => {
  if (typeof window === 'undefined') {
    return null
  }
  try {
    return window.localStorage.getItem(STORAGE_KEY)
  } catch (error) {
    console.warn('Unable to read tree preference', error)
    return null
  }
}

const persistTreeSelection = (treeId: string | null) => {
  if (typeof window === 'undefined') {
    return
  }
  try {
    if (treeId) {
      window.localStorage.setItem(STORAGE_KEY, treeId)
    } else {
      window.localStorage.removeItem(STORAGE_KEY)
    }
  } catch (error) {
    console.warn('Unable to persist tree preference', error)
  }
}

/**
 * Hook for managing tech tree selection and list.
 * Handles fetching available trees, persisting selection to localStorage,
 * and providing tree metadata.
 */
const useTechTreeSelection = () => {
  const [techTrees, setTechTrees] = useState<TechTreeSummary[]>([])
  const [treeLoading, setTreeLoading] = useState(true)
  const [selectedTreeId, setSelectedTreeId] = useState<string | null>(() => readInitialTree())

  // Persist selection to localStorage
  useEffect(() => {
    persistTreeSelection(selectedTreeId)
  }, [selectedTreeId])

  // Load available trees on mount
  useEffect(() => {
    let cancelled = false

    const loadTrees = async () => {
      setTreeLoading(true)
      try {
        const response = await fetchTechTrees()
        if (cancelled) {
          return
        }
        const treeList = Array.isArray(response.trees) ? response.trees : []
        setTechTrees(treeList)
        if (!treeList.length) {
          setSelectedTreeId(null)
        } else {
          setSelectedTreeId((current) => {
            const hasCurrent = current && treeList.some((entry) => entry?.tree?.id === current)
            return hasCurrent ? current : treeList[0]?.tree?.id || null
          })
        }
      } catch (error) {
        console.warn('Failed to fetch tech trees', error)
        if (!cancelled) {
          setTechTrees([])
          setSelectedTreeId(null)
        }
      } finally {
        if (!cancelled) {
          setTreeLoading(false)
        }
      }
    }

    loadTrees()
    return () => {
      cancelled = true
    }
  }, [])

  const selectedTreeSummary = useMemo(
    () => techTrees.find((entry) => entry?.tree?.id === selectedTreeId) || null,
    [techTrees, selectedTreeId]
  )

  const buildTreeAwarePath = useCallback(
    (path: string, treeOverride?: string | null) => {
      const targetTreeId = treeOverride ?? selectedTreeId
      if (!targetTreeId) {
        return path
      }
      const separator = path.includes('?') ? '&' : '?'
      return `${path}${separator}tree_id=${encodeURIComponent(targetTreeId)}`
    },
    [selectedTreeId]
  )

  return {
    techTrees,
    treeLoading,
    selectedTreeId,
    setSelectedTreeId,
    selectedTreeSummary,
    buildTreeAwarePath
  }
}

export default useTechTreeSelection
