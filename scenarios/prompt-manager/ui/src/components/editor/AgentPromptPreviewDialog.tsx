/**
 * AgentPromptPreviewDialog - Modal for viewing the fully constructed prompt.
 */

import { useCallback, useEffect, useRef, useState } from 'react'
import { Copy, RefreshCw, X, Eye, FileText, Code } from 'lucide-react'
import { cn } from '@/lib/utils'
import { toast } from '@/hooks/use-toast'
import { MarkdownRenderer } from '@/components/markdown/MarkdownRenderer'
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
  const [viewMode, setViewMode] = useState<'markdown' | 'raw'>('markdown')

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
          'bg-card border border-border rounded-xl shadow-2xl',
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
            'text-muted-foreground hover:text-foreground hover:bg-muted transition-colors'
          )}
          aria-label="Close dialog"
        >
          <X className="h-5 w-5" />
        </button>

        <div className="flex items-start justify-between gap-4">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/15 text-primary">
                <Eye className="h-4 w-4" />
              </div>
              <h2 id="prompt-preview-title" className="text-lg font-semibold text-foreground">
                Prompt Preview
              </h2>
            </div>
            <p className="text-xs text-muted-foreground">
              Agent: <span className="text-foreground">{agentName}</span>
              {teamId ? (
                <>
                  {' '}
                  • Team: <span className="text-foreground">{teamId}</span>
                </>
              ) : null}
            </p>
          </div>

          <div className="flex items-center gap-2">
            <div className="flex items-center rounded-md border border-border">
              <button
                type="button"
                onClick={() => setViewMode('markdown')}
                className={cn(
                  'inline-flex items-center gap-1 px-2 py-1.5 text-xs font-medium rounded-l-md transition-colors',
                  viewMode === 'markdown'
                    ? 'bg-primary/15 text-primary'
                    : 'text-muted-foreground hover:text-foreground hover:bg-muted',
                )}
                title="Rendered markdown"
              >
                <FileText className="h-3.5 w-3.5" />
              </button>
              <button
                type="button"
                onClick={() => setViewMode('raw')}
                className={cn(
                  'inline-flex items-center gap-1 px-2 py-1.5 text-xs font-medium rounded-r-md transition-colors',
                  viewMode === 'raw'
                    ? 'bg-primary/15 text-primary'
                    : 'text-muted-foreground hover:text-foreground hover:bg-muted',
                )}
                title="Raw text"
              >
                <Code className="h-3.5 w-3.5" />
              </button>
            </div>
            <button
              type="button"
              onClick={() => void loadPreview()}
              disabled={isLoading}
              className={cn(
                'inline-flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1.5 text-xs font-medium',
                'text-foreground hover:bg-muted transition-colors',
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
                'inline-flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1.5 text-xs font-medium',
                'text-foreground hover:bg-muted transition-colors',
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
            <div className="rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-700 dark:text-amber-200/90">
              This preview uses the last saved agent.json + markdown files. Save changes to update the
              prompt.
            </div>
          )}
          {!hasUnsavedChanges && (
            <div className="text-[11px] text-muted-foreground">
              Preview uses saved agent.json + markdown files.
            </div>
          )}
        </div>

        <div className="mt-3 flex-1 min-h-0 overflow-hidden">
          {isLoading ? (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
              Building prompt preview...
            </div>
          ) : error ? (
            <div className="flex h-full items-center justify-center text-sm text-amber-700 dark:text-amber-200">
              {error}
            </div>
          ) : (
            <div className="h-full overflow-y-auto rounded-lg border border-border bg-muted/30 p-4">
              {viewMode === 'markdown' ? (
                <MarkdownRenderer content={prompt || 'No prompt content available.'} />
              ) : (
                <pre className="whitespace-pre-wrap text-xs leading-relaxed text-foreground">
                  {prompt || 'No prompt content available.'}
                </pre>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
