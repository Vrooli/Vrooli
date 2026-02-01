/**
 * SkillsTab - Agent skill assignment tab.
 *
 * Features:
 * - List of assigned skills
 * - Add/remove skills
 * - Uses centralized form state for dirty tracking
 * - Drag-drop reordering (future)
 */

import { useState, useCallback } from 'react'
import { Plus, X, GripVertical, Zap } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { NormalizedAgentFormState } from '@/stores/agentEditorStore'
import type { Skill } from '@/types'

interface SkillsTabProps {
  /** Form state from the editor store */
  formState: NormalizedAgentFormState
  /** All available skills for the picker */
  allSkills?: Skill[]
  /** Update a single field */
  updateField: <K extends keyof NormalizedAgentFormState>(field: K, value: NormalizedAgentFormState[K]) => void
}

/**
 * Skills assignment tab component.
 */
export function SkillsTab({ formState, allSkills = [], updateField }: SkillsTabProps) {
  const [showPicker, setShowPicker] = useState(false)

  // Get skills from form state
  const skills = formState.skills

  // Handle skill removal
  const handleRemoveSkill = useCallback(
    (skillId: string) => {
      const newSkills = skills.filter((id) => id !== skillId)
      updateField('skills', newSkills)
    },
    [skills, updateField]
  )

  // Handle skill addition
  const handleAddSkill = useCallback(
    (skillId: string) => {
      if (!skills.includes(skillId)) {
        updateField('skills', [...skills, skillId])
      }
      setShowPicker(false)
    },
    [skills, updateField]
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

  // Get available skills (not already assigned)
  const availableSkills = allSkills.filter((s) => !skills.includes(s.id))

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">
          Assigned Skills ({skills.length})
        </h3>
        <button
          type="button"
          onClick={() => setShowPicker(true)}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg',
            'bg-primary text-primary-foreground hover:bg-primary/90 transition-colors'
          )}
        >
          <Plus className="h-3.5 w-3.5" />
          Add Skills
        </button>
      </div>

      {/* Skills List */}
      {skills.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-8 text-center">
          <Zap className="h-10 w-10 text-muted-foreground/50 mb-3" />
          <p className="text-sm text-muted-foreground">No skills assigned</p>
          <p className="text-xs text-muted-foreground/70 mt-1">
            Click &quot;Add Skills&quot; to assign capabilities to this agent
          </p>
        </div>
      ) : (
        <ul className="space-y-2">
          {skills.map((skillId, index) => {
            const { name, description } = getSkillInfo(skillId)
            return (
              <li
                key={skillId}
                className={cn(
                  'flex items-center gap-2 px-3 py-2',
                  'bg-muted rounded-lg group',
                  'hover:bg-muted/80 transition-colors'
                )}
              >
                {/* Drag handle (visual only for now) */}
                <GripVertical className="h-4 w-4 text-muted-foreground/50 cursor-grab" />

                {/* Skill info */}
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{name}</p>
                  {description && (
                    <p className="text-xs text-muted-foreground truncate">
                      {description}
                    </p>
                  )}
                </div>

                {/* Order badge */}
                <span className="px-1.5 py-0.5 text-[10px] text-muted-foreground bg-background rounded">
                  #{index + 1}
                </span>

                {/* Remove button */}
                <button
                  type="button"
                  onClick={() => handleRemoveSkill(skillId)}
                  className={cn(
                    'p-1 rounded opacity-0 group-hover:opacity-100',
                    'text-muted-foreground hover:text-destructive hover:bg-destructive/10',
                    'transition-all'
                  )}
                  title="Remove skill"
                >
                  <X className="h-4 w-4" />
                </button>
              </li>
            )
          })}
        </ul>
      )}

      {/* Skill Picker Modal */}
      {showPicker && (
        <SkillPickerModal
          availableSkills={availableSkills}
          onSelect={handleAddSkill}
          onClose={() => setShowPicker(false)}
        />
      )}
    </div>
  )
}

/**
 * Modal for selecting skills to add.
 */
interface SkillPickerModalProps {
  availableSkills: Skill[]
  onSelect: (skillId: string) => void
  onClose: () => void
}

function SkillPickerModal({ availableSkills, onSelect, onClose }: SkillPickerModalProps) {
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
          <h3 className="font-medium">Add Skill</h3>
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
                ? 'All skills are already assigned'
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
