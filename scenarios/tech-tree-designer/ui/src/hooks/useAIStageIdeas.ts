import { useState, useCallback } from 'react'
import { requestStageIdeas, createStage } from '../services/techTree'
import type { Sector, StageIdea } from '../types/techTree'

interface UseAIStageIdeasParams {
  selectedTreeId: string | null
  onSuccess?: (message: string) => void
  onError?: (error: string) => void
}

interface UseAIStageIdeasResult {
  isOpen: boolean
  targetSector: Sector | null
  open: (sector: Sector) => void
  close: () => void
  handleGenerate: (sectorId: string, prompt: string, count: number) => Promise<{ ideas: StageIdea[] }>
  handleCreateStages: (ideas: StageIdea[], sectorId: string) => Promise<void>
}

/**
 * Hook for managing AI Stage Ideas modal state and operations.
 *
 * Handles:
 * - Modal open/close state
 * - AI idea generation via OpenRouter
 * - Batch creation of stages from selected ideas
 */
export const useAIStageIdeas = ({
  selectedTreeId,
  onSuccess,
  onError
}: UseAIStageIdeasParams): UseAIStageIdeasResult => {
  const [isOpen, setIsOpen] = useState(false)
  const [targetSector, setTargetSector] = useState<Sector | null>(null)

  const open = useCallback((sector: Sector) => {
    setTargetSector(sector)
    setIsOpen(true)
  }, [])

  const close = useCallback(() => {
    setIsOpen(false)
    // Delay clearing sector to avoid flash
    setTimeout(() => setTargetSector(null), 200)
  }, [])

  const handleGenerate = useCallback(
    async (sectorId: string, prompt: string, count: number) => {
      try {
        const response = await requestStageIdeas(
          { sector_id: sectorId, prompt, count },
          selectedTreeId || undefined
        )
        return response
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to generate ideas'
        if (onError) {
          onError(message)
        }
        throw error
      }
    },
    [selectedTreeId, onError]
  )

  const handleCreateStages = useCallback(
    async (ideas: StageIdea[], sectorId: string) => {
      if (ideas.length === 0) {
        return
      }

      try {
        // Create stages sequentially to avoid race conditions
        for (const idea of ideas) {
          await createStage(
            {
              sector_id: sectorId,
              name: idea.name,
              description: idea.description,
              stage_type: idea.stage_type,
              stage_order: 1, // Let backend auto-assign if needed
              examples: idea.suggested_scenarios || []
            },
            selectedTreeId || undefined
          )
        }

        if (onSuccess) {
          onSuccess(`Successfully created ${ideas.length} stage${ideas.length > 1 ? 's' : ''}`)
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to create stages'
        if (onError) {
          onError(message)
        }
        throw error
      }
    },
    [selectedTreeId, onSuccess, onError]
  )

  return {
    isOpen,
    targetSector,
    open,
    close,
    handleGenerate,
    handleCreateStages
  }
}
