/**
 * TeamSkillsTab - Team skill grants configuration tab.
 *
 * Features:
 * - Show skillGrantsByRole configuration
 * - For each role, list assigned skills
 * - Add/remove skills to roles
 */

import { useState, useCallback, useMemo } from 'react'
import { Plus, X, Zap, Shield } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TeamDetails, UpdateTeamRequest } from '@/types/team'
import type { Skill } from '@/types'

interface TeamSkillsTabProps {
  team: TeamDetails
  /** All available skills */
  allSkills?: Skill[]
  onUpdate: (updates: UpdateTeamRequest) => Promise<void>
}

/**
 * Team skills configuration tab component.
 */
export function TeamSkillsTab({ team, allSkills = [], onUpdate }: TeamSkillsTabProps) {
  const [expandedRole, setExpandedRole] = useState<string | null>(null)
  const [showSkillPicker, setShowSkillPicker] = useState<string | null>(null)

  // Get skill grants by role
  const skillGrantsByRole = useMemo(() => {
    return team.defaults?.skillGrantsByRole ?? {}
  }, [team.defaults])

  // Handle adding a skill to a role
  const handleAddSkillToRole = useCallback(
    async (roleId: string, skillId: string) => {
      const currentSkills = skillGrantsByRole[roleId] ?? []
      if (currentSkills.includes(skillId)) {
        setShowSkillPicker(null)
        return
      }

      const updatedGrants = {
        ...skillGrantsByRole,
        [roleId]: [...currentSkills, skillId],
      }

      await onUpdate({
        defaults: { skillGrantsByRole: updatedGrants },
      })
      setShowSkillPicker(null)
    },
    [skillGrantsByRole, onUpdate]
  )

  // Handle removing a skill from a role
  const handleRemoveSkillFromRole = useCallback(
    async (roleId: string, skillId: string) => {
      const currentSkills = skillGrantsByRole[roleId] ?? []
      const updatedGrants = {
        ...skillGrantsByRole,
        [roleId]: currentSkills.filter((id) => id !== skillId),
      }

      await onUpdate({
        defaults: { skillGrantsByRole: updatedGrants },
      })
    },
    [skillGrantsByRole, onUpdate]
  )

  // Get skill info by ID
  const getSkillInfo = useCallback(
    (skillId: string): { name: string; description?: string } => {
      const skill = allSkills.find((s) => s.id === skillId)
      return {
        name: skill?.name ?? skillId,
        description: skill?.description,
      }
    },
    [allSkills]
  )

  // Get available skills for a role (not already assigned)
  const getAvailableSkillsForRole = useCallback(
    (roleId: string) => {
      const currentSkills = skillGrantsByRole[roleId] ?? []
      return allSkills.filter((s) => !currentSkills.includes(s.id))
    },
    [allSkills, skillGrantsByRole]
  )

  if (team.roles.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-center">
        <Shield className="h-10 w-10 text-muted-foreground/50 mb-3" />
        <p className="text-sm text-muted-foreground">No roles defined</p>
        <p className="text-xs text-muted-foreground/70 mt-1">
          Create team roles first to configure skill grants
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div>
        <h3 className="text-sm font-medium">Skill Grants by Role</h3>
        <p className="text-xs text-muted-foreground mt-1">
          Configure which skills are granted to team members based on their roles
        </p>
      </div>

      {/* Role sections */}
      <div className="space-y-3">
        {team.roles.map((role) => {
          const roleSkills = skillGrantsByRole[role.id] ?? []
          const isExpanded = expandedRole === role.id || roleSkills.length > 0

          return (
            <div
              key={role.id}
              className={cn(
                'rounded-lg border transition-colors',
                isExpanded ? 'border-border bg-muted/30' : 'border-transparent'
              )}
            >
              {/* Role header */}
              <button
                type="button"
                onClick={() => setExpandedRole(expandedRole === role.id ? null : role.id)}
                className={cn(
                  'w-full flex items-center justify-between px-3 py-2',
                  'text-left hover:bg-muted/50 rounded-lg transition-colors'
                )}
              >
                <div className="flex items-center gap-2">
                  <Shield className="h-4 w-4 text-primary" />
                  <span className="text-sm font-medium">{role.name}</span>
                  <span className="text-xs text-muted-foreground">
                    ({roleSkills.length} skill{roleSkills.length !== 1 ? 's' : ''})
                  </span>
                </div>
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation()
                    setShowSkillPicker(role.id)
                  }}
                  className="p-1 rounded text-muted-foreground hover:text-primary hover:bg-primary/10"
                  title="Add skill to role"
                >
                  <Plus className="h-4 w-4" />
                </button>
              </button>

              {/* Role skills */}
              {isExpanded && roleSkills.length > 0 && (
                <div className="px-3 pb-3">
                  <ul className="space-y-1">
                    {roleSkills.map((skillId) => {
                      const { name } = getSkillInfo(skillId)
                      return (
                        <li
                          key={skillId}
                          className={cn(
                            'flex items-center justify-between px-2 py-1.5',
                            'bg-background rounded group'
                          )}
                        >
                          <div className="flex items-center gap-2 min-w-0">
                            <Zap className="h-3.5 w-3.5 text-primary flex-shrink-0" />
                            <span className="text-sm truncate">{name}</span>
                          </div>
                          <button
                            type="button"
                            onClick={() => void handleRemoveSkillFromRole(role.id, skillId)}
                            className={cn(
                              'p-0.5 rounded opacity-0 group-hover:opacity-100',
                              'text-muted-foreground hover:text-destructive',
                              'transition-opacity'
                            )}
                            title="Remove skill"
                          >
                            <X className="h-3.5 w-3.5" />
                          </button>
                        </li>
                      )
                    })}
                  </ul>
                </div>
              )}
            </div>
          )
        })}
      </div>

      {/* Skill Picker Modal */}
      {showSkillPicker && (
        <SkillPickerModal
          roleId={showSkillPicker}
          roleName={team.roles.find((r) => r.id === showSkillPicker)?.name ?? ''}
          availableSkills={getAvailableSkillsForRole(showSkillPicker)}
          onSelect={(skillId) => void handleAddSkillToRole(showSkillPicker, skillId)}
          onClose={() => setShowSkillPicker(null)}
        />
      )}
    </div>
  )
}

