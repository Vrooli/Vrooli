/**
 * MemberInboxTab - Gmail-style view of a member's unrouted inbox entries.
 *
 * Shows one section per intake prefix declared on the member's topics.json.
 * Each section lists the unrouted entries under that prefix, with inline
 * actions to promote (retag to a destination prefix) or drop (delete).
 *
 * Promote/drop translates to the same CLI verbs the agent uses:
 *   prompt-manager team knowledge-update <team> <id> --topic=<destination>
 *   prompt-manager team knowledge-delete <team> <id>
 *
 * After every action, the entry leaves the unrouted set — the inbox view is
 * the unrouted set by the inbox-router-drain invariant.
 *
 * DOC: docs/agent-system/INTAKE_PIPELINE.md
 */

import { useCallback, useEffect, useState } from 'react'
import { ExternalLink, Inbox, RefreshCw, Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import * as memberFlowService from '@/services/memberFlowService'
import type { KnowledgeEntry } from '@/services/heartbeatService'
import type { TopicIntakeEntry, TopicOutputEntry } from '@/types/topicsGraph'

interface MemberInboxTabProps {
  teamId: string
  intake: TopicIntakeEntry[]
  output: TopicOutputEntry[]
}

interface IntakeSectionProps {
  teamId: string
  intake: TopicIntakeEntry
  output: TopicOutputEntry[]
}

const ENTRY_LIMIT = 100

function formatAge(at?: string): string {
  if (!at) return ''
  const t = new Date(at).getTime()
  if (Number.isNaN(t)) return ''
  const diff = Date.now() - t
  if (diff < 60_000) return 'just now'
  if (diff < 3_600_000) {
    const m = Math.round(diff / 60_000)
    return `${m}m ago`
  }
  if (diff < 86_400_000) {
    const h = Math.round(diff / 3_600_000)
    return `${h}h ago`
  }
  const d = Math.round(diff / 86_400_000)
  return `${d}d ago`
}

function previewContent(content: string, max = 180): string {
  const trimmed = content.trim()
  if (trimmed.length <= max) return trimmed
  return trimmed.slice(0, max).trimEnd() + '…'
}

/**
 * Derive a destination topic from a destination prefix and an entry's topic.
 *
 * Examples:
 *   prefix="audience-scan/*" entry.topic="research-inbox/audience/foo" -> "audience-scan/foo"
 *   prefix="audience-scan/*" entry.topic="research-inbox/audience" -> "audience-scan/audience"
 *   prefix="competitor"       entry.topic="..."                   -> "competitor"   (exact prefix)
 *
 * The slug is the final path segment of the entry's topic.
 */
function buildDestinationTopic(prefix: string, entryTopic: string): string {
  if (!prefix.endsWith('/*')) return prefix
  const base = prefix.slice(0, -2) // strip trailing /*
  const segments = entryTopic.split('/').filter(Boolean)
  const slug = segments[segments.length - 1] || 'entry'
  return `${base}/${slug}`
}

function IntakeSection({ teamId, intake, output }: IntakeSectionProps) {
  const [entries, setEntries] = useState<KnowledgeEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [destinations, setDestinations] = useState<Record<string, string>>({})

  // Strip the trailing /* for the actual list call (the API does prefix matching).
  const listPrefix = intake.prefix.endsWith('/*')
    ? intake.prefix.slice(0, -1) // keep the trailing slash
    : intake.prefix

  const reload = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const list = await memberFlowService.listInboxEntries(teamId, listPrefix, {
        limit: ENTRY_LIMIT,
      })
      setEntries(list)
    } catch (err) {
      console.warn('Failed to load inbox entries:', err)
      setError(err instanceof Error ? err.message : 'Failed to load inbox entries')
      setEntries([])
    } finally {
      setLoading(false)
    }
  }, [teamId, listPrefix])

  useEffect(() => {
    void reload()
  }, [reload])

  const handlePromote = useCallback(
    async (entry: KnowledgeEntry) => {
      const destPrefix = destinations[entry.id] ?? output[0]?.prefix
      if (!destPrefix) return
      const destTopic = buildDestinationTopic(destPrefix, entry.topic)
      setBusyId(entry.id)
      try {
        await memberFlowService.promoteInboxEntry(teamId, entry.id, destTopic)
        setEntries((prev) => prev.filter((e) => e.id !== entry.id))
      } catch (err) {
        console.warn('Failed to promote inbox entry:', err)
        setError(err instanceof Error ? err.message : 'Failed to promote entry')
      } finally {
        setBusyId(null)
      }
    },
    [teamId, destinations, output],
  )

  const handleDrop = useCallback(
    async (entry: KnowledgeEntry) => {
      setBusyId(entry.id)
      try {
        await memberFlowService.dropInboxEntry(teamId, entry.id)
        setEntries((prev) => prev.filter((e) => e.id !== entry.id))
      } catch (err) {
        console.warn('Failed to drop inbox entry:', err)
        setError(err instanceof Error ? err.message : 'Failed to drop entry')
      } finally {
        setBusyId(null)
      }
    },
    [teamId],
  )

  const knowledgeOutputs = output.filter((o) => o.destination_kind === 'knowledge')
  const hasDestinations = knowledgeOutputs.length > 0

  return (
    <section
      data-testid={`inbox-section-${intake.prefix}`}
      className="border border-border rounded-lg overflow-hidden"
    >
      <header className="flex items-center justify-between gap-3 px-4 py-3 bg-muted/40 border-b border-border">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <Inbox className="h-4 w-4 text-muted-foreground flex-shrink-0" />
            <code className="text-sm font-mono text-foreground truncate">{intake.prefix}</code>
            <span
              className="px-2 py-0.5 text-xs rounded-full bg-primary/10 text-primary border border-primary/30"
              data-testid={`inbox-count-${intake.prefix}`}
            >
              {entries.length} unrouted
            </span>
          </div>
          {intake.taxonomy && (
            <p className="text-xs text-muted-foreground mt-1">
              Taxonomy: <code>{intake.taxonomy}</code>
              {intake.classifier_skill && (
                <>
                  {' · Classifier: '}
                  <code>{intake.classifier_skill}</code>
                </>
              )}
            </p>
          )}
        </div>
        <button
          type="button"
          onClick={() => void reload()}
          disabled={loading}
          aria-label="Refresh inbox"
          className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0 disabled:opacity-40"
        >
          <RefreshCw className={cn('h-3.5 w-3.5', loading && 'animate-spin')} />
        </button>
      </header>

      {error && (
        <div className="px-4 py-2 text-xs text-destructive bg-destructive/10 border-b border-destructive/30">
          {error}
        </div>
      )}

      {loading ? (
        <div className="p-4 space-y-2 animate-pulse">
          <div className="h-4 bg-muted rounded w-1/2" />
          <div className="h-4 bg-muted rounded w-3/4" />
        </div>
      ) : entries.length === 0 ? (
        <div className="p-6 text-center text-sm text-muted-foreground" data-testid={`inbox-empty-${intake.prefix}`}>
          Inbox is clear. Unrouted entries will appear here.
        </div>
      ) : (
        <ul className="divide-y divide-border">
          {entries.map((entry) => {
            const selectedDest = destinations[entry.id] ?? knowledgeOutputs[0]?.prefix ?? ''
            const isBusy = busyId === entry.id
            return (
              <li
                key={entry.id}
                data-testid={`inbox-entry-${entry.id}`}
                className="p-4 flex flex-col gap-2"
              >
                <div className="flex items-start justify-between gap-3 text-xs text-muted-foreground">
                  <div className="flex items-center gap-2 min-w-0 flex-wrap">
                    <code className="text-foreground/80">{entry.topic}</code>
                    <span>·</span>
                    <span>by {entry.caller || 'unknown'}</span>
                    <span>·</span>
                    <span>{formatAge(entry.at)}</span>
                  </div>
                  {entry.source && (
                    <a
                      href={entry.source}
                      target="_blank"
                      rel="noreferrer"
                      className="flex items-center gap-1 text-primary hover:underline flex-shrink-0"
                    >
                      source <ExternalLink className="h-3 w-3" />
                    </a>
                  )}
                </div>

                <p className="text-sm text-foreground/90 whitespace-pre-wrap">
                  {previewContent(entry.content)}
                </p>

                <div className="flex items-center gap-2 flex-wrap">
                  {hasDestinations ? (
                    <>
                      <select
                        value={selectedDest}
                        onChange={(e) =>
                          setDestinations((prev) => ({ ...prev, [entry.id]: e.target.value }))
                        }
                        disabled={isBusy}
                        aria-label="Destination prefix"
                        data-testid={`inbox-destination-${entry.id}`}
                        className={cn(
                          'px-2 py-1 text-xs font-mono',
                          'bg-muted border border-border rounded',
                          'focus:outline-none focus:ring-2 focus:ring-primary',
                        )}
                      >
                        {knowledgeOutputs.map((o) => (
                          <option key={o.prefix} value={o.prefix}>
                            {o.prefix}
                          </option>
                        ))}
                      </select>
                      <button
                        type="button"
                        onClick={() => void handlePromote(entry)}
                        disabled={isBusy || !selectedDest}
                        data-testid={`inbox-promote-${entry.id}`}
                        className={cn(
                          'px-3 py-1 text-xs font-medium rounded',
                          'bg-primary text-primary-foreground hover:bg-primary/90',
                          'disabled:opacity-50 disabled:cursor-not-allowed',
                        )}
                      >
                        Promote
                      </button>
                    </>
                  ) : (
                    <span className="text-xs text-muted-foreground italic">
                      No knowledge destinations declared in topics.json output[]
                    </span>
                  )}
                  <button
                    type="button"
                    onClick={() => void handleDrop(entry)}
                    disabled={isBusy}
                    data-testid={`inbox-drop-${entry.id}`}
                    className={cn(
                      'flex items-center gap-1 px-2 py-1 text-xs font-medium rounded',
                      'text-destructive hover:bg-destructive/10',
                      'disabled:opacity-50 disabled:cursor-not-allowed',
                    )}
                  >
                    <Trash2 className="h-3 w-3" />
                    Drop
                  </button>
                </div>
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}

export function MemberInboxTab({ teamId, intake, output }: MemberInboxTabProps) {
  if (intake.length === 0) {
    return (
      <div className="p-6 text-center text-sm text-muted-foreground" data-testid="inbox-no-intake">
        This member declares no <code>intake[]</code> in their <code>topics.json</code>, so
        they don&apos;t drain any team knowledge inbox.
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <p className="text-xs text-muted-foreground">
        Unrouted entries under each declared intake prefix. Promote to retag to a destination,
        or drop to delete. Routed entries leave this view by the inbox-router-drain invariant.
      </p>
      {intake.map((entry) => (
        <IntakeSection
          key={entry.prefix}
          teamId={teamId}
          intake={entry}
          output={output}
        />
      ))}
    </div>
  )
}
