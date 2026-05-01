import { useState, useEffect, useMemo, useCallback } from 'react'
import { copyToClipboard } from '@/lib/clipboard'
import { RefreshCw, Copy, ExternalLink, Pencil, Link } from 'lucide-react'
import { cn } from '@/lib/utils'
import { toast } from '@/hooks/use-toast'
import * as agentService from '@/services/agentService'
import type { PromptSection } from '@/lib/schemas'

interface MemberPromptPipelineSectionProps {
  teamId: string
  memberId: string
  onNavigateToTab?: (tab: 'responsibilities' | 'heartbeat') => void
  onNavigateToAgentFiles?: (filePath?: string) => void
}

interface PipelineSectionMeta {
  description: string
}

interface AgentFileBlock {
  path: string
  content: string
}

const SECTION_META: Record<string, PipelineSectionMeta> = {
  'agent-file': {
    description: 'SOUL.md and other agent markdown files.',
  },
  'team-shared-charter': {
    description: 'Team-level charter and operating model.',
  },
  'execution-brief': {
    description: 'Generated orientation for this member and heartbeat prompt.',
  },
  'team-operating-contract': {
    description: 'Resolved team/member policy generated from team.json.',
  },
  'team-responsibilities': {
    description: 'Role-specific instructions for this team member.',
  },
  'team-org-context': {
    description: 'Reporting context for this team member.',
  },
  'team-coordination': {
    description: 'Coordination policy and available teammate interactions.',
  },
  'team-storage-map': {
    description: 'Persistent storage primitives, authority order, and available commands.',
  },
  'team-inbox': {
    description: 'Pending messages from other team members.',
  },
  'last-handoff': {
    description: 'Continuity notes from the member’s previous heartbeat.',
  },
  'heartbeat-task': {
    description: 'The exact task this member will execute on each heartbeat.',
  },
}

function reassemblePrompt(sections: PromptSection[]): string {
  if (sections.length === 0) return ''
  const parts: string[] = []
  for (let i = 0; i < sections.length; i += 1) {
    const section = sections[i]
    if (!section) continue
    if (section.kind === 'agent-file') {
      let block = '# Agent Files (Markdown)\n\n'
      while (i < sections.length && sections[i]?.kind === 'agent-file') {
        block += sections[i]?.content ?? ''
        i += 1
      }
      i -= 1
      parts.push(block)
    } else {
      parts.push(section.content)
    }
  }
  return parts.join('\n\n---\n\n')
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
  const [promptSections, setPromptSections] = useState<PromptSection[]>([])
  const [promptError, setPromptError] = useState<string | null>(null)
  const [isPromptLoading, setIsPromptLoading] = useState(false)

  // DOC: docs/concepts/HEARTBEATS.md#prompt-pipeline-ui
  const loadPromptPreview = useCallback(async () => {
    setIsPromptLoading(true)
    setPromptError(null)
    try {
      const response = await agentService.previewAgentPromptStructured(memberId, teamId)
      setPromptSections(response.sections)
    } catch (err) {
      console.error('Failed to load prompt preview:', err)
      setPromptSections([])
      setPromptError('Unable to build prompt preview. Check the API and try again.')
    } finally {
      setIsPromptLoading(false)
    }
  }, [memberId, teamId])

  const promptPreview = useMemo(() => reassemblePrompt(promptSections), [promptSections])

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
    if (promptSections.length > 0 || isPromptLoading) return
    void loadPromptPreview()
  }, [promptSections.length, isPromptLoading, loadPromptPreview])

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
          {promptSections.map((section, index) => {
            const agentFiles = section.kind === 'agent-file'
              ? extractAgentFileBlocks(section.content)
              : []
            const meta = SECTION_META[section.kind]

            // Determine navigation action for this section
            const navAction = section.kind === 'agent-file' && onNavigateToAgentFiles
              ? () => onNavigateToAgentFiles()
              : section.kind === 'team-responsibilities' && onNavigateToTab
                ? () => onNavigateToTab('responsibilities')
                : section.kind === 'heartbeat-task' && onNavigateToTab
                  ? () => onNavigateToTab('heartbeat')
                  : null

            const navLabel = section.kind === 'agent-file'
              ? 'Open in Agent Editor'
              : section.kind === 'team-responsibilities' || section.kind === 'heartbeat-task'
                ? 'Edit'
                : null

            return (
              <div
                key={`${section.kind}-${section.label}-${index}`}
                className="rounded-lg border border-border bg-background px-3 py-2"
              >
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <span className="text-[11px] font-semibold text-muted-foreground">
                      {index + 1}
                    </span>
                    <p className="text-xs font-medium text-foreground">{section.label}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    {navAction && navLabel && (
                      <button
                        type="button"
                        onClick={navAction}
                        className="inline-flex items-center gap-1 px-2 py-0.5 text-[11px] font-medium text-primary hover:text-primary/80 transition-colors"
                      >
                        {section.kind === 'agent-file' ? (
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
                        'bg-emerald-500/10 text-emerald-500'
                      )}
                    >
                      Included
                    </span>
                  </div>
                </div>
                <p className="text-[11px] text-muted-foreground mt-1">
                  {meta?.description ?? section.kind}
                </p>
                {section.kind === 'agent-file' && agentFiles.length > 0 ? (
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
                    {section.content || 'Empty section.'}
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
