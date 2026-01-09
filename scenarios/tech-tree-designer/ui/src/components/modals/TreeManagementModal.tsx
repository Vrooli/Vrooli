import React, { useCallback, useEffect, useState } from 'react'
import type { TechTreeSummary } from '../../types/techTree'

export interface TreeFormData {
  name: string
  slug: string
  description: string
  tree_type: string
  status: string
  version: string
  is_active: boolean
}

interface TreeManagementModalProps {
  isOpen: boolean
  mode: 'create' | 'clone' | 'edit'
  sourceTree?: TechTreeSummary | null
  formState: TreeFormData
  onFieldChange: (field: keyof TreeFormData, value: string | boolean) => void
  onSubmit: () => void
  onClose: () => void
  isSaving: boolean
  errorMessage?: string
}

const treeTypeOptions = [
  { value: 'official', label: 'Official' },
  { value: 'draft', label: 'Draft' },
  { value: 'experimental', label: 'Experimental' }
]

const statusOptions = [
  { value: 'active', label: 'Active' },
  { value: 'proposed', label: 'Proposed' },
  { value: 'archived', label: 'Archived' }
]

/**
 * Modal for creating, cloning, or editing tech trees.
 * Supports all tree management operations with validation.
 */
const TreeManagementModal: React.FC<TreeManagementModalProps> = ({
  isOpen,
  mode,
  sourceTree,
  formState,
  onFieldChange,
  onSubmit,
  onClose,
  isSaving,
  errorMessage
}) => {
  const [localSlug, setLocalSlug] = useState(formState.slug)

  // Auto-generate slug from name
  useEffect(() => {
    if (mode === 'create' && formState.name && !localSlug) {
      const generatedSlug = formState.name
        .toLowerCase()
        .replace(/[^a-z0-9-]+/g, '-')
        .replace(/^-+|-+$/g, '')
      onFieldChange('slug', generatedSlug)
      setLocalSlug(generatedSlug)
    }
  }, [formState.name, mode, localSlug, onFieldChange])

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault()
      onSubmit()
    },
    [onSubmit]
  )

  const modalTitle = mode === 'create' ? 'Create New Tech Tree' : mode === 'clone' ? `Clone: ${sourceTree?.tree.name}` : 'Edit Tech Tree'

  const submitLabel = isSaving ? 'Saving...' : mode === 'create' ? 'Create Tree' : mode === 'clone' ? 'Clone Tree' : 'Save Changes'

  if (!isOpen) return null

  return (
    <div
      className="modal-overlay"
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        background: 'rgba(2, 6, 23, 0.85)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
        backdropFilter: 'blur(4px)'
      }}
      onClick={onClose}
    >
      <div
        className="modal-content"
        style={{
          background: 'linear-gradient(135deg, #0f172a 0%, #1e293b 100%)',
          border: '1px solid rgba(59, 130, 246, 0.3)',
          borderRadius: '16px',
          padding: '32px',
          maxWidth: '600px',
          width: '90%',
          maxHeight: '90vh',
          overflowY: 'auto',
          boxShadow: '0 24px 48px rgba(2, 6, 23, 0.6)'
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <h2
          style={{
            color: '#f8fafc',
            fontSize: '24px',
            fontWeight: 700,
            marginBottom: '24px',
            borderBottom: '2px solid rgba(59, 130, 246, 0.3)',
            paddingBottom: '12px'
          }}
        >
          {modalTitle}
        </h2>

        {mode === 'clone' && sourceTree && (
          <div
            style={{
              background: 'rgba(59, 130, 246, 0.1)',
              border: '1px solid rgba(59, 130, 246, 0.3)',
              borderRadius: '8px',
              padding: '12px',
              marginBottom: '20px',
              color: '#94a3b8'
            }}
          >
            <strong>Cloning from:</strong> {sourceTree.tree.name} ({sourceTree.tree.tree_type})
          </div>
        )}

        {errorMessage && (
          <div
            style={{
              background: 'rgba(239, 68, 68, 0.1)',
              border: '1px solid rgba(239, 68, 68, 0.3)',
              borderRadius: '8px',
              padding: '12px',
              marginBottom: '20px',
              color: '#fca5a5'
            }}
          >
            {errorMessage}
          </div>
        )}

        <form onSubmit={handleSubmit}>
          {/* Name */}
          <div style={{ marginBottom: '20px' }}>
            <label
              htmlFor="tree-name"
              style={{
                display: 'block',
                color: '#e2e8f0',
                fontSize: '14px',
                fontWeight: 600,
                marginBottom: '8px'
              }}
            >
              Tree Name *
            </label>
            <input
              id="tree-name"
              type="text"
              value={formState.name}
              onChange={(e) => onFieldChange('name', e.target.value)}
              required
              placeholder="e.g., Interstellar Civilization Roadmap"
              style={{
                width: '100%',
                padding: '10px 14px',
                background: 'rgba(15, 23, 42, 0.6)',
                border: '1px solid rgba(148, 163, 184, 0.3)',
                borderRadius: '8px',
                color: '#f8fafc',
                fontSize: '14px'
              }}
            />
          </div>

          {/* Slug */}
          <div style={{ marginBottom: '20px' }}>
            <label
              htmlFor="tree-slug"
              style={{
                display: 'block',
                color: '#e2e8f0',
                fontSize: '14px',
                fontWeight: 600,
                marginBottom: '8px'
              }}
            >
              Slug * <span style={{ color: '#94a3b8', fontWeight: 400 }}>(URL-friendly identifier)</span>
            </label>
            <input
              id="tree-slug"
              type="text"
              value={formState.slug}
              onChange={(e) => {
                setLocalSlug(e.target.value)
                onFieldChange('slug', e.target.value)
              }}
              required
              placeholder="e.g., interstellar-civilization"
              pattern="[a-z0-9-]+"
              title="Only lowercase letters, numbers, and hyphens allowed"
              style={{
                width: '100%',
                padding: '10px 14px',
                background: 'rgba(15, 23, 42, 0.6)',
                border: '1px solid rgba(148, 163, 184, 0.3)',
                borderRadius: '8px',
                color: '#f8fafc',
                fontSize: '14px'
              }}
            />
          </div>

          {/* Description */}
          <div style={{ marginBottom: '20px' }}>
            <label
              htmlFor="tree-description"
              style={{
                display: 'block',
                color: '#e2e8f0',
                fontSize: '14px',
                fontWeight: 600,
                marginBottom: '8px'
              }}
            >
              Description *
            </label>
            <textarea
              id="tree-description"
              value={formState.description}
              onChange={(e) => onFieldChange('description', e.target.value)}
              required
              placeholder="Describe the purpose and scope of this tech tree..."
              rows={3}
              style={{
                width: '100%',
                padding: '10px 14px',
                background: 'rgba(15, 23, 42, 0.6)',
                border: '1px solid rgba(148, 163, 184, 0.3)',
                borderRadius: '8px',
                color: '#f8fafc',
                fontSize: '14px',
                resize: 'vertical'
              }}
            />
          </div>

          {/* Tree Type */}
          <div style={{ marginBottom: '20px' }}>
            <label
              htmlFor="tree-type"
              style={{
                display: 'block',
                color: '#e2e8f0',
                fontSize: '14px',
                fontWeight: 600,
                marginBottom: '8px'
              }}
            >
              Tree Type *
            </label>
            <select
              id="tree-type"
              value={formState.tree_type}
              onChange={(e) => onFieldChange('tree_type', e.target.value)}
              required
              style={{
                width: '100%',
                padding: '10px 14px',
                background: 'rgba(15, 23, 42, 0.6)',
                border: '1px solid rgba(148, 163, 184, 0.3)',
                borderRadius: '8px',
                color: '#f8fafc',
                fontSize: '14px'
              }}
            >
              {treeTypeOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>

          {/* Status */}
          <div style={{ marginBottom: '20px' }}>
            <label
              htmlFor="tree-status"
              style={{
                display: 'block',
                color: '#e2e8f0',
                fontSize: '14px',
                fontWeight: 600,
                marginBottom: '8px'
              }}
            >
              Status *
            </label>
            <select
              id="tree-status"
              value={formState.status}
              onChange={(e) => onFieldChange('status', e.target.value)}
              required
              style={{
                width: '100%',
                padding: '10px 14px',
                background: 'rgba(15, 23, 42, 0.6)',
                border: '1px solid rgba(148, 163, 184, 0.3)',
                borderRadius: '8px',
                color: '#f8fafc',
                fontSize: '14px'
              }}
            >
              {statusOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>

          {/* Version (Create mode only) */}
          {mode === 'create' && (
            <div style={{ marginBottom: '20px' }}>
              <label
                htmlFor="tree-version"
                style={{
                  display: 'block',
                  color: '#e2e8f0',
                  fontSize: '14px',
                  fontWeight: 600,
                  marginBottom: '8px'
                }}
              >
                Version
              </label>
              <input
                id="tree-version"
                type="text"
                value={formState.version}
                onChange={(e) => onFieldChange('version', e.target.value)}
                placeholder="e.g., 1.0.0"
                style={{
                  width: '100%',
                  padding: '10px 14px',
                  background: 'rgba(15, 23, 42, 0.6)',
                  border: '1px solid rgba(148, 163, 184, 0.3)',
                  borderRadius: '8px',
                  color: '#f8fafc',
                  fontSize: '14px'
                }}
              />
            </div>
          )}

          {/* Active checkbox */}
          <div style={{ marginBottom: '24px' }}>
            <label
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '10px',
                color: '#e2e8f0',
                fontSize: '14px',
                cursor: 'pointer'
              }}
            >
              <input
                type="checkbox"
                checked={formState.is_active}
                onChange={(e) => onFieldChange('is_active', e.target.checked)}
                style={{
                  width: '18px',
                  height: '18px',
                  cursor: 'pointer'
                }}
              />
              Set as active tree (switches UI to this tree)
            </label>
          </div>

          {/* Actions */}
          <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
            <button
              type="button"
              onClick={onClose}
              disabled={isSaving}
              style={{
                padding: '10px 24px',
                background: 'transparent',
                border: '1px solid rgba(148, 163, 184, 0.3)',
                borderRadius: '8px',
                color: '#e2e8f0',
                fontSize: '14px',
                fontWeight: 600,
                cursor: isSaving ? 'not-allowed' : 'pointer',
                opacity: isSaving ? 0.5 : 1
              }}
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSaving}
              style={{
                padding: '10px 24px',
                background: 'linear-gradient(135deg, #3B82F6, #2563EB)',
                border: 'none',
                borderRadius: '8px',
                color: '#ffffff',
                fontSize: '14px',
                fontWeight: 600,
                cursor: isSaving ? 'not-allowed' : 'pointer',
                opacity: isSaving ? 0.5 : 1,
                boxShadow: '0 4px 12px rgba(59, 130, 246, 0.3)'
              }}
            >
              {submitLabel}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default TreeManagementModal
