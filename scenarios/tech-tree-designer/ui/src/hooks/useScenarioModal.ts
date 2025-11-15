import { useCallback, useState } from 'react'
import type { FormEvent } from 'react'
import { defaultScenarioForm } from '../utils/constants'
import { linkScenario } from '../services/techTree'
import type { ProgressionStage } from '../types/techTree'
import type { ScenarioFormState } from '../types/forms'

const cloneScenarioForm = (): ScenarioFormState => ({ ...(defaultScenarioForm as ScenarioFormState) })

interface UseScenarioModalParams {
  selectedTreeId: string | null
  onSuccess?: () => void
}

export interface ScenarioModalResult {
  isOpen: boolean
  formState: ScenarioFormState
  targetStage: ProgressionStage | null
  isSaving: boolean
  errorMessage: string | null
  openForStage: (stage: ProgressionStage) => void
  close: () => void
  onFieldChange: (field: keyof ScenarioFormState, value: string) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => Promise<void>
}

/**
 * Hook for managing scenario linking modal state and operations.
 * Handles linking scenarios to progression stages.
 */
export const useScenarioModal = ({
  selectedTreeId,
  onSuccess
}: UseScenarioModalParams): ScenarioModalResult => {
  const [isOpen, setIsOpen] = useState(false)
  const [formState, setFormState] = useState<ScenarioFormState>(cloneScenarioForm())
  const [isSaving, setIsSaving] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [targetStage, setTargetStage] = useState<ProgressionStage | null>(null)

  const onFieldChange = useCallback(
    (field: keyof ScenarioFormState, value: string) => {
      setFormState((previous) => ({ ...previous, [field]: value }))
    },
    []
  )

  const openForStage = useCallback((stage: ProgressionStage) => {
    if (!stage?.id) {
      return
    }
    setTargetStage(stage)
    setFormState({
      ...cloneScenarioForm(),
      stageId: stage.id
    })
    setErrorMessage(null)
    setIsOpen(true)
  }, [])

  const close = useCallback(() => {
    setIsOpen(false)
    setErrorMessage(null)
    setTargetStage(null)
  }, [])

  const onSubmit = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      if (!formState.stageId) {
        setErrorMessage('Stage identifier missing; reopen the form and try again.')
        return
      }

      setIsSaving(true)
      setErrorMessage(null)

      try {
        await linkScenario(
          {
            stage_id: formState.stageId,
            scenario_name: formState.scenarioName.trim(),
            completion_status: formState.status,
            contribution_weight: Number(formState.contributionWeight) || 1,
            priority: Number(formState.priority) || 3,
            estimated_impact: Number(formState.estimatedImpact) || 5,
            notes: formState.notes.trim()
          },
          selectedTreeId || undefined
        )
        setIsOpen(false)
        setFormState(cloneScenarioForm())
        onSuccess?.()
      } catch (error) {
        setErrorMessage(error instanceof Error ? error.message : 'Failed to link scenario')
      } finally {
        setIsSaving(false)
      }
    },
    [formState, selectedTreeId, onSuccess]
  )

  return {
    isOpen,
    formState,
    targetStage,
    isSaving,
    errorMessage,
    openForStage,
    close,
    onFieldChange,
    onSubmit
  }
}
