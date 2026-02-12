/**
 * CrossReferencePanel - Collapsible panel showing which agents, teams,
 * and skills reference a given skill.
 *
 * Uses the graph API (GET /api/v1/graph/node/{id}) to discover inbound edges
 * where this skill is the target, and displays them grouped by source entity type.
 */

import { useEffect, useState } from 'react'
import { ChevronRight, ChevronDown, Bot, Users, Sparkles } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useSelectionStore } from '@/stores/selectionStore'
import { getGraphNode } from '@/services/graphService'
import { selectors } from '@/constants/selectors'
import type { GraphEdge, Reference } from '@/lib/schemas'

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
  const [expanded, setExpanded] = useState(false)
  const [refs, setRefs] = useState<ResolvedRef[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)

  const setSelectedAgentId = useSelectionStore((s) => s.setSelectedAgentId)
  const setSelectedTeamId = useSelectionStore((s) => s.setSelectedTeamId)
  const setSelectedSkillId = useSelectionStore((s) => s.setSelectedSkillId)

  useEffect(() => {
    let cancelled = false
    setLoading(true)

    void getGraphNode(skillId).then((result) => {
      if (cancelled) return
      if (result) {
        // Find edges where this skill is the target (inbound references)
        const inboundEdges = result.adjacentEdges.filter(
          (e: GraphEdge) => e.to === skillId
        )

        // For inbound edges, the 'from' is the entity that references us.
        const resolved: ResolvedRef[] = []
        for (const edge of inboundEdges) {
          // Determine entity type from edge kind
          let entityType: EntityType = 'skill'
          if (edge.kind === 'membership') {
            entityType = 'team'
          } else if (edge.kind === 'cli-read' || edge.kind === 'bold-listed' || edge.kind === 'code-usage') {
            entityType = 'agent'
          }

          resolved.push({
            entityType,
            entityId: edge.from,
            entityName: edge.from, // best we have without a full graph lookup
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
    })

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
      return
    }
    if (ref.entityType === 'agent') setSelectedAgentId(ref.entityId)
    else if (ref.entityType === 'team') setSelectedTeamId(ref.entityId)
    else setSelectedSkillId(ref.entityId)
  }

  // Group references by entity type
  const grouped = new Map<EntityType, ResolvedRef[]>()
  for (const ref of refs) {
    const list = grouped.get(ref.entityType) ?? []
    list.push(ref)
    grouped.set(ref.entityType, list)
  }

  return (
    <div className={cn('text-sm', className)} data-testid={selectors.xrefs.panel}>
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-1 text-muted-foreground hover:text-foreground transition-colors"
        data-testid={selectors.xrefs.toggle}
      >
        {expanded ? (
          <ChevronDown className="h-3.5 w-3.5" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5" />
        )}
        <span>
          Referenced by ({loading ? '...' : total})
        </span>
      </button>

      {expanded && (
        <div className="mt-1 ml-4 space-y-1.5" data-testid={selectors.xrefs.list}>
          {loading && (
            <div className="text-xs text-muted-foreground">Loading...</div>
          )}

          {!loading && refs.length === 0 && (
            <div className="text-xs text-muted-foreground">No references found</div>
          )}

          {!loading &&
            GROUP_ORDER.filter((type) => grouped.has(type)).map((type) => {
              const config = GROUP_CONFIG[type]
              const Icon = config.icon
              const groupRefs = grouped.get(type) ?? []

              return (
                <div key={type}>
                  <div className="flex items-center gap-1 text-xs font-medium text-muted-foreground mb-0.5">
                    <Icon className="h-3 w-3" />
                    <span>{config.label}</span>
                  </div>
                  {groupRefs.map((ref, i) => (
                    <button
                      key={`${ref.entityId}-${ref.filePath}-${i}`}
                      type="button"
                      onClick={() => handleClick(ref)}
                      className="block w-full text-left pl-4 py-0.5 text-xs text-foreground/80 hover:text-foreground hover:bg-muted/50 rounded transition-colors"
                      data-testid={selectors.xrefs.item}
                    >
                      <span className="font-medium">{ref.entityName}</span>
                      {ref.filePath && (
                        <span className="text-muted-foreground ml-1.5">
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
      )}
    </div>
  )
}
