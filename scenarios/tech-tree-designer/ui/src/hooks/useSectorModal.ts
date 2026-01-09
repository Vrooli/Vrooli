import { useCallback, useState } from 'react'
import type { FormEvent } from 'react'
import { defaultSectorForm } from '../utils/constants'
import { createSector, updateSector } from '../services/techTree'
import type { Sector } from '../types/techTree'
import type { SectorFormState } from '../types/forms'

const cloneSectorForm = (): SectorFormState => ({ ...(defaultSectorForm as SectorFormState) })

interface UseSectorModalParams {
  selectedTreeId: string | null
  defaultCategory?: string
  onSuccess?: () => void
}

export interface SectorModalResult {
  isOpen: boolean
  formState: SectorFormState
  isSaving: boolean
  errorMessage: string | null
  editMode: boolean
  sectorId: string | null
  open: () => void
  openForEdit: (sector: Sector) => void
  close: () => void
  onFieldChange: (field: keyof SectorFormState, value: string) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => Promise<void>
}

/**
 * Hook for managing sector creation/editing modal state and operations.
 * Handles form state, validation, submission, and edit mode.
 */
export const useSectorModal = ({
  selectedTreeId,
  defaultCategory = 'software',
  onSuccess
}: UseSectorModalParams): SectorModalResult => {
  const [isOpen, setIsOpen] = useState(false)
  const [formState, setFormState] = useState<SectorFormState>(cloneSectorForm())
  const [isSaving, setIsSaving] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [editingSectorId, setEditingSectorId] = useState<string | null>(null)

  const onFieldChange = useCallback(
    (field: keyof SectorFormState, value: string) => {
      setFormState((previous) => ({ ...previous, [field]: value }))
    },
    []
  )

  const open = useCallback(() => {
    setEditingSectorId(null)
    setFormState((previous) => ({
      ...cloneSectorForm(),
      category: defaultCategory || previous.category || 'software'
    }))
    setErrorMessage(null)
    setIsOpen(true)
  }, [defaultCategory])

  const openForEdit = useCallback((sector: Sector) => {
    setEditingSectorId(sector.id)
    setFormState({
      name: sector.name,
      category: sector.category,
      description: sector.description || '',
      color: sector.color || '#0ea5e9',
      positionX: sector.position_x?.toString() || '',
      positionY: sector.position_y?.toString() || ''
    })
    setErrorMessage(null)
    setIsOpen(true)
  }, [])

  const close = useCallback(() => {
    setIsOpen(false)
    setErrorMessage(null)
    setEditingSectorId(null)
  }, [])

  const onSubmit = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      if (!selectedTreeId) {
        setErrorMessage('Select a tech tree before creating sectors.')
        return
      }

      setIsSaving(true)
      setErrorMessage(null)

      try {
        const payload = {
          name: formState.name.trim(),
          category: formState.category.trim(),
          description: formState.description.trim(),
          color: formState.color.trim(),
          position_x: formState.positionX ? Number(formState.positionX) : undefined,
          position_y: formState.positionY ? Number(formState.positionY) : undefined
        }

        if (editingSectorId) {
          await updateSector(editingSectorId, payload, selectedTreeId)
        } else {
          await createSector(payload, selectedTreeId)
        }

        setIsOpen(false)
        setFormState(cloneSectorForm())
        setEditingSectorId(null)
        onSuccess?.()
      } catch (error) {
        setErrorMessage(error instanceof Error ? error.message : 'Failed to save sector')
      } finally {
        setIsSaving(false)
      }
    },
    [formState, selectedTreeId, editingSectorId, onSuccess]
  )

  return {
    isOpen,
    formState,
    isSaving,
    errorMessage,
    editMode: !!editingSectorId,
    sectorId: editingSectorId,
    open,
    openForEdit,
    close,
    onFieldChange,
    onSubmit
  }
}
