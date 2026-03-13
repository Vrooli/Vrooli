/**
 * CrossReferencePanel - Icon button that opens a modal showing which agents,
 * teams, and skills reference the current skill.
 *
 * Uses the graph API (GET /api/v1/graph/node/{id}) to discover inbound edges
 * where this skill is the target, and displays them grouped by source entity type.
 */

import { useEffect, useState } from 'react'
import { Bot, Users, Sparkles, Link2, Network } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useSelectionStore } from '@/stores/selectionStore'
import { useGraphStore } from '@/stores/graphStore'
import { getGraphNode } from '@/services/graphService'
import { selectors } from '@/constants/selectors'
import type { GraphEdge, Reference } from '@/lib/schemas'
import { Dialog } from '@/components/shared/Dialog'

interface CrossReferencePanelProps {
  skillId: string
  /** Called when a reference is clicked, for highlight navigation */
  onNavigateToReference?: (ref: Reference) => void
  className?: string
}

type EntityType = 'agent' | 'team' | 'skill'

const GROUP_CONFIG: Record<EntityType, { label: string; icon: typeof Bot }> = {
  agent: { label: 'Agents', icon: Bot },
  team: { label: 'Teams', icon: Users },
  skill: { label: 'Skills', icon: Sparkles },
}

const GROUP_ORDER: EntityType[] = ['agent', 'team', 'skill']

function inferEntityTypeFromEdge(edge: GraphEdge): EntityType {
  if (edge.kind === 'membership') {
    return 'team'
  }
  if (edge.kind === 'cli-read' || edge.kind === 'bold-listed' || edge.kind === 'code-usage') {
    return 'agent'
  }
  return 'skill'
}

function mapNodeTypeToEntityType(nodeType: string | undefined): EntityType | null {
  if (nodeType === 'agent' || nodeType === 'team' || nodeType === 'skill') {
    return nodeType
  }
  return null
}

/** Represents a resolved cross-reference from the graph API. */
interface ResolvedRef {
  entityType: EntityType
  entityId: string
  entityName: string
  filePath: string
  lineNumber: number
  edgeKind: string
}

