/**
 * SkillListView — Flat sorted list of skills with metadata.
 */

import { type ReactNode } from 'react'
import { Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Skill } from '@/types'
import type { DetailMode } from '@/types/filterSort'
import { SkillMetadataBadges } from './SkillMetadataBadges'

interface SkillListViewProps {
  skills: Skill[]
  selectedItemId: string | null
  onSelectItem: (id: string) => void
  dirtyItemIds: Set<string>
  detailMode: DetailMode
  healthScoreMap?: Map<string, number>
  renderItemIcon?: (skill: Skill) => ReactNode
  onSkillContextMenu?: (
    skillId: string,
    skillName: string,
    x: number,
    y: number
  ) => void
  combineMode?: boolean
  combineSelectedIds?: Set<string>
  onCombineToggleSkill?: (skillId: string) => void
}

export function SkillListView({
  skills,
  selectedItemId,
  onSelectItem,
  dirtyItemIds,
  detailMode,
  healthScoreMap,
  renderItemIcon,
  onSkillContextMenu,
  combineMode = false,
  combineSelectedIds,
  onCombineToggleSkill,
}: SkillListViewProps) {
  if (skills.length === 0) {
    return (
      <div className="flex items-center justify-center py-8 text-xs text-muted-foreground">
        No skills match your filters
      </div>
    )
  }

  const showDetails = detailMode === 'full'

  return (
    <div className="flex flex-col" role="listbox" data-testid="skill-list-view">
      {skills.map((skill) => {
        const isSelected = skill.id === selectedItemId
        const isDirty = dirtyItemIds.has(skill.id)
        const isCombineSelected = combineMode && combineSelectedIds?.has(skill.id)

        const handleClick = () => {
          if (combineMode && onCombineToggleSkill) {
            onCombineToggleSkill(skill.id)
          } else {
            onSelectItem(skill.id)
          }
        }

        return (
          <button
            key={skill.id}
            type="button"
            role="option"
            aria-selected={combineMode ? !!isCombineSelected : isSelected}
            onClick={handleClick}
            onContextMenu={(e) => {
              if (onSkillContextMenu && !combineMode) {
                e.preventDefault()
                onSkillContextMenu(skill.id, skill.name, e.clientX, e.clientY)
              }
            }}
            className={cn(
              'flex flex-col gap-0.5 px-3 text-left transition-colors border-b border-border/50',
              showDetails ? 'py-2' : 'py-1.5',
              combineMode
                ? isCombineSelected
                  ? 'bg-primary/10 text-foreground'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                : isSelected
                  ? 'bg-primary/30 text-foreground'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground'
            )}
            data-testid="skill-list-item"
            data-skill-id={skill.id}
          >
            {/* Top row: checkbox/icon + name + dirty dot */}
            <div className="flex items-center gap-2 min-w-0">
              {combineMode ? (
                <span className={cn(
                  'flex-shrink-0 w-4 h-4 rounded border flex items-center justify-center transition-colors',
                  isCombineSelected
                    ? 'bg-primary border-primary'
                    : 'border-muted-foreground/40'
                )}>
                  {isCombineSelected && <Check className="h-3 w-3 text-primary-foreground" />}
                </span>
              ) : renderItemIcon ? (
                <span className="flex-shrink-0">{renderItemIcon(skill)}</span>
              ) : null}
              <span className="text-xs font-medium truncate flex-1">
                {skill.name}
              </span>
              {isDirty && (
                <span className="w-2 h-2 bg-amber-500 rounded-full flex-shrink-0" />
              )}
            </div>

            {showDetails && (
              <>
                {/* Description preview */}
                {skill.description && (
                  <p className="text-[10px] text-muted-foreground truncate">
                    {skill.description}
                  </p>
                )}

                {/* Tags */}
                {skill.tags.length > 0 && (
                  <div className="flex items-center gap-1 flex-wrap">
                    {skill.tags.slice(0, 3).map((tag) => (
                      <span
                        key={tag}
                        className="px-1.5 py-0.5 text-[9px] bg-muted rounded-full text-muted-foreground"
                      >
                        {tag}
                      </span>
                    ))}
                    {skill.tags.length > 3 && (
                      <span className="text-[9px] text-muted-foreground">
                        +{skill.tags.length - 3}
                      </span>
                    )}
                  </div>
                )}

                {/* Metadata badges */}
                <SkillMetadataBadges
                  skill={skill}
                  healthScore={healthScoreMap?.get(skill.id) ?? null}
                />
              </>
            )}
          </button>
        )
      })}
    </div>
  )
}
