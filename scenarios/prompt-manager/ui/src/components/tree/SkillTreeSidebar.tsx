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
// AI_CHECK: SIDEBAR_PROMPT_SUBSCRIPTION=1 | LAST: 2026-02-17

import { type ReactNode, type RefObject, type KeyboardEvent as ReactKeyboardEvent, useState, useRef, useCallback, useEffect, useMemo } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import { Home, PanelLeftClose, PanelLeftOpen, Search, Plus, ChevronDown, ChevronUp, ChevronRight, Settings, User, Users, Sparkles, Layers, Loader2, Activity, AlertCircle, Bolt } from 'lucide-react'
import { TabList, TabTrigger } from '../shared/TabTrigger'
import { cn } from '@/lib/utils'
import type { TreeNode } from '@/types/editor'
import type { Skill, FolderType, ContentSearchOptions, SkillSearchMode } from '@/types'
import type { Agent } from '@/types/agent'
import { useCombineStore, type CombineFormat } from '@/stores/combineStore'
import type { ContentSearchMatch, AISearchResponse, AIActionSearchResponse, AIAgentSearchResponse, AITeamSearchResponse, TopicMatchResponse, DiscoverResponse, BudgetConfig, DiscoverFilterConfig } from '@/lib/schemas'
import type { UseRunningAgentsResult } from '@/hooks/useRunningAgents'
import type { UsePendingDecisionsResult } from '@/hooks/usePendingDecisions'
import type { FilterState, SortConfig, ViewMode, DetailMode } from '@/types/filterSort'
import { DEFAULT_FILTER_STATE } from '@/types/filterSort'
import { TreeNodeComponent } from './TreeNode'
import { FilterSortToolbar } from '../sidebar/FilterSortToolbar'
import { ActiveFilterChips } from '../sidebar/ActiveFilterChips'
import { SkillListView } from '../sidebar/SkillListView'
import { SkillCardView } from '../sidebar/SkillCardView'
import { isFilterEmpty } from '@/services/filterSortService'
import { AgentListPanel } from '../agent/AgentListPanel'
import { TeamListPanel } from '../team/TeamListPanel'
import { RunListPanel } from '../run/RunListPanel'
import { TopicListPanel } from '../topic/TopicListPanel'
import { TopicTreeView } from '../topic/TopicTreeView'
import { TopicCardView } from '../topic/TopicCardView'
import { ActionListPanel } from '../action/ActionListPanel'
import { ViewModeToggle } from '../sidebar/ViewModeToggle'
import { useTopics } from '@/hooks/useTopicData'
import { useTeamData } from '@/hooks/useTeamData'
import { useActionsData } from '@/hooks/useActionsData'
import { FolderContextMenu } from './FolderContextMenu'
import { SkillContextMenu } from './SkillContextMenu'
import { SearchResultsList } from '../search/SearchResultsList'
import { DiscoverControls } from '../search/DiscoverControls'
import { api } from '@/lib/api'
import type { CombineEntityType } from '@/stores/combineStore'
import { CombineActionBar } from './CombineActionBar'
import { SavedSetsPanel } from './SavedSetsPanel'
import { SavedSetEditor } from './SavedSetEditor'
import type { CopySetEntry } from '@/lib/copySetStorage'
import { UnsavedChangesMenu, UnsavedChangesCollapsedBadge } from './UnsavedChangesMenu'
import { RunningAgentsPopover } from './RunningAgentsPopover'
import { PendingDecisionsPopover } from './PendingDecisionsPopover'
import { getModesPathFromNode, getAllItemIdsInSubtree } from '@/services/treeService'
import { buildDirtyCountIndex, buildSelectionStateIndex } from '@/services/treeService'
import { getAISearchStatus, searchSkillContent } from '@/services/skillService'
import { selectors } from '@/constants/selectors'
import { useEditorStore } from '@/stores/editorStore'

const CONTENT_SNIPPET_LENGTH = 120
const CONTENT_SEARCH_MIN_CHARS = 2

/**
 * Per-tab search feature availability.
 * Flip booleans to enable features as entity types reach search parity with Skills.
 */
const TAB_SEARCH_FEATURES = {
  skills:  { contentSearch: true, aiSearch: true, tagFilter: true },
  agents:  { contentSearch: false, aiSearch: true, tagFilter: false },
  teams:   { contentSearch: false, aiSearch: true, tagFilter: false },
  runs:    { contentSearch: false, aiSearch: false, tagFilter: false },
  topics:  { contentSearch: false, aiSearch: true, tagFilter: false },
  actions: { contentSearch: false, aiSearch: true, tagFilter: false },
} as const

/** Map sidebar tab names to CombineEntityType */
const TAB_TO_ENTITY_TYPE: Record<string, CombineEntityType> = {
  skills: 'skills',
  agents: 'agents',
  teams: 'teams',
  topics: 'topics',
  actions: 'actions',
}

type SearchableTab = keyof typeof TAB_SEARCH_FEATURES

const TAB_SEARCH_PLACEHOLDERS: Record<SearchableTab, string> = {
  skills: 'Search skills... (Ctrl+K)',
  agents: 'Search agents...',
  teams: 'Search teams...',
  runs: 'Search runs...',
  topics: 'Search topics...',
  actions: 'Search actions...',
}

interface ContentMatchGroup {
  file: string
  skillId: string
  skillName: string
  folder: string
  matches: ContentSearchMatch[]
}

function flattenVisibleTree(nodes: TreeNode[], expandedNodes: Set<string>): TreeNode[] {
  const rows: TreeNode[] = []

  const visit = (node: TreeNode) => {
    rows.push(node)
    if (!node.isCategory || !expandedNodes.has(node.id)) return
    node.children.forEach(visit)
  }

  nodes.forEach(visit)
  return rows
}

function areMatchRangesEqual(a: { start: number; end: number }[], b: { start: number; end: number }[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    const left = a[i]
    const right = b[i]
    if (!left || !right) return false
    if (left.start !== right.start || left.end !== right.end) return false
  }
  return true
}

function areContentMatchesEqual(a: ContentSearchMatch[], b: ContentSearchMatch[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    const left = a[i]
    const right = b[i]
    if (!left || !right) return false
    if (
      left.skillId !== right.skillId ||
      left.skillName !== right.skillName ||
      left.file !== right.file ||
      left.folder !== right.folder ||
      left.lineNumber !== right.lineNumber ||
      left.line !== right.line ||
      !areMatchRangesEqual(left.matchRanges, right.matchRanges)
    ) {
      return false
    }
  }
  return true
}

function areStringSetsEqual(a: Set<string>, b: Set<string>): boolean {
  if (a.size !== b.size) return false
  for (const value of a) {
    if (!b.has(value)) return false
  }
  return true
}

