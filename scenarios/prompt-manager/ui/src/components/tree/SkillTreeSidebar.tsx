/**
 * SkillTreeSidebar - Full tree sidebar for skill navigation.
 *
 * Adapted from agent-inbox ItemTreeSidebar for full-page experience.
 * Features:
 * - Mode-based tree navigation
 * - Search filtering
 * - Tag filtering
 * - Dirty indicators
 * - Collapse/expand controls
 * - New skill button
 */

import { type ReactNode, type RefObject, useState, useRef, useCallback, useEffect, useMemo } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import { PanelLeftClose, PanelLeftOpen, Search, Plus, ChevronDown, ChevronUp, ChevronRight, Settings, User, Users, Sparkles, Layers, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TreeNode } from '@/types/editor'
import type { Skill, FolderType, ContentSearchOptions, SkillSearchMode } from '@/types'
import type { Agent } from '@/types/agent'
import type { CombineFormat } from '@/stores/combineStore'
import type { ContentSearchMatch } from '@/lib/schemas'
import { TreeNodeComponent } from './TreeNode'
import { TagFilterChips } from './TagFilterChips'
import { TagFilterPopover } from './TagFilterPopover'
import { FolderFilterChips } from './FolderFilterChips'
import { AgentListPanel } from '../agent/AgentListPanel'
import { TeamListPanel } from '../team/TeamListPanel'
import { FolderContextMenu } from './FolderContextMenu'
import { SkillContextMenu } from './SkillContextMenu'
import { AISearchModal } from '../search/AISearchModal'
import { CombineActionBar } from './CombineActionBar'
import { UnsavedChangesMenu, UnsavedChangesCollapsedBadge } from './UnsavedChangesMenu'
import { getModesPathFromNode, getAllItemIdsInSubtree } from '@/services/treeService'
import { getAISearchStatus, searchSkillContent } from '@/services/skillService'
import { selectors } from '@/constants/selectors'
import { useSelectionStore } from '@/stores/selectionStore'

const CONTENT_SNIPPET_LENGTH = 120
const CONTENT_SEARCH_MIN_CHARS = 2

interface ContentMatchGroup {
  file: string
  skillId: string
  skillName: string
  folder: string
  matches: ContentSearchMatch[]
}

function groupContentMatches(matches: ContentSearchMatch[]): ContentMatchGroup[] {
  const groups = new Map<string, ContentMatchGroup>()
  const order: string[] = []

  for (const match of matches) {
    if (!groups.has(match.file)) {
      groups.set(match.file, {
        file: match.file,
        skillId: match.skillId,
        skillName: match.skillName,
        folder: match.folder,
        matches: [],
      })
      order.push(match.file)
    }
    groups.get(match.file)?.matches.push(match)
  }

  const results: ContentMatchGroup[] = []
  for (const file of order) {
    const group = groups.get(file)
    if (group) {
      results.push(group)
    }
  }
  return results
}

function buildSnippet(line: string, ranges: { start: number; end: number }[], maxLen: number) {
  if (ranges.length === 0) {
    return {
      text: line,
      ranges: [],
      prefix: false,
      suffix: false,
    }
  }

  if (line.length <= maxLen) {
    return {
      text: line,
      ranges,
      prefix: false,
      suffix: false,
    }
  }

  const sorted = [...ranges].sort((a, b) => a.start - b.start)
  const focus = sorted[0]
  if (!focus) {
    return {
      text: line,
      ranges: [],
      prefix: false,
      suffix: false,
    }
  }
  let start = Math.max(0, Math.floor((focus.start + focus.end) / 2) - Math.floor(maxLen / 2))
  const end = Math.min(line.length, start + maxLen)

  if (end - start < maxLen && start > 0) {
    start = Math.max(0, end - maxLen)
  }

  const text = line.slice(start, end)
  const adjusted = sorted
    .filter((range) => range.end > start && range.start < end)
    .map((range) => ({
      start: Math.max(0, range.start - start),
      end: Math.min(end - start, range.end - start),
    }))

  return {
    text,
    ranges: adjusted,
    prefix: start > 0,
    suffix: end < line.length,
  }
}

