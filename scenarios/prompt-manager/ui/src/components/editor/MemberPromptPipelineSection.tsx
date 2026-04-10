import { useState, useEffect, useMemo, useCallback } from 'react'
import { copyToClipboard } from '@/lib/clipboard'
import { RefreshCw, Copy, ExternalLink, Pencil, Link } from 'lucide-react'
import { cn } from '@/lib/utils'
import { toast } from '@/hooks/use-toast'
import * as agentService from '@/services/agentService'

interface MemberPromptPipelineSectionProps {
  teamId: string
  memberId: string
  onNavigateToTab?: (tab: 'responsibilities' | 'heartbeat') => void
  onNavigateToAgentFiles?: (filePath?: string) => void
}

type PipelineSectionKey =
  | 'agent-files'
  | 'responsibilities'
  | 'relationships'
  | 'inbox'
  | 'heartbeat-task'

interface PipelineSectionDefinition {
  key: PipelineSectionKey
  title: string
  headers: string[]
  description: string
  emptyMessage: string
}

interface PipelineSection extends PipelineSectionDefinition {
  content: string
  missing: boolean
  note?: string
}

interface AgentFileBlock {
  path: string
  content: string
}

const PIPELINE_SECTIONS: PipelineSectionDefinition[] = [
  {
    key: 'agent-files',
    title: 'Agent Files',
    headers: ['Agent Files (Markdown)'],
    description: 'SOUL.md and other agent markdown files (personality + operating notes).',
    emptyMessage: 'No agent markdown files were included.',
  },
  {
    key: 'responsibilities',
    title: 'Responsibilities',
    headers: ['Team Responsibilities (RESPONSIBILITIES.md)'],
    description: 'Role-specific instructions for this team member.',
    emptyMessage: 'No responsibilities are set for this member yet.',
  },
  {
    key: 'relationships',
    title: 'Relationships',
    headers: ['Team Relationships'],
    description: 'Reporting lines plus coordination commands.',
    emptyMessage: 'No relationship context is available yet.',
  },
  {
    key: 'inbox',
    title: 'Inbox',
    headers: ['Team Inbox'],
    description: 'Pending messages from other team members.',
    emptyMessage: 'No pending inbox messages.',
  },
  {
    key: 'heartbeat-task',
    title: 'Heartbeat Task',
    headers: ['Heartbeat Task (HEARTBEAT.md)', 'Heartbeat Task'],
    description: 'The exact task this member will execute on each heartbeat.',
    emptyMessage: 'No heartbeat task is defined yet.',
  },
]

