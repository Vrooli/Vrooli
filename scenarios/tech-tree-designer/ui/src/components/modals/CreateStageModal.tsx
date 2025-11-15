import React from 'react'
import { formatStageTypeLabel, stageTypeOptions } from '../../utils/constants'
import type { Sector } from '../../types/techTree'
import type { StageFormState } from '../../types/forms'

interface CreateStageModalProps {
  isOpen: boolean
  sectors: Sector[]
  formState: StageFormState
  onFieldChange: (field: keyof StageFormState, value: string) => void
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void
  onClose: () => void
  onGenerateIdea: () => void
  isGeneratingIdea: boolean
  isSaving: boolean
  errorMessage: string | null
  editMode?: boolean
  stageId?: string | null
}

const CreateStageModal: React.FC<CreateStageModalProps> = ({
  isOpen,
  sectors,
  formState,
  onFieldChange,
  onSubmit,
  onClose,
  onGenerateIdea,
  isGeneratingIdea,
  isSaving,
  errorMessage,
  editMode = false,
  stageId = null
}) => {
  if (!isOpen) {
    return null
  }

  const modalTitle = editMode ? 'Edit Stage' : 'Create Stage'
  const submitLabel = editMode ? 'Update Stage' : 'Create Stage'

  return (
    <div className="modal-overlay" role="dialog" aria-modal="true" aria-label={modalTitle}>
      <div className="modal">
        <h3>{modalTitle}</h3>
        {errorMessage ? <p className="form-error">{errorMessage}</p> : null}
        <form onSubmit={onSubmit} className="modal-form">
          <div className="modal-field">
            <label htmlFor="stage-sector">Sector</label>
            <select
              id="stage-sector"
              value={formState.sectorId}
              onChange={(event) => onFieldChange('sectorId', event.target.value)}
              required
            >
              <option value="">Select a sector</option>
              {sectors.map((sector) => (
                <option key={sector.id} value={sector.id}>
                  {sector.name}
                </option>
              ))}
            </select>
          </div>
          <div className="modal-grid">
            <div className="modal-field">
              <label htmlFor="stage-type">Stage Type</label>
              <select
                id="stage-type"
                value={formState.stageType}
                onChange={(event) => onFieldChange('stageType', event.target.value)}
              >
                {stageTypeOptions.map((option) => (
                  <option key={option} value={option}>
                    {formatStageTypeLabel(option)}
                  </option>
                ))}
              </select>
            </div>
            <div className="modal-field">
              <label htmlFor="stage-order">Stage Order</label>
              <input
                id="stage-order"
                type="number"
                value={formState.stageOrder}
                onChange={(event) => onFieldChange('stageOrder', event.target.value)}
              />
            </div>
            <div className="modal-field">
              <label htmlFor="stage-progress">Progress %</label>
              <input
                id="stage-progress"
                type="number"
                value={formState.progress}
                onChange={(event) => onFieldChange('progress', event.target.value)}
              />
            </div>
          </div>
          <div className="modal-field">
            <label htmlFor="stage-name">Stage Name</label>
            <input
              id="stage-name"
              type="text"
              value={formState.name}
              onChange={(event) => onFieldChange('name', event.target.value)}
              required
            />
          </div>
          <div className="modal-field">
            <label htmlFor="stage-description">Description</label>
            <textarea
              id="stage-description"
              rows={3}
              value={formState.description}
              onChange={(event) => onFieldChange('description', event.target.value)}
            />
          </div>
          <div className="modal-grid">
            <div className="modal-field">
              <label htmlFor="stage-position-x">Position X</label>
              <input
                id="stage-position-x"
                type="number"
                value={formState.positionX}
                onChange={(event) => onFieldChange('positionX', event.target.value)}
              />
            </div>
            <div className="modal-field">
              <label htmlFor="stage-position-y">Position Y</label>
              <input
                id="stage-position-y"
                type="number"
                value={formState.positionY}
                onChange={(event) => onFieldChange('positionY', event.target.value)}
              />
            </div>
          </div>
          <div className="modal-field">
            <label htmlFor="stage-examples">Examples (comma separated)</label>
            <input
              id="stage-examples"
              type="text"
              value={formState.examples}
              onChange={(event) => onFieldChange('examples', event.target.value)}
            />
          </div>
          <div className="modal-actions modal-actions--spread">
            <button type="button" className="button button--ghost" onClick={onClose}>
              Cancel
            </button>
            <button type="button" className="button button--ghost" onClick={onGenerateIdea} disabled={isGeneratingIdea}>
              {isGeneratingIdea ? 'Asking AI…' : 'Generate Idea'}
            </button>
            <button type="submit" className="button" disabled={isSaving}>
              {isSaving ? 'Saving…' : submitLabel}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default CreateStageModal