function renderHighlightedSnippet(line: string, ranges: { start: number; end: number }[]) {
  const snippet = buildSnippet(line, ranges, CONTENT_SNIPPET_LENGTH)
  if (snippet.ranges.length === 0) {
    return line.length > CONTENT_SNIPPET_LENGTH
      ? `${line.slice(0, CONTENT_SNIPPET_LENGTH)}...`
      : line
  }

  const nodes: ReactNode[] = []
  let cursor = 0

  snippet.ranges.forEach((range, index) => {
    if (range.start > cursor) {
      nodes.push(<span key={`text-${index}`}>{snippet.text.slice(cursor, range.start)}</span>)
    }
    nodes.push(
      <mark
        key={`mark-${index}`}
        className="bg-primary/20 text-primary rounded px-0.5"
      >
        {snippet.text.slice(range.start, range.end)}
      </mark>
    )
    cursor = range.end
  })

  if (cursor < snippet.text.length) {
    nodes.push(<span key="text-tail">{snippet.text.slice(cursor)}</span>)
  }

  return (
    <>
      {snippet.prefix && <span className="text-muted-foreground">...</span>}
      {nodes}
      {snippet.suffix && <span className="text-muted-foreground">...</span>}
    </>
  )
}

interface SkillTreeSidebarProps {
  treeNodes: TreeNode[]
  skills: Skill[]
  /** All agents for name lookup in unsaved changes menu */
  agents?: Agent[]
  selectedItemId: string | null
  onSelectItem: (id: string) => void
  dirtyItemIds: Set<string>
  /** Separate dirty skill IDs for unsaved menu (defaults to dirtyItemIds if not provided) */
  dirtySkillIds?: Set<string>
  /** Dirty agent IDs for unsaved menu */
  dirtyAgentIds?: Set<string>
  /** Dirty team member IDs for unsaved menu */
  dirtyTeamMemberIds?: Set<string>
  expandedNodes: Set<string>
  onToggleNode: (nodeId: string) => void
  renderItemIcon?: (skill: Skill) => ReactNode
  searchQuery: string
  onSearchChange: (query: string) => void
  searchMode: SkillSearchMode
  onSearchModeChange: (mode: SkillSearchMode) => void
  contentSearchOptions: ContentSearchOptions
  onContentSearchOptionsChange: (options: ContentSearchOptions) => void
  isCollapsed: boolean
  onToggleCollapse: () => void
  onExpandAll: () => void
  onCollapseAll: () => void
  onCreateNew: (modes?: string[]) => void
  /** Ref for the search input (for keyboard shortcuts) */
  searchInputRef?: RefObject<HTMLInputElement>
  /** Callback to open settings modal */
  onOpenSettings?: () => void
  // Tag filter props
  selectedTags: string[]
  onSelectedTagsChange: (tags: string[]) => void
  availableTags: string[]
  // Folder filter props
  selectedFolders: string[]
  onSelectedFoldersChange: (folders: string[]) => void
  availableFolders: string[]
  // Context menu callbacks
  onDeleteFolder: (skillIds: string[], folderLabel: string) => void
  onCopySkill: (skillId: string) => void
  onMoveToFolder: (skillId: string, path: string[]) => void
  onChangeStorage: (skillId: string, folder: FolderType) => void
  onCreateNewFolder: (skillId: string) => void
  // Combine mode props
  combineMode?: boolean
  combineSelectedIds?: Set<string>
  combineFormat?: CombineFormat
  onCombineFormatChange?: (format: CombineFormat) => void
  onCombineToggle?: (node: TreeNode) => void
  getCombineSelectionState?: (node: TreeNode) => 'none' | 'partial' | 'all'
  onEnterCombineMode?: () => void
  onExitCombineMode?: () => void
  onCombineCopy?: () => void
  isCombineCopying?: boolean
  combineCopySuccess?: boolean
  /** Initial active tab (for persistence) */
  initialActiveTab?: string
  /** Callback when active tab changes (for persistence) */
  onActiveTabChange?: (tab: string) => void
  // Unsaved changes menu callbacks
  /** Callback to select/open a skill from unsaved menu */
  onSelectSkillFromMenu?: (skillId: string) => void
  /** Callback to select/open an agent from unsaved menu */
  onSelectAgentFromMenu?: (agentId: string) => void
  /** Callback to save a specific skill */
  onSaveSkill?: (skillId: string) => Promise<void>
  /** Callback to discard changes for a specific skill */
  onDiscardSkill?: (skillId: string) => void
  /** Callback to save a specific agent */
  onSaveAgent?: (agentId: string) => Promise<void>
  /** Callback to discard changes for a specific agent */
  onDiscardAgent?: (agentId: string) => void
  /** Callback to save all changes */
  onSaveAll?: () => Promise<void>
  /** Callback to discard all changes */
  onDiscardAll?: () => void
  /** Whether save operation is in progress */
  isSaving?: boolean
  /** Callback when content search matches change (for editor highlighting) */
  onContentMatchesChange?: (matches: ContentSearchMatch[]) => void
  className?: string
}

/**
 * Full tree sidebar component.
 */
