import { useCallback, useState } from 'react'
import type { FormEvent } from 'react'
import { createMilestone, updateMilestone } from '../services/techTree'
import type { StrategicMilestone } from '../types/techTree'
import type { MilestoneFormState } from '../types/forms'

const defaultMilestoneForm: MilestoneFormState = {
  name: '',
  description: '',
  milestoneType: 'sector_complete',
  completion: '0',
  businessValue: '',
  confidence: '0.6',
  estimatedDate: '',
  targetSectors: [],
  targetStages: []
}

const cloneDefaultForm = (): MilestoneFormState => ({ ...defaultMilestoneForm })

interface UseMilestoneModalParams {
  selectedTreeId: string | null
  onSuccess?: () => void
}

export interface MilestoneModalResult {
  isOpen: boolean
  formState: MilestoneFormState
  isSaving: boolean
  errorMessage: string | null
  editMode: boolean
  milestoneId: string | null
  open: () => void
  openForEdit: (milestone: StrategicMilestone) => void
  close: () => void
  onFieldChange: (field: keyof MilestoneFormState, value: string | string[]) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => Promise<void>
}

const parseNumber = (value: string, precision?: number): number | undefined => {
  const trimmed = value.trim()
  if (!trimmed) {
    return undefined
  }
  const parsed = Number(trimmed)
  if (Number.isNaN(parsed)) {
    return undefined
  }
  if (typeof precision === 'number') {
    const factor = Math.pow(10, precision)
    return Math.round(parsed * factor) / factor
  }
  return parsed
}

export const useMilestoneModal = ({ selectedTreeId, onSuccess }: UseMilestoneModalParams) => {
  const [isOpen, setIsOpen] = useState(false)
  const [formState, setFormState] = useState<MilestoneFormState>(cloneDefaultForm())
  const [isSaving, setIsSaving] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [editingMilestoneId, setEditingMilestoneId] = useState<string | null>(null)

  const onFieldChange = useCallback(
    (field: keyof MilestoneFormState, value: string | string[]) => {
      setFormState((previous) => ({
        ...previous,
        [field]: Array.isArray(value) ? value : value
      }))
    },
    []
  )

  const open = useCallback(() => {
    setEditingMilestoneId(null)
    setFormState(cloneDefaultForm())
    setErrorMessage(null)
    setIsOpen(true)
  }, [])

  const openForEdit = useCallback((milestone: StrategicMilestone) => {
    setEditingMilestoneId(milestone.id)
    setFormState({
      name: milestone.name,
      description: milestone.description || '',
      milestoneType: milestone.milestone_type || 'sector_complete',
      completion: String(milestone.completion_percentage ?? ''),
      businessValue: String(milestone.business_value_estimate ?? ''),
      confidence: String(milestone.confidence_level ?? ''),
      estimatedDate: milestone.estimated_completion_date || '',
      targetSectors: milestone.target_sector_ids || [],
      targetStages: milestone.target_stage_ids || []
    })
    setErrorMessage(null)
    setIsOpen(true)
  }, [])

  const close = useCallback(() => {
    setIsOpen(false)
    setErrorMessage(null)
    setEditingMilestoneId(null)
  }, [])

  const onSubmit = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      if (!selectedTreeId) {
        setErrorMessage('Select a tech tree before managing milestones.')
        return
      }

      setIsSaving(true)
      setErrorMessage(null)

      try {
        const payload = {
          name: formState.name.trim(),
          description: formState.description.trim(),
          milestone_type: formState.milestoneType,
          completion_percentage: parseNumber(formState.completion, 2),
          business_value_estimate: parseNumber(formState.businessValue),
          confidence_level: parseNumber(formState.confidence, 3),
          estimated_completion_date: formState.estimatedDate.trim() || undefined,
          target_sector_ids: formState.targetSectors,
          target_stage_ids: formState.targetStages
        }

        if (editingMilestoneId) {
          await updateMilestone(editingMilestoneId, payload, selectedTreeId)
        } else {
          await createMilestone(payload, selectedTreeId)
        }

        setIsOpen(false)
        setEditingMilestoneId(null)
        setFormState(cloneDefaultForm())
        onSuccess?.()
      } catch (error) {
        setErrorMessage(error instanceof Error ? error.message : 'Failed to save milestone')
      } finally {
        setIsSaving(false)
      }
    },
    [editingMilestoneId, formState, onSuccess, selectedTreeId]
  )

  return {
    isOpen,
    formState,
    isSaving,
    errorMessage,
    editMode: !!editingMilestoneId,
    milestoneId: editingMilestoneId,
    open,
    openForEdit,
    close,
    onFieldChange,
    onSubmit
  }
}
