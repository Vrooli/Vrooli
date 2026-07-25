/**
 * SkillEditorPanel - Container component for the editor sections.
 *
 * Brings together:
 * - SkillContentEditor (full width)
 * - WorldCanvas (when no skill selected)
 *
 * Header contains:
 * - Icon selector
 * - Inline editable name
 * - Draft toggle
 * - Actions menu (discard/delete)
 * - Expandable description
 * - Tag chips editor
 * - Mode path display
 *
 * Also handles:
 * - Empty state with 3D world visualization
 */

import { useState } from 'react'
import { ChevronDown, ChevronUp, MoreHorizontal, RotateCcw, Trash2, Menu, X, MessageSquare, GitBranch, FolderOpen } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { NormalizedFormState, ValidationResult } from '@/types/editorStore'
import type { Skill } from '@/types'
import type { DisplayFormat } from '@/types/world'
import type { ContentSearchMatch, Reference } from '@/lib/schemas'
import { SkillContentEditor } from './SkillContentEditor'
import { FilePathMenu } from './FilePathMenu'
import { ScopeSelector } from './ScopeSelector'
import { ToolbarDropdown, DropdownItem } from './ToolbarDropdown'
import { WorldCanvas } from '@/components/world'
import { WorldSettingsContent } from '@/components/world/WorldSettingsContent'
import { WorldHelpContent } from '@/components/world/WorldHelpContent'
import { GraphView } from '@/components/graph/GraphView'
import { OperatingMapFlow } from '@/components/graph/OperatingMapFlow'
import { GraphSettingsContent } from '@/components/graph/GraphSettingsContent'
import { GraphHelpContent } from '@/components/graph/GraphHelpContent'
import { GraphQueryPanel } from '@/components/graph/GraphQueryPanel'
import { ViewOverlay } from '../shared/ViewOverlay'
import { IconSelector } from '../shared/IconSelector'
import { InlineEditableText } from '../shared/InlineEditableText'
import { DraftStatusChip } from '../shared/DraftStatusChip'
import { ExpandableDescription } from '../shared/ExpandableDescription'
import { TagChipsEditor } from '../shared/TagChipsEditor'
import { CrossReferencePanel } from './CrossReferencePanel'
import { StartChatDialog } from '../chat/StartChatDialog'
import { LineagePanel, type LineageTab } from './LineagePanel'
import { PanelErrorBoundary } from '../PanelErrorBoundary'
import { selectors } from '@/constants/selectors'

interface SkillEditorPanelProps {
  // Current state
  currentSkill: Skill | null
  formState: NormalizedFormState
  originalContent?: string | null
  validation: ValidationResult

  // All skills for skill tree
  allSkills?: Skill[]

  // Dirty tracking
  isDirty: boolean
  dirtyCount: number

  // Form operations
  onFieldChange: <K extends keyof NormalizedFormState>(field: K, value: NormalizedFormState[K]) => void

  // Available tags for autocomplete
  availableTags?: string[]

  // Undo/Redo
  onUndo?: () => void
  onRedo?: () => void
  canUndo?: boolean
  canRedo?: boolean

  // Actions
  onSave: () => void
  onSaveAll: () => void
  onDiscard: () => void
  onDelete: () => void
  onClose?: () => void
  onSelectSkill?: (skillId: string) => void
  onSelectTeam?: (teamId: string) => void
  onDisplaySkills?: (combined: string, format: DisplayFormat) => void

  // Loading states
  isSaving: boolean
  isDeleting: boolean
  isLoadingContent?: boolean

  // Search highlighting
  searchMatches?: ContentSearchMatch[]
  /** Line number to scroll to in the editor */
  scrollToLine?: number | null
  /** Called after the editor has scrolled to the requested line */
  onScrollToLineHandled?: () => void

  /** Called when a cross-reference is clicked, for highlight navigation */
  onNavigateToXRef?: (ref: Reference) => void
  onOpenSidebar?: () => void
  onOpenMobileSidebar?: () => void
  /** Notification counts for mobile hamburger badge */
  pendingDecisionCount?: number
  runningAgentCount?: number
  homeView?: 'world' | 'graph'
  onHomeViewChange?: (view: 'world' | 'graph') => void

