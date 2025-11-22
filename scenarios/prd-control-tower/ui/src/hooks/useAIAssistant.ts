import { useCallback, useRef, useState } from 'react'
import { buildApiUrl } from '../utils/apiClient'

interface UseAIAssistantOptions {
  draftId: string
  editorContent: string
  onContentChange: (value: string) => void
}

export type AIAction = 'improve' | 'expand' | 'simplify' | 'grammar' | 'technical' | 'clarify' | ''

interface ReferencePRDData {
  name: string
  content: string
}

interface UseAIAssistantResult {
  textareaRef: React.RefObject<HTMLTextAreaElement>
  aiDialogOpen: boolean
  aiSection: string
  aiContext: string
  aiAction: AIAction
  aiModel: string
  aiIncludeExistingContent: boolean
  aiReferencePRDs: ReferencePRDData[]
  aiResult: string | null
  aiGenerating: boolean
  aiError: string | null
  selectionStart: number
  selectionEnd: number
  hasSelection: boolean
  openAIDialog: () => void
  closeAIDialog: () => void
  setAISection: (section: string) => void
  setAIContext: (context: string) => void
  setAIAction: (action: AIAction) => void
  setAIModel: (model: string) => void
  setAIIncludeExistingContent: (value: boolean) => void
  setAIReferencePRDs: (prds: ReferencePRDData[]) => void
  handleAIGenerate: () => Promise<void>
  handleQuickAction: (action: AIAction) => Promise<void>
  applyAIResult: (mode: 'insert' | 'replace') => void
}

/**
 * Hook for managing AI assistant state and operations.
 * Encapsulates AI generation logic, dialog state, and text insertion.
 */
export function useAIAssistant({ draftId, editorContent, onContentChange }: UseAIAssistantOptions): UseAIAssistantResult {
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)
  const [aiDialogOpen, setAIDialogOpen] = useState(false)
  const [aiSection, setAISection] = useState('🎯 Full PRD')
  const [aiContext, setAIContext] = useState('')
  const [aiAction, setAIAction] = useState<AIAction>('')
  const [aiModel, setAIModel] = useState<string>('default')
  const [aiIncludeExistingContent, setAIIncludeExistingContent] = useState<boolean>(true)
  const [aiReferencePRDs, setAIReferencePRDs] = useState<ReferencePRDData[]>([])
  const [aiResult, setAIResult] = useState<string | null>(null)
  const [aiGenerating, setAIGenerating] = useState(false)
  const [aiError, setAIError] = useState<string | null>(null)
  const [selectionStart, setSelectionStart] = useState(0)
  const [selectionEnd, setSelectionEnd] = useState(0)
  const hasSelection = selectionEnd > selectionStart

  const openAIDialog = useCallback(() => {
    // Extract selected text from textarea if any
    if (textareaRef.current) {
      const start = textareaRef.current.selectionStart
      const end = textareaRef.current.selectionEnd
      setSelectionStart(start)
      setSelectionEnd(end)

      const selection = textareaRef.current.value.slice(start, end)
      if (selection.trim()) {
        setAIContext(selection.trim())
      }
    }
    setAIAction('')
    setAIResult(null)
    setAIError(null)
    setAIDialogOpen(true)
  }, [])

  const closeAIDialog = useCallback(() => {
    setAIDialogOpen(false)
  }, [])

  const handleAIGenerate = useCallback(async () => {
    setAIGenerating(true)
    setAIError(null)
    try {
      const payload: Record<string, unknown> = {
        section: aiSection,
        context: aiContext,
        action: aiAction,
        include_existing_content: aiIncludeExistingContent,
      }

      // Include reference PRDs if any
      if (aiReferencePRDs.length > 0) {
        payload.reference_prds = aiReferencePRDs
      }

      // Only include model if it's not 'default'
      if (aiModel && aiModel !== 'default') {
        payload.model = aiModel
      }

      const response = await fetch(buildApiUrl(`/drafts/${draftId}/ai/generate-section`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!response.ok) {
        const message = await response.text()
        throw new Error(message || 'AI request failed')
      }
      const data = (await response.json()) as { generated_text?: string; success?: boolean; message?: string }
      if (!data.generated_text) {
        throw new Error(data.message || 'AI response missing content')
      }
      setAIResult(data.generated_text.trim())
    } catch (err) {
      setAIError(err instanceof Error ? err.message : 'Unexpected AI error')
    } finally {
      setAIGenerating(false)
    }
  }, [draftId, aiSection, aiContext, aiAction, aiModel, aiIncludeExistingContent, aiReferencePRDs])

  const handleQuickAction = useCallback(async (action: AIAction) => {
    if (!aiContext.trim()) {
      setAIError('Please select some text first')
      return
    }
    setAIAction(action)
    setAIGenerating(true)
    setAIError(null)
    try {
      const payload: Record<string, unknown> = {
        section: '',
        context: aiContext,
        action,
        include_existing_content: false, // Quick actions don't need existing content
      }

      // Only include model if it's not 'default'
      if (aiModel && aiModel !== 'default') {
        payload.model = aiModel
      }

      const response = await fetch(buildApiUrl(`/drafts/${draftId}/ai/generate-section`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!response.ok) {
        const message = await response.text()
        throw new Error(message || 'AI request failed')
      }
      const data = (await response.json()) as { generated_text?: string; success?: boolean; message?: string }
      if (!data.generated_text) {
        throw new Error(data.message || 'AI response missing content')
      }
      setAIResult(data.generated_text.trim())
    } catch (err) {
      setAIError(err instanceof Error ? err.message : 'Unexpected AI error')
    } finally {
      setAIGenerating(false)
    }
  }, [draftId, aiContext, aiModel])

  const applyAIResult = useCallback(
    (mode: 'insert' | 'replace') => {
      if (!aiResult || !textareaRef.current) {
        return
      }
      const textarea = textareaRef.current
      const start = textarea.selectionStart
      const end = textarea.selectionEnd
      const snippet = aiResult
      const before = editorContent.slice(0, start)
      const after = mode === 'replace' ? editorContent.slice(end) : editorContent.slice(start)
      const nextContent = `${before}${snippet}${after}`
      const cursorPosition = before.length + snippet.length
      onContentChange(nextContent)
      // Focus textarea and set cursor position after content update
      requestAnimationFrame(() => {
        if (textareaRef.current) {
          textareaRef.current.setSelectionRange(cursorPosition, cursorPosition)
          textareaRef.current.focus()
        }
      })
      setAIDialogOpen(false)
    },
    [aiResult, editorContent, onContentChange],
  )

  return {
    textareaRef,
    aiDialogOpen,
    aiSection,
    aiContext,
    aiAction,
    aiModel,
    aiIncludeExistingContent,
    aiReferencePRDs,
    aiResult,
    aiGenerating,
    aiError,
    selectionStart,
    selectionEnd,
    hasSelection,
    openAIDialog,
    closeAIDialog,
    setAISection,
    setAIContext,
    setAIAction,
    setAIModel,
    setAIIncludeExistingContent,
    setAIReferencePRDs,
    handleAIGenerate,
    handleQuickAction,
    applyAIResult,
  }
}
