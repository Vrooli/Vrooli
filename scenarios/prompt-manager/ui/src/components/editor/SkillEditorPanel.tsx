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
import { ChevronDown, ChevronUp, MoreHorizontal, RotateCcw, Trash2, Menu, X, ToggleLeft, ToggleRight, MessageSquare } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useSelectionStore } from '@/stores/selectionStore'
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
import { GraphSettingsContent } from '@/components/graph/GraphSettingsContent'
import { GraphHelpContent } from '@/components/graph/GraphHelpContent'
import { GraphQueryPanel } from '@/components/graph/GraphQueryPanel'
import { ViewOverlay } from '../shared/ViewOverlay'
import { IconSelector } from '../shared/IconSelector'
import { InlineEditableText } from '../shared/InlineEditableText'
import { DraftToggle } from '../shared/DraftToggle'
import { ExpandableDescription } from '../shared/ExpandableDescription'
import { TagChipsEditor } from '../shared/TagChipsEditor'
import { CrossReferencePanel } from './CrossReferencePanel'
import { StartChatDialog } from '../chat/StartChatDialog'
import { PanelErrorBoundary } from '../PanelErrorBoundary'
import { selectors } from '@/constants/selectors'
import { useIsCompactHeader } from '@/hooks/useMediaQuery'

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
  onSelectSkill?: (skillId: string) => void
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
  onSelectSkill,
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
  className,
}: SkillEditorPanelProps) {
  const [isDescriptionExpanded, setIsDescriptionExpanded] = useState(false)
  const [showChatDialog, setShowChatDialog] = useState(false)
  const isCompactHeader = useIsCompactHeader()
  const isMobileSidebarToggle = Boolean(onOpenSidebar)

  // Access the selection store for closing the editor
  const setSelectedSkillId = useSelectionStore((state) => state.setSelectedSkillId)

  // Handle close - return to skill tree view
  const handleClose = () => {
    if (onOpenSidebar) {
      onOpenSidebar()
      return
    }
    setSelectedSkillId(null)
  }

  // Graph view toggle
  const graphViewActive = useSelectionStore((state) => state.graphViewActive)

  // Show 3D world or graph view when no skill selected
  if (!currentSkill) {
    return (
      <div className={cn('relative h-full', className)}>
        {graphViewActive ? (
          <PanelErrorBoundary panelName="Graph View" className="h-full">
            <GraphView className="h-full" />
          </PanelErrorBoundary>
        ) : (
          <PanelErrorBoundary panelName="3D World" className="h-full">
            <WorldCanvas
              skills={allSkills}
              onSelectSkill={onSelectSkill}
              onDisplaySkills={onDisplaySkills}
            />
          </PanelErrorBoundary>
        )}
        <ViewOverlay
          onOpenMobileSidebar={onOpenMobileSidebar}
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
          {/* Row 1: Close, Icon, Name, Draft toggle, Unsaved indicator, Actions */}
          <div className="flex items-center gap-2 min-w-0">
            <div className="flex items-center gap-2 flex-1 min-w-0">
              {/* Close button */}
              <button
                type="button"
                onClick={handleClose}
                className="h-9 w-9 flex items-center justify-center rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
                aria-label={isMobileSidebarToggle ? 'Open sidebar' : 'Close editor and return to world'}
                title={isMobileSidebarToggle ? 'Open sidebar' : 'Close (Esc)'}
              >
                {isMobileSidebarToggle ? <Menu className="h-5 w-5" /> : <X className="h-5 w-5" />}
              </button>

              {/* Icon selector */}
              <IconSelector
                value={formState.icon}
                onChange={(v) => onFieldChange('icon', v)}
                isLoading={isLoadingContent}
                className="flex-shrink-0"
              />

              {/* Editable name */}
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

            <div className="flex items-center gap-2 flex-shrink-0">
              {/* Draft toggle */}
              <DraftToggle
                isDraft={formState.draft}
                onChange={(v) => onFieldChange('draft', v)}
                isLoading={isLoadingContent}
                className="flex-shrink-0 max-[389px]:hidden"
              />

              {/* Unsaved indicator */}
              {isDirty && (
                <div
                  className="hidden min-[390px]:flex items-center gap-1.5 px-2.5 py-1 bg-amber-500/20 text-amber-300 rounded-md text-xs font-medium flex-shrink-0"
                  data-testid={selectors.editor.unsavedIndicator}
                >
                  Unsaved
                </div>
              )}

              {/* Actions menu */}
              <ToolbarDropdown
                icon={<MoreHorizontal className="h-4 w-4" />}
                label="Skill actions"
                showChevron={false}
                align="right"
                className="h-9 w-9 p-0 rounded-lg"
                testId={selectors.editor.actionsMenu}
              >
                {isCompactHeader && (
                  <DropdownItem
                    onClick={() => onFieldChange('draft', !formState.draft)}
                    icon={formState.draft ? <ToggleRight className="h-4 w-4" /> : <ToggleLeft className="h-4 w-4" />}
                    label={formState.draft ? 'Set published' : 'Set draft'}
                  />
                )}
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
                  label={isDeleting ? 'Deleting...' : 'Delete skill'}
                />
              </ToolbarDropdown>
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

            {/* Start agent chat */}
            <button
              type="button"
              onClick={() => setShowChatDialog(true)}
              className="h-8 w-8 flex items-center justify-center rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
              aria-label="Start agent chat"
              title="Start agent chat with this skill"
            >
              <MessageSquare className="h-4 w-4" />
            </button>

            {/* File path menu with filename, breadcrumb, copy actions, and storage toggle */}
            <FilePathMenu
              file={formState.file}
              modes={formState.modes}
              folder={formState.folder}
              onFileChange={(v) => onFieldChange('file', v)}
              onFolderChange={(v) => onFieldChange('folder', v)}
              skillDir={currentSkill.skillDir ?? undefined}
              contentPath={currentSkill.contentPath ?? undefined}
              className="flex-shrink-0"
            />
          </div>
        </div>

        {/* Content area - full width */}
        <div className="flex-1 overflow-hidden">
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
