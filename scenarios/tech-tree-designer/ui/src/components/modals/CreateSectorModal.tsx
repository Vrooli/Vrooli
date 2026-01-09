import React from 'react'
import type { SectorFormState } from '../../types/forms'

interface CreateSectorModalProps {
  isOpen: boolean
  formState: SectorFormState
  categoryOptions: string[]
  onFieldChange: (field: keyof SectorFormState, value: string) => void
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void
  onClose: () => void
  isSaving: boolean
  errorMessage: string | null
  editMode?: boolean
  sectorId?: string | null
}

const CreateSectorModal: React.FC<CreateSectorModalProps> = ({
  isOpen,
  formState,
  categoryOptions,
  onFieldChange,
  onSubmit,
  onClose,
  isSaving,
  errorMessage,
  editMode = false,
  sectorId = null
}) => {
  if (!isOpen) {
    return null
  }

  const modalTitle = editMode ? 'Edit Sector' : 'Create Sector'
  const submitLabel = editMode ? 'Update Sector' : 'Create Sector'

  return (
    <div className="modal-overlay" role="dialog" aria-modal="true" aria-label={modalTitle}>
      <div className="modal">
        <h3>{modalTitle}</h3>
        {errorMessage ? <p className="form-error">{errorMessage}</p> : null}
        <form onSubmit={onSubmit} className="modal-form">
          <div className="modal-field">
            <label htmlFor="sector-name">Name</label>
            <input
              id="sector-name"
              type="text"
              value={formState.name}
              onChange={(event) => onFieldChange('name', event.target.value)}
              required
            />
          </div>
          <div className="modal-field">
            <label htmlFor="sector-category">Category</label>
            <select
              id="sector-category"
              value={formState.category}
              onChange={(event) => onFieldChange('category', event.target.value)}
            >
              {categoryOptions.map((option) => (
                <option key={option} value={option}>
                  {option}
                </option>
              ))}
            </select>
          </div>
          <div className="modal-field">
            <label htmlFor="sector-description">Description</label>
            <textarea
              id="sector-description"
              rows={3}
              value={formState.description}
              onChange={(event) => onFieldChange('description', event.target.value)}
            />
          </div>
          <div className="modal-field">
            <label htmlFor="sector-color">Color</label>
            <input
              id="sector-color"
              type="text"
              value={formState.color}
              onChange={(event) => onFieldChange('color', event.target.value)}
            />
          </div>
          <div className="modal-grid">
            <div className="modal-field">
              <label htmlFor="sector-position-x">Position X</label>
              <input
                id="sector-position-x"
                type="number"
                value={formState.positionX}
                onChange={(event) => onFieldChange('positionX', event.target.value)}
              />
            </div>
            <div className="modal-field">
              <label htmlFor="sector-position-y">Position Y</label>
              <input
                id="sector-position-y"
                type="number"
                value={formState.positionY}
                onChange={(event) => onFieldChange('positionY', event.target.value)}
              />
            </div>
          </div>
          <div className="modal-actions">
            <button type="button" className="button button--ghost" onClick={onClose}>
              Cancel
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

export default CreateSectorModal
