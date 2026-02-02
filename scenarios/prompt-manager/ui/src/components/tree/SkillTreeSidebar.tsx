/**
 * SkillTreeSidebar - Full tree sidebar for skill navigation.
 *
 * Adapted from agent-inbox ItemTreeSidebar for full-page experience.
 * Features:
 * - Mode-based tree navigation
 * - Search filtering
 * - Tag filtering
 * - Skill selection mode for agents
 * - Dirty indicators
 * - Collapse/expand controls
 * - New skill button
 */

import { type ReactNode, type RefObject, useState, useRef, useCallback, useEffect } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import { PanelLeftClose, PanelLeftOpen, Search, Plus, ChevronDown, ChevronUp, Settings, User, Users, Check, X, Sparkles, Layers } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TreeNode } from '@/types/editor'
import type { Skill, FolderType } from '@/types'
import type { Agent } from '@/types/agent'
import type { CombineFormat } from '@/stores/combineStore'
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
import { getAISearchStatus } from '@/services/skillService'
import { selectors } from '@/constants/selectors'
import { useSelectionStore } from '@/stores/selectionStore'

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
  // Skill selection mode props
  skillSelectionMode: boolean
  skillSelectedIds: Set<string>
  currentAgent: Agent | null
  onSkillSelectionSave: () => void
  onSkillSelectionCancel: () => void
  getSkillSelectionState: (node: TreeNode) => 'none' | 'partial' | 'all'
  onSkillCheckboxChange: (node: TreeNode) => void
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
  skillSelectionMode,
  skillSelectedIds,
  currentAgent,
  onSkillSelectionSave,
  onSkillSelectionCancel,
  getSkillSelectionState,
  onSkillCheckboxChange,
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

  // Active tab state - locked to skills when in skill selection mode
  const [activeTab, setActiveTab] = useState(initialActiveTab)
  const effectiveTab = skillSelectionMode ? 'skills' : activeTab

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
            ) : skillSelectionMode && currentAgent ? (
              <div className="flex items-center gap-2">
                <div
                  className="w-6 h-6 rounded-full flex items-center justify-center"
                  style={{ backgroundColor: currentAgent.appearance?.body ?? '#6366f1' }}
                >
                  <div
                    className="w-3 h-3 rounded-full"
                    style={{ backgroundColor: currentAgent.appearance?.head ?? '#818cf8' }}
                  />
                </div>
                <span className="text-xs font-medium text-foreground">
                  {skillSelectedIds.size} skill{skillSelectedIds.size !== 1 ? 's' : ''} selected
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
            {onOpenSettings && !skillSelectionMode && !combineMode && (
              <button
                type="button"
                onClick={onOpenSettings}
                className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                title="Settings (,)"
              >
                <Settings className="h-4 w-4" />
              </button>
            )}
            {!skillSelectionMode && !combineMode && (
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
        value={effectiveTab}
        onValueChange={skillSelectionMode ? undefined : handleTabChange}
        className="flex flex-col flex-1 min-h-0"
      >
        {/* Tab triggers */}
        <Tabs.List className="flex-shrink-0 flex border-b border-border">
          <Tabs.Trigger
            value="skills"
            disabled={skillSelectionMode}
            className={cn(
              'flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium',
              'border-b-2 border-transparent',
              'text-muted-foreground hover:text-foreground',
              'data-[state=active]:text-foreground data-[state=active]:border-primary',
              'transition-colors',
              skillSelectionMode && 'cursor-default'
            )}
            data-testid={selectors.sidebar.tabSkills}
          >
            <Search className="h-3.5 w-3.5" />
            Skills
          </Tabs.Trigger>
          <Tabs.Trigger
            value="agents"
            disabled={skillSelectionMode}
            className={cn(
              'flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium',
              'border-b-2 border-transparent',
              'text-muted-foreground hover:text-foreground',
              'data-[state=active]:text-foreground data-[state=active]:border-primary',
              'transition-colors',
              skillSelectionMode && 'opacity-50 cursor-not-allowed'
            )}
            data-testid={selectors.sidebar.tabAgents}
          >
            <User className="h-3.5 w-3.5" />
            Agents
          </Tabs.Trigger>
          <Tabs.Trigger
            value="teams"
            disabled={skillSelectionMode}
            className={cn(
              'flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium',
              'border-b-2 border-transparent',
              'text-muted-foreground hover:text-foreground',
              'data-[state=active]:text-foreground data-[state=active]:border-primary',
              'transition-colors',
              skillSelectionMode && 'opacity-50 cursor-not-allowed'
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
                placeholder={skillSelectionMode ? 'Search skills...' : 'Search skills... (Ctrl+K)'}
                className={cn(
                  'w-full pl-8 pr-3 py-1.5 text-xs',
                  'bg-muted border border-border rounded-md',
                  'text-foreground placeholder:text-muted-foreground',
                  'focus:outline-none focus:ring-2 focus:ring-primary'
                )}
                data-testid={selectors.sidebar.searchInput}
              />
            </div>

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
                {!skillSelectionMode && onEnterCombineMode && (
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
                  showCheckbox={skillSelectionMode || combineMode}
                  onCheckboxChange={combineMode ? onCombineToggle : onSkillCheckboxChange}
                  getSelectionState={combineMode ? getCombineSelectionState : getSkillSelectionState}
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
            ) : skillSelectionMode ? (
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={onSkillSelectionCancel}
                  className={cn(
                    'flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm',
                    'bg-muted hover:bg-muted/80 text-foreground rounded-lg transition-colors'
                  )}
                >
                  <X className="h-4 w-4" />
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={onSkillSelectionSave}
                  className={cn(
                    'flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm',
                    'bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg transition-colors'
                  )}
                >
                  <Check className="h-4 w-4" />
                  Save Skills
                </button>
              </div>
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
