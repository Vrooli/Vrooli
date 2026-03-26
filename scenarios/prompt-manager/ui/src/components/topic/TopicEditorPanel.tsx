/**
 * TopicEditorPanel - Full-panel editor for topics.
 *
 * Features:
 * - Editable name and description fields
 * - Icon selector
 * - Parent topic dropdown
 * - Skill multi-select
 * - Save/discard actions, delete via header menu
 * - Dirty tracking
 */

import { useState, useEffect, useCallback, useMemo } from 'react'
import { Menu, X, Save, Trash2, RotateCcw, Layers, ChevronDown, MoreHorizontal } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useTopics, useTopic } from '@/hooks/useTopicData'
import { useSkillsData } from '@/hooks/useSkillsData'
import { SkillPicker } from '@/components/shared/SkillPicker'
import { ToolbarDropdown, DropdownItem } from '@/components/editor/ToolbarDropdown'
import type { Topic, UpdateTopicRequest } from '@/lib/schemas'

interface TopicEditorPanelProps {
  topicId: string
  onClose: () => void
  /** Optional callback to open sidebar (used on mobile) */
  onOpenSidebar?: () => void
  className?: string
}

/**
 * Determine if a topic ID would create a circular parent reference.
 * A topic cannot be its own parent, or a descendant's parent.
 */
function wouldCreateCycle(
  topicId: string,
  candidateParentId: string,
  topics: Topic[]
): boolean {
  if (topicId === candidateParentId) return true
  const topicMap = new Map(topics.map((t) => [t.id, t]))
  let current = topicMap.get(candidateParentId)
  while (current?.parentTopicId) {
    if (current.parentTopicId === topicId) return true
    current = topicMap.get(current.parentTopicId)
  }
  return false
}

/**
 * Topic editor panel component.
 */
