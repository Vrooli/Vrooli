import React, { useState, useCallback } from 'react'
import { Bot, Rocket, Lightbulb } from 'lucide-react'
import type { StageIdea, Sector } from '../../types/techTree'

interface AIStageIdeasModalProps {
  isOpen: boolean
  sector: Sector | null
  onClose: () => void
  onGenerate: (sectorId: string, prompt: string, count: number) => Promise<{ ideas: StageIdea[] }>
  onCreateStages: (ideas: StageIdea[], sectorId: string) => Promise<void>
}

/**
 * Modal for generating AI-powered stage suggestions and batch-creating them.
 *
 * Features:
 * - Input prompt to guide AI generation
 * - Number of suggestions slider
 * - Preview generated ideas with details
 * - Select/deselect individual ideas
 * - Batch create selected stages
 */
const AIStageIdeasModal: React.FC<AIStageIdeasModalProps> = ({
  isOpen,
  sector,
  onClose,
  onGenerate,
  onCreateStages
}) => {
  const [prompt, setPrompt] = useState('')
  const [count, setCount] = useState(3)
  const [ideas, setIdeas] = useState<StageIdea[]>([])
  const [selectedIdeas, setSelectedIdeas] = useState<Set<number>>(new Set())
  const [isGenerating, setIsGenerating] = useState(false)
  const [isCreating, setIsCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleGenerate = useCallback(async () => {
    if (!sector) return

    setIsGenerating(true)
    setError(null)

    try {
      const response = await onGenerate(sector.id, prompt, count)
      setIdeas(response.ideas || [])
      // Auto-select all generated ideas
      setSelectedIdeas(new Set(response.ideas.map((_, index) => index)))
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to generate ideas'
      setError(errorMessage)
    } finally {
      setIsGenerating(false)
    }
  }, [sector, prompt, count, onGenerate])

  const handleToggleIdea = useCallback((index: number) => {
    setSelectedIdeas(prev => {
      const next = new Set(prev)
      if (next.has(index)) {
        next.delete(index)
      } else {
        next.add(index)
      }
      return next
    })
  }, [])

  const handleCreateSelected = useCallback(async () => {
    if (!sector || selectedIdeas.size === 0) return

    setIsCreating(true)
    setError(null)

    try {
      const selected = ideas.filter((_, index) => selectedIdeas.has(index))
      await onCreateStages(selected, sector.id)

      // Close modal on success
      onClose()

      // Reset state
      setIdeas([])
      setSelectedIdeas(new Set())
      setPrompt('')
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to create stages'
      setError(errorMessage)
    } finally {
      setIsCreating(false)
    }
  }, [sector, ideas, selectedIdeas, onCreateStages, onClose])

  const handleClose = useCallback(() => {
    setIdeas([])
    setSelectedIdeas(new Set())
    setPrompt('')
    setError(null)
    onClose()
  }, [onClose])

  if (!isOpen || !sector) return null

  const hasIdeas = ideas.length > 0
  const selectedCount = selectedIdeas.size

  return (
    <div className="modal-overlay" onClick={handleClose}>
      <div
        className="modal-container"
        onClick={(e) => e.stopPropagation()}
        style={{ maxWidth: '800px', maxHeight: '90vh', overflow: 'auto' }}
      >
        <div className="modal-header">
          <h2>AI Stage Ideas for {sector.name}</h2>
          <button className="modal-close-btn" onClick={handleClose} aria-label="Close">
            ×
          </button>
        </div>

        <div className="modal-body" style={{ padding: '24px' }}>
          {/* Generation Controls */}
          <div style={{ marginBottom: '24px', padding: '16px', background: 'rgba(59,130,246,0.1)', borderRadius: '8px' }}>
            <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>
              Prompt (Optional)
            </label>
            <input
              type="text"
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="E.g., 'Focus on real-time monitoring' or 'integration with IoT'"
              style={{
                width: '100%',
                padding: '10px',
                borderRadius: '6px',
                border: '1px solid rgba(148,163,184,0.3)',
                background: 'rgba(15,23,42,0.6)',
                color: '#f8fafc',
                marginBottom: '16px'
              }}
            />

            <label style={{ display: 'block', marginBottom: '8px', fontWeight: 600 }}>
              Number of Ideas: {count}
            </label>
            <input
              type="range"
              min="1"
              max="5"
              value={count}
              onChange={(e) => setCount(parseInt(e.target.value, 10))}
              style={{ width: '100%', marginBottom: '16px' }}
            />

            <button
              onClick={handleGenerate}
              disabled={isGenerating}
              style={{
                padding: '10px 20px',
                borderRadius: '6px',
                border: 'none',
                background: isGenerating ? 'rgba(148,163,184,0.3)' : 'rgba(59,130,246,0.8)',
                color: '#fff',
                fontWeight: 600,
                cursor: isGenerating ? 'wait' : 'pointer',
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                gap: '8px'
              }}
            >
              {isGenerating ? (
                <>
                  <Bot size={16} />
                  Generating Ideas...
                </>
              ) : (
                <>
                  <Rocket size={16} />
                  Generate AI Suggestions
                </>
              )}
            </button>
          </div>

          {/* Error Display */}
          {error && (
            <div style={{
              padding: '12px',
              marginBottom: '16px',
              background: 'rgba(239,68,68,0.1)',
              border: '1px solid rgba(239,68,68,0.3)',
              borderRadius: '6px',
              color: '#fca5a5'
            }}>
              {error}
            </div>
          )}

          {/* Ideas List */}
          {hasIdeas && (
            <div style={{ marginBottom: '16px' }}>
              <div style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginBottom: '12px'
              }}>
                <h3 style={{ margin: 0 }}>Generated Ideas ({ideas.length})</h3>
                <span style={{ fontSize: '13px', color: '#94a3b8' }}>
                  {selectedCount} selected
                </span>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                {ideas.map((idea, index) => (
                  <div
                    key={index}
                    onClick={() => handleToggleIdea(index)}
                    style={{
                      padding: '16px',
                      border: `2px solid ${selectedIdeas.has(index) ? 'rgba(59,130,246,0.6)' : 'rgba(148,163,184,0.2)'}`,
                      borderRadius: '8px',
                      background: selectedIdeas.has(index) ? 'rgba(59,130,246,0.05)' : 'rgba(15,23,42,0.4)',
                      cursor: 'pointer',
                      transition: 'all 0.2s'
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'flex-start', gap: '12px' }}>
                      <input
                        type="checkbox"
                        checked={selectedIdeas.has(index)}
                        onChange={() => {}}
                        style={{ marginTop: '4px' }}
                      />
                      <div style={{ flex: 1 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                          <h4 style={{ margin: 0, fontSize: '15px' }}>{idea.name}</h4>
                          <span style={{
                            padding: '2px 8px',
                            borderRadius: '4px',
                            fontSize: '11px',
                            background: 'rgba(94,234,212,0.2)',
                            color: '#5eead4',
                            textTransform: 'capitalize'
                          }}>
                            {idea.stage_type.replace('_', ' ')}
                          </span>
                        </div>
                        <p style={{
                          margin: '0 0 8px 0',
                          fontSize: '13px',
                          color: '#cbd5e1',
                          lineHeight: '1.5'
                        }}>
                          {idea.description}
                        </p>
                        {idea.suggested_scenarios && idea.suggested_scenarios.length > 0 && (
                          <div style={{ fontSize: '12px', color: '#94a3b8' }}>
                            <strong>Suggested Scenarios:</strong> {idea.suggested_scenarios.join(', ')}
                          </div>
                        )}
                        {idea.strategic_rationale && (
                          <div style={{
                            fontSize: '11px',
                            color: '#64748b',
                            marginTop: '6px',
                            fontStyle: 'italic',
                            display: 'flex',
                            alignItems: 'flex-start',
                            gap: '4px'
                          }}>
                            <Lightbulb size={12} style={{ marginTop: '2px', flexShrink: 0 }} />
                            <span>{idea.strategic_rationale}</span>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Footer Actions */}
        {hasIdeas && (
          <div className="modal-footer" style={{
            padding: '16px 24px',
            borderTop: '1px solid rgba(148,163,184,0.2)',
            display: 'flex',
            gap: '12px',
            justifyContent: 'flex-end'
          }}>
            <button
              onClick={handleClose}
              style={{
                padding: '10px 20px',
                borderRadius: '6px',
                border: '1px solid rgba(148,163,184,0.3)',
                background: 'transparent',
                color: '#94a3b8',
                cursor: 'pointer'
              }}
            >
              Cancel
            </button>
            <button
              onClick={handleCreateSelected}
              disabled={selectedCount === 0 || isCreating}
              style={{
                padding: '10px 20px',
                borderRadius: '6px',
                border: 'none',
                background: selectedCount === 0 || isCreating
                  ? 'rgba(148,163,184,0.3)'
                  : 'rgba(34,197,94,0.8)',
                color: '#fff',
                fontWeight: 600,
                cursor: selectedCount === 0 || isCreating ? 'not-allowed' : 'pointer'
              }}
            >
              {isCreating ? 'Creating...' : `Create ${selectedCount} Stage${selectedCount !== 1 ? 's' : ''}`}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

export default AIStageIdeasModal
