/**
 * AgentEditorPanel - Full-panel editor for agents.
 *
 * Similar to SkillEditorPanel, provides a comprehensive editing interface for agents.
 * Features:
 * - Header with close button, 3D preview, editable name, and status toggle
 * - Expandable description
 * - Tabbed interface: Appearance, Skills, Persona, Info
 *
 * Appearance Tab:
 * - Live 3D preview
 * - Color pickers for body, head, accent
 *
 * Skills Tab:
 * - Assigned skills list with drag-drop reordering
 * - Add/remove skills
 *
 * Persona Tab:
 * - SOUL.md content editor (reuses SkillContentEditor)
 *
 * Info Tab:
 * - Read-only metadata (runtime, teams, created date)
 */

import { useState, useCallback, useMemo } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import { X, Palette, Zap, User, Info, ChevronDown, ChevronUp } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Agent, UpdateAgentRequest } from '@/types/agent'
import { DEFAULT_AGENT_COLORS } from '@/types/agent'
import { InlineEditableText } from '../shared/InlineEditableText'
import { ExpandableDescription } from '../shared/ExpandableDescription'
import { selectors } from '@/constants/selectors'

import { AppearanceTab, SkillsTab, PersonaTab, InfoTab } from './tabs'
import type { Skill } from '@/types'

interface AgentEditorPanelProps {
  /** Current agent being edited */
  agent: Agent | null
  /** All available skills for the skill picker */
  allSkills?: Skill[]
  /** Callback when agent data changes */
  onUpdate: (updates: UpdateAgentRequest) => Promise<void>
  /** Callback to close the editor */
  onClose: () => void
  /** Whether the agent is being saved */
  isSaving?: boolean
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
  allSkills = [],
  onUpdate,
  onClose,
  isSaving = false,
  className,
}: AgentEditorPanelProps) {
  // Active tab state
  const [activeTab, setActiveTab] = useState('appearance')

  // Description expanded state
  const [isDescriptionExpanded, setIsDescriptionExpanded] = useState(false)

  // Handle name change
  const handleNameChange = useCallback(
    async (newName: string) => {
      if (agent && newName !== agent.displayName) {
        await onUpdate({ displayName: newName })
      }
    },
    [agent, onUpdate]
  )

  // Handle description change
  const handleDescriptionChange = useCallback(
    async (newDescription: string) => {
      if (agent) {
        await onUpdate({ description: newDescription })
      }
    },
    [agent, onUpdate]
  )

  // Handle status change
  const handleStatusChange = useCallback(
    async (newStatus: string) => {
      if (agent && newStatus !== agent.status) {
        await onUpdate({ status: newStatus as 'active' | 'inactive' | 'suspended' })
      }
    },
    [agent, onUpdate]
  )

  // Extract colors with defaults
  const colors = useMemo(() => ({
    body: agent?.appearance?.body ?? DEFAULT_AGENT_COLORS.body,
    head: agent?.appearance?.head ?? DEFAULT_AGENT_COLORS.head,
    accent: agent?.appearance?.accent ?? DEFAULT_AGENT_COLORS.accent,
  }), [agent?.appearance])

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
        data-testid={selectors.agentEditor?.header ?? 'agent-editor-header'}
      >
        {/* Row 1: Close, Preview, Name, Status */}
        <div className="flex items-center gap-3">
          {/* Close button */}
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
            aria-label="Close editor"
            title="Close (Esc)"
          >
            <X className="h-5 w-5" />
          </button>

          {/* Mini 3D preview */}
          <div className="flex-shrink-0">
            <div
              className="w-10 h-10 rounded-full flex items-center justify-center"
              style={{ backgroundColor: colors.body }}
            >
              <div
                className="w-5 h-5 rounded-full"
                style={{ backgroundColor: colors.head }}
              />
            </div>
          </div>

          {/* Editable name */}
          <div className="flex-1 min-w-0">
            <InlineEditableText
              value={agent.displayName}
              onChange={(value) => void handleNameChange(value)}
              placeholder="Agent name"
              className="text-lg font-semibold"
            />
          </div>

          {/* Status toggle */}
          <AgentStatusToggle
            status={agent.status}
            onChange={handleStatusChange}
            disabled={isSaving}
          />
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
              value={agent.description ?? ''}
              onChange={(value) => void handleDescriptionChange(value)}
              placeholder="Add a description..."
              className="flex-1"
            />
          ) : (
            <p className="flex-1 text-sm text-muted-foreground truncate">
              {agent.description || 'No description'}
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
          <TabTrigger value="appearance" icon={<Palette className="h-4 w-4" />} label="Appearance" />
          <TabTrigger value="skills" icon={<Zap className="h-4 w-4" />} label="Skills" />
          <TabTrigger value="persona" icon={<User className="h-4 w-4" />} label="Persona" />
          <TabTrigger value="info" icon={<Info className="h-4 w-4" />} label="Info" />
        </Tabs.List>

        {/* Tab Content */}
        <div className="flex-1 overflow-y-auto">
          <Tabs.Content value="appearance" className="h-full p-4">
            <AppearanceTab agent={agent} onUpdate={onUpdate} />
          </Tabs.Content>

          <Tabs.Content value="skills" className="h-full p-4">
            <SkillsTab agent={agent} allSkills={allSkills} onUpdate={onUpdate} />
          </Tabs.Content>

          <Tabs.Content value="persona" className="h-full p-4">
            <PersonaTab agent={agent} onUpdate={onUpdate} />
          </Tabs.Content>

          <Tabs.Content value="info" className="h-full p-4">
            <InfoTab agent={agent} />
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
}

function AgentStatusToggle({ status, onChange, disabled }: AgentStatusToggleProps) {
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
        disabled && 'opacity-50 cursor-not-allowed'
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