/**
 * Modal for selecting skills to add to a role.
 */
interface SkillPickerModalProps {
  roleId: string
  roleName: string
  availableSkills: Skill[]
  onSelect: (skillId: string) => void
  onClose: () => void
}

function SkillPickerModal({
  roleName,
  availableSkills,
  onSelect,
  onClose,
}: SkillPickerModalProps) {
  const [search, setSearch] = useState('')

  // Filter skills by search query
  const filteredSkills = availableSkills.filter(
    (skill) =>
      skill.name.toLowerCase().includes(search.toLowerCase()) ||
      skill.description.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative w-full max-w-md mx-4 bg-card border border-border rounded-xl shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <h3 className="font-medium">Add Skill to {roleName}</h3>
          <button
            type="button"
            onClick={onClose}
            className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Search */}
        <div className="px-4 py-3 border-b border-border">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search skills..."
            className={cn(
              'w-full px-3 py-2 text-sm',
              'bg-muted border border-border rounded-lg',
              'text-foreground placeholder:text-muted-foreground',
              'focus:outline-none focus:ring-2 focus:ring-primary'
            )}
            autoFocus
          />
        </div>

        {/* Skills list */}
        <div className="max-h-64 overflow-y-auto">
          {filteredSkills.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-muted-foreground">
              {availableSkills.length === 0
                ? 'All skills are already assigned to this role'
                : 'No skills match your search'}
            </div>
          ) : (
            <ul className="p-2 space-y-1">
              {filteredSkills.map((skill) => (
                <li key={skill.id}>
                  <button
                    type="button"
                    onClick={() => onSelect(skill.id)}
                    className={cn(
                      'w-full flex items-start gap-3 px-3 py-2',
                      'rounded-lg text-left',
                      'hover:bg-muted transition-colors'
                    )}
                  >
                    <Zap className="h-4 w-4 mt-0.5 text-primary flex-shrink-0" />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium">{skill.name}</p>
                      {skill.description && (
                        <p className="text-xs text-muted-foreground line-clamp-2">
                          {skill.description}
                        </p>
                      )}
                    </div>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}
