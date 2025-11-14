import { useCallback, useRef, useState } from 'react'
import { buildApiUrl } from '../utils/apiClient'

interface UseAIAssistantOptions {
  draftId: string
  editorContent: string
  onContentChange: (value: string) => void
}

interface UseAIAssistantResult {
  textareaRef: React.RefObject<HTMLTextAreaElement>
  aiDialogOpen: boolean
  aiSection: string
  aiContext: string
  aiResult: string | null
  aiGenerating: boolean
  aiError: string | null
  openAIDialog: () => void
  closeAIDialog: () => void
  setAISection: (section: string) => void
  setAIContext: (context: string) => void
  handleAIGenerate: () => Promise<void>
  applyAIResult: (mode: 'insert' | 'replace') => void
}

/**
 * Hook for managing AI assistant state and operations.
 * Encapsulates AI generation logic, dialog state, and text insertion.
 */
export function useAIAssistant({ draftId, editorContent, onContentChange }: UseAIAssistantOptions): UseAIAssistantResult {
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)
  const [aiDialogOpen, setAIDialogOpen] = useState(false)
  const [aiSection, setAISection] = useState('Executive Summary')
  const [aiContext, setAIContext] = useState('')
  const [aiResult, setAIResult] = useState<string | null>(null)
  const [aiGenerating, setAIGenerating] = useState(false)
  const [aiError, setAIError] = useState<string | null>(null)

  const openAIDialog = useCallback(() => {
    // Extract selected text from textarea if any
    if (textareaRef.current) {
      const selection = textareaRef.current.value.slice(textareaRef.current.selectionStart, textareaRef.current.selectionEnd)
      if (selection.trim()) {
        setAIContext(selection.trim())
      }
    }
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
      const response = await fetch(buildApiUrl(`/drafts/${draftId}/ai/generate-section`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ section: aiSection, context: aiContext }),
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
  }, [draftId, aiSection, aiContext])

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
    aiResult,
    aiGenerating,
    aiError,
    openAIDialog,
    closeAIDialog,
    setAISection,
    setAIContext,
    handleAIGenerate,
    applyAIResult,
  }
}
