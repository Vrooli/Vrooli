/**
 * PersonaTab - Agent persona configuration tab with full markdown editor.
 *
 * Features:
 * - Full markdown editor for system prompt (code/WYSIWYG modes, preview, diff)
 * - Voice style selector
 * - Traits editor
 * - Uses centralized form state for dirty tracking
 */

import { useState, useCallback } from 'react'
import { ChevronDown, ChevronUp } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { AgentVoice } from '@/types/agent'
import type { NormalizedAgentFormState } from '@/stores/agentEditorStore'
import { SkillContentEditor } from '../SkillContentEditor'

interface PersonaTabProps {
  /** Form state from the editor store */
  formState: NormalizedAgentFormState
  /** Original state for diff comparison */
  originalState: NormalizedAgentFormState | null
  /** Update a single field */
  updateField: <K extends keyof NormalizedAgentFormState>(field: K, value: NormalizedAgentFormState[K]) => void
  /** Update multiple fields at once */
  updateFields: (updates: Partial<NormalizedAgentFormState>) => void
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
  /** Save current changes */
  onSave: () => void
  /** Discard current changes */
  onDiscard: () => void
  /** Whether saving is in progress */
  isSaving: boolean
  /** Whether the form is valid */
  isValid: boolean
}

/** Available voice styles */
const VOICE_STYLES: Array<{ value: AgentVoice; label: string; description: string }> = [
  { value: 'professional', label: 'Professional', description: 'Formal and business-like' },
  { value: 'casual', label: 'Casual', description: 'Friendly and relaxed' },
  { value: 'technical', label: 'Technical', description: 'Precise and detail-oriented' },
  { value: 'empathetic', label: 'Empathetic', description: 'Warm and understanding' },
  { value: 'terse', label: 'Terse', description: 'Brief and to-the-point' },
]

/**
 * Persona configuration tab component with markdown editor.
 */
export function PersonaTab({
  formState,
  originalState,
  updateField,
  updateFields: _updateFields,
  isDirty,
  dirtyCount,
  onUndo,
  onRedo,
  canUndo,
  canRedo,
  onSave,
  onDiscard,
  isSaving,
  isValid,
}: PersonaTabProps) {
  // TODO: Use updateFields for batch updates if needed
  void _updateFields

  // Local state for UI (settings panel expansion, new trait input)
  const [newTrait, setNewTrait] = useState('')
  const [showSettings, setShowSettings] = useState(false)

  // Extract values from form state (persona is always defined in NormalizedAgentFormState)
  const content = formState.persona.systemPromptPrefix ?? ''
  const voice = formState.persona.voice ?? 'professional'
  const traits = formState.persona.traits

  // Original content for diff view
  const originalContent = originalState?.persona.systemPromptPrefix ?? ''

  // Handle content change
  const handleContentChange = useCallback((newContent: string) => {
    updateField('persona', {
      ...formState.persona,
      systemPromptPrefix: newContent,
    })
  }, [formState.persona, updateField])

  // Handle voice change
  const handleVoiceChange = useCallback((newVoice: AgentVoice) => {
    updateField('persona', {
      ...formState.persona,
      voice: newVoice,
    })
  }, [formState.persona, updateField])

  // Handle trait addition
  const handleAddTrait = useCallback(() => {
    if (newTrait.trim() && !traits.includes(newTrait.trim())) {
      updateField('persona', {
        ...formState.persona,
        traits: [...traits, newTrait.trim()],
      })
      setNewTrait('')
    }
  }, [newTrait, traits, formState.persona, updateField])

  // Handle trait removal
  const handleRemoveTrait = useCallback((trait: string) => {
    updateField('persona', {
      ...formState.persona,
      traits: traits.filter((t) => t !== trait),
    })
  }, [traits, formState.persona, updateField])

  return (
    <div className="flex flex-col h-full">
      {/* Settings Panel (collapsible) */}
      <div className="flex-shrink-0 border-b border-border">
        <button
          type="button"
          onClick={() => setShowSettings(!showSettings)}
          className={cn(
            'w-full flex items-center justify-between px-4 py-2',
            'text-sm font-medium text-muted-foreground hover:text-foreground',
            'transition-colors'
          )}
        >
          <span>Voice &amp; Traits Settings</span>
          {showSettings ? (
            <ChevronUp className="h-4 w-4" />
          ) : (
            <ChevronDown className="h-4 w-4" />
          )}
        </button>

        {showSettings && (
          <div className="px-4 pb-4 space-y-4">
            {/* Voice Style */}
            <div>
              <label className="block text-xs font-medium text-muted-foreground mb-2">
                Voice Style
              </label>
              <div className="flex flex-wrap gap-1.5">
                {VOICE_STYLES.map((style) => (
                  <button
                    key={style.value}
                    type="button"
                    onClick={() => handleVoiceChange(style.value)}
                    className={cn(
                      'px-2.5 py-1 text-xs font-medium rounded-md transition-colors',
                      voice === style.value
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-muted text-muted-foreground hover:bg-muted/80'
                    )}
                    title={style.description}
                  >
                    {style.label}
                  </button>
                ))}
              </div>
            </div>

            {/* Traits */}
            <div>
              <label className="block text-xs font-medium text-muted-foreground mb-2">
                Personality Traits
              </label>
              <div className="flex flex-wrap gap-1.5 mb-2">
                {traits.map((trait) => (
                  <span
                    key={trait}
                    className="flex items-center gap-1 px-2 py-0.5 text-xs bg-primary/20 text-primary rounded-full"
                  >
                    {trait}
                    <button
                      type="button"
                      onClick={() => handleRemoveTrait(trait)}
                      className="hover:text-destructive"
                    >
                      &times;
                    </button>
                  </span>
                ))}
                {traits.length === 0 && (
                  <span className="text-xs text-muted-foreground">No traits defined</span>
                )}
              </div>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={newTrait}
                  onChange={(e) => setNewTrait(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault()
                      handleAddTrait()
                    }
                  }}
                  placeholder="Add a trait..."
                  className={cn(
                    'flex-1 px-2 py-1 text-xs',
                    'bg-muted border border-border rounded',
                    'text-foreground placeholder:text-muted-foreground',
                    'focus:outline-none focus:ring-1 focus:ring-primary'
                  )}
                />
                <button
                  type="button"
                  onClick={handleAddTrait}
                  disabled={!newTrait.trim()}
                  className={cn(
                    'px-2 py-1 text-xs font-medium rounded',
                    'bg-primary text-primary-foreground hover:bg-primary/90',
                    'disabled:opacity-50 disabled:cursor-not-allowed'
                  )}
                >
                  Add
                </button>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Markdown Editor */}
      <div className="flex-1 min-h-0">
        <SkillContentEditor
          value={content}
          originalValue={originalContent}
          onChange={handleContentChange}
          isDirty={isDirty}
          dirtyCount={dirtyCount}
          onUndo={onUndo}
          onRedo={onRedo}
          canUndo={canUndo}
          canRedo={canRedo}
          onSave={onSave}
          onDiscard={onDiscard}
          isSaving={isSaving}
          isValid={isValid}
          className="h-full"
        />
      </div>
    </div>
  )
}
