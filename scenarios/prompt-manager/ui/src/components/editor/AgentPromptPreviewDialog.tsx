/**
 * AgentPromptPreviewDialog - Modal for viewing the fully constructed prompt.
 */

import { useCallback, useEffect, useRef, useState } from 'react'
import { Copy, RefreshCw, X, Eye } from 'lucide-react'
import { cn } from '@/lib/utils'
import { toast } from '@/hooks/use-toast'
import * as agentService from '@/services/agentService'

interface AgentPromptPreviewDialogProps {
  isOpen: boolean
  onClose: () => void
  agentId: string
  agentName: string
  teamId?: string
  hasUnsavedChanges?: boolean
}

export function AgentPromptPreviewDialog({
  isOpen,
  onClose,
  agentId,
  agentName,
  teamId,
  hasUnsavedChanges = false,
}: AgentPromptPreviewDialogProps) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const [prompt, setPrompt] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadPreview = useCallback(async () => {
    if (!agentId) return
    setIsLoading(true)
    setError(null)
    try {
      const response = await agentService.previewAgentPrompt(agentId, teamId)
      setPrompt(response.prompt)
    } catch (err) {
      console.error('[AgentPromptPreviewDialog] Failed to fetch prompt preview:', err)
      setPrompt('')
      setError('Unable to build prompt preview. Check the API and try again.')
    } finally {
      setIsLoading(false)
    }
  }, [agentId, teamId])

  const handleCopy = useCallback(async () => {
    if (!prompt) return
    try {
      await navigator.clipboard.writeText(prompt)
      toast({
        title: 'Prompt copied',
        description: 'The full prompt is now in your clipboard.',
      })
    } catch (err) {
      console.error('[AgentPromptPreviewDialog] Failed to copy prompt:', err)
      toast({
        title: 'Copy failed',
        description: 'Unable to copy the prompt. Try again.',
      })
    }
  }, [prompt])

  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    },
    [onClose]
  )

  const handleClickOutside = useCallback(
    (event: MouseEvent) => {
      if (dialogRef.current && !dialogRef.current.contains(event.target as Node)) {
        onClose()
      }
    },
    [onClose]
  )

  useEffect(() => {
    if (!isOpen) return
    void loadPreview()
  }, [isOpen, loadPreview])

  useEffect(() => {
    if (!isOpen) return
    document.addEventListener('keydown', handleKeyDown)
    document.addEventListener('mousedown', handleClickOutside)
    document.body.style.overflow = 'hidden'

    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.removeEventListener('mousedown', handleClickOutside)
      document.body.style.overflow = ''
    }
  }, [handleClickOutside, handleKeyDown, isOpen])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" />

      <div
        ref={dialogRef}
        className={cn(
          'relative w-full max-w-5xl mx-4 p-6',
          'bg-slate-900 border border-white/10 rounded-xl shadow-2xl',
          'animate-in fade-in-0 zoom-in-95 duration-150',
          'max-h-[85vh] overflow-hidden flex flex-col'
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby="prompt-preview-title"
      >
        <button
          type="button"
          onClick={onClose}
          className={cn(
            'absolute top-4 right-4 p-1 rounded',
            'text-slate-400 hover:text-white hover:bg-white/10 transition-colors'
          )}
          aria-label="Close dialog"
        >
          <X className="h-5 w-5" />
        </button>

        <div className="flex items-start justify-between gap-4">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-500/15 text-indigo-300">
                <Eye className="h-4 w-4" />
              </div>
              <h2 id="prompt-preview-title" className="text-lg font-semibold text-white">
                Prompt Preview
              </h2>
            </div>
            <p className="text-xs text-slate-400">
              Agent: <span className="text-slate-200">{agentName}</span>
              {teamId ? (
                <>
                  {' '}
                  • Team: <span className="text-slate-200">{teamId}</span>
                </>
              ) : null}
            </p>
          </div>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => void loadPreview()}
              disabled={isLoading}
              className={cn(
                'inline-flex items-center gap-1.5 rounded-md border border-white/10 px-2.5 py-1.5 text-xs font-medium',
                'text-slate-200 hover:bg-white/10 transition-colors',
                isLoading && 'opacity-50 cursor-not-allowed'
              )}
            >
              <RefreshCw className="h-3.5 w-3.5" />
              Refresh
            </button>
            <button
              type="button"
              onClick={() => void handleCopy()}
              disabled={!prompt}
              className={cn(
                'inline-flex items-center gap-1.5 rounded-md border border-white/10 px-2.5 py-1.5 text-xs font-medium',
                'text-slate-200 hover:bg-white/10 transition-colors',
                !prompt && 'opacity-50 cursor-not-allowed'
              )}
            >
              <Copy className="h-3.5 w-3.5" />
              Copy
            </button>
          </div>
        </div>

        <div className="mt-4 space-y-3">
          {hasUnsavedChanges && (
            <div className="rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-200/90">
              This preview uses the last saved agent.json + markdown files. Save changes to update the
              prompt.
            </div>
          )}
          {!hasUnsavedChanges && (
            <div className="text-[11px] text-slate-400">
              Preview uses saved agent.json + markdown files.
            </div>
          )}
        </div>

        <div className="mt-3 flex-1 overflow-hidden">
          {isLoading ? (
            <div className="flex h-full items-center justify-center text-sm text-slate-400">
              Building prompt preview...
            </div>
          ) : error ? (
            <div className="flex h-full items-center justify-center text-sm text-amber-200">
              {error}
            </div>
          ) : (
            <div className="h-full overflow-y-auto rounded-lg border border-white/10 bg-slate-950 p-4">
              <pre className="whitespace-pre-wrap text-xs leading-relaxed text-slate-200">
                {prompt || 'No prompt content available.'}
              </pre>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
