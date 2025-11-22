import { useCallback, useRef, useState } from 'react'
import { buildApiUrl } from '../utils/apiClient'
import type { Draft } from '../types'

interface UseAIAssistantOptions {
  draftId: string
  draft: Draft
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
 * Build the AI prompt that will be sent to the backend.
 * This mirrors the backend buildPrompt function for preview purposes.
 */
export function buildPrompt(
  draft: Draft,
  section: string,
  context: string,
  action: AIAction,
  includeExisting: boolean,
  referencePRDs: ReferencePRDData[]
): string {
  // If action is specified, use action-based prompt
  if (action) {
    return buildActionPrompt(action, context)
  }

  // Otherwise, use section-based prompt
  let prompt = ''

  prompt += 'You are an expert technical writer helping to create a Product Requirements Document (PRD).\n\n'
  prompt += `Entity Type: ${draft.entity_type}\n`
  prompt += `Entity Name: ${draft.entity_name}\n\n`

  // Include reference PRDs if provided
  if (referencePRDs.length > 0) {
    prompt += 'Reference PRD Examples (for style and structure guidance):\n'
    prompt += '='.repeat(71) + '\n\n'
    referencePRDs.forEach((ref, i) => {
      prompt += `Reference PRD ${i + 1}: ${ref.name}\n`
      prompt += '-'.repeat(70) + '\n'
      prompt += ref.content
      prompt += '\n'
      prompt += '-'.repeat(70) + '\n\n'
    })
    prompt += '='.repeat(71) + '\n\n'
  }

  // Include existing PRD content if requested
  if (includeExisting && draft.content) {
    prompt += 'Current PRD Content:\n'
    prompt += '='.repeat(71) + '\n'
    prompt += draft.content
    prompt += '\n'
    prompt += '='.repeat(71) + '\n\n'
  }

  // Determine task description based on section
  const isFullPRD = section === 'Full PRD' || section === '🎯 Full PRD'
  if (isFullPRD) {
    prompt += 'Task: Generate a complete, comprehensive PRD for this entity.\n\n'
  } else {
    prompt += `Task: Generate the "${section}" section for this PRD.\n\n`
  }

  // Include additional context if provided
  if (context) {
    prompt += 'Additional Context:\n'
    prompt += '='.repeat(71) + '\n'
    prompt += context
    prompt += '\n'
    prompt += '='.repeat(71) + '\n\n'
  }

  // Add requirements
  prompt += 'Requirements:\n'
  prompt += '- Follow Vrooli PRD standards and structure\n'
  prompt += '- Be specific and actionable\n'
  prompt += '- Include measurable criteria where applicable\n'
  prompt += '- Use markdown formatting\n'
  prompt += '- Focus on business value and technical clarity\n'
  if (referencePRDs.length > 0) {
    prompt += '- Use the reference PRD examples above as inspiration for style, structure, and level of detail\n'
  }
  prompt += '\n'

  if (isFullPRD) {
    prompt += 'Generate a complete PRD including all standard sections (Overview, Operational Targets, Tech Direction, Dependencies, UX & Branding, etc.). Do not include markdown headers for the document title.'
  } else {
    prompt += `Generate only the content for the "${section}" section. Do not include the section header itself.`
  }

  return prompt
}

function buildActionPrompt(action: AIAction, selectedText: string): string {
  const actionPrompts: Record<string, string> = {
    improve: `Improve the following text to be more professional, clear, and actionable:

${selectedText}

Requirements:
- Maintain the original meaning and key points
- Use stronger, more precise language
- Add clarity and remove ambiguity
- Keep markdown formatting
- Return only the improved text without explanations`,
    expand: `Expand the following text with more detail, examples, and context:

${selectedText}

Requirements:
- Add relevant details and examples
- Explain technical concepts more thoroughly
- Include practical implications
- Maintain markdown formatting
- Return only the expanded text without explanations`,
    simplify: `Simplify the following text to be more concise and easier to understand:

${selectedText}

Requirements:
- Remove unnecessary jargon and complexity
- Use simpler language where possible
- Keep only essential information
- Maintain markdown formatting
- Return only the simplified text without explanations`,
    grammar: `Fix grammar, spelling, and formatting issues in the following text:

${selectedText}

Requirements:
- Correct all grammatical errors
- Fix spelling mistakes
- Improve sentence structure
- Ensure consistent markdown formatting
- Return only the corrected text without explanations`,
    technical: `Make the following text more technical and precise:

${selectedText}

Requirements:
- Add technical accuracy and specificity
- Include relevant technical terms
- Remove vague language
- Add measurable criteria where applicable
- Maintain markdown formatting
- Return only the enhanced text without explanations`,
    clarify: `Clarify and restructure the following text to be more understandable:

${selectedText}

Requirements:
- Break down complex ideas into clear points
- Add structure with headings or lists if helpful
- Explain ambiguous terms
- Improve logical flow
- Maintain markdown formatting
- Return only the clarified text without explanations`,
  }

  return actionPrompts[action] || actionPrompts.improve
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
