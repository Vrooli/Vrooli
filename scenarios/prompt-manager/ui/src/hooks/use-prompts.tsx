/**
 * Custom hook for prompt-related state management.
 *
 * This hook encapsulates:
 * - Folder selection state
 * - Prompt selection state
 * - View filter state (folders, favorites, recent, popular)
 * - Search state
 * - Computed filtered prompts based on current view
 * - Sidebar counts
 *
 * Following boundary-of-responsibility: state management lives here,
 * App.tsx handles orchestration and layout.
 */

import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { Folder, Prompt } from '@/types'

// View filter types for the sidebar
export type ViewFilter = 'folders' | 'favorites' | 'recent' | 'popular'

// Props for filter display info
export interface FilterInfo {
  label: string
  description: string
}

// Sidebar counts for badges
export interface SidebarCounts {
  favorites: number
  recent: number
  popular: number
}

interface UsePromptsOptions {
  favorites: Set<string>
}

interface UsePromptsReturn {
  // Data
  folders: Folder[]
  filteredPrompts: Prompt[]
  sidebarCounts: SidebarCounts

  // Selection state
  selectedFolder: Folder | null
  selectedPrompt: Prompt | null

  // Filter state
  viewFilter: ViewFilter
  searchQuery: string
  filterInfo: FilterInfo | null

  // Loading states
  isLoading: boolean
  foldersLoading: boolean

  // Actions
  setSelectedFolder: (folder: Folder | null) => void
  setSelectedPrompt: (prompt: Prompt | null) => void
  setViewFilter: (filter: ViewFilter) => void
  setSearchQuery: (query: string) => void
  handleFilterChange: (filter: ViewFilter) => void

  // Computed
  showPromptList: boolean
}

/**
 * Custom hook for managing prompt-related state and computed data.
 *
 * @param options.favorites - Set of favorited prompt IDs (from useFavorites hook)
 * @returns Prompt state, computed values, and state setters
 */
export function usePrompts({ favorites }: UsePromptsOptions): UsePromptsReturn {
  // Selection state
  const [selectedFolder, setSelectedFolder] = useState<Folder | null>(null)
  const [selectedPrompt, setSelectedPrompt] = useState<Prompt | null>(null)

  // Filter state
  const [viewFilter, setViewFilter] = useState<ViewFilter>('folders')
  const [searchQuery, setSearchQuery] = useState('')

  // Fetch folders (computed from prompts)
  const { data: folders = [], isLoading: foldersLoading } = useQuery({
    queryKey: ['folders'],
    queryFn: () => api.getFolders(),
  })

  // Fetch all prompts
  const { data: allPrompts = [], isLoading: allPromptsLoading } = useQuery({
    queryKey: ['prompts', 'all'],
    queryFn: () => api.getPrompts(),
  })

  // Fetch prompts for selected folder
  const { data: folderPrompts = [], isLoading: folderPromptsLoading } = useQuery({
    queryKey: ['prompts', 'folder', selectedFolder?.id],
    queryFn: () => selectedFolder ? api.getPromptsByFolder(selectedFolder.id) : Promise.resolve([]),
    enabled: !!selectedFolder && viewFilter === 'folders',
  })

  // Handle search
  const { data: searchResults = [] } = useQuery({
    queryKey: ['search', searchQuery],
    queryFn: () => api.searchPrompts(searchQuery),
    enabled: searchQuery.length > 2,
  })

  // Compute filtered prompts based on view filter
  const filteredPrompts = useMemo(() => {
    if (searchQuery.length > 2) {
      return searchResults
    }

    switch (viewFilter) {
      case 'favorites':
        return allPrompts.filter(p => favorites.has(p.id))
      case 'recent': {
        const weekAgo = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000)
        return allPrompts
          .filter(p => new Date(p.updatedAt) > weekAgo)
          .sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
      }
      case 'popular':
        return [...allPrompts]
          .sort((a, b) => b.usageCount - a.usageCount)
          .slice(0, 20)
      case 'folders':
      default:
        return selectedFolder ? folderPrompts : []
    }
  }, [viewFilter, allPrompts, folderPrompts, searchResults, searchQuery, selectedFolder, favorites])

  // Compute counts for sidebar badges
  const sidebarCounts = useMemo((): SidebarCounts => {
    const weekAgo = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000)
    return {
      favorites: allPrompts.filter(p => favorites.has(p.id)).length,
      recent: allPrompts.filter(p => new Date(p.updatedAt) > weekAgo).length,
      popular: allPrompts.filter(p => p.usageCount > 0).length,
    }
  }, [allPrompts, favorites])

  // Get filter display info
  const filterInfo = useMemo((): FilterInfo | null => {
    switch (viewFilter) {
      case 'favorites':
        return { label: 'Favorites', description: 'Your starred prompts' }
      case 'recent':
        return { label: 'Recent', description: 'Updated in the last 7 days' }
      case 'popular':
        return { label: 'Popular', description: 'Most used prompts' }
      default:
        return null
    }
  }, [viewFilter])

  // Compute loading state
  const isLoading = viewFilter === 'folders' ? folderPromptsLoading : allPromptsLoading

  // Handle filter change - clears folder selection when switching to non-folder views
  const handleFilterChange = (filter: ViewFilter) => {
    setViewFilter(filter)
    if (filter !== 'folders') {
      setSelectedFolder(null)
    }
  }

  // Determine whether to show prompt list
  const showPromptList = searchQuery.length > 2 || viewFilter !== 'folders' || selectedFolder !== null

  return {
    // Data
    folders,
    filteredPrompts,
    sidebarCounts,

    // Selection state
    selectedFolder,
    selectedPrompt,

    // Filter state
    viewFilter,
    searchQuery,
    filterInfo,

    // Loading states
    isLoading,
    foldersLoading,

    // Actions
    setSelectedFolder,
    setSelectedPrompt,
    setViewFilter,
    setSearchQuery,
    handleFilterChange,

    // Computed
    showPromptList,
  }
}
