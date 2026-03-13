/**
 * AgentEditorPanel - Full-panel editor for agents.
 *
 * Similar to SkillEditorPanel, provides a comprehensive editing interface for agents.
 * Uses centralized form state through useAgentEditor hook.
 *
 * Features:
 * - Header with close button, color badge, editable name, and status toggle
 * - Expandable description
 * - Tabbed interface: Files, Info, Prompt
 * - Dirty tracking with save/discard buttons
 * - Undo/redo support
 */

import { useEffect, useState } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import { Menu, X, Folder, User, Info, ChevronDown, ChevronUp, MoreHorizontal, Copy, Trash2, Eye, Circle } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Agent } from '@/types/agent'
import type { NormalizedAgentFormState } from '@/stores/agentEditorStore'
import type { ValidationResult } from '@/types/entityEditorStore'
import { useIsCompactHeader } from '@/hooks/useMediaQuery'
import { InlineEditableText } from '../shared/InlineEditableText'
import { ExpandableDescription } from '../shared/ExpandableDescription'
import { selectors } from '@/constants/selectors'

import type { HighlightRequest } from '@/lib/highlight'
import { InfoTab, FilesTab, PromptTab } from './tabs'
import { AgentColorBadge } from '../shared/AgentColorBadge'
import { ToolbarDropdown, DropdownItem } from './ToolbarDropdown'

interface AgentEditorPanelProps {
  /** Current agent being edited (for read-only metadata) */
  agent: Agent | null
  /** Form state from the editor store */
  formState: NormalizedAgentFormState
  /** Update a single field */
  updateField: <K extends keyof NormalizedAgentFormState>(field: K, value: NormalizedAgentFormState[K]) => void
  /** Update multiple fields at once */
  updateFields: (updates: Partial<NormalizedAgentFormState>) => void
  /** Sync file order on rename */
  renameFileOrderPath: (fromPath: string, toPath: string, isDir: boolean) => void
  /** Validation result */
  validation: ValidationResult
  /** Whether the form has unsaved changes */
  isDirty: boolean
  /** Count of dirty entities */
  dirtyCount: number
  /** Undo last change */
  onUndo: () => void
  /** Redo last undone change */
  onRedo: () => void
  /** Whether undo is available */
  canUndo: boolean
  /** Whether redo is available */
  canRedo: boolean
  /** Save current agent */
  onSave: () => void
  /** Save all dirty agents */
  onSaveAll: () => void
  /** Discard current changes */
  onDiscard: () => void
  /** Delete current agent */
  onDelete: () => void
  /** Duplicate current agent */
  onDuplicate: () => void
  /** Callback to close the editor */
  onClose: () => void
  /** Optional callback to open sidebar (used on mobile) */
  onOpenSidebar?: () => void
  /** Whether the agent is being saved */
  isSaving?: boolean
  /** Whether the agent is being deleted */
  isDeleting?: boolean
  /** Cross-reference highlight request */
  highlightRequest?: HighlightRequest | null
  /** Called after highlight is applied (clears URL params) */
  onHighlightHandled?: () => void
  /** Tab to open initially (e.g. 'prompt' from context menu) */
  initialTab?: string
  /** Additional class names */
  className?: string
}

/** Status options for agents */
const STATUS_OPTIONS = ['active', 'inactive', 'suspended'] as const

/**
 * Agent editor panel component.
 */