export function SkillTreeSidebar({
  treeNodes,
  skills,
  agents = [],
  selectedItemId,
  onSelectItem,
  dirtyItemIds,
  dirtySkillIds,
  dirtyAgentIds = new Set(),
  dirtyTeamMemberIds = new Set(),
  expandedNodes,
  onToggleNode,
  renderItemIcon,
  searchQuery,
  onSearchChange,
  searchMode,
  onSearchModeChange,
  contentSearchOptions,
  onContentSearchOptionsChange,
  isCollapsed,
  onToggleCollapse,
  onExpandAll,
  onCollapseAll,
  onCreateNew,
  searchInputRef,
  onOpenSettings,
  selectedTags,
  onSelectedTagsChange,
  availableTags,
  selectedFolders,
  onSelectedFoldersChange,
  availableFolders,
  onDeleteFolder,
  onCopySkill,
  onMoveToFolder,
  onChangeStorage,
  onCreateNewFolder,
  combineMode = false,
  combineSelectedIds = new Set(),
  combineFormat = 'xml',
  onCombineFormatChange,
  onCombineToggle,
  getCombineSelectionState,
  onEnterCombineMode,
  onExitCombineMode,
  onCombineCopy,
  isCombineCopying = false,
  combineCopySuccess = false,
  initialActiveTab = 'skills',
  onActiveTabChange,
  onSelectSkillFromMenu,
  onSelectAgentFromMenu,
  onSaveSkill,
  onDiscardSkill,
  onSaveAgent,
  onDiscardAgent,
  onSaveAll,
  onDiscardAll,
  isSaving = false,
  onContentMatchesChange,
  className = '',
}: SkillTreeSidebarProps) {
  // Count total dirty items
  const dirtyCount = dirtyItemIds.size

  // Tag filter popover state
  const [isTagPopoverOpen, setIsTagPopoverOpen] = useState(false)
  const tagFilterRef = useRef<HTMLDivElement>(null)

  // Agent selection from centralized store
  const selectedAgentId = useSelectionStore((state) => state.selectedAgentId)
  const setSelectedAgentId = useSelectionStore((state) => state.setSelectedAgentId)

  // Team selection from centralized store
  const selectedTeamId = useSelectionStore((state) => state.selectedTeamId)
  const setSelectedTeamId = useSelectionStore((state) => state.setSelectedTeamId)

  // Active tab state
  const [activeTab, setActiveTab] = useState(initialActiveTab)

  // Notify parent when tab changes (for persistence)
  const handleTabChange = useCallback((tab: string) => {
    setActiveTab(tab)
    onActiveTabChange?.(tab)
  }, [onActiveTabChange])

  // Folder context menu state
  const [folderContextMenu, setFolderContextMenu] = useState<{
    node: TreeNode
    x: number
    y: number
  } | null>(null)

  // Skill context menu state
  const [skillContextMenu, setSkillContextMenu] = useState<{
    skillId: string
    skillName: string
    currentModes: string[]
    currentFolder: FolderType
    x: number
    y: number
  } | null>(null)

  // AI Search modal state
  const [isAISearchOpen, setIsAISearchOpen] = useState(false)
  const [aiSearchAvailable, setAISearchAvailable] = useState(false)

  // Content search state
  const [contentMatches, setContentMatches] = useState<ContentSearchMatch[]>([])
  const [contentLoading, setContentLoading] = useState(false)
  const [contentError, setContentError] = useState<string | null>(null)
  const [debouncedQuery, setDebouncedQuery] = useState('')
  const [expandedContentFiles, setExpandedContentFiles] = useState<Set<string>>(new Set())

  const groupedContentMatches = useMemo(
    () => groupContentMatches(contentMatches),
    [contentMatches]
  )

  // Check AI search availability on mount
  useEffect(() => {
    getAISearchStatus()
      .then((status) => setAISearchAvailable(status.available))
      .catch(() => setAISearchAvailable(false))
  }, [])

  const handleAISearch = useCallback(() => {
    setIsAISearchOpen(true)
  }, [])

  const handleAISearchSelect = useCallback((skillId: string) => {
    onSelectItem(skillId)
    setIsAISearchOpen(false)
  }, [onSelectItem])

  const handleToggleContentGroup = useCallback((file: string) => {
    setExpandedContentFiles((prev) => {
      const next = new Set(prev)
      if (next.has(file)) {
        next.delete(file)
      } else {
        next.add(file)
      }
      return next
    })
  }, [])

  useEffect(() => {
    if (searchMode !== 'content') return
    const trimmed = searchQuery.trim()
    const timer = setTimeout(() => {
      setDebouncedQuery(trimmed)
    }, 250)

    return () => clearTimeout(timer)
  }, [searchQuery, searchMode])

  useEffect(() => {
    if (searchMode !== 'content') return
    if (debouncedQuery.length < CONTENT_SEARCH_MIN_CHARS) {
      setContentMatches([])
      setContentError(null)
      setContentLoading(false)
      return
    }

    let cancelled = false
    setContentLoading(true)
    setContentError(null)

    searchSkillContent(debouncedQuery, {
      tags: selectedTags,
      folders: selectedFolders,
      caseSensitive: contentSearchOptions.caseSensitive,
      wholeWord: contentSearchOptions.wholeWord,
      regex: contentSearchOptions.regex,
      limit: 200,
    })
      .then((response) => {
        if (cancelled) return
        setContentMatches(response.matches)
        setContentLoading(false)
      })
      .catch((err: unknown) => {
        if (cancelled) return
        const message = err instanceof Error ? err.message : 'Content search failed'
        setContentError(message)
        setContentMatches([])
        setContentLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [debouncedQuery, searchMode, selectedTags, selectedFolders, contentSearchOptions])

  useEffect(() => {
    if (searchMode !== 'content') return
    setExpandedContentFiles(new Set(groupedContentMatches.map((group) => group.file)))
  }, [groupedContentMatches, searchMode])

  // Notify parent when content matches change (for editor highlighting)
  useEffect(() => {
    console.log('[SearchHighlight] Sidebar notifying parent:', {
      matchCount: contentMatches.length,
      hasCallback: !!onContentMatchesChange,
    })
    onContentMatchesChange?.(contentMatches)
  }, [contentMatches, onContentMatchesChange])

  const handleCategoryContextMenu = useCallback((node: TreeNode, x: number, y: number) => {
    setSkillContextMenu(null) // Close any open skill menu
    setFolderContextMenu({ node, x, y })
  }, [])

  const handleSkillContextMenu = useCallback((skillId: string, skillName: string, x: number, y: number) => {
    setFolderContextMenu(null) // Close any open folder menu
    // Find the skill to get its current modes and folder
    const skill = skills.find((s) => s.id === skillId)
    setSkillContextMenu({
      skillId,
      skillName,
      currentModes: skill?.modes || [],
      currentFolder: skill?.folder || 'local',
      x,
      y,
    })
  }, [skills])

  const handleCloseFolderContextMenu = useCallback(() => {
    setFolderContextMenu(null)
  }, [])

  const handleCloseSkillContextMenu = useCallback(() => {
    setSkillContextMenu(null)
  }, [])

  const handleAddSkillInFolder = useCallback(() => {
    if (folderContextMenu) {
      const modes = getModesPathFromNode(folderContextMenu.node)
      onCreateNew(modes)
      setFolderContextMenu(null)
    }
  }, [folderContextMenu, onCreateNew])

  const handleDeleteFolder = useCallback(() => {
    if (folderContextMenu) {
      const skillIds = getAllItemIdsInSubtree(folderContextMenu.node)
      onDeleteFolder(skillIds, folderContextMenu.node.label)
      setFolderContextMenu(null)
    }
  }, [folderContextMenu, onDeleteFolder])

  const handleCopySkill = useCallback(() => {
    if (skillContextMenu) {
      onCopySkill(skillContextMenu.skillId)
      setSkillContextMenu(null)
    }
  }, [skillContextMenu, onCopySkill])

  const handleMoveToFolder = useCallback((path: string[]) => {
    if (skillContextMenu) {
      onMoveToFolder(skillContextMenu.skillId, path)
      setSkillContextMenu(null)
    }
  }, [skillContextMenu, onMoveToFolder])

  const handleChangeStorage = useCallback((folder: FolderType) => {
    if (skillContextMenu) {
      onChangeStorage(skillContextMenu.skillId, folder)
      setSkillContextMenu(null)
    }
  }, [skillContextMenu, onChangeStorage])

  const handleCreateNewFolder = useCallback(() => {
    if (skillContextMenu) {
      onCreateNewFolder(skillContextMenu.skillId)
      setSkillContextMenu(null)
    }
  }, [skillContextMenu, onCreateNewFolder])

  // Get all available mode paths from skills
  const availableModePaths = skills
    .filter((s) => s.modes.length > 0)
    .map((s) => s.modes)

  // Collapsed state - show narrow strip with expand button
  if (isCollapsed) {
    return (
      <div
        className={cn(
          'flex flex-col h-full border-r border-border w-full bg-card/50',
          className
        )}
      >
        <div className="flex flex-col items-center py-3 gap-3">
          <button
            type="button"
            onClick={onToggleCollapse}
            className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
            title="Expand sidebar"
          >
            <PanelLeftOpen className="h-4 w-4" />
          </button>
          <UnsavedChangesCollapsedBadge
            dirtyCount={dirtyCount}
            onClick={onToggleCollapse}
          />
          {onOpenSettings && (
            <button
              type="button"
              onClick={onOpenSettings}
              className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
              title="Settings (,)"
            >
              <Settings className="h-4 w-4" />
            </button>
          )}
          <button
            type="button"
            onClick={() => onCreateNew()}
            className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
            title="New skill (Ctrl+N)"
          >
            <Plus className="h-4 w-4" />
          </button>
        </div>
      </div>
    )
  }

  // Expanded state - full sidebar with tabs
  return (
    <div
      className={cn(
        'flex flex-col h-full border-r border-border w-full bg-card/50',
        className
      )}
      data-testid={selectors.sidebar.container}
    >
      {/* Header with tabs */}
      <div className="flex-shrink-0 border-b border-border">
        {/* Top bar with settings and collapse */}
        <div className="flex items-center justify-between px-3 py-2">
          <div className="flex items-center gap-1">
            {combineMode ? (
              <div className="flex items-center gap-2">
                <Layers className="h-4 w-4 text-primary" />
                <span className="text-xs font-medium text-foreground">
                  Combine Mode
                </span>
              </div>
            ) : dirtyCount > 0 ? (
              <UnsavedChangesMenu
                dirtyCount={dirtyCount}
                dirtySkillIds={dirtySkillIds ?? dirtyItemIds}
                dirtyAgentIds={dirtyAgentIds}
                dirtyTeamMemberIds={dirtyTeamMemberIds}
                skills={skills}
                agents={agents}
                onSelectSkill={onSelectSkillFromMenu}
                onSelectAgent={onSelectAgentFromMenu}
                onSaveSkill={onSaveSkill}
                onDiscardSkill={onDiscardSkill}
                onSaveAgent={onSaveAgent}
                onDiscardAgent={onDiscardAgent}
                onSaveAll={onSaveAll}
                onDiscardAll={onDiscardAll}
                isSaving={isSaving}
              />
            ) : null}
          </div>
          <div className="flex items-center gap-1">
            {onOpenSettings && !combineMode && (
              <button
                type="button"
                onClick={onOpenSettings}
                className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                title="Settings (,)"
              >
                <Settings className="h-4 w-4" />
              </button>
            )}
            {!combineMode && (
              <button
                type="button"
                onClick={onToggleCollapse}
                className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                title="Collapse sidebar"
              >
                <PanelLeftClose className="h-4 w-4" />
              </button>
            )}
          </div>
        </div>
      </div>

      <Tabs.Root
        value={activeTab}
        onValueChange={handleTabChange}
        className="flex flex-col flex-1 min-h-0"
      >
        {/* Tab triggers */}
        <Tabs.List className="flex-shrink-0 flex border-b border-border">
          <Tabs.Trigger
            value="skills"
            className={cn(
              'flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium',
              'border-b-2 border-transparent',
              'text-muted-foreground hover:text-foreground',
              'data-[state=active]:text-foreground data-[state=active]:border-primary',
              'transition-colors'
            )}
            data-testid={selectors.sidebar.tabSkills}
          >
            <Search className="h-3.5 w-3.5" />
            Skills
          </Tabs.Trigger>
          <Tabs.Trigger
            value="agents"
            className={cn(
              'flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium',
              'border-b-2 border-transparent',
              'text-muted-foreground hover:text-foreground',
              'data-[state=active]:text-foreground data-[state=active]:border-primary',
              'transition-colors'
            )}
            data-testid={selectors.sidebar.tabAgents}
          >
            <User className="h-3.5 w-3.5" />
            Agents
          </Tabs.Trigger>
          <Tabs.Trigger
            value="teams"
            className={cn(
              'flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium',
              'border-b-2 border-transparent',
              'text-muted-foreground hover:text-foreground',
              'data-[state=active]:text-foreground data-[state=active]:border-primary',
              'transition-colors'
            )}
          >
            <Users className="h-3.5 w-3.5" />
            Teams
          </Tabs.Trigger>
        </Tabs.List>

        {/* Skills Tab */}
        <Tabs.Content value="skills" className="flex-1 flex flex-col min-h-0 data-[state=inactive]:hidden">
          {/* Search */}
          <div className="flex-shrink-0 px-3 py-2">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
              <input
                ref={searchInputRef}
                type="text"
                value={searchQuery}
                onChange={(e) => onSearchChange(e.target.value)}
                placeholder={searchMode === 'content' ? 'Search content... (Ctrl+K)' : 'Search skills... (Ctrl+K)'}
                className={cn(
                  'w-full pl-8 pr-3 py-1.5 text-xs',
                  'bg-muted border border-border rounded-md',
                  'text-foreground placeholder:text-muted-foreground',
                  'focus:outline-none focus:ring-2 focus:ring-primary'
                )}
                data-testid={selectors.sidebar.searchInput}
              />
            </div>

            <div className="flex items-center justify-between mt-2 gap-2">
              <div className="flex items-center gap-1">
                <button
                  type="button"
                  onClick={() => onSearchModeChange('quick')}
                  className={cn(
                    'px-2 py-1 text-[10px] rounded border transition-colors',
                    searchMode === 'quick'
                      ? 'bg-primary/10 text-primary border-primary/40'
                      : 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/50'
                  )}
                >
                  Quick
                </button>
                <button
                  type="button"
                  onClick={() => onSearchModeChange('content')}
                  className={cn(
                    'px-2 py-1 text-[10px] rounded border transition-colors',
                    searchMode === 'content'
                      ? 'bg-primary/10 text-primary border-primary/40'
                      : 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/50'
                  )}
                >
                  Content
                </button>
              </div>
              <button
                type="button"
                onClick={handleAISearch}
                disabled={!aiSearchAvailable}
                className={cn(
                  'flex items-center gap-1 px-2 py-1 text-[10px] rounded border transition-colors',
                  aiSearchAvailable
                    ? 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/50'
                    : 'text-muted-foreground/60 border-border/60 cursor-not-allowed'
                )}
                title={aiSearchAvailable ? 'Open AI search' : 'AI search unavailable'}
              >
                <Sparkles className="h-3 w-3" />
                AI
              </button>
            </div>

            {searchMode === 'content' && (
              <div className="flex items-center gap-1 mt-2">
                <button
                  type="button"
                  onClick={() => onContentSearchOptionsChange({
                    ...contentSearchOptions,
                    caseSensitive: !contentSearchOptions.caseSensitive,
                  })}
                  className={cn(
                    'px-1.5 py-1 text-[10px] rounded border transition-colors',
                    contentSearchOptions.caseSensitive
                      ? 'bg-primary/10 text-primary border-primary/40'
                      : 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/50'
                  )}
                  title="Case sensitive"
                >
                  Aa
                </button>
                <button
                  type="button"
                  onClick={() => onContentSearchOptionsChange({
                    ...contentSearchOptions,
                    wholeWord: !contentSearchOptions.wholeWord,
                  })}
                  className={cn(
                    'px-1.5 py-1 text-[10px] rounded border transition-colors',
                    contentSearchOptions.wholeWord
                      ? 'bg-primary/10 text-primary border-primary/40'
                      : 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/50'
                  )}
                  title="Whole word"
                >
                  W
                </button>
                <button
                  type="button"
                  onClick={() => onContentSearchOptionsChange({
                    ...contentSearchOptions,
                    regex: !contentSearchOptions.regex,
                  })}
                  className={cn(
                    'px-1.5 py-1 text-[10px] rounded border transition-colors',
                    contentSearchOptions.regex
                      ? 'bg-primary/10 text-primary border-primary/40'
                      : 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/50'
                  )}
                  title="Regex"
                >
                  .*
                </button>
              </div>
            )}

            {/* Filters row: Tag filter + Folder filter + Controls */}
            <div className="flex items-center justify-between mt-2 gap-2" ref={tagFilterRef}>
              <div className="relative flex-1 min-w-0">
                <TagFilterChips
                  selectedTags={selectedTags}
                  onRemoveTag={(tag) => onSelectedTagsChange(selectedTags.filter((t) => t !== tag))}
                  onAddFilter={() => setIsTagPopoverOpen(true)}
                  onClearAll={() => onSelectedTagsChange([])}
                />
                <TagFilterPopover
                  availableTags={availableTags}
                  selectedTags={selectedTags}
                  isOpen={isTagPopoverOpen}
                  onClose={() => setIsTagPopoverOpen(false)}
                  onApply={onSelectedTagsChange}
                  className="left-0 top-full"
                />
              </div>
              {searchMode === 'quick' && (
                <div className="flex items-center gap-1 flex-shrink-0">
                  <button
                    type="button"
                    onClick={onExpandAll}
                    className="flex items-center gap-1 px-2 py-1 text-[10px] text-muted-foreground hover:text-foreground hover:bg-muted/50 rounded transition-colors"
                    title="Expand all"
                    data-testid={selectors.sidebar.expandAllButton}
                  >
                    <ChevronDown className="h-3 w-3" />
                  </button>
                  <button
                    type="button"
                    onClick={onCollapseAll}
                    className="flex items-center gap-1 px-2 py-1 text-[10px] text-muted-foreground hover:text-foreground hover:bg-muted/50 rounded transition-colors"
                    title="Collapse all"
                  >
                    <ChevronUp className="h-3 w-3" />
                  </button>
                  {onEnterCombineMode && (
                    <button
                      type="button"
                      onClick={combineMode ? onExitCombineMode : onEnterCombineMode}
                      className={cn(
                        'flex items-center gap-1 px-2 py-1 text-[10px] rounded transition-colors',
                        combineMode
                          ? 'bg-primary/20 text-primary'
                          : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
                      )}
                      title={combineMode ? 'Exit combine mode' : 'Combine skills'}
                    >
                      <Layers className="h-3 w-3" />
                    </button>
                  )}
                </div>
              )}
            </div>

            {/* Folder filter row */}
            {availableFolders.length > 1 && (
              <div className="flex items-center gap-2 mt-2">
                <span className="text-[10px] text-muted-foreground flex-shrink-0">Storage:</span>
                <FolderFilterChips
                  selectedFolders={selectedFolders}
                  availableFolders={availableFolders}
                  onToggleFolder={(folder) => {
                    if (selectedFolders.includes(folder)) {
                      onSelectedFoldersChange(selectedFolders.filter((f) => f !== folder))
                    } else {
                      onSelectedFoldersChange([...selectedFolders, folder])
                    }
                  }}
                />
              </div>
            )}
          </div>

          {/* Tree */}
          <div className="flex-1 overflow-y-auto py-1">
            {searchMode === 'content' ? (
              <div className="px-3 py-4">
                {contentError && (
                  <div className="px-3 py-2 text-xs text-destructive bg-destructive/10 rounded-md">
                    {contentError}
                  </div>
                )}

                {!contentError && searchQuery.trim().length < CONTENT_SEARCH_MIN_CHARS && (
                  <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <Search className="h-8 w-8 mb-2 opacity-60" />
                    <p className="text-xs">Type at least {CONTENT_SEARCH_MIN_CHARS} characters to search</p>
                  </div>
                )}

                {!contentError && searchQuery.trim().length >= CONTENT_SEARCH_MIN_CHARS && contentLoading && (
                  <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <Loader2 className="h-6 w-6 mb-2 animate-spin" />
                    <p className="text-xs">Searching content...</p>
                  </div>
                )}

                {!contentError && searchQuery.trim().length >= CONTENT_SEARCH_MIN_CHARS && !contentLoading && groupedContentMatches.length === 0 && (
                  <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <Search className="h-8 w-8 mb-2 opacity-60" />
                    <p className="text-xs">No matches found</p>
                  </div>
                )}

                {!contentError && groupedContentMatches.length > 0 && (
                  <div className="space-y-2">
                    {groupedContentMatches.map((group) => {
                      const isExpanded = expandedContentFiles.has(group.file)
                      return (
                        <div key={group.file} className="border border-border rounded-md overflow-hidden">
                          <button
                            type="button"
                            onClick={() => handleToggleContentGroup(group.file)}
                            className={cn(
                              'w-full flex items-center gap-2 px-3 py-2 text-left transition-colors',
                              'text-muted-foreground hover:text-foreground hover:bg-muted/40'
                            )}
                          >
                            <ChevronRight className={cn(
                              'h-4 w-4 transition-transform',
                              isExpanded ? 'rotate-90' : ''
                            )} />
                            <div className="flex flex-col min-w-0">
                              <span className="text-xs font-medium text-foreground truncate">{group.skillName}</span>
                              <span className="text-[10px] text-muted-foreground truncate">{group.file}</span>
                            </div>
                            <span className="text-[10px] text-muted-foreground ml-auto">
                              {group.matches.length} {group.matches.length === 1 ? 'match' : 'matches'}
                            </span>
                          </button>

                          {isExpanded && (
                            <div className="divide-y divide-border">
                              {group.matches.map((match) => (
                                <button
                                  key={`${match.skillId}-${match.lineNumber}-${match.line}`}
                                  type="button"
                                  onClick={() => onSelectItem(match.skillId)}
                                  className={cn(
                                    'w-full flex items-start gap-3 px-3 py-2 text-left transition-colors',
                                    'hover:bg-muted/40'
                                  )}
                                >
                                  <span className="text-[10px] text-muted-foreground font-mono min-w-[2.5rem] text-right pt-0.5">
                                    {match.lineNumber}
                                  </span>
                                  <span className="text-xs font-mono text-foreground truncate flex-1">
                                    {renderHighlightedSnippet(match.line, match.matchRanges)}
                                  </span>
                                </button>
                              ))}
                            </div>
                          )}
                        </div>
                      )
                    })}
                  </div>
                )}
              </div>
            ) : (
              <>
                {treeNodes.length === 0 ? (
                  <div
                    className="px-3 py-8 text-center"
                    data-testid={selectors.sidebar.emptyState}
                  >
                    <p className="text-xs text-muted-foreground">
                      {searchQuery || selectedTags.length > 0 || selectedFolders.length > 0 ? 'No skills match your filters' : 'No skills yet'}
                    </p>
                    {searchQuery && aiSearchAvailable && (
                      <button
                        type="button"
                        onClick={handleAISearch}
                        className={cn(
                          'mt-3 inline-flex items-center gap-1.5 px-3 py-1.5 text-xs',
                          'bg-primary/10 hover:bg-primary/20 text-primary rounded-lg transition-colors'
                        )}
                      >
                        <Sparkles className="h-3.5 w-3.5" />
                        Try AI Search
                      </button>
                    )}
                  </div>
                ) : (
                  treeNodes.map((node) => (
                    <TreeNodeComponent
                      key={node.id}
                      node={node}
                      skills={skills}
                      selectedItemId={selectedItemId}
                      onSelectItem={onSelectItem}
                      dirtyItemIds={dirtyItemIds}
                      expandedNodes={expandedNodes}
                      onToggleNode={onToggleNode}
                      renderItemIcon={renderItemIcon}
                      showCheckbox={combineMode}
                      onCheckboxChange={combineMode ? onCombineToggle : undefined}
                      getSelectionState={combineMode ? getCombineSelectionState : undefined}
                      onCategoryContextMenu={handleCategoryContextMenu}
                      onSkillContextMenu={handleSkillContextMenu}
                    />
                  ))
                )}

                {/* Folder context menu */}
                {folderContextMenu && (
                  <FolderContextMenu
                    x={folderContextMenu.x}
                    y={folderContextMenu.y}
                    folderLabel={folderContextMenu.node.label}
                    skillCount={getAllItemIdsInSubtree(folderContextMenu.node).length}
                    onClose={handleCloseFolderContextMenu}
                    onAddSkill={handleAddSkillInFolder}
                    onDeleteFolder={handleDeleteFolder}
                  />
                )}

                {/* Skill context menu */}
                {skillContextMenu && (
                  <SkillContextMenu
                    x={skillContextMenu.x}
                    y={skillContextMenu.y}
                    skillId={skillContextMenu.skillId}
                    skillName={skillContextMenu.skillName}
                    currentModes={skillContextMenu.currentModes}
                    currentFolder={skillContextMenu.currentFolder}
                    availableModePaths={availableModePaths}
                    onClose={handleCloseSkillContextMenu}
                    onCopySkill={handleCopySkill}
                    onMoveToFolder={handleMoveToFolder}
                    onChangeStorage={handleChangeStorage}
                    onCreateNewFolder={handleCreateNewFolder}
                  />
                )}
              </>
            )}
          </div>

          {/* Footer - Context dependent */}
          <div className="flex-shrink-0 px-3 py-3 border-t border-border">
            {combineMode && onCombineCopy && onExitCombineMode && onCombineFormatChange ? (
              <CombineActionBar
                selectedCount={combineSelectedIds.size}
                format={combineFormat}
                onFormatChange={onCombineFormatChange}
                onCopy={onCombineCopy}
                onCancel={onExitCombineMode}
                isCopying={isCombineCopying}
                copySuccess={combineCopySuccess}
              />
            ) : (
              <button
                type="button"
                onClick={() => onCreateNew()}
                title="Create new skill (Ctrl+N)"
                className={cn(
                  'w-full flex items-center justify-center gap-2 px-3 py-2 text-sm',
                  'bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg transition-colors'
                )}
                data-testid={selectors.sidebar.newSkillButton}
              >
                <Plus className="h-4 w-4" />
                New Skill
              </button>
            )}
          </div>
        </Tabs.Content>

        {/* Agents Tab */}
        <Tabs.Content value="agents" className="flex-1 flex flex-col min-h-0 data-[state=inactive]:hidden">
          <AgentListPanel
            selectedAgentId={selectedAgentId}
            onSelectAgent={setSelectedAgentId}
            className="flex-1"
          />
        </Tabs.Content>

        {/* Teams Tab */}
        <Tabs.Content value="teams" className="flex-1 flex flex-col min-h-0 data-[state=inactive]:hidden">
          <TeamListPanel
            selectedTeamId={selectedTeamId}
            onSelectTeam={setSelectedTeamId}
            className="flex-1"
          />
        </Tabs.Content>
      </Tabs.Root>

      {/* AI Search Modal */}
      <AISearchModal
        isOpen={isAISearchOpen}
        onClose={() => setIsAISearchOpen(false)}
        initialQuery={searchQuery}
        onSelectSkill={handleAISearchSelect}
      />
    </div>
  )
}