  className?: string
}

/**
 * Main editor panel component.
 */
export function SkillEditorPanel({
  currentSkill,
  formState,
  originalContent,
  validation,
  allSkills = [],
  isDirty,
  dirtyCount,
  onFieldChange,
  availableTags = [],
  onUndo,
  onRedo,
  canUndo = false,
  canRedo = false,
  onSave,
  onSaveAll,
  onDiscard,
  onDelete,
  onClose,
  onSelectSkill,
  onSelectTeam,
  onDisplaySkills,
  isSaving,
  isDeleting,
  isLoadingContent = false,
  searchMatches = [],
  scrollToLine,
  onScrollToLineHandled,
  onNavigateToXRef,
  onOpenSidebar,
  onOpenMobileSidebar,
  pendingDecisionCount,
  runningAgentCount,
  homeView = 'world',
  onHomeViewChange,
  className,
}: SkillEditorPanelProps) {
  const [isDescriptionExpanded, setIsDescriptionExpanded] = useState(false)
  const [showChatDialog, setShowChatDialog] = useState(false)
  const [lineageOpen, setLineageOpen] = useState(false)
  const [lineageTab, setLineageTab] = useState<LineageTab>('history')
  const [graphProjection, setGraphProjection] = useState<'relationships' | 'flow'>(() =>
    typeof window !== 'undefined' && localStorage.getItem('pm.graphProjection') === 'flow' ? 'flow' : 'relationships',
  )
  const selectGraphProjection = (projection: 'relationships' | 'flow') => {
    setGraphProjection(projection)
    localStorage.setItem('pm.graphProjection', projection)
  }
  const isMobileSidebarToggle = Boolean(onOpenSidebar)

  const handleClose = () => {
    if (onOpenSidebar) {
      onOpenSidebar()
      return
    }
    onClose?.()
  }

  const graphViewActive = homeView === 'graph'

  // Show 3D world or graph view when no skill selected
  if (!currentSkill) {
    return (
      <div className={cn('relative h-full', className)}>
        {graphViewActive ? (
          <PanelErrorBoundary panelName="Graph View" className="h-full">
            <div className="h-full">
              <div
                data-testid="graph-projection-toggle"
                className="absolute bottom-32 left-4 z-40 flex gap-1 rounded-lg border bg-background/95 p-1 shadow-sm sm:bottom-auto sm:left-3 sm:top-3"
              >
                {(['relationships', 'flow'] as const).map((projection) => (
                  <button
                    key={projection}
                    type="button"
                    className={cn(
                      'rounded-md px-2.5 py-1.5 text-xs font-medium transition-colors',
                      graphProjection === projection
                        ? 'bg-primary text-primary-foreground'
                        : 'text-muted-foreground hover:bg-muted hover:text-foreground',
                    )}
                    aria-pressed={graphProjection === projection}
                    onClick={() => selectGraphProjection(projection)}
                  >
                    {projection === 'relationships' ? 'Relationships' : 'Flow'}
                  </button>
                ))}
              </div>
              {graphProjection === 'flow' ? <OperatingMapFlow /> : <GraphView className="h-full" />}
            </div>
          </PanelErrorBoundary>
        ) : (
          <PanelErrorBoundary panelName="3D World" className="h-full">
            <WorldCanvas
              skills={allSkills}
              onSelectSkill={onSelectSkill}
              onSelectTeam={onSelectTeam}
              onDisplaySkills={onDisplaySkills}
            />
          </PanelErrorBoundary>
        )}
        <ViewOverlay
          onOpenMobileSidebar={onOpenMobileSidebar}
          pendingDecisionCount={pendingDecisionCount}
          runningAgentCount={runningAgentCount}
          homeView={homeView}
          onHomeViewChange={onHomeViewChange}
          leftPanelContent={graphViewActive ? (
            <>
              <PanelErrorBoundary panelName="Graph Queries">
                <GraphQueryPanel />
              </PanelErrorBoundary>
            </>
          ) : undefined}
          settingsContent={graphViewActive ? <GraphSettingsContent /> : <WorldSettingsContent />}
          settingsTitle={graphViewActive ? 'Graph Settings' : 'World Settings'}
          helpContent={graphViewActive ? <GraphHelpContent /> : <WorldHelpContent />}
          helpTitle={graphViewActive ? 'Graph Help' : 'Avatar Environment'}
        />
      </div>
    )
  }

  const canDiscard = isDirty && !isSaving
  const canDelete = !isDeleting

  return (
    <div className={cn('h-full', className)}>
      <div className="flex flex-col h-full bg-card/50">
        {/* Header with all metadata */}
        <div
          className="flex-shrink-0 px-4 py-3 border-b border-border space-y-2"
          data-testid={selectors.editor.header}
        >
          {/* Row 1: Sidebar/close, icon, title — nothing else so the title gets all remaining space. */}
          <div className="flex items-center gap-2 min-w-0">
            <button
              type="button"
              onClick={handleClose}
              className="h-9 w-9 flex items-center justify-center rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
              aria-label={isMobileSidebarToggle ? 'Open sidebar' : 'Close editor and return to world'}
              title={isMobileSidebarToggle ? 'Open sidebar' : 'Close (Esc)'}
            >
              {isMobileSidebarToggle ? <Menu className="h-5 w-5" /> : <X className="h-5 w-5" />}
            </button>

            <IconSelector
              value={formState.icon}
              onChange={(v) => onFieldChange('icon', v)}
              isLoading={isLoadingContent}
              className="flex-shrink-0"
            />

            <div className="flex-1 min-w-0">
              <InlineEditableText
                value={formState.name}
                onChange={(v) => onFieldChange('name', v)}
                placeholder="Untitled Skill"
                error={validation.errors.name}
                as="h2"
                isLoading={isLoadingContent}
                className="text-foreground"
                displayTestId={selectors.editor.nameDisplay}
                inputTestId={selectors.editor.nameInput}
              />
            </div>
          </div>

          {/* Row 2: Description */}
          <div className="flex items-start gap-2">
            <button
              type="button"
              onClick={() => setIsDescriptionExpanded((prev) => !prev)}
              className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
              aria-label={isDescriptionExpanded ? 'Collapse description' : 'Expand description'}
            >
              {isDescriptionExpanded ? (
                <ChevronUp className="h-4 w-4" />
              ) : (
                <ChevronDown className="h-4 w-4" />
              )}
            </button>
            {isDescriptionExpanded ? (
              <ExpandableDescription
                value={formState.description}
                onChange={(v) => onFieldChange('description', v)}
                placeholder="Click to add description..."
                error={validation.errors.description}
                isLoading={isLoadingContent}
                className="flex-1 text-muted-foreground"
              />
            ) : (
              <p className="flex-1 text-sm text-muted-foreground truncate">
                {formState.description || 'No description'}
              </p>
            )}
          </div>

          {/* Row 3: Tags, DefaultScope (for steer skills), references, and file path menu */}
          <div className="flex items-center gap-4 flex-wrap">
            <TagChipsEditor
              value={formState.tags}
              onChange={(v) => onFieldChange('tags', v)}
              availableTags={availableTags}
              placeholder="Add tags..."
              isLoading={isLoadingContent}
              className="flex-1 min-w-0"
            />

            {/* Draft status is shown only when it needs attention. Published is the default. */}
            {formState.draft && (
              <DraftStatusChip
                isDraft={formState.draft}
                onChange={(v) => onFieldChange('draft', v)}
                isLoading={isLoadingContent}
                className="flex-shrink-0"
              />
            )}

            {/* Default scope selector - only show for steer skills */}
            {formState.modes.includes('steer') && (
              <ScopeSelector
                value={formState.defaultScope}
                onChange={(v) => onFieldChange('defaultScope', v)}
                allSkills={allSkills}
                className="flex-shrink-0"
              />
            )}

            {/* Cross-references icon button */}
            <CrossReferencePanel
              skillId={currentSkill.id}
              onNavigateToReference={onNavigateToXRef}
              className="flex-shrink-0"
            />

            <FilePathMenu
              file={formState.file}
              modes={formState.modes}
              folder={formState.folder}
              onFileChange={(v) => onFieldChange('file', v)}
              onFolderChange={(v) => onFieldChange('folder', v)}
              skillDir={currentSkill.skillDir ?? undefined}
              contentPath={currentSkill.contentPath ?? undefined}
              triggerIcon={<FolderOpen className="h-4 w-4 text-muted-foreground" />}
              className="flex-shrink-0"
            />

            {/* Lineage button: opens tabbed panel with history, variants, experiments */}
            <button
              type="button"
              onClick={() => setLineageOpen((v) => !v)}
              className={cn(
                'h-8 px-2.5 flex items-center gap-1.5 rounded-lg border border-border/50 text-sm transition-colors flex-shrink-0',
                lineageOpen
                  ? 'bg-primary/20 text-primary'
                  : 'text-muted-foreground hover:text-foreground hover:bg-muted'
              )}
              aria-label={lineageOpen ? 'Close lineage panel' : 'Open lineage panel'}
              aria-expanded={lineageOpen}
              title="Lineage (history, variants, experiments)"
              data-testid="lineage-toggle"
            >
              <GitBranch className="h-4 w-4" />
              <span className="hidden sm:inline">Lineage</span>
            </button>

            {/* More overflow menu: chat, discard, delete */}
            <span className="relative flex-shrink-0">
                <ToolbarDropdown
                  icon={<MoreHorizontal className="h-4 w-4" />}
                  label="More actions"
                  showChevron={false}
                  align="right"
                  className="h-8 w-8 p-0 rounded-lg"
                  testId={selectors.editor.actionsMenu}
                >
                  <DropdownItem
                    onClick={() => setShowChatDialog(true)}
                    icon={<MessageSquare className="h-4 w-4" />}
                    label="Start agent chat"
                  />
                  <DropdownItem
                    onClick={onDiscard}
                    disabled={!canDiscard}
                    icon={<RotateCcw className="h-4 w-4" />}
                    label="Discard changes"
                    testId={selectors.editor.discardAction}
                  />
                  <DropdownItem
                    onClick={onDelete}
                    disabled={!canDelete}
                    icon={<Trash2 className="h-4 w-4 text-destructive" />}
                    label={isDeleting ? 'Deleting…' : 'Delete skill'}
                  />
                </ToolbarDropdown>
            </span>
          </div>
        </div>

        {/* Content area with optional right sidebar */}
        <div className="flex-1 overflow-hidden flex">
          <div className={cn('flex-1 overflow-hidden', lineageOpen && 'min-w-0')}>
            <SkillContentEditor
              value={formState.content}
              originalValue={originalContent ?? undefined}
              onChange={(v) => onFieldChange('content', v)}
              error={validation.errors.content}
              isDirty={isDirty}
              dirtyCount={dirtyCount}
              onUndo={onUndo}
              onRedo={onRedo}
              canUndo={canUndo}
              canRedo={canRedo}
              onSave={onSave}
              onSaveAll={onSaveAll}
              onDiscard={onDiscard}
              isSaving={isSaving}
              isValid={validation.valid}
              searchMatches={searchMatches}
              scrollToLine={scrollToLine}
              onScrollToLineHandled={onScrollToLineHandled}
              className="h-full"
            />
          </div>

          {/* Right sidebar: lineage panel with tabs */}
          {lineageOpen && (
            <div className="w-72 flex-shrink-0 border-l border-border overflow-hidden flex flex-col">
              <PanelErrorBoundary panelName="Lineage panel">
                <LineagePanel
                  skillId={currentSkill.id}
                  currentContent={formState.content}
                  activeTab={lineageTab}
                  onActiveTabChange={setLineageTab}
                  className="h-full"
                />
              </PanelErrorBoundary>
            </div>
          )}
        </div>

        {/* Agent chat dialog */}
        <StartChatDialog
          isOpen={showChatDialog}
          onClose={() => setShowChatDialog(false)}
          initialSkill={currentSkill}
          allSkills={allSkills}
        />
      </div>
    </div>
  )
}