export function AgentEditorPanel({
  agent,
  formState,
  updateField,
  updateFields,
  renameFileOrderPath,
  validation,
  isDirty,
  dirtyCount,
  onUndo,
  onRedo,
  canUndo,
  canRedo,
  onSave,
  onSaveAll: _onSaveAll,
  onDiscard,
  onDelete,
  onDuplicate,
  onClose,
  onOpenSidebar,
  isSaving = false,
  isDeleting = false,
  highlightRequest,
  onHighlightHandled,
  initialTab,
  className,
}: AgentEditorPanelProps) {
  // TODO: Wire up save all button in the actions menu
  void _onSaveAll
  // Active tab state
  const [activeTab, setActiveTab] = useState(initialTab ?? 'files')
  const isCompactHeader = useIsCompactHeader()

  // Sync when initialTab changes (e.g. context menu → prompt tab)
  useEffect(() => {
    if (initialTab) setActiveTab(initialTab)
  }, [initialTab])

  // Description expanded state
  const [isDescriptionExpanded, setIsDescriptionExpanded] = useState(false)
  const isMobileSidebarToggle = Boolean(onOpenSidebar)

  // Empty state when no agent selected
  if (!agent) {
    return (
      <div className={cn('h-full flex items-center justify-center', className)}>
        <div className="text-center">
          <User className="h-16 w-16 mx-auto mb-4 text-muted-foreground/50" />
          <h3 className="text-lg font-medium text-muted-foreground">No Agent Selected</h3>
          <p className="text-sm text-muted-foreground/70 max-w-xs mx-auto mt-2">
            Select an agent from the list to view and edit their details.
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className={cn('h-full flex flex-col bg-card/50', className)}>
      {/* Header */}
      <div
        className="flex-shrink-0 px-4 py-3 border-b border-border space-y-2"
        data-testid={selectors.agentEditor.header}
      >
        {/* Row 1: Close, Name, Status */}
        <div className="flex items-center gap-2 min-w-0">
          {/* Close button */}
          <button
            type="button"
            onClick={onOpenSidebar ?? onClose}
            className="h-9 w-9 flex items-center justify-center rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
            aria-label={isMobileSidebarToggle ? 'Open sidebar' : 'Close editor'}
            title={isMobileSidebarToggle ? 'Open sidebar' : 'Close (Esc)'}
          >
            {isMobileSidebarToggle ? <Menu className="h-5 w-5" /> : <X className="h-5 w-5" />}
          </button>

          {/* Agent color badge - uses form state */}
          <AgentColorBadge appearance={formState.appearance} size="md" />

          {/* Editable name - uses form state */}
          <div className="flex-1 min-w-0">
            <InlineEditableText
              value={formState.displayName}
              onChange={(value) => updateField('displayName', value)}
              placeholder="Agent name"
              className="text-lg font-semibold"
              error={validation.errors.displayName}
            />
          </div>

          {/* Status toggle - uses form state */}
          <AgentStatusToggle
            status={formState.status}
            onChange={(status) => updateField('status', status as 'active' | 'inactive' | 'suspended')}
            disabled={isSaving}
            className="max-[389px]:hidden"
          />

          {/* Unsaved indicator */}
          {isDirty && (
            <div className="hidden min-[390px]:flex items-center gap-1.5 px-2.5 py-1 bg-amber-500/20 text-amber-300 rounded-md text-xs font-medium flex-shrink-0">
              Unsaved
            </div>
          )}

          {/* Actions menu */}
          <ToolbarDropdown
            icon={<MoreHorizontal className="h-4 w-4" />}
            label="Agent actions"
            showChevron={false}
            align="right"
            className="h-9 w-9 p-0 rounded-lg"
          >
            {isCompactHeader && (
              <>
                <DropdownItem
                  onClick={() => updateField('status', 'active')}
                  isActive={formState.status === 'active'}
                  disabled={isSaving}
                  icon={<Circle className="h-4 w-4" />}
                  label="Status: Active"
                />
                <DropdownItem
                  onClick={() => updateField('status', 'inactive')}
                  isActive={formState.status === 'inactive'}
                  disabled={isSaving}
                  icon={<Circle className="h-4 w-4" />}
                  label="Status: Inactive"
                />
                <DropdownItem
                  onClick={() => updateField('status', 'suspended')}
                  isActive={formState.status === 'suspended'}
                  disabled={isSaving}
                  icon={<Circle className="h-4 w-4" />}
                  label="Status: Suspended"
                />
              </>
            )}
            <DropdownItem
              onClick={onDuplicate}
              disabled={isSaving || isDeleting}
              icon={<Copy className="h-4 w-4" />}
              label="Duplicate agent"
            />
            <DropdownItem
              onClick={onDelete}
              disabled={isDeleting}
              icon={<Trash2 className="h-4 w-4 text-destructive" />}
              label={isDeleting ? 'Deleting...' : 'Delete agent'}
            />
          </ToolbarDropdown>
        </div>

        {/* Row 2: Expandable description */}
        <div className="flex items-start gap-2">
          <button
            type="button"
            onClick={() => setIsDescriptionExpanded(!isDescriptionExpanded)}
            className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
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
              onChange={(value) => updateField('description', value)}
              placeholder="Add a description..."
              className="flex-1"
            />
          ) : (
            <p className="flex-1 text-sm text-muted-foreground truncate">
              {formState.description || 'No description'}
            </p>
          )}
        </div>
      </div>

      {/* Tabs */}
      <Tabs.Root
        value={activeTab}
        onValueChange={setActiveTab}
        className="flex-1 flex flex-col min-h-0"
      >
        {/* Tab List */}
        <Tabs.List className="flex-shrink-0 flex border-b border-border px-4">
          <TabTrigger value="files" icon={<Folder className="h-4 w-4" />} label="Files" />
          <TabTrigger value="prompt" icon={<Eye className="h-4 w-4" />} label="Prompt" />
          <TabTrigger value="info" icon={<Info className="h-4 w-4" />} label="Info" />
        </Tabs.List>

        {/* Tab Content */}
        <div className="flex-1 overflow-y-auto">
          <Tabs.Content value="files" className="h-full">
            <FilesTab
              agentId={agent.id}
              agentDir={agent.agentDir ?? undefined}
              formState={formState}
              updateField={updateField}
              updateFields={updateFields}
              renameFileOrderPath={renameFileOrderPath}
              isDirty={isDirty}
              dirtyCount={dirtyCount}
              onUndo={onUndo}
              onRedo={onRedo}
              canUndo={canUndo}
              canRedo={canRedo}
              onSave={onSave}
              onDiscard={onDiscard}
              isSaving={isSaving}
              isValid={validation.valid}
              highlightRequest={highlightRequest}
              onHighlightHandled={onHighlightHandled}
            />
          </Tabs.Content>

          <Tabs.Content value="info" className="h-full p-4">
            <InfoTab agent={agent} />
          </Tabs.Content>

          <Tabs.Content value="prompt" className="h-full p-4">
            <PromptTab
              agent={agent}
              hasUnsavedChanges={isDirty}
              onNavigateToFile={() => setActiveTab('files')}
            />
          </Tabs.Content>
        </div>
      </Tabs.Root>
    </div>
  )
}