function findFirstSkillId(nodes: TreeNode[]): string | null {
  for (const node of nodes) {
    if (node.isCategory) {
      const childSkillId = findFirstSkillId(node.children)
      if (childSkillId) {
        return childSkillId
      }
      continue
    }

    if (node.itemId) {
      return node.itemId
    }
  }

  return null
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
  onSelectItem: (id: string, lineNumber?: number) => void
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
  // Filter/sort/view props
  filterState: FilterState
  onFilterStateChange: (state: FilterState) => void
  sortConfig: SortConfig
  onSortConfigChange: (config: SortConfig) => void
  viewMode: ViewMode
  onViewModeChange: (mode: ViewMode) => void
  detailMode: DetailMode
  onDetailModeChange: (mode: DetailMode) => void
  healthScoreMap?: Map<string, number>
  filteredSortedSkills: Skill[]
  availableTags: string[]
  availableFolders: string[]
  // Context menu callbacks
  onDeleteFolder: (skillIds: string[], folderLabel: string) => void
  onCopySkill: (skillId: string) => void
  onMoveToFolder: (skillId: string, path: string[]) => void
  onChangeStorage: (skillId: string, folder: FolderType) => void
  onCreateNewFolder: (skillId: string) => void
  // Combine / select mode props
  combineMode?: boolean
  combineSelectedIds?: Set<string>
  combineFormat?: CombineFormat
  combineEntityType?: CombineEntityType
  onCombineFormatChange?: (format: CombineFormat) => void
  onCombineToggle?: (node: TreeNode) => void
  onEnterCombineMode?: () => void
  onExitCombineMode?: () => void
  /** Enter select mode for a specific entity type (used for non-skills tabs and AI mode) */
  onEnterSelectMode?: (entityType: CombineEntityType) => void
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
  selectedAgentId?: string | null
  /** Callback to select/open a team from sidebar (wraps selection + sidebar close on mobile) */
  onSelectTeamFromMenu?: (teamId: string) => void
  selectedTeamId?: string | null
  /** Callback to select/open a run from sidebar (wraps selection + sidebar close on mobile) */
  onSelectRunFromMenu?: (runId: string) => void
  selectedRunId?: string | null
  /** Callback to select/open a topic from sidebar (wraps selection + sidebar close on mobile) */
  onSelectTopicFromMenu?: (topicId: string) => void
  selectedTopicId?: string | null
  /** Callback to select/open an Action from sidebar (wraps selection + sidebar close on mobile) */
  onSelectActionFromMenu?: (actionId: string) => void
  selectedActionId?: string | null
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
  /** Callback to navigate to a running agent's team member view */
  onNavigateToRunningAgent?: (teamId: string, agentId: string) => void
  /** Pre-fetched running agents data from the sync hook (eliminates duplicate polling) */
  runningAgentsData?: UseRunningAgentsResult
  /** Pre-fetched pending decisions data from the sync hook */
  pendingDecisionsData?: UsePendingDecisionsResult
  /** Callback to navigate to a team's decision log */
  onNavigateToDecision?: (teamId: string) => void
  /** Callback to open the topic discovery wizard route */
  onOpenTopicWizard?: () => void
  // Agent context menu callbacks
  /** Called when user requests to duplicate an agent via context menu */
  onDuplicateAgent?: (agentId: string) => void
  /** Called when user requests to customize an agent via context menu */
  onCustomizeAgent?: (agentId: string) => void
  /** Called when user requests to preview an agent's prompt via context menu */
  onPreviewPrompt?: (agentId: string) => void
  // Team context menu callbacks
  /** Called when user toggles team enabled/disabled via context menu */
  onToggleTeamEnabled?: (teamId: string) => void
  /** Navigate back to the primary home/graph surface */
  onGoHome?: () => void
  /** Hide the top controls row (running/unsaved/settings/collapse) */
  hideTopControlsRow?: boolean
  className?: string
}