export function CrossReferencePanel({ skillId, onNavigateToReference, className }: CrossReferencePanelProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [refs, setRefs] = useState<ResolvedRef[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)

  const setSelectedAgentId = useSelectionStore((s) => s.setSelectedAgentId)
  const setSelectedTeamId = useSelectionStore((s) => s.setSelectedTeamId)
  const setSelectedSkillId = useSelectionStore((s) => s.setSelectedSkillId)
  const setGraphViewActive = useSelectionStore((s) => s.setGraphViewActive)
  const focusNodes = useGraphStore((s) => s.focusNodes)

  useEffect(() => {
    let cancelled = false
    setLoading(true)

    void (async () => {
      const result = await getGraphNode(skillId)
      if (cancelled as boolean) return

      if (result) {
        // Find edges where this skill is the target (inbound references)
        const inboundEdges = result.adjacentEdges.filter((e: GraphEdge) => e.to === skillId)
        const sourceIds = Array.from(new Set(inboundEdges.map((edge) => edge.from)))
        const sourceNodeMap = new Map<string, Awaited<ReturnType<typeof getGraphNode>>>()

        // Resolve source node types so entity routing is accurate.
        await Promise.all(sourceIds.map(async (sourceId) => {
          const sourceNode = await getGraphNode(sourceId)
          sourceNodeMap.set(sourceId, sourceNode)
        }))
        if (cancelled as boolean) return

        // For inbound edges, the 'from' is the entity that references us.
        const resolved: ResolvedRef[] = []
        for (const edge of inboundEdges) {
          const sourceNode = sourceNodeMap.get(edge.from)
          const entityType = mapNodeTypeToEntityType(sourceNode?.node.type) ?? inferEntityTypeFromEdge(edge)
          resolved.push({
            entityType,
            entityId: edge.from,
            entityName: sourceNode?.node.label || edge.from,
            filePath: edge.sourceFile,
            lineNumber: edge.lineNumber,
            edgeKind: edge.kind,
          })
        }

        setRefs(resolved)
        setTotal(resolved.length)
      } else {
        setRefs([])
        setTotal(0)
      }
      setLoading(false)
    })()

    return () => {
      cancelled = true
    }
  }, [skillId])

  const handleClick = (ref: ResolvedRef) => {
    // If there's an onNavigateToReference callback, adapt the resolved ref
    // to the legacy Reference shape for compatibility
    if (onNavigateToReference) {
      const legacyRef: Reference = {
        skillId,
        refType: 'bold-listed',
        source: {
          entityType: ref.entityType,
          entityId: ref.entityId,
          entityName: ref.entityName,
          filePath: ref.filePath,
          lineNumber: ref.lineNumber,
        },
      }
      onNavigateToReference(legacyRef)
      setIsOpen(false)
      return
    }
    if (ref.entityType === 'agent') setSelectedAgentId(ref.entityId)
    else if (ref.entityType === 'team') setSelectedTeamId(ref.entityId)
    else setSelectedSkillId(ref.entityId)
    setIsOpen(false)
  }

  // Group references by entity type
  const grouped = new Map<EntityType, ResolvedRef[]>()
  for (const ref of refs) {
    const list = grouped.get(ref.entityType) ?? []
    list.push(ref)
    grouped.set(ref.entityType, list)
  }

  return (
    <div className={cn('text-sm', className)}>
      <button
        type="button"
        onClick={() => setIsOpen(true)}
        className={cn(
          'relative flex items-center gap-1 px-1.5 sm:px-2 py-0.5 rounded text-xs',
          'text-foreground hover:bg-muted transition-colors',
          'border border-transparent hover:border-border',
        )}
        data-testid={selectors.xrefs.toggle}
        title={`Referenced by (${loading ? '...' : total})`}
        aria-label={`Open references modal${total > 0 ? ` (${total} references)` : ''}`}
      >
        <Link2 className="h-4 w-4" />
        {!loading && total > 0 && (
          <span
            className={cn(
              'absolute -top-1 -right-1 min-w-[16px] h-4 px-1 rounded-full',
              'bg-primary/20 text-primary text-[10px] leading-4 font-medium text-center',
              'border border-primary/30',
            )}
            aria-hidden="true"
          >
            {total > 99 ? '99+' : total}
          </span>
        )}
      </button>

      <Dialog
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        title={`Referenced by (${loading ? '...' : total})`}
        maxWidth="max-w-2xl"
        testId={selectors.xrefs.panel}
      >
        <div className="space-y-3">
          <div className="flex justify-end">
            <button
              type="button"
              onClick={() => {
                focusNodes([skillId])
                setGraphViewActive(true)
                setSelectedAgentId(null)
                setSelectedTeamId(null)
                setSelectedSkillId(null)
                setIsOpen(false)
              }}
              className={cn(
                'inline-flex items-center gap-2 px-3 py-1.5 rounded-md text-sm',
                'bg-indigo-600/20 text-indigo-200 border border-indigo-500/30',
                'hover:bg-indigo-600/30 transition-colors',
              )}
            >
              <Network className="h-4 w-4" />
              Open in Graph View
            </button>
          </div>

          <div className="space-y-1.5" data-testid={selectors.xrefs.list}>
            {loading && (
              <div className="text-sm text-slate-300">Loading...</div>
            )}

            {!loading && refs.length === 0 && (
              <div className="text-sm text-slate-300">No references found</div>
            )}

            {!loading &&
              GROUP_ORDER.filter((type) => grouped.has(type)).map((type) => {
                const config = GROUP_CONFIG[type]
                const Icon = config.icon
                const groupRefs = grouped.get(type) ?? []

                return (
                  <div key={type}>
                    <div className="flex items-center gap-1.5 text-sm font-medium text-slate-300 mb-1">
                      <Icon className="h-4 w-4" />
                      <span>{config.label}</span>
                    </div>
                    {groupRefs.map((ref, i) => (
                      <button
                        key={`${ref.entityId}-${ref.filePath}-${i}`}
                        type="button"
                        onClick={() => handleClick(ref)}
                        className={cn(
                          'block w-full text-left px-3 py-2 text-sm rounded-md transition-colors',
                          'text-slate-200 hover:text-white hover:bg-white/10'
                        )}
                        data-testid={selectors.xrefs.item}
                      >
                        <span className="font-medium">{ref.entityName}</span>
                        {ref.filePath && (
                          <span className="text-slate-400 ml-2">
                            {ref.filePath}
                            {ref.lineNumber > 0 && `:${ref.lineNumber}`}
                          </span>
                        )}
                      </button>
                    ))}
                  </div>
                )
              })}
          </div>
        </div>
      </Dialog>
    </div>
  )
}
