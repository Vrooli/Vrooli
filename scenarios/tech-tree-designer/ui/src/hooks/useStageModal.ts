import { useCallback, useState } from 'react'
import type { FormEvent } from 'react'
import { defaultStageForm } from '../utils/constants'
import { createStage, updateStage, requestStageIdeas } from '../services/techTree'
import type { ProgressionStage, Sector } from '../types/techTree'
import type { StageFormState } from '../types/forms'

const cloneStageForm = (): StageFormState => ({ ...(defaultStageForm as StageFormState) })

export interface StageModalOverrides {
  sectorId?: string
  stageType?: string
  parentStageId?: string
}

interface UseStageModalParams {
  selectedTreeId: string | null
  selectedSector: Sector | null
  sectors: Sector[]
  onSuccess?: () => void
}

export interface StageModalResult {
  isOpen: boolean
  formState: StageFormState
  isSaving: boolean
  isGeneratingIdea: boolean
  errorMessage: string | null
  editMode: boolean
  stageId: string | null
  open: (overrides?: StageModalOverrides) => void
  openForEdit: (stage: ProgressionStage) => void
  close: () => void
  onFieldChange: (field: keyof StageFormState, value: string) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => Promise<void>
  onGenerateIdea: () => Promise<void>
}

/**
 * Hook for managing stage creation/editing modal state and operations.
 * Handles form state, validation, submission, edit mode, and AI generation.
 */
export const useStageModal = ({
  selectedTreeId,
  selectedSector,
  sectors,
  onSuccess
}: UseStageModalParams): StageModalResult => {
  const [isOpen, setIsOpen] = useState(false)
  const [formState, setFormState] = useState<StageFormState>(cloneStageForm())
  const [isSaving, setIsSaving] = useState(false)
  const [isGeneratingIdea, setIsGeneratingIdea] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [editingStageId, setEditingStageId] = useState<string | null>(null)

  const onFieldChange = useCallback(
    (field: keyof StageFormState, value: string) => {
      setFormState((previous) => ({ ...previous, [field]: value }))
    },
    []
  )

  const open = useCallback(
    (overrides: StageModalOverrides = {}) => {
      setEditingStageId(null)
      const fallbackSectorId = overrides.sectorId || selectedSector?.id || sectors[0]?.id || ''
      setFormState({
        ...cloneStageForm(),
        sectorId: fallbackSectorId,
        stageType: overrides.stageType || defaultStageForm.stageType,
        parentStageId: overrides.parentStageId
      })
      setErrorMessage(null)
      setIsOpen(true)
    },
    [sectors, selectedSector?.id]
  )

  const openForEdit = useCallback((stage: ProgressionStage) => {
    setEditingStageId(stage.id)
    const examplesArray = Array.isArray(stage.examples) ? stage.examples : []
    setFormState({
      sectorId: stage.sector_id,
      stageType: stage.stage_type || 'foundation',
      stageOrder: stage.stage_order?.toString() || '',
      name: stage.name,
      description: stage.description || '',
      progress: stage.progress_percentage?.toString() || '0',
      positionX: stage.position_x?.toString() || '',
      positionY: stage.position_y?.toString() || '',
      examples: examplesArray.join(', ')
    })
    setErrorMessage(null)
    setIsOpen(true)
  }, [])

  const close = useCallback(() => {
    setIsOpen(false)
    setErrorMessage(null)
    setEditingStageId(null)
  }, [])

  const onSubmit = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      const targetSectorId = formState.sectorId || selectedSector?.id
      if (!targetSectorId) {
        setErrorMessage('Select a sector before creating a stage.')
        return
      }

      setIsSaving(true)
      setErrorMessage(null)

      try {
        const payload = {
          sector_id: targetSectorId,
          stage_type: formState.stageType,
          stage_order: formState.stageOrder ? Number(formState.stageOrder) : undefined,
          name: formState.name.trim(),
          description: formState.description.trim(),
          progress_percentage: formState.progress ? Number(formState.progress) : undefined,
          position_x: formState.positionX ? Number(formState.positionX) : undefined,
          position_y: formState.positionY ? Number(formState.positionY) : undefined,
          examples: formState.examples
            ? formState.examples
                .split(',')
                .map((item) => item.trim())
                .filter(Boolean)
            : []
        }

        if (editingStageId) {
          await updateStage(editingStageId, payload, selectedTreeId || undefined)
        } else {
          await createStage(payload, selectedTreeId || undefined)
        }

        setIsOpen(false)
        setFormState(cloneStageForm())
        setEditingStageId(null)
        onSuccess?.()
      } catch (error) {
        setErrorMessage(error instanceof Error ? error.message : 'Failed to save stage')
      } finally {
        setIsSaving(false)
      }
    },
    [formState, selectedSector?.id, selectedTreeId, editingStageId, onSuccess]
  )

  const onGenerateIdea = useCallback(async () => {
    const targetSectorId = formState.sectorId || selectedSector?.id
    if (!targetSectorId) {
      setErrorMessage('Select a sector before asking for AI suggestions.')
      return
    }

    setIsGeneratingIdea(true)
    setErrorMessage(null)

    try {
      const response = await requestStageIdeas(
        {
          sector_id: targetSectorId,
          prompt: formState.description,
          count: 1
        },
        selectedTreeId || undefined
      )
      const nextIdea = response.ideas?.[0]
      if (nextIdea) {
        setFormState((previous) => ({
          ...previous,
          name: nextIdea.name || previous.name,
          description: nextIdea.description || previous.description,
          stageType: nextIdea.stage_type || previous.stageType
        }))
      }
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : 'Failed to generate idea')
    } finally {
      setIsGeneratingIdea(false)
    }
  }, [selectedSector?.id, selectedTreeId, formState.description, formState.sectorId])

  return {
    isOpen,
    formState,
    isSaving,
    isGeneratingIdea,
    errorMessage,
    editMode: !!editingStageId,
    stageId: editingStageId,
    open,
    openForEdit,
    close,
    onFieldChange,
    onSubmit,
    onGenerateIdea
  }
}