const EMPTY_SET = new Set<string>()

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
  dirtyAgentIds = EMPTY_SET,
  dirtyTeamMemberIds = EMPTY_SET,
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
  filterState,
  onFilterStateChange,
  sortConfig,
  onSortConfigChange,
  viewMode,
  onViewModeChange,
  detailMode,
  onDetailModeChange,
  healthScoreMap,
  filteredSortedSkills,
  availableTags,
  availableFolders,
  onDeleteFolder,
  onCopySkill,
  onMoveToFolder,
  onChangeStorage,
  onCreateNewFolder,
  combineMode = false,
  combineSelectedIds = EMPTY_SET,
  combineFormat = 'xml',
  combineEntityType = 'skills',
  onCombineFormatChange,
  onCombineToggle,
  onEnterCombineMode,
  onExitCombineMode,
  onEnterSelectMode,
  onCombineCopy,
  isCombineCopying = false,
  combineCopySuccess = false,
  initialActiveTab = 'skills',
  onActiveTabChange,
  onSelectSkillFromMenu,
  onSelectAgentFromMenu,
  selectedAgentId = null,
  onSelectTeamFromMenu,
  selectedTeamId = null,
  onSelectRunFromMenu,
  selectedRunId = null,
  onSelectTopicFromMenu,
  selectedTopicId = null,
  onSelectActionFromMenu,
  selectedActionId = null,
  onSaveSkill,
  onDiscardSkill,
  onSaveAgent,
  onDiscardAgent,
  onSaveAll,
  onDiscardAll,
  isSaving = false,
  onContentMatchesChange,
  onNavigateToRunningAgent,
  runningAgentsData,
  pendingDecisionsData,
  onNavigateToDecision,
  onOpenTopicWizard,
  onDuplicateAgent,
  onCustomizeAgent,
  onPreviewPrompt,
  onToggleTeamEnabled,
  onGoHome,
  hideTopControlsRow = false,
  className = '',
}: SkillTreeSidebarProps) {
  // Count total dirty items
  const dirtyCount = dirtyItemIds.size

  const tabsListRef = useRef<HTMLDivElement>(null)

  // Convert vertical mouse wheel to horizontal scroll on tab triggers
  useEffect(() => {
    const el = tabsListRef.current
    if (!el) return
    const handler = (e: WheelEvent) => {
      if (Math.abs(e.deltaY) > Math.abs(e.deltaX)) {
        el.scrollLeft += e.deltaY
        e.preventDefault()
      }
    }
    el.addEventListener('wheel', handler, { passive: false })
    return () => el.removeEventListener('wheel', handler)
  }, [])

  // Active tab state
  const [activeTab, setActiveTab] = useState(initialActiveTab)

  // Search state for agents/teams/runs/topics tabs (skills search is managed by parent)
  const [agentSearchQuery, setAgentSearchQuery] = useState('')
  const [teamSearchQuery, setTeamSearchQuery] = useState('')
  const [runSearchQuery, setRunSearchQuery] = useState('')
  const [topicSearchQuery, setTopicSearchQuery] = useState('')
  const [actionSearchQuery, setActionSearchQuery] = useState('')
  const [topicViewMode, setTopicViewMode] = useState<ViewMode>('tree')
  const [topicDetailMode, setTopicDetailMode] = useState<DetailMode>('compact')
  const { topics: allTopics } = useTopics()
  const { teams: allTeams } = useTeamData()
  const { actions: allActions } = useActionsData()

  const filteredTopics = useMemo(() => {
    if (!topicSearchQuery) return allTopics
    const lower = topicSearchQuery.toLowerCase()
    return allTopics.filter(
      (t) => t.name.toLowerCase().includes(lower) || t.description.toLowerCase().includes(lower)
    )
  }, [allTopics, topicSearchQuery])

  // Saved sets state
  const [showSavedSets, setShowSavedSets] = useState(false)
  const [editingSet, setEditingSet] = useState<CopySetEntry | null>(null)
  const [savedSetsRefreshKey, setSavedSetsRefreshKey] = useState(0)

  // Build entity lookup maps for saved sets display
  const entityLookup = useMemo(() => {
    const map = new Map<string, string>()
    for (const s of skills) map.set(s.id, s.name)
    for (const a of agents) map.set(a.id, a.displayName)
    for (const t of allTeams) map.set(t.id, t.displayName)
    for (const t of allTopics) map.set(t.id, t.name)
    for (const a of allActions) map.set(a.id, a.name)
    return map
  }, [skills, agents, allTeams, allTopics, allActions])

  // Build allEntities list for set editor
  const allEntitiesForEditor = useMemo(() => {
    const entityType = TAB_TO_ENTITY_TYPE[activeTab]
    if (entityType === 'skills') return skills.map((s) => ({ id: s.id, name: s.name }))
    if (entityType === 'agents') return agents.map((a) => ({ id: a.id, name: a.displayName }))
    if (entityType === 'teams') return allTeams.map((t) => ({ id: t.id, name: t.displayName }))
    if (entityType === 'topics') return allTopics.map((t) => ({ id: t.id, name: t.name }))
    if (entityType === 'actions') return allActions.map((a) => ({ id: a.id, name: a.name }))
    return []
  }, [activeTab, skills, agents, allTeams, allTopics, allActions])

  const handleApplySavedSet = useCallback((ids: string[]) => {
    const entityType = TAB_TO_ENTITY_TYPE[activeTab]
    if (!entityType) return
    // Enter combine mode and select all IDs from the saved set
    if (activeTab === 'skills' && searchMode !== 'ai') {
      onEnterCombineMode?.()
    } else {
      onEnterSelectMode?.(entityType)
    }
    // Apply selection after a tick to let mode activate
    setTimeout(() => {
      useCombineStore.getState().selectMultiple(ids)
    }, 0)
    setShowSavedSets(false)
    setEditingSet(null)
  }, [activeTab, searchMode, onEnterCombineMode, onEnterSelectMode])

  const handleSavedSetEditorSave = useCallback(() => {
    setEditingSet(null)
    setSavedSetsRefreshKey((n) => n + 1)
  }, [])

  // Unified search query for the current tab
  const currentSearchQuery = activeTab === 'skills' ? searchQuery
    : activeTab === 'agents' ? agentSearchQuery
    : activeTab === 'runs' ? runSearchQuery
    : activeTab === 'topics' ? topicSearchQuery
    : activeTab === 'actions' ? actionSearchQuery
    : teamSearchQuery

  const handleCurrentSearchChange = useCallback((query: string) => {
    if (activeTab === 'skills') onSearchChange(query)
    else if (activeTab === 'agents') setAgentSearchQuery(query)
    else if (activeTab === 'runs') setRunSearchQuery(query)
    else if (activeTab === 'topics') setTopicSearchQuery(query)
    else if (activeTab === 'actions') setActionSearchQuery(query)
    else setTeamSearchQuery(query)
  }, [activeTab, onSearchChange])

  const tabFeatures: { contentSearch: boolean; aiSearch: boolean; tagFilter: boolean } | undefined =
    (activeTab in TAB_SEARCH_FEATURES) ? TAB_SEARCH_FEATURES[activeTab as SearchableTab] : undefined

  // Notify parent when tab changes (for persistence)
  const handleTabChange = useCallback((tab: string) => {
    setActiveTab(tab)
    onActiveTabChange?.(tab)
    setShowSavedSets(false)
    setEditingSet(null)
    // Keep combineEntityType in sync when switching tabs while in select mode
    if (combineMode) {
      const entityType = TAB_TO_ENTITY_TYPE[tab]
      if (entityType) {
        useCombineStore.getState().setEntityType(entityType)
      }
    }
  }, [onActiveTabChange, combineMode])

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

  // AI Search state (inline, no modal)
  const [aiSearchAvailable, setAISearchAvailable] = useState(false)
  const [aiLoading, setAILoading] = useState(false)
  const [aiError, setAIError] = useState<string | null>(null)
  const [skillAIResults, setSkillAIResults] = useState<AISearchResponse | null>(null)
  const [actionAIResults, setActionAIResults] = useState<AIActionSearchResponse | null>(null)
  const [agentAIResults, setAgentAIResults] = useState<AIAgentSearchResponse | null>(null)
  const [teamAIResults, setTeamAIResults] = useState<AITeamSearchResponse | null>(null)
  const [topicAIResults, setTopicAIResults] = useState<TopicMatchResponse | null>(null)
  const [discoverResults, setDiscoverResults] = useState<DiscoverResponse | null>(null)
  const [useDiscover, setUseDiscover] = useState(true)
  const [complexity, setComplexity] = useState<string | undefined>(undefined)
  const [discoverType, setDiscoverType] = useState<'skill' | 'action' | 'all'>('skill')
  const [budgetConfig, setBudgetConfig] = useState<BudgetConfig | null>(null)
  const [filterConfig, setFilterConfig] = useState<DiscoverFilterConfig | null>(null)

  // Reactive selected content chars for budget gauge in selection mode
  const combineContentCharsMap = useCombineStore((s) => s.contentCharsMap)
  const selectedContentChars = useMemo(() => {
    if (!combineMode) return undefined
    let total = 0
    for (const id of combineSelectedIds) {
      total += combineContentCharsMap.get(id) ?? 0
    }
    return total
  }, [combineMode, combineSelectedIds, combineContentCharsMap])
  const [aiDebouncedQuery, setAIDebouncedQuery] = useState('')

  // Toggle select/combine mode for the current tab
  const handleSelectModeToggle = useCallback(() => {
    if (combineMode) {
      setShowSavedSets(false)
      setEditingSet(null)
      onExitCombineMode?.()
    } else {
      const entityType = TAB_TO_ENTITY_TYPE[activeTab]
      if (!entityType) return
      if (activeTab === 'skills' && searchMode !== 'ai') {
        onEnterCombineMode?.()
      } else {
        onEnterSelectMode?.(entityType)

        // Auto-select discover results when entering selection mode on skills tab
        if (activeTab === 'skills' && useDiscover && discoverResults?.results.length) {
          const results = discoverResults.results
          const budget = discoverResults.budgetChars

          let idsToSelect: string[]
          const charsEntries: Array<[string, number]> = results.map((r) => [r.id, r.contentChars])

          if (complexity && budget) {
            // With complexity: select only what fits within budget
            idsToSelect = []
            let cumulative = 0
            for (const r of results) {
              if (cumulative + r.contentChars > budget) break
              cumulative += r.contentChars
              idsToSelect.push(r.id)
            }
          } else {
            // No complexity: select all
            idsToSelect = results.map((r) => r.id)
          }

          // Schedule after enterAISelectMode completes (it resets selection)
          requestAnimationFrame(() => {
            useCombineStore.getState().selectMultiple(idsToSelect, charsEntries)
          })
        }
      }
    }
  }, [combineMode, activeTab, searchMode, onExitCombineMode, onEnterCombineMode, onEnterSelectMode, useDiscover, discoverResults, complexity])

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

  const editedNameSignature = useEditorStore((state) => {
    const entries: string[] = []
    for (const [id, promptState] of state.prompts) {
      if (promptState.current.name !== promptState.original.name) {
        entries.push(`${JSON.stringify(id)}\t${JSON.stringify(promptState.current.name)}`)
      }
    }
    entries.sort()
    return entries.join('\n')
  })

  const editedNameById = useMemo(() => {
    const next = new Map<string, string>()
    if (!editedNameSignature) return next

    for (const entry of editedNameSignature.split('\n')) {
      const separator = entry.indexOf('\t')
      if (separator <= 0) continue
      const idJson = entry.slice(0, separator)
      const nameJson = entry.slice(separator + 1)
      try {
        const id = JSON.parse(idJson) as string
        const name = JSON.parse(nameJson) as string
        next.set(id, name)
      } catch {
        // Skip malformed entry; keep sidebar rendering resilient.
      }
    }

    return next
  }, [editedNameSignature])

  const skillsById = useMemo(() => {
    const map = new Map<string, Skill>()
    for (const skill of skills) {
      map.set(skill.id, skill)
    }
    return map
  }, [skills])

  const dirtyCountByNodeId = useMemo(
    () => buildDirtyCountIndex(treeNodes, dirtyItemIds),
    [treeNodes, dirtyItemIds]
  )

  const selectionStateByNodeId = useMemo(
    () => (combineMode ? buildSelectionStateIndex(treeNodes, combineSelectedIds) : undefined),
    [combineMode, treeNodes, combineSelectedIds]
  )

  const visibleTreeRows = useMemo(
    () => flattenVisibleTree(treeNodes, expandedNodes),
    [treeNodes, expandedNodes]
  )

  // Check AI search availability on mount
  useEffect(() => {
    getAISearchStatus()
      .then((status) => setAISearchAvailable(status.available))
      .catch(() => setAISearchAvailable(false))
  }, [])

  const handleSearchInputKeyDown = useCallback((event: ReactKeyboardEvent<HTMLInputElement>) => {
    if (event.key !== 'Enter') return
    if (searchMode === 'ai') return // AI search is debounced, no Enter action needed

    if (searchMode !== 'quick') return

    event.preventDefault()

    const firstSkillId = findFirstSkillId(treeNodes)
    if (firstSkillId) {
      onSelectItem(firstSkillId)
      return
    }

    // No results in quick mode — switch to AI mode if available
    if (searchQuery.trim() && aiSearchAvailable) {
      onSearchModeChange('ai')
    }
  }, [searchMode, treeNodes, onSelectItem, searchQuery, aiSearchAvailable, onSearchModeChange])

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
      tags: filterState.tags,
      folders: filterState.storage,
      caseSensitive: contentSearchOptions.caseSensitive,
      wholeWord: contentSearchOptions.wholeWord,
      regex: contentSearchOptions.regex,
      limit: 200,
    })
      .then((response) => {
        if (cancelled) return
        setContentMatches((prev) =>
          areContentMatchesEqual(prev, response.matches) ? prev : response.matches
        )
        setContentLoading(false)
      })
      .catch((err: unknown) => {
        if (cancelled) return
        const message = err instanceof Error ? err.message : 'Content search failed'
        setContentError(message)
        setContentMatches((prev) => (prev.length === 0 ? prev : []))
        setContentLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [debouncedQuery, searchMode, filterState.tags, filterState.storage, contentSearchOptions])

  useEffect(() => {
    if (searchMode !== 'content') return
    const next = new Set(groupedContentMatches.map((group) => group.file))
    setExpandedContentFiles((prev) => (areStringSetsEqual(prev, next) ? prev : next))
  }, [groupedContentMatches, searchMode])

  // Notify parent when content matches change (for editor highlighting)
  useEffect(() => {
    onContentMatchesChange?.(contentMatches)
  }, [contentMatches, onContentMatchesChange])

  // AI search: debounce the query
  useEffect(() => {
    if (searchMode !== 'ai') return
    const trimmed = currentSearchQuery.trim()
    const timer = setTimeout(() => {
      setAIDebouncedQuery(trimmed)
    }, 300)
    return () => clearTimeout(timer)
  }, [currentSearchQuery, searchMode])

  // Load budget config and filter config on mount
  useEffect(() => {
    api.getBudgetConfig().then(setBudgetConfig).catch(() => {})
    api.getDiscoverFilterConfig().then(setFilterConfig).catch(() => {})
  }, [])

  // AI search: clear results when switching away from AI mode
  useEffect(() => {
    if (searchMode !== 'ai') {
      setSkillAIResults(null)
      setActionAIResults(null)
      setAgentAIResults(null)
      setTeamAIResults(null)
      setTopicAIResults(null)
      setDiscoverResults(null)
      setAIError(null)
      setAILoading(false)
    }
  }, [searchMode])

  // AI search: execute search when debounced query or params change
  useEffect(() => {
    if (searchMode !== 'ai') return
    if (!aiDebouncedQuery) {
      setSkillAIResults(null)
      setActionAIResults(null)
      setAgentAIResults(null)
      setTeamAIResults(null)
      setTopicAIResults(null)
      setDiscoverResults(null)
      setAIError(null)
      setAILoading(false)
      return
    }

    let cancelled = false
    setAILoading(true)
    setAIError(null)

    const doSearch = async () => {
      try {
        if (activeTab === 'skills') {
          if (useDiscover) {
            const result = await api.discover([aiDebouncedQuery], complexity, 10, discoverType)
            if (!cancelled) {
              setDiscoverResults(result)
              setSkillAIResults(null)
            }
          } else {
            const result = await api.aiSearch(aiDebouncedQuery, 10)
            if (!cancelled) {
              setSkillAIResults(result)
              setDiscoverResults(null)
            }
          }
        } else if (activeTab === 'agents') {
          const result = await api.aiSearchAgents(aiDebouncedQuery, 10)
          if (!cancelled) setAgentAIResults(result)
        } else if (activeTab === 'actions') {
          const result = await api.aiSearchActions(aiDebouncedQuery, 10)
          if (!cancelled) setActionAIResults(result)
        } else if (activeTab === 'teams') {
          const result = await api.aiSearchTeams(aiDebouncedQuery, 10)
          if (!cancelled) setTeamAIResults(result)
        } else if (activeTab === 'topics') {
          const result = await api.matchTopics([aiDebouncedQuery], 10)
          if (!cancelled) setTopicAIResults(result)
        }
      } catch (err: unknown) {
        if (!cancelled) {
          setAIError(err instanceof Error ? err.message : 'Search failed')
        }
      } finally {
        if (!cancelled) setAILoading(false)
      }
    }

    void doSearch()
    return () => { cancelled = true }
  }, [aiDebouncedQuery, searchMode, activeTab, useDiscover, complexity, discoverType])

  // AI search: compute over-budget IDs for discover mode
  const overBudgetIds = useMemo(() => {
    if (!discoverResults?.results || !discoverResults.budgetChars) return undefined
    const ids = new Set<string>()
    let cumulative = 0
    for (const r of discoverResults.results) {
      cumulative += r.contentChars
      if (cumulative > discoverResults.budgetChars) {
        ids.add(r.id)
      }
    }
    return ids.size > 0 ? ids : undefined
  }, [discoverResults])

  // Sync discover results contentChars into combineStore so budget gauge works in selection mode
  useEffect(() => {
    if (!discoverResults?.results.length) return
    const entries: Array<[string, number]> = discoverResults.results.map((r) => [r.id, r.contentChars])
    // selectMultiple with empty ids array just populates the contentCharsMap
    useCombineStore.getState().selectMultiple([], entries)
  }, [discoverResults])

  // Helper: navigate to entity from AI results
  const handleAIResultNavigate = useCallback((id: string, type?: 'skill' | 'action') => {
    if (type === 'action') {
      if (onSelectActionFromMenu) onSelectActionFromMenu(id)
    } else if (activeTab === 'skills') {
      onSelectItem(id)
    } else if (activeTab === 'agents') {
      if (onSelectAgentFromMenu) onSelectAgentFromMenu(id)
    } else if (activeTab === 'teams') {
      if (onSelectTeamFromMenu) onSelectTeamFromMenu(id)
    } else if (activeTab === 'topics') {
      if (onSelectTopicFromMenu) onSelectTopicFromMenu(id)
    } else if (activeTab === 'actions') {
      if (onSelectActionFromMenu) onSelectActionFromMenu(id)
    }
  }, [activeTab, onSelectItem, onSelectAgentFromMenu, onSelectTeamFromMenu, onSelectTopicFromMenu, onSelectActionFromMenu])

  // Helper: toggle selection from AI results
  const handleAIResultToggle = useCallback((id: string, _contentChars?: number) => {
    if (combineMode) {
      onCombineToggle?.({ id: `item-${id}`, label: '', isCategory: false, children: [], itemId: id, depth: 0 })
    }
  }, [combineMode, onCombineToggle])

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
  const availableModePaths = useMemo(
    () => skills
      .filter((s) => s.modes.length > 0)
      .map((s) => s.modes),
    [skills]
  )

  // Distinct modes and tags for filter config UI
  const availableModes = useMemo(() => {
    const set = new Set<string>()
    for (const skill of skills) {
      for (const mode of skill.modes) set.add(mode)
    }
    return Array.from(set).sort()
  }, [skills])

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
          {onGoHome && (
            <button
              type="button"
              onClick={onGoHome}
              className="p-2 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
              title="Go home"
              aria-label="Go home"
            >
              <Home className="h-4 w-4" />
            </button>
          )}
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
        'flex flex-col h-full overflow-hidden border-r border-border w-full bg-card/50',
        className
      )}
      data-testid={selectors.sidebar.container}
    >
      {/* Header with tabs */}
      <div className="flex-shrink-0 border-b border-border">
        {/* Top bar with settings and collapse */}
        {!hideTopControlsRow && (
          <div className="flex items-center justify-between gap-2 px-3 py-2">
            <div className="flex items-center gap-1">
              {combineMode ? (
                <div className="flex items-center gap-2">
                  <Layers className="h-4 w-4 text-primary" />
                  <span className="text-xs font-medium text-foreground">
                    Combine Mode
                  </span>
                </div>
              ) : (
                <>
                  {onGoHome && (
                    <button
                      type="button"
                      onClick={onGoHome}
                      className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                      title="Go home"
                      aria-label="Go home"
                    >
                      <Home className="h-4 w-4" />
                    </button>
                  )}
                  {onNavigateToRunningAgent && (
                    <RunningAgentsPopover
                      onNavigateToMember={onNavigateToRunningAgent}
                      groupedByTeam={runningAgentsData?.groupedByTeam}
                      count={runningAgentsData?.count}
                      stopAgent={runningAgentsData?.stopAgent}
                      stoppingIds={runningAgentsData?.stoppingIds}
                    />
                  )}
                  {onNavigateToDecision && (
                    <PendingDecisionsPopover
                      onNavigateToDecision={onNavigateToDecision}
                      groupedByTeam={pendingDecisionsData?.groupedByTeam}
                      count={pendingDecisionsData?.count}
                      acceptDecision={pendingDecisionsData?.acceptDecision}
                      rejectDecision={pendingDecisionsData?.rejectDecision}
                      processingIds={pendingDecisionsData?.processingIds}
                    />
                  )}
                  {dirtyCount > 0 && (
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
                  )}
                </>
              )}
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
        )}
      </div>

      <Tabs.Root
        value={activeTab}
        onValueChange={handleTabChange}
        className="flex flex-col flex-1 min-h-0 overflow-hidden"
      >
        {/* Search -- above tabs, visible for all entity types */}
        <div className="flex-shrink-0 px-3 py-2">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <input
              ref={searchInputRef}
              type="text"
              value={currentSearchQuery}
              onChange={(e) => handleCurrentSearchChange(e.target.value)}
              onKeyDown={activeTab === 'skills' ? handleSearchInputKeyDown : undefined}
              placeholder={(activeTab in TAB_SEARCH_PLACEHOLDERS ? TAB_SEARCH_PLACEHOLDERS[activeTab as SearchableTab] : undefined) ?? 'Search...'}
              className={cn(
                'w-full pl-8 pr-3 py-1.5 text-base md:text-xs',
                'bg-muted border border-border rounded-md',
                'text-foreground placeholder:text-muted-foreground',
                'focus:outline-none focus:ring-2 focus:ring-primary'
              )}
              data-testid={selectors.sidebar.searchInput}
            />
          </div>

          {/* Search mode toggle + Select button */}
          {(tabFeatures?.contentSearch || tabFeatures?.aiSearch || activeTab in TAB_TO_ENTITY_TYPE) && (
            <div className="flex min-w-0 flex-wrap items-center justify-between gap-2 mt-2">
              <div className="flex min-w-0 flex-wrap items-center gap-1">
                {(tabFeatures?.contentSearch || tabFeatures?.aiSearch) && (
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
                )}
                {tabFeatures?.contentSearch && (
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
                )}
                {tabFeatures?.aiSearch && (
                  <button
                    type="button"
                    onClick={() => onSearchModeChange('ai')}
                    disabled={!aiSearchAvailable}
                    className={cn(
                      'flex items-center gap-1 px-2 py-1 text-[10px] rounded border transition-colors',
                      searchMode === 'ai'
                        ? 'bg-primary/10 text-primary border-primary/40'
                        : aiSearchAvailable
                          ? 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/50'
                          : 'text-muted-foreground/60 border-border/60 cursor-not-allowed'
                    )}
                    title={aiSearchAvailable ? 'AI semantic search' : 'AI search unavailable (Ollama not running)'}
                  >
                    <Sparkles className="h-3 w-3" />
                    AI
                  </button>
                )}
              </div>
              {/* Select (combine) toggle + Saved Sets — available on all entity tabs */}
              {activeTab in TAB_TO_ENTITY_TYPE && (onEnterCombineMode || onEnterSelectMode) && (
                <div className="flex min-w-0 flex-wrap items-center gap-1">
                  {combineMode && (
                    <button
                      type="button"
                      onClick={() => {
                        if (showSavedSets) {
                          setShowSavedSets(false)
                          setEditingSet(null)
                        } else {
                          setShowSavedSets(true)
                          setEditingSet(null)
                          setSavedSetsRefreshKey((n) => n + 1)
                        }
                      }}
                      className={cn(
                        'flex items-center gap-1 px-2 py-1 text-[10px] rounded border transition-colors',
                        showSavedSets
                          ? 'bg-primary/10 text-primary border-primary/40'
                          : 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/50'
                      )}
                      title={showSavedSets ? 'Back to browse' : 'View saved sets'}
                    >
                      Saved
                    </button>
                  )}
                  <button
                    type="button"
                    onClick={handleSelectModeToggle}
                    className={cn(
                      'flex items-center gap-1 px-2 py-1 text-[10px] rounded border transition-colors',
                      combineMode
                        ? 'bg-primary/10 text-primary border-primary/40'
                        : 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/50'
                    )}
                    title={combineMode ? 'Exit select mode' : 'Select items to copy'}
                    data-testid="combine-mode-toggle"
                  >
                    <Layers className="h-3 w-3" />
                    Select
                  </button>
                </div>
              )}
            </div>
          )}

          {/* AI mode: discover controls (skills tab only) */}
          {searchMode === 'ai' && activeTab === 'skills' && !showSavedSets && (
            <div className="mt-2">
              <DiscoverControls
                useDiscover={useDiscover}
                onToggleDiscover={setUseDiscover}
                complexity={complexity}
                onComplexityChange={setComplexity}
                discoverType={discoverType}
                onDiscoverTypeChange={setDiscoverType}
                budgetChars={discoverResults?.budgetChars}
                totalContentChars={discoverResults?.totalContentChars}
                selectedContentChars={selectedContentChars}
                budgetConfig={budgetConfig}
                onBudgetConfigSave={(config) => {
                  api.setBudgetConfig(config).then(setBudgetConfig).catch(() => {})
                }}
                filterConfig={filterConfig}
                onFilterConfigSave={(config) => {
                  api.setDiscoverFilterConfig(config).then(setFilterConfig).catch(() => {})
                }}
                availableModes={availableModes}
                availableTags={availableTags}
              />
            </div>
          )}

          {/* Skills-only: content search options */}
          {tabFeatures?.contentSearch && searchMode === 'content' && (
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

          {/* Skills-only: filter/sort/view toolbar (hidden in AI mode) */}
          {tabFeatures?.tagFilter && searchMode !== 'ai' && (
            <>
              <FilterSortToolbar
                filterState={filterState}
                onFilterStateChange={onFilterStateChange}
                sortConfig={sortConfig}
                onSortConfigChange={onSortConfigChange}
                viewMode={viewMode}
                onViewModeChange={onViewModeChange}
                detailMode={detailMode}
                onDetailModeChange={onDetailModeChange}
                availableTags={availableTags}
                availableFolders={availableFolders}
                className="mt-2"
              />
              {!isFilterEmpty(filterState) && (
                <ActiveFilterChips
                  filterState={filterState}
                  onFilterStateChange={onFilterStateChange}
                  className="mt-1.5"
                />
              )}
              {skills.length > 0 && filteredSortedSkills.length < skills.length && (
                <div className="mt-1.5 flex items-center justify-between gap-2 text-[10px] text-muted-foreground">
                  <span>
                    Showing {filteredSortedSkills.length} of {skills.length} skills
                  </span>
                  {!isFilterEmpty(filterState) && (
                    <button
                      type="button"
                      onClick={() => onFilterStateChange(DEFAULT_FILTER_STATE)}
                      className="text-primary hover:text-primary/80 transition-colors"
                    >
                      Clear filters
                    </button>
                  )}
                </div>
              )}
            </>
          )}
        </div>

        {/* Tab triggers — wheel converts vertical scroll to horizontal */}
        <TabList ref={tabsListRef}>
          <TabTrigger value="skills" icon={<Search className="h-3.5 w-3.5" />} label="Skills" alwaysShowLabel testId={selectors.sidebar.tabSkills} />
          <TabTrigger value="agents" icon={<User className="h-3.5 w-3.5" />} label="Agents" alwaysShowLabel testId={selectors.sidebar.tabAgents} />
          <TabTrigger value="teams" icon={<Users className="h-3.5 w-3.5" />} label="Teams" alwaysShowLabel />
          <TabTrigger value="runs" icon={<Activity className="h-3.5 w-3.5" />} label="Runs" alwaysShowLabel />
          <TabTrigger value="topics" icon={<Layers className="h-3.5 w-3.5" />} label="Topics" alwaysShowLabel />
          <TabTrigger value="actions" icon={<Bolt className="h-3.5 w-3.5" />} label="Actions" alwaysShowLabel />
        </TabList>

        {/* Skills Tab */}
        <Tabs.Content value="skills" className="flex-1 flex flex-col min-h-0 data-[state=inactive]:hidden">
          {showSavedSets ? (
            editingSet ? (
              <SavedSetEditor
                entry={editingSet}
                entityType="skills"
                allEntities={allEntitiesForEditor}
                onSave={handleSavedSetEditorSave}
                onCancel={() => setEditingSet(null)}
              />
            ) : (
              <SavedSetsPanel
                entityType="skills"
                onApplySet={handleApplySavedSet}
                onEditSet={setEditingSet}
                entityLookup={entityLookup}
                refreshKey={savedSetsRefreshKey}
              />
            )
          ) : (
          <>
          {/* Tree / Results */}
          <div className="flex-1 overflow-y-auto py-1">
            {searchMode === 'ai' && currentSearchQuery.trim() ? (
              <div className="px-3 py-2">
                {aiLoading && (
                  <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <Loader2 className="h-6 w-6 mb-2 animate-spin" />
                    <p className="text-xs">Searching...</p>
                  </div>
                )}

                {aiError && (
                  <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <AlertCircle className="h-6 w-6 mb-2 text-destructive" />
                    <p className="text-xs text-destructive">{aiError}</p>
                  </div>
                )}

                {!aiLoading && !aiError && (() => {
                  const results = activeTab === 'skills'
                    ? (useDiscover ? discoverResults?.results : skillAIResults?.results)
                    : activeTab === 'agents' ? agentAIResults?.results
                    : activeTab === 'actions' ? actionAIResults?.results
                    : activeTab === 'teams' ? teamAIResults?.results
                    : activeTab === 'topics' ? topicAIResults
                    : undefined
                  const method = activeTab === 'skills'
                    ? (useDiscover ? undefined : skillAIResults?.method)
                    : activeTab === 'agents' ? agentAIResults?.method
                    : activeTab === 'actions' ? actionAIResults?.method
                    : activeTab === 'teams' ? teamAIResults?.method
                    : undefined
                  const hasResults = results && results.length > 0

                  return (
                    <>
                      {method === 'text' && (
                        <div className="mb-2 px-2 py-1 text-[10px] bg-yellow-500/10 text-yellow-500 rounded">
                          Using text fallback (AI unavailable)
                        </div>
                      )}

                      {hasResults ? (
                        <SearchResultsList
                          entityType={TAB_TO_ENTITY_TYPE[activeTab] ?? 'skills'}
                          skillResults={!useDiscover ? skillAIResults?.results : undefined}
                          actionResults={actionAIResults?.results}
                          discoverResults={useDiscover ? discoverResults?.results : undefined}
                          agentResults={agentAIResults?.results}
                          teamResults={teamAIResults?.results}
                          topicResults={topicAIResults ?? undefined}
                          isSelectMode={combineMode}
                          selectedIds={combineSelectedIds}
                          onToggleSelection={handleAIResultToggle}
                          onNavigate={handleAIResultNavigate}
                          discoverMode={activeTab === 'skills' && useDiscover}
                          overBudgetIds={overBudgetIds}
                          compact
                        />
                      ) : (
                        <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                          <Search className="h-8 w-8 mb-2 opacity-60" />
                          <p className="text-xs">No results found</p>
                        </div>
                      )}
                    </>
                  )
                })()}
              </div>
            ) : searchMode === 'content' ? (
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
                                  onClick={() => onSelectItem(match.skillId, match.lineNumber)}
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
                {treeNodes.length === 0 && filteredSortedSkills.length === 0 ? (
                  <div
                    className="px-3 py-8 text-center"
                    data-testid={selectors.sidebar.emptyState}
                  >
                    <p className="text-xs text-muted-foreground">
                      {searchQuery || !isFilterEmpty(filterState) ? 'No skills match your filters' : 'No skills yet'}
                    </p>
                    {searchQuery && aiSearchAvailable && (
                      <button
                        type="button"
                        onClick={() => onSearchModeChange('ai')}
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
                ) : viewMode === 'list' ? (
                  <SkillListView
                    skills={filteredSortedSkills}
                    selectedItemId={selectedItemId}
                    onSelectItem={onSelectItem}
                    dirtyItemIds={dirtyItemIds}
                    detailMode={detailMode}
                    healthScoreMap={healthScoreMap}
                    renderItemIcon={renderItemIcon}
                    onSkillContextMenu={handleSkillContextMenu}
                    combineMode={combineMode}
                    combineSelectedIds={combineSelectedIds}
                    onCombineToggleSkill={onCombineToggle ? (skillId) => {
                      onCombineToggle({ id: `item-${skillId}`, label: '', isCategory: false, children: [], itemId: skillId, depth: 0 })
                    } : undefined}
                  />
                ) : viewMode === 'card' ? (
                  <SkillCardView
                    skills={filteredSortedSkills}
                    selectedItemId={selectedItemId}
                    onSelectItem={onSelectItem}
                    dirtyItemIds={dirtyItemIds}
                    detailMode={detailMode}
                    healthScoreMap={healthScoreMap}
                    renderItemIcon={renderItemIcon}
                    onSkillContextMenu={handleSkillContextMenu}
                    combineMode={combineMode}
                    combineSelectedIds={combineSelectedIds}
                    onCombineToggleSkill={onCombineToggle ? (skillId) => {
                      onCombineToggle({ id: `item-${skillId}`, label: '', isCategory: false, children: [], itemId: skillId, depth: 0 })
                    } : undefined}
                  />
                ) : (
                  <div className="flex h-full flex-col">
                    {/* Tree expand/collapse controls */}
                    <div className="flex flex-shrink-0 items-center gap-1 px-3 py-1 border-b border-border/50">
                      <button
                        type="button"
                        onClick={onExpandAll}
                        className="flex items-center gap-1 px-1.5 py-0.5 text-[10px] text-muted-foreground hover:text-foreground hover:bg-muted/50 rounded transition-colors"
                        title="Expand all"
                        data-testid={selectors.sidebar.expandAllButton}
                      >
                        <ChevronDown className="h-3 w-3" />
                        <span>Expand</span>
                      </button>
                      <button
                        type="button"
                        onClick={onCollapseAll}
                        className="flex items-center gap-1 px-1.5 py-0.5 text-[10px] text-muted-foreground hover:text-foreground hover:bg-muted/50 rounded transition-colors"
                        title="Collapse all"
                      >
                        <ChevronUp className="h-3 w-3" />
                        <span>Collapse</span>
                      </button>
                    </div>
                    <div className="min-h-0 flex-1 overflow-y-auto">
                      {visibleTreeRows.map((node) => (
                        <TreeNodeComponent
                          key={node.id}
                          node={node}
                          skillsById={skillsById}
                          editedNameById={editedNameById}
                          selectedItemId={selectedItemId}
                          onSelectItem={onSelectItem}
                          dirtyItemIds={dirtyItemIds}
                          dirtyCountByNodeId={dirtyCountByNodeId}
                          expandedNodes={expandedNodes}
                          onToggleNode={onToggleNode}
                          renderItemIcon={renderItemIcon}
                          showCheckbox={combineMode}
                          onCheckboxChange={combineMode ? onCombineToggle : undefined}
                          selectionStateByNodeId={selectionStateByNodeId}
                          detailMode={detailMode}
                          healthScoreMap={healthScoreMap}
                          onCategoryContextMenu={handleCategoryContextMenu}
                          onSkillContextMenu={handleSkillContextMenu}
                          renderChildren={false}
                        />
                      ))}
                    </div>
                  </div>
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
            {combineMode && combineEntityType === 'skills' && onCombineCopy && onExitCombineMode && onCombineFormatChange ? (
              <CombineActionBar
                selectedCount={combineSelectedIds.size}
                format={combineFormat}
                onFormatChange={onCombineFormatChange}
                onCopy={onCombineCopy}
                onCancel={onExitCombineMode}
                isCopying={isCombineCopying}
                copySuccess={combineCopySuccess}
                entityLabel="skill"
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
          </>
          )}
        </Tabs.Content>

        {/* Agents Tab */}
        <Tabs.Content value="agents" className="flex-1 flex flex-col min-h-0 data-[state=inactive]:hidden">
          {showSavedSets ? (
            editingSet ? (
              <SavedSetEditor
                entry={editingSet}
                entityType="agents"
                allEntities={allEntitiesForEditor}
                onSave={handleSavedSetEditorSave}
                onCancel={() => setEditingSet(null)}
              />
            ) : (
              <SavedSetsPanel
                entityType="agents"
                onApplySet={handleApplySavedSet}
                onEditSet={setEditingSet}
                entityLookup={entityLookup}
                refreshKey={savedSetsRefreshKey}
              />
            )
          ) : (
          <>
          {searchMode === 'ai' && agentSearchQuery.trim() ? (
            <div className="flex-1 overflow-y-auto px-3 py-2">
              {aiLoading && (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                  <Loader2 className="h-6 w-6 mb-2 animate-spin" />
                  <p className="text-xs">Searching agents...</p>
                </div>
              )}
              {aiError && (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                  <AlertCircle className="h-6 w-6 mb-2 text-destructive" />
                  <p className="text-xs text-destructive">{aiError}</p>
                </div>
              )}
              {!aiLoading && !aiError && (
                agentAIResults?.results && agentAIResults.results.length > 0 ? (
                  <SearchResultsList
                    entityType="agents"
                    agentResults={agentAIResults.results}
                    isSelectMode={combineMode}
                    selectedIds={combineSelectedIds}
                    onToggleSelection={handleAIResultToggle}
                    onNavigate={handleAIResultNavigate}
                    compact
                  />
                ) : (
                  <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <Search className="h-8 w-8 mb-2 opacity-60" />
                    <p className="text-xs">No results found</p>
                  </div>
                )
              )}
            </div>
          ) : (
            <AgentListPanel
              selectedAgentId={selectedAgentId}
              onSelectAgent={onSelectAgentFromMenu ?? (() => {})}
              searchQuery={agentSearchQuery}
              onDuplicateAgent={onDuplicateAgent}
              onCustomizeAgent={onCustomizeAgent}
              onPreviewPrompt={onPreviewPrompt}
              className="flex-1"
              isSelectMode={combineMode && combineEntityType === 'agents'}
              selectedIds={combineSelectedIds}
              onToggleSelection={(id) => handleAIResultToggle(id)}
            />
          )}
          {/* Footer for agents */}
          {combineMode && combineEntityType === 'agents' && onCombineCopy && onExitCombineMode && onCombineFormatChange && (
            <div className="flex-shrink-0 px-3 py-3 border-t border-border">
              <CombineActionBar
                selectedCount={combineSelectedIds.size}
                format={combineFormat}
                onFormatChange={onCombineFormatChange}
                onCopy={onCombineCopy}
                onCancel={onExitCombineMode}
                isCopying={isCombineCopying}
                copySuccess={combineCopySuccess}
                entityLabel="agent"
              />
            </div>
          )}
          </>
          )}
        </Tabs.Content>

        {/* Teams Tab */}
        <Tabs.Content value="teams" className="flex-1 flex flex-col min-h-0 data-[state=inactive]:hidden">
          {showSavedSets ? (
            editingSet ? (
              <SavedSetEditor
                entry={editingSet}
                entityType="teams"
                allEntities={allEntitiesForEditor}
                onSave={handleSavedSetEditorSave}
                onCancel={() => setEditingSet(null)}
              />
            ) : (
              <SavedSetsPanel
                entityType="teams"
                onApplySet={handleApplySavedSet}
                onEditSet={setEditingSet}
                entityLookup={entityLookup}
                refreshKey={savedSetsRefreshKey}
              />
            )
          ) : (
          <>
          {searchMode === 'ai' && teamSearchQuery.trim() ? (
            <div className="flex-1 overflow-y-auto px-3 py-2">
              {aiLoading && (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                  <Loader2 className="h-6 w-6 mb-2 animate-spin" />
                  <p className="text-xs">Searching teams...</p>
                </div>
              )}
              {aiError && (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                  <AlertCircle className="h-6 w-6 mb-2 text-destructive" />
                  <p className="text-xs text-destructive">{aiError}</p>
                </div>
              )}
              {!aiLoading && !aiError && (
                teamAIResults?.results && teamAIResults.results.length > 0 ? (
                  <SearchResultsList
                    entityType="teams"
                    teamResults={teamAIResults.results}
                    isSelectMode={combineMode}
                    selectedIds={combineSelectedIds}
                    onToggleSelection={handleAIResultToggle}
                    onNavigate={handleAIResultNavigate}
                    compact
                  />
                ) : (
                  <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <Search className="h-8 w-8 mb-2 opacity-60" />
                    <p className="text-xs">No results found</p>
                  </div>
                )
              )}
            </div>
          ) : (
            <TeamListPanel
              selectedTeamId={selectedTeamId}
              onSelectTeam={onSelectTeamFromMenu ?? (() => {})}
              searchQuery={teamSearchQuery}
              onToggleTeamEnabled={onToggleTeamEnabled}
              className="flex-1"
              isSelectMode={combineMode && combineEntityType === 'teams'}
              selectedIds={combineSelectedIds}
              onToggleSelection={(id) => handleAIResultToggle(id)}
            />
          )}
          {/* Footer for teams */}
          {combineMode && combineEntityType === 'teams' && onCombineCopy && onExitCombineMode && onCombineFormatChange && (
            <div className="flex-shrink-0 px-3 py-3 border-t border-border">
              <CombineActionBar
                selectedCount={combineSelectedIds.size}
                format={combineFormat}
                onFormatChange={onCombineFormatChange}
                onCopy={onCombineCopy}
                onCancel={onExitCombineMode}
                isCopying={isCombineCopying}
                copySuccess={combineCopySuccess}
                entityLabel="team"
              />
            </div>
          )}
          </>
          )}
        </Tabs.Content>

        {/* Runs Tab */}
        <Tabs.Content value="runs" className="flex-1 flex flex-col min-h-0 data-[state=inactive]:hidden">
          <RunListPanel
            selectedRunId={selectedRunId}
            onSelectRun={onSelectRunFromMenu ?? (() => {})}
            searchQuery={runSearchQuery}
            className="flex-1"
          />
        </Tabs.Content>

        {/* Topics Tab */}
        <Tabs.Content value="topics" className="flex-1 flex flex-col min-h-0 data-[state=inactive]:hidden">
          {showSavedSets ? (
            editingSet ? (
              <SavedSetEditor
                entry={editingSet}
                entityType="topics"
                allEntities={allEntitiesForEditor}
                onSave={handleSavedSetEditorSave}
                onCancel={() => setEditingSet(null)}
              />
            ) : (
              <SavedSetsPanel
                entityType="topics"
                onApplySet={handleApplySavedSet}
                onEditSet={setEditingSet}
                entityLookup={entityLookup}
                refreshKey={savedSetsRefreshKey}
              />
            )
          ) : (
          <>
          {searchMode === 'ai' && topicSearchQuery.trim() ? (
            <div className="flex-1 overflow-y-auto px-3 py-2">
              {aiLoading && (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                  <Loader2 className="h-6 w-6 mb-2 animate-spin" />
                  <p className="text-xs">Searching topics...</p>
                </div>
              )}
              {aiError && (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                  <AlertCircle className="h-6 w-6 mb-2 text-destructive" />
                  <p className="text-xs text-destructive">{aiError}</p>
                </div>
              )}
              {!aiLoading && !aiError && (
                topicAIResults && topicAIResults.length > 0 ? (
                  <SearchResultsList
                    entityType="topics"
                    topicResults={topicAIResults}
                    isSelectMode={combineMode}
                    selectedIds={combineSelectedIds}
                    onToggleSelection={handleAIResultToggle}
                    onNavigate={handleAIResultNavigate}
                    compact
                  />
                ) : (
                  <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <Search className="h-8 w-8 mb-2 opacity-60" />
                    <p className="text-xs">No results found</p>
                  </div>
                )
              )}
            </div>
          ) : (
            <>
              {/* View mode toggle + Discover button */}
              <div className="flex items-center justify-between px-2 py-1 border-b border-border">
                <ViewModeToggle
                  viewMode={topicViewMode}
                  onViewModeChange={setTopicViewMode}
                  detailMode={topicDetailMode}
                  onDetailModeChange={setTopicDetailMode}
                />
                <button
                  onClick={onOpenTopicWizard}
                  className="flex items-center gap-1 px-2 py-1 text-xs rounded bg-primary/10 text-primary hover:bg-primary/20 transition-colors"
                  title="Discover skills through topics"
                >
                  <Sparkles className="w-3 h-3" />
                  <span className="hidden sm:inline">Discover</span>
                </button>
              </div>
              {topicViewMode === 'tree' ? (
                <TopicTreeView
                  topics={filteredTopics}
                  selectedTopicId={selectedTopicId}
                  onSelectTopic={onSelectTopicFromMenu ?? (() => {})}
                  className="flex-1"
                  detailMode={topicDetailMode}
                  isSelectMode={combineMode && combineEntityType === 'topics'}
                  selectedIds={combineSelectedIds}
                  onToggleSelection={(id) => handleAIResultToggle(id)}
                />
              ) : topicViewMode === 'list' ? (
                <TopicListPanel
                  selectedTopicId={selectedTopicId}
                  onSelectTopic={onSelectTopicFromMenu ?? (() => {})}
                  searchQuery={topicSearchQuery}
                  className="flex-1"
                  detailMode={topicDetailMode}
                  isSelectMode={combineMode && combineEntityType === 'topics'}
                  selectedIds={combineSelectedIds}
                  onToggleSelection={(id) => handleAIResultToggle(id)}
                />
              ) : (
                <TopicCardView
                  topics={filteredTopics}
                  selectedTopicId={selectedTopicId}
                  onSelectTopic={onSelectTopicFromMenu ?? (() => {})}
                  detailMode={topicDetailMode}
                  className="flex-1"
                  isSelectMode={combineMode && combineEntityType === 'topics'}
                  selectedIds={combineSelectedIds}
                  onToggleSelection={(id) => handleAIResultToggle(id)}
                />
              )}
            </>
          )}
          {/* Footer for topics */}
          {combineMode && combineEntityType === 'topics' && onCombineCopy && onExitCombineMode && onCombineFormatChange && (
            <div className="flex-shrink-0 px-3 py-3 border-t border-border">
              <CombineActionBar
                selectedCount={combineSelectedIds.size}
                format={combineFormat}
                onFormatChange={onCombineFormatChange}
                onCopy={onCombineCopy}
                onCancel={onExitCombineMode}
                isCopying={isCombineCopying}
                copySuccess={combineCopySuccess}
                entityLabel="topic"
              />
            </div>
          )}
          </>
          )}
        </Tabs.Content>

        <Tabs.Content value="actions" className="flex-1 flex flex-col min-h-0 data-[state=inactive]:hidden">
          {showSavedSets ? (
            editingSet ? (
              <SavedSetEditor
                entry={editingSet}
                entityType="actions"
                allEntities={allEntitiesForEditor}
                onSave={handleSavedSetEditorSave}
                onCancel={() => setEditingSet(null)}
              />
            ) : (
              <SavedSetsPanel
                entityType="actions"
                onApplySet={handleApplySavedSet}
                onEditSet={setEditingSet}
                entityLookup={entityLookup}
                refreshKey={savedSetsRefreshKey}
              />
            )
          ) : (
          <>
          {searchMode === 'ai' && actionSearchQuery.trim() ? (
            <div className="flex-1 overflow-y-auto px-3 py-2">
              {aiLoading && (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                  <Loader2 className="h-6 w-6 mb-2 animate-spin" />
                  <p className="text-xs">Searching actions...</p>
                </div>
              )}
              {aiError && (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                  <AlertCircle className="h-6 w-6 mb-2 text-destructive" />
                  <p className="text-xs text-destructive">{aiError}</p>
                </div>
              )}
              {!aiLoading && !aiError && (
                actionAIResults?.results && actionAIResults.results.length > 0 ? (
                  <SearchResultsList
                    entityType="actions"
                    actionResults={actionAIResults.results}
                    isSelectMode={combineMode}
                    selectedIds={combineSelectedIds}
                    onToggleSelection={handleAIResultToggle}
                    onNavigate={handleAIResultNavigate}
                    compact
                  />
                ) : (
                  <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <Search className="h-8 w-8 mb-2 opacity-60" />
                    <p className="text-xs">No results found</p>
                  </div>
                )
              )}
            </div>
          ) : (
            <ActionListPanel
              selectedActionId={selectedActionId}
              onSelectAction={onSelectActionFromMenu ?? (() => {})}
              searchQuery={actionSearchQuery}
              className="flex-1"
              isSelectMode={combineMode && combineEntityType === 'actions'}
              selectedIds={combineSelectedIds}
              onToggleSelection={(id) => handleAIResultToggle(id)}
            />
          )}
          {combineMode && combineEntityType === 'actions' && onCombineCopy && onExitCombineMode && onCombineFormatChange && (
            <div className="flex-shrink-0 px-3 py-3 border-t border-border">
              <CombineActionBar
                selectedCount={combineSelectedIds.size}
                format={combineFormat}
                onFormatChange={onCombineFormatChange}
                onCopy={onCombineCopy}
                onCancel={onExitCombineMode}
                isCopying={isCombineCopying}
                copySuccess={combineCopySuccess}
                entityLabel="action"
              />
            </div>
          )}
          </>
          )}
        </Tabs.Content>
      </Tabs.Root>

    </div>
  )
}
