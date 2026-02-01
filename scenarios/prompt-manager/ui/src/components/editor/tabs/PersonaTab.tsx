/**
 * PersonaTab - Agent persona configuration tab.
 *
 * Features:
 * - SOUL.md content editor (reuses SkillContentEditor pattern)
 * - Voice style selector
 * - Traits editor
 * - System prompt prefix
 */

import { useState, useCallback, useEffect } from 'react'
import { cn } from '@/lib/utils'
import type { Agent, UpdateAgentRequest, AgentPersona, AgentVoice } from '@/types/agent'

interface PersonaTabProps {
  agent: Agent
  onUpdate: (updates: UpdateAgentRequest) => Promise<void>
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
 * Persona configuration tab component.
 */
export function PersonaTab({ agent, onUpdate }: PersonaTabProps) {
  // Local state for editing
  const [voice, setVoice] = useState<AgentVoice>(agent.persona?.voice ?? 'professional')
  const [traits, setTraits] = useState<string[]>(agent.persona?.traits ?? [])
  const [newTrait, setNewTrait] = useState('')
  const [systemPromptPrefix, setSystemPromptPrefix] = useState(
    agent.persona?.systemPromptPrefix ?? ''
  )
  const [isDirty, setIsDirty] = useState(false)

  // Initialize from agent
  useEffect(() => {
    setVoice(agent.persona?.voice ?? 'professional')
    setTraits(agent.persona?.traits ?? [])
    setSystemPromptPrefix(agent.persona?.systemPromptPrefix ?? '')
    setIsDirty(false)
  }, [agent.persona])

  // Handle voice change
  const handleVoiceChange = useCallback(async (newVoice: AgentVoice) => {
    setVoice(newVoice)
    const updatedPersona: AgentPersona = {
      ...agent.persona,
      traits: agent.persona?.traits ?? [],
      voice: newVoice,
    }
    await onUpdate({ persona: updatedPersona })
  }, [agent.persona, onUpdate])

  // Handle trait addition
  const handleAddTrait = useCallback(async () => {
    if (newTrait.trim() && !traits.includes(newTrait.trim())) {
      const updatedTraits = [...traits, newTrait.trim()]
      setTraits(updatedTraits)
      setNewTrait('')
      const updatedPersona: AgentPersona = {
        ...agent.persona,
        traits: updatedTraits,
      }
      await onUpdate({ persona: updatedPersona })
    }
  }, [newTrait, traits, agent.persona, onUpdate])

  // Handle trait removal
  const handleRemoveTrait = useCallback(async (trait: string) => {
    const updatedTraits = traits.filter((t) => t !== trait)
    setTraits(updatedTraits)
    const updatedPersona: AgentPersona = {
      ...agent.persona,
      traits: updatedTraits,
    }
    await onUpdate({ persona: updatedPersona })
  }, [traits, agent.persona, onUpdate])

  // Handle system prompt prefix change (debounced save)
  const handleSystemPromptChange = useCallback((value: string) => {
    setSystemPromptPrefix(value)
    setIsDirty(true)
  }, [])

  // Save system prompt prefix
  const handleSaveSystemPrompt = useCallback(async () => {
    const updatedPersona: AgentPersona = {
      ...agent.persona,
      traits: agent.persona?.traits ?? [],
      systemPromptPrefix,
    }
    await onUpdate({ persona: updatedPersona })
    setIsDirty(false)
  }, [systemPromptPrefix, agent.persona, onUpdate])

  return (
    <div className="space-y-6">
      {/* Voice Style */}
      <div>
        <label className="block text-sm font-medium text-foreground mb-2">
          Voice Style
        </label>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
          {VOICE_STYLES.map((style) => (
            <button
              key={style.value}
              type="button"
              onClick={() => void handleVoiceChange(style.value)}
              className={cn(
                'flex flex-col items-start p-3 rounded-lg border transition-colors text-left',
                voice === style.value
                  ? 'border-primary bg-primary/10'
                  : 'border-border hover:border-muted-foreground hover:bg-muted'
              )}
            >
              <span className="text-sm font-medium">{style.label}</span>
              <span className="text-xs text-muted-foreground">{style.description}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Traits */}
      <div>
        <label className="block text-sm font-medium text-foreground mb-2">
          Personality Traits
        </label>
        <div className="flex flex-wrap gap-2 mb-2">
          {traits.map((trait) => (
            <span
              key={trait}
              className="flex items-center gap-1 px-2 py-1 text-sm bg-primary/20 text-primary rounded-full"
            >
              {trait}
              <button
                type="button"
                onClick={() => void handleRemoveTrait(trait)}
                className="hover:text-destructive ml-1"
              >
                &times;
              </button>
            </span>
          ))}
          {traits.length === 0 && (
            <span className="text-sm text-muted-foreground">No traits defined</span>
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
                void handleAddTrait()
              }
            }}
            placeholder="Add a trait..."
            className={cn(
              'flex-1 px-3 py-2 text-sm',
              'bg-muted border border-border rounded-lg',
              'text-foreground placeholder:text-muted-foreground',
              'focus:outline-none focus:ring-2 focus:ring-primary'
            )}
          />
          <button
            type="button"
            onClick={() => void handleAddTrait()}
            disabled={!newTrait.trim()}
            className={cn(
              'px-3 py-2 text-sm font-medium rounded-lg',
              'bg-primary text-primary-foreground hover:bg-primary/90',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            Add
          </button>
        </div>
      </div>

      {/* System Prompt Prefix */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="text-sm font-medium text-foreground">
            System Prompt Prefix
          </label>
          {isDirty && (
            <button
              type="button"
              onClick={() => void handleSaveSystemPrompt()}
              className="text-xs text-primary hover:underline"
            >
              Save changes
            </button>
          )}
        </div>
        <textarea
          value={systemPromptPrefix}
          onChange={(e) => handleSystemPromptChange(e.target.value)}
          placeholder="Optional prefix added to the agent's system prompt..."
          rows={4}
          className={cn(
            'w-full px-3 py-2 text-sm',
            'bg-muted border border-border rounded-lg',
            'text-foreground placeholder:text-muted-foreground',
            'focus:outline-none focus:ring-2 focus:ring-primary',
            'resize-none'
          )}
        />
        <p className="mt-1 text-xs text-muted-foreground">
          For complex personas, create a SOUL.md file in the agent&apos;s directory.
        </p>
      </div>

      {/* Persona File Reference */}
      <div className="p-3 bg-muted/50 rounded-lg">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium">Persona Entry File</span>
          <span className="text-sm font-mono text-muted-foreground">
            {agent.persona?.entry ?? 'SOUL.md'}
          </span>
        </div>
        <p className="mt-1 text-xs text-muted-foreground">
          External persona files are loaded from the agent&apos;s storage directory.
        </p>
      </div>
    </div>
  )
}
