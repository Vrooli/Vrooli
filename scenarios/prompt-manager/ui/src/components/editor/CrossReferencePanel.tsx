/**
 * CrossReferencePanel - Collapsible panel showing which agents, teams,
 * and skills reference a given skill.
 */

import { useEffect, useState } from 'react'
import { ChevronRight, ChevronDown, Bot, Users, Sparkles } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useSelectionStore } from '@/stores/selectionStore'
import { getSkillXRefs } from '@/services/skillService'
import { selectors } from '@/constants/selectors'
import type { Reference } from '@/lib/schemas'

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

export function CrossReferencePanel({ skillId, onNavigateToReference, className }: CrossReferencePanelProps) {
  const [expanded, setExpanded] = useState(false)
  const [references, setReferences] = useState<Reference[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)

  const setSelectedAgentId = useSelectionStore((s) => s.setSelectedAgentId)
  const setSelectedTeamId = useSelectionStore((s) => s.setSelectedTeamId)
  const setSelectedSkillId = useSelectionStore((s) => s.setSelectedSkillId)

  useEffect(() => {
    let cancelled = false
    setLoading(true)

    void getSkillXRefs(skillId).then((result) => {
      if (cancelled) return
      if (result) {
        setReferences(result.references)
        setTotal(result.total)
      } else {
        setReferences([])
        setTotal(0)
      }
      setLoading(false)
    })

    return () => {
      cancelled = true
    }
  }, [skillId])

  const handleClick = (ref: Reference) => {
    if (onNavigateToReference) {
      onNavigateToReference(ref)
      return
    }
    const { entityType, entityId } = ref.source
    if (entityType === 'agent') setSelectedAgentId(entityId)
    else if (entityType === 'team') setSelectedTeamId(entityId)
    else setSelectedSkillId(entityId)
  }

  // Group references by entity type
  const grouped = new Map<EntityType, Reference[]>()
  for (const ref of references) {
    const type = ref.source.entityType
    const list = grouped.get(type) ?? []
    list.push(ref)
    grouped.set(type, list)
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

          {!loading && references.length === 0 && (
            <div className="text-xs text-muted-foreground">No references found</div>
          )}

          {!loading &&
            GROUP_ORDER.filter((type) => grouped.has(type)).map((type) => {
              const config = GROUP_CONFIG[type]
              const Icon = config.icon
              const refs = grouped.get(type) ?? []

              return (
                <div key={type}>
                  <div className="flex items-center gap-1 text-xs font-medium text-muted-foreground mb-0.5">
                    <Icon className="h-3 w-3" />
                    <span>{config.label}</span>
                  </div>
                  {refs.map((ref, i) => (
                    <button
                      key={`${ref.source.entityId}-${ref.source.filePath}-${i}`}
                      type="button"
                      onClick={() => handleClick(ref)}
                      className="block w-full text-left pl-4 py-0.5 text-xs text-foreground/80 hover:text-foreground hover:bg-muted/50 rounded transition-colors"
                      data-testid={selectors.xrefs.item}
                    >
                      <span className="font-medium">{ref.source.entityName}</span>
                      <span className="text-muted-foreground ml-1.5">
                        {ref.source.filePath}
                        {ref.source.lineNumber > 0 && `:${ref.source.lineNumber}`}
                      </span>
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
