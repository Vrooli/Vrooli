/**
 * AgentPromptPreviewDialog - Modal for viewing the fully constructed prompt.
 */

import { useCallback, useEffect, useState } from 'react'
import { Copy, RefreshCw, Eye, FileText, Code } from 'lucide-react'
import { cn } from '@/lib/utils'
import { toast } from '@/hooks/use-toast'
import { Dialog } from '@/components/shared/Dialog'
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

  useEffect(() => {
    if (!isOpen) return
    void loadPreview()
  }, [isOpen, loadPreview])

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      maxWidth="max-w-5xl"
      titleId="prompt-preview-title"
      className="bg-card border-border max-h-[85vh] overflow-hidden flex flex-col"
    >
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
    </Dialog>
  )
}
