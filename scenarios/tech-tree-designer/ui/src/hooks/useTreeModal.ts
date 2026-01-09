import { useCallback, useState } from 'react'
import { createTechTree, cloneTechTree, updateTechTree } from '../services/techTree'
import type { TechTreeSummary } from '../types/techTree'
import type { TreeFormData } from '../components/modals/TreeManagementModal'

interface UseTreeModalParams {
  onSuccess?: () => void
  onError?: (error: string) => void
}

interface UseTreeModalResult {
  isOpen: boolean
  mode: 'create' | 'clone' | 'edit'
  sourceTree: TechTreeSummary | null
  formState: TreeFormData
  isSaving: boolean
  errorMessage: string
  openForCreate: () => void
  openForClone: (tree: TechTreeSummary) => void
  openForEdit: (tree: TechTreeSummary) => void
  close: () => void
  onFieldChange: (field: keyof TreeFormData, value: string | boolean) => void
  onSubmit: () => Promise<void>
}

const defaultFormState: TreeFormData = {
  name: '',
  slug: '',
  description: '',
  tree_type: 'draft',
  status: 'active',
  version: '1.0.0',
  is_active: false
}

/**
 * Custom hook for managing tech tree creation/cloning/editing modal.
 * Handles form state, validation, and API calls.
 */
export const useTreeModal = ({
  onSuccess,
  onError
}: UseTreeModalParams = {}): UseTreeModalResult => {
  const [isOpen, setIsOpen] = useState(false)
  const [mode, setMode] = useState<'create' | 'clone' | 'edit'>('create')
  const [sourceTree, setSourceTree] = useState<TechTreeSummary | null>(null)
  const [formState, setFormState] = useState<TreeFormData>(defaultFormState)
  const [isSaving, setIsSaving] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')

  const openForCreate = useCallback(() => {
    setMode('create')
    setSourceTree(null)
    setFormState(defaultFormState)
    setErrorMessage('')
    setIsOpen(true)
  }, [])

  const openForClone = useCallback((tree: TechTreeSummary) => {
    setMode('clone')
    setSourceTree(tree)
    setFormState({
      name: `${tree.tree.name} (Copy)`,
      slug: `${tree.tree.slug}-copy`,
      description: tree.tree.description || '',
      tree_type: 'draft',
      status: 'active',
      version: tree.tree.version,
      is_active: false
    })
    setErrorMessage('')
    setIsOpen(true)
  }, [])

  const openForEdit = useCallback((tree: TechTreeSummary) => {
    setMode('edit')
    setSourceTree(tree)
    setFormState({
      name: tree.tree.name,
      slug: tree.tree.slug,
      description: tree.tree.description || '',
      tree_type: tree.tree.tree_type,
      status: tree.tree.status,
      version: tree.tree.version,
      is_active: tree.tree.is_active
    })
    setErrorMessage('')
    setIsOpen(true)
  }, [])

  const close = useCallback(() => {
    setIsOpen(false)
    setFormState(defaultFormState)
    setErrorMessage('')
  }, [])

  const onFieldChange = useCallback((field: keyof TreeFormData, value: string | boolean) => {
    setFormState((prev) => ({
      ...prev,
      [field]: value
    }))
    setErrorMessage('') // Clear error when user makes changes
  }, [])

  const onSubmit = useCallback(async () => {
    // Validation
    if (!formState.name.trim()) {
      setErrorMessage('Tree name is required')
      return
    }
    if (!formState.slug.trim()) {
      setErrorMessage('Slug is required')
      return
    }
    if (!formState.description.trim()) {
      setErrorMessage('Description is required')
      return
    }

    setIsSaving(true)
    setErrorMessage('')

    try {
      if (mode === 'create') {
        await createTechTree({
          name: formState.name,
          slug: formState.slug,
          description: formState.description,
          tree_type: formState.tree_type,
          status: formState.status,
          version: formState.version,
          is_active: formState.is_active
        })
      } else if (mode === 'clone' && sourceTree) {
        await cloneTechTree(sourceTree.tree.id, {
          name: formState.name,
          slug: formState.slug,
          description: formState.description,
          tree_type: formState.tree_type,
          status: formState.status,
          is_active: formState.is_active
        })
      } else if (mode === 'edit' && sourceTree) {
        await updateTechTree(sourceTree.tree.id, {
          name: formState.name,
          slug: formState.slug,
          description: formState.description,
          tree_type: formState.tree_type,
          status: formState.status,
          is_active: formState.is_active
        })
      }

      close()
      if (onSuccess) {
        onSuccess()
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Operation failed'
      setErrorMessage(message)
      if (onError) {
        onError(message)
      }
    } finally {
      setIsSaving(false)
    }
  }, [mode, formState, sourceTree, close, onSuccess, onError])

  return {
    isOpen,
    mode,
    sourceTree,
    formState,
    isSaving,
    errorMessage,
    openForCreate,
    openForClone,
    openForEdit,
    close,
    onFieldChange,
    onSubmit
  }
}