/**
 * Individual tab trigger button.
 */
interface TabTriggerProps {
  value: string
  icon: React.ReactNode
  label: string
}

function TabTrigger({ value, icon, label }: TabTriggerProps) {
  return (
    <Tabs.Trigger
      value={value}
      className={cn(
        'flex items-center gap-1.5 px-3 py-2 text-sm font-medium',
        'border-b-2 transition-colors',
        'data-[state=active]:border-primary data-[state=active]:text-primary',
        'data-[state=inactive]:border-transparent data-[state=inactive]:text-muted-foreground',
        'hover:text-foreground'
      )}
    >
      {icon}
      {label}
    </Tabs.Trigger>
  )
}

/**
 * Agent status toggle component.
 */
interface AgentStatusToggleProps {
  status: string
  onChange: (status: string) => void
  disabled?: boolean
  className?: string
}

function AgentStatusToggle({ status, onChange, disabled, className }: AgentStatusToggleProps) {
  const statusColors: Record<string, string> = {
    active: 'bg-green-500/20 text-green-500 border-green-500/30',
    inactive: 'bg-slate-500/20 text-slate-400 border-slate-500/30',
    suspended: 'bg-yellow-500/20 text-yellow-500 border-yellow-500/30',
  }

  return (
    <select
      value={status}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
      className={cn(
        'px-2 py-1 text-xs font-medium rounded-full border cursor-pointer',
        'focus:outline-none focus:ring-2 focus:ring-primary',
        statusColors[status] ?? statusColors.inactive,
        disabled && 'opacity-50 cursor-not-allowed',
        className
      )}
    >
      {STATUS_OPTIONS.map((opt) => (
        <option key={opt} value={opt}>
          {opt.charAt(0).toUpperCase() + opt.slice(1)}
        </option>
      ))}
    </select>
  )
}