export function TopicEditorPanel({
  topicId,
  onClose,
  onOpenSidebar,
  className,
}: TopicEditorPanelProps) {
  const { topic, isLoading: isLoadingTopic } = useTopic(topicId)
  const { topics, updateTopic, deleteTopic, isUpdating, isDeleting } = useTopics()
  const { skills } = useSkillsData()

  // Local form state
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [icon, setIcon] = useState('')
  const [parentTopicId, setParentTopicId] = useState<string | null>(null)
  const [selectedSkills, setSelectedSkills] = useState<string[]>([])
  const [isDirty, setIsDirty] = useState(false)
  const [showParentDropdown, setShowParentDropdown] = useState(false)

  // Sync form state when topic loads
  useEffect(() => {
    if (topic) {
      setName(topic.name)
      setDescription(topic.description)
      setIcon(topic.icon ?? '')
      setParentTopicId(topic.parentTopicId ?? null)
      setSelectedSkills(topic.skills)
      setIsDirty(false)
    }
  }, [topic])

  const markDirty = useCallback(() => setIsDirty(true), [])
  const isMobileSidebarToggle = Boolean(onOpenSidebar)

  // Parent topic options (excluding self and descendants to prevent cycles)
  const parentOptions = useMemo(() => {
    return topics.filter((t) => t.id !== topicId && !wouldCreateCycle(topicId, t.id, topics))
  }, [topics, topicId])

  const handleSave = async () => {
    const updates: UpdateTopicRequest = {
      name,
      description,
      icon: icon || undefined,
      parentTopicId: parentTopicId ?? '',
      skills: selectedSkills,
    }
    await updateTopic(topicId, updates)
    setIsDirty(false)
  }

  const handleDelete = async () => {
    await deleteTopic(topicId)
    onClose()
  }

  const handleDiscard = () => {
    if (topic) {
      setName(topic.name)
      setDescription(topic.description)
      setIcon(topic.icon ?? '')
      setParentTopicId(topic.parentTopicId ?? null)
      setSelectedSkills(topic.skills)
      setIsDirty(false)
    }
  }

  const selectedSkillSet = useMemo(() => new Set(selectedSkills), [selectedSkills])

  const handleSkillToggle = useCallback((skillId: string) => {
    setSelectedSkills((prev) =>
      prev.includes(skillId)
        ? prev.filter((s) => s !== skillId)
        : [...prev, skillId]
    )
    markDirty()
  }, [markDirty])

  const availableTags = useMemo(
    () => [...new Set(skills.flatMap((s) => s.tags))].sort(),
    [skills],
  )

  if (isLoadingTopic) {
    return (
      <div className={cn('flex items-center justify-center py-16', className)}>
        <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  if (!topic) {
    return (
      <div className={cn('flex flex-col items-center justify-center py-16', className)}>
        <Layers className="h-12 w-12 text-muted-foreground mb-4" />
        <p className="text-sm text-muted-foreground">Topic not found</p>
        <button
          type="button"
          onClick={onClose}
          className="mt-4 text-sm text-primary hover:underline"
        >
          Go back
        </button>
      </div>
    )
  }

  return (
    <div className={cn('flex flex-col h-full', className)}>
      {/* Header */}
      <div className="flex items-center gap-2 px-4 py-3 border-b border-border">
        {/* Close / hamburger button */}
        <button
          type="button"
          onClick={onOpenSidebar ?? onClose}
          className="h-9 w-9 flex items-center justify-center rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
          aria-label={isMobileSidebarToggle ? 'Open sidebar' : 'Close editor'}
          title={isMobileSidebarToggle ? 'Open sidebar' : 'Close (Esc)'}
        >
          {isMobileSidebarToggle ? <Menu className="h-5 w-5" /> : <X className="h-5 w-5" />}
        </button>

        <div className="flex-shrink-0 w-8 h-8 rounded-md bg-muted flex items-center justify-center">
          {icon ? (
            <span className="text-lg">{icon}</span>
          ) : (
            <Layers className="h-4 w-4 text-muted-foreground" />
          )}
        </div>
        <div className="flex-1 min-w-0">
          <h2 className="text-sm font-semibold text-foreground truncate">{name || 'Untitled Topic'}</h2>
        </div>

        {/* Unsaved indicator */}
        {isDirty && (
          <div className="hidden min-[390px]:flex items-center gap-1.5 px-2.5 py-1 bg-amber-500/20 text-amber-300 rounded-md text-xs font-medium flex-shrink-0">
            Unsaved
          </div>
        )}

        {/* Actions menu */}
        <ToolbarDropdown
          icon={<MoreHorizontal className="h-4 w-4" />}
          label="Topic actions"
          showChevron={false}
          align="right"
          className="h-9 w-9 p-0 rounded-lg"
        >
          <DropdownItem
            onClick={handleDiscard}
            disabled={!isDirty}
            icon={<RotateCcw className="h-4 w-4" />}
            label="Discard changes"
          />
          <DropdownItem
            onClick={() => void handleDelete()}
            disabled={isDeleting}
            icon={<Trash2 className="h-4 w-4 text-destructive" />}
            label={isDeleting ? 'Deleting...' : 'Delete topic'}
          />
        </ToolbarDropdown>
      </div>

      {/* Form */}
      <div className="flex-1 overflow-y-auto px-4 py-4 space-y-5">
        {/* Name */}
        <div>
          <label htmlFor="topic-name" className="block text-xs font-medium text-muted-foreground mb-1.5">
            Name
          </label>
          <input
            id="topic-name"
            type="text"
            value={name}
            onChange={(e) => { setName(e.target.value); markDirty() }}
            className="w-full px-3 py-2 text-sm bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-primary/50"
            placeholder="Topic name"
          />
        </div>

        {/* Description */}
        <div>
          <label htmlFor="topic-description" className="block text-xs font-medium text-muted-foreground mb-1.5">
            Description
          </label>
          <textarea
            id="topic-description"
            value={description}
            onChange={(e) => { setDescription(e.target.value); markDirty() }}
            rows={3}
            className="w-full px-3 py-2 text-sm bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-primary/50 resize-y"
            placeholder="Optional description"
          />
        </div>

        {/* Icon */}
        <div>
          <label htmlFor="topic-icon" className="block text-xs font-medium text-muted-foreground mb-1.5">
            Icon (emoji)
          </label>
          <input
            id="topic-icon"
            type="text"
            value={icon}
            onChange={(e) => { setIcon(e.target.value); markDirty() }}
            className="w-20 px-3 py-2 text-sm bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-primary/50 text-center"
            placeholder="..."
            maxLength={4}
          />
        </div>

        {/* Parent Topic */}
        <div className="relative">
          <label className="block text-xs font-medium text-muted-foreground mb-1.5">
            Parent Topic
          </label>
          <button
            type="button"
            onClick={() => setShowParentDropdown((v) => !v)}
            className="w-full flex items-center justify-between px-3 py-2 text-sm bg-background border border-border rounded-md hover:border-primary/50 transition-colors"
          >
            <span className={cn(!parentTopicId && 'text-muted-foreground')}>
              {parentTopicId
                ? parentOptions.find((t) => t.id === parentTopicId)?.name ?? 'Unknown'
                : 'None (root topic)'}
            </span>
            <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
          </button>
          {showParentDropdown && (
            <div className="absolute z-10 mt-1 w-full max-h-48 overflow-y-auto bg-popover border border-border rounded-md shadow-lg">
              <button
                type="button"
                onClick={() => { setParentTopicId(null); setShowParentDropdown(false); markDirty() }}
                className={cn(
                  'w-full px-3 py-2 text-sm text-left hover:bg-muted/50 transition-colors',
                  !parentTopicId && 'bg-primary/10 text-primary'
                )}
              >
                None (root topic)
              </button>
              {parentOptions.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => { setParentTopicId(t.id); setShowParentDropdown(false); markDirty() }}
                  className={cn(
                    'w-full px-3 py-2 text-sm text-left hover:bg-muted/50 transition-colors',
                    parentTopicId === t.id && 'bg-primary/10 text-primary'
                  )}
                >
                  {t.icon ? `${t.icon} ` : ''}{t.name}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Skills */}
        <div>
          <label className="block text-xs font-medium text-muted-foreground mb-1.5">
            Skills
          </label>

          {/* Selected skill chips */}
          {selectedSkills.length > 0 && (
            <div className="flex flex-wrap gap-1.5 mb-2">
              {selectedSkills.map((skillId) => {
                const skill = skills.find((s) => s.id === skillId)
                return (
                  <span
                    key={skillId}
                    className="inline-flex items-center gap-1 px-2 py-0.5 text-xs bg-primary/10 text-primary rounded-full"
                  >
                    {skill?.name ?? skillId}
                    <button
                      type="button"
                      onClick={() => handleSkillToggle(skillId)}
                      className="hover:text-destructive transition-colors"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </span>
                )
              })}
            </div>
          )}

          <SkillPicker
            skills={skills}
            selectedIds={selectedSkillSet}
            onToggle={handleSkillToggle}
            availableTags={availableTags}
          />
        </div>
      </div>

      {/* Footer actions */}
      <div className="flex-shrink-0 flex items-center gap-2 px-4 py-3 border-t border-border">
        <button
          type="button"
          onClick={() => void handleSave()}
          disabled={!isDirty || isUpdating}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-md transition-colors',
            isDirty
              ? 'bg-primary text-primary-foreground hover:bg-primary/90'
              : 'bg-muted text-muted-foreground cursor-not-allowed'
          )}
        >
          <Save className="h-3.5 w-3.5" />
          {isUpdating ? 'Saving...' : 'Save'}
        </button>

        <button
          type="button"
          onClick={handleDiscard}
          disabled={!isDirty}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-md transition-colors',
            isDirty
              ? 'bg-muted hover:bg-muted/80 text-foreground'
              : 'text-muted-foreground cursor-not-allowed'
          )}
        >
          <RotateCcw className="h-3.5 w-3.5" />
          Discard
        </button>
      </div>
    </div>
  )
}
