/**
 * TeamEditorPanel - Full-panel editor for teams.
 *
 * Features:
 * - Header with close button, editable name, member count badge
 * - Editable mission statement
 * - Tabbed interface: Members, Roles, Skills, Info
 */

import { useState, useCallback } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import { X, Users, Shield, Zap, Info, ChevronDown, ChevronUp } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamDetails, UpdateTeamRequest, TeamRole, TeamMember, AddMemberRequest, UpdateMemberRequest } from '@/types/team'
import type { Skill } from '@/types'
import type { Agent } from '@/types/agent'
import { InlineEditableText } from '../shared/InlineEditableText'
import { ExpandableDescription } from '../shared/ExpandableDescription'
import { selectors } from '@/constants/selectors'

import { MembersTab, RolesTab, TeamSkillsTab, TeamInfoTab } from './teamTabs'

interface TeamEditorPanelProps {
  /** Current team being edited */
  team: TeamDetails | null
  /** All available skills for the skill picker */
  allSkills?: Skill[]
  /** All available agents for the member picker */
  allAgents?: Agent[]
  /** Callback when team data changes */
  onUpdate: (updates: UpdateTeamRequest) => Promise<void>
  /** Callback to add a member */
  onAddMember: (request: AddMemberRequest) => Promise<TeamMember>
  /** Callback to update a member */
  onUpdateMember: (agentId: string, request: UpdateMemberRequest) => Promise<TeamMember>
  /** Callback to remove a member */
  onRemoveMember: (agentId: string) => Promise<void>
  /** Callback to set roles */
  onSetRoles: (roles: TeamRole[]) => Promise<TeamRole[]>
  /** Callback to close the editor */
  onClose: () => void
  /** Additional class names */
  className?: string
}

/**
 * Team editor panel component.
 */
export function TeamEditorPanel({
  team,
  allSkills = [],
  allAgents = [],
  onUpdate,
  onAddMember,
  onUpdateMember,
  onRemoveMember,
  onSetRoles,
  onClose,
  className,
}: TeamEditorPanelProps) {
  // Active tab state
  const [activeTab, setActiveTab] = useState('members')

  // Mission expanded state
  const [isMissionExpanded, setIsMissionExpanded] = useState(false)

  // Handle name change
  const handleNameChange = useCallback(
    async (newName: string) => {
      if (team && newName !== team.displayName) {
        await onUpdate({ displayName: newName })
      }
    },
    [team, onUpdate]
  )

  // Handle mission change
  const handleMissionChange = useCallback(
    async (newMission: string) => {
      if (team) {
        await onUpdate({ mission: newMission })
      }
    },
    [team, onUpdate]
  )

  // Empty state when no team selected
  if (!team) {
    return (
      <div className={cn('h-full flex items-center justify-center', className)}>
        <div className="text-center">
          <Users className="h-16 w-16 mx-auto mb-4 text-muted-foreground/50" />
          <h3 className="text-lg font-medium text-muted-foreground">No Team Selected</h3>
          <p className="text-sm text-muted-foreground/70 max-w-xs mx-auto mt-2">
            Select a team from the list to view and edit its details.
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
        data-testid={selectors.teamEditor?.header ?? 'team-editor-header'}
      >
        {/* Row 1: Close, Icon, Name, Member count */}
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

          {/* Team icon */}
          <div className="flex-shrink-0">
            <div className="w-10 h-10 rounded-full flex items-center justify-center bg-primary/20">
              <Users className="h-5 w-5 text-primary" />
            </div>
          </div>

          {/* Editable name */}
          <div className="flex-1 min-w-0">
            <InlineEditableText
              value={team.displayName}
              onChange={(value) => void handleNameChange(value)}
              placeholder="Team name"
              className="text-lg font-semibold"
            />
          </div>

          {/* Member count badge */}
          <span className="px-2 py-1 text-xs font-medium bg-muted text-muted-foreground rounded-full">
            {team.memberCount} member{team.memberCount !== 1 ? 's' : ''}
          </span>
        </div>

        {/* Row 2: Expandable mission */}
        <div className="flex items-start gap-2">
          <button
            type="button"
            onClick={() => setIsMissionExpanded(!isMissionExpanded)}
            className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
          >
            {isMissionExpanded ? (
              <ChevronUp className="h-4 w-4" />
            ) : (
              <ChevronDown className="h-4 w-4" />
            )}
          </button>
          {isMissionExpanded ? (
            <ExpandableDescription
              value={team.mission ?? ''}
              onChange={(value) => void handleMissionChange(value)}
              placeholder="Add a mission statement..."
              className="flex-1"
            />
          ) : (
            <p className="flex-1 text-sm text-muted-foreground truncate">
              {team.mission || 'No mission statement'}
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
          <TabTrigger value="members" icon={<Users className="h-4 w-4" />} label="Members" />
          <TabTrigger value="roles" icon={<Shield className="h-4 w-4" />} label="Roles" />
          <TabTrigger value="skills" icon={<Zap className="h-4 w-4" />} label="Skills" />
          <TabTrigger value="info" icon={<Info className="h-4 w-4" />} label="Info" />
        </Tabs.List>

        {/* Tab Content */}
        <div className="flex-1 overflow-y-auto">
          <Tabs.Content value="members" className="h-full p-4">
            <MembersTab
              team={team}
              allAgents={allAgents}
              onAddMember={onAddMember}
              onUpdateMember={onUpdateMember}
              onRemoveMember={onRemoveMember}
            />
          </Tabs.Content>

          <Tabs.Content value="roles" className="h-full p-4">
            <RolesTab team={team} onSetRoles={onSetRoles} />
          </Tabs.Content>

          <Tabs.Content value="skills" className="h-full p-4">
            <TeamSkillsTab team={team} allSkills={allSkills} onUpdate={onUpdate} />
          </Tabs.Content>

          <Tabs.Content value="info" className="h-full p-4">
            <TeamInfoTab team={team} />
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