function parsePromptSections(prompt: string): Map<string, string> {
  const sections = new Map<string, string>()
  if (!prompt) {
    return sections
  }
  const chunks = prompt.split(/\n\n---\n\n/)
  for (const chunk of chunks) {
    const trimmed = chunk.trim()
    if (!trimmed) continue
    const firstLine = trimmed.split('\n')[0]?.trim()
    if (!firstLine) continue
    const header = firstLine.replace(/^#+\s*/, '').trim()
    if (!header) continue
    sections.set(header, trimmed)
  }
  return sections
}

function stripHeader(section: string): string {
  const lines = section.split('\n')
  if (lines.length <= 1) return ''
  return lines.slice(1).join('\n').trim()
}

function buildPipelineSections(prompt: string): PipelineSection[] {
  const sections = parsePromptSections(prompt)
  return PIPELINE_SECTIONS.map((def) => {
    const matchedHeader = def.headers.find((entry) => sections.has(entry))
    const rawSection = matchedHeader ? sections.get(matchedHeader) ?? '' : ''
    const content = rawSection ? stripHeader(rawSection) : ''
    const missing = !rawSection || !content
    let note: string | undefined
    if (def.key === 'heartbeat-task' && matchedHeader === 'Heartbeat Task') {
      note = 'No heartbeat instructions defined. Default task inserted.'
    }
    return {
      ...def,
      content,
      missing,
      note,
    }
  })
}

function extractAgentFileBlocks(sectionContent: string): AgentFileBlock[] {
  if (!sectionContent) return []
  const matches = [...sectionContent.matchAll(/^##\s+(.+\.md)\s*$/gm)]
  if (matches.length === 0) return []

  const blocks: AgentFileBlock[] = []
  for (let i = 0; i < matches.length; i += 1) {
    const match = matches[i]
    if (!match) continue
    const fullMatch = match[0]
    const heading = match[1]
    if (!heading) continue
    const start = match.index + fullMatch.length
    const end = matches[i + 1]?.index ?? sectionContent.length
    const content = sectionContent.slice(start, end).trim()
    blocks.push({ path: heading.trim(), content })
  }
  return blocks
}

export function MemberPromptPipelineSection({ teamId, memberId, onNavigateToTab, onNavigateToAgentFiles }: MemberPromptPipelineSectionProps) {
  const [promptPreview, setPromptPreview] = useState('')
  const [promptError, setPromptError] = useState<string | null>(null)
  const [isPromptLoading, setIsPromptLoading] = useState(false)

  // DOC: docs/concepts/HEARTBEATS.md#prompt-pipeline-ui
  const loadPromptPreview = useCallback(async () => {
    setIsPromptLoading(true)
    setPromptError(null)
    try {
      const response = await agentService.previewAgentPrompt(memberId, teamId)
      setPromptPreview(response.prompt)
    } catch (err) {
      console.error('Failed to load prompt preview:', err)
      setPromptPreview('')
      setPromptError('Unable to build prompt preview. Check the API and try again.')
    } finally {
      setIsPromptLoading(false)
    }
  }, [memberId, teamId])

  const pipelineSections = useMemo(() => buildPipelineSections(promptPreview), [promptPreview])

  const handleCopyPrompt = useCallback(async () => {
    if (!promptPreview) return
    try {
      await copyToClipboard(promptPreview)
      toast({
        title: 'Prompt copied',
        description: 'The full prompt is now in your clipboard.',
      })
    } catch (err) {
      console.error('Failed to copy prompt:', err)
      toast({
        title: 'Copy failed',
        description: 'Unable to copy the prompt. Try again.',
      })
    }
  }, [promptPreview])

  // Auto-load prompt preview on mount
  useEffect(() => {
    if (promptPreview || isPromptLoading) return
    void loadPromptPreview()
  }, [promptPreview, isPromptLoading, loadPromptPreview])

  return (
    <div className="space-y-4">
      {/* Full prompt preview card */}
      {promptPreview && !isPromptLoading && !promptError && (
        <div className="rounded-lg border border-border bg-muted/20 px-3 py-2">
          <p className="text-[11px] font-medium text-foreground mb-2">Full prompt preview</p>
          <pre className="max-h-48 overflow-y-auto whitespace-pre-wrap text-[11px] text-muted-foreground">
            {promptPreview}
          </pre>
        </div>
      )}

      {/* Toolbar: description + refresh/copy */}
      <div className="flex items-start justify-between gap-4">
        <p className="text-xs text-muted-foreground">
          Preview uses saved agent + team files. Save changes, then refresh to update.
        </p>
        <div className="flex items-center gap-2 flex-shrink-0">
          <button
            type="button"
            onClick={() => void loadPromptPreview()}
            disabled={isPromptLoading}
            className={cn(
              'inline-flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1.5 text-xs font-medium',
              'text-muted-foreground hover:text-foreground hover:bg-muted/80 transition-colors',
              isPromptLoading && 'opacity-50 cursor-not-allowed'
            )}
          >
            <RefreshCw className="h-3.5 w-3.5" />
            Refresh
          </button>
          <button
            type="button"
            onClick={() => void handleCopyPrompt()}
            disabled={!promptPreview}
            className={cn(
              'inline-flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1.5 text-xs font-medium',
              'text-muted-foreground hover:text-foreground hover:bg-muted/80 transition-colors',
              !promptPreview && 'opacity-50 cursor-not-allowed'
            )}
          >
            <Copy className="h-3.5 w-3.5" />
            Copy
          </button>
        </div>
      </div>

      {isPromptLoading ? (
        <div className="flex items-center justify-center py-6 text-xs text-muted-foreground">
          Building prompt preview...
        </div>
      ) : promptError ? (
        <div className="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive">
          {promptError}
        </div>
      ) : (
        <div className="space-y-3">
          {pipelineSections.map((section, index) => {
            const agentFiles = section.key === 'agent-files'
              ? extractAgentFileBlocks(section.content)
              : []

            // Determine navigation action for this section
            const navAction = section.key === 'agent-files' && onNavigateToAgentFiles
              ? () => onNavigateToAgentFiles()
              : section.key === 'responsibilities' && onNavigateToTab
                ? () => onNavigateToTab('responsibilities')
                : section.key === 'heartbeat-task' && onNavigateToTab
                  ? () => onNavigateToTab('heartbeat')
                  : null

            const navLabel = section.key === 'agent-files'
              ? 'Open in Agent Editor'
              : section.key === 'responsibilities' || section.key === 'heartbeat-task'
                ? 'Edit'
                : null

            return (
              <div
                key={section.key}
                className="rounded-lg border border-border bg-background px-3 py-2"
              >
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <span className="text-[11px] font-semibold text-muted-foreground">
                      {index + 1}
                    </span>
                    <p className="text-xs font-medium text-foreground">{section.title}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    {navAction && navLabel && (
                      <button
                        type="button"
                        onClick={navAction}
                        className="inline-flex items-center gap-1 px-2 py-0.5 text-[11px] font-medium text-primary hover:text-primary/80 transition-colors"
                      >
                        {section.key === 'agent-files' ? (
                          <ExternalLink className="h-3 w-3" />
                        ) : (
                          <Pencil className="h-3 w-3" />
                        )}
                        {navLabel}
                      </button>
                    )}
                    <span
                      className={cn(
                        'px-2 py-0.5 text-[11px] rounded-full',
                        section.missing
                          ? 'bg-amber-500/10 text-amber-500'
                          : 'bg-emerald-500/10 text-emerald-500'
                      )}
                    >
                      {section.missing ? 'Not set' : 'Included'}
                    </span>
                  </div>
                </div>
                <p className="text-[11px] text-muted-foreground mt-1">{section.description}</p>
                {section.note && (
                  <p className="text-[11px] text-amber-500 mt-2">{section.note}</p>
                )}
                {section.missing ? (
                  <p className="text-[11px] text-muted-foreground mt-2">{section.emptyMessage}</p>
                ) : section.key === 'agent-files' && agentFiles.length > 0 ? (
                  <div className="mt-3 space-y-2">
                    {agentFiles.map((file) => (
                      <details
                        key={file.path}
                        className="rounded-lg border border-border bg-muted/40 px-3 py-2"
                      >
                        <summary className="cursor-pointer text-[11px] font-medium text-foreground flex items-center gap-1.5">
                          <span className="flex-1">{file.path}</span>
                          {onNavigateToAgentFiles && (
                            <button
                              type="button"
                              onClick={(e) => {
                                e.preventDefault()
                                e.stopPropagation()
                                onNavigateToAgentFiles(file.path)
                              }}
                              className="p-0.5 text-muted-foreground hover:text-primary transition-colors flex-shrink-0"
                              title={`Open ${file.path} in Agent Editor`}
                            >
                              <Link className="h-3 w-3" />
                            </button>
                          )}
                        </summary>
                        <pre className="mt-2 whitespace-pre-wrap text-[11px] text-muted-foreground">
                          {file.content || 'Empty file.'}
                        </pre>
                      </details>
                    ))}
                  </div>
                ) : (
                  <pre className="mt-3 max-h-40 overflow-y-auto whitespace-pre-wrap text-[11px] text-muted-foreground">
                    {section.content || section.emptyMessage}
                  </pre>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
