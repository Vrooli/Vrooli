/**
 * SkillCardView — Card grid of skills with metadata and description preview.
 */

import { Profiler, type ReactNode } from 'react'
import { Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Skill } from '@/types'
import type { DetailMode } from '@/types/filterSort'
import { SkillMetadataBadges } from './SkillMetadataBadges'
import { onProfilerRender } from '@/lib/profiler'
import { useVirtualRows } from '@/hooks/useVirtualRows'

interface SkillCardViewProps {
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

export function SkillCardView({
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
}: SkillCardViewProps) {
  return (
    <Profiler id="SkillCardView" onRender={onProfilerRender}>
      <SkillCardViewImpl
        skills={skills}
        selectedItemId={selectedItemId}
        onSelectItem={onSelectItem}
        dirtyItemIds={dirtyItemIds}
        detailMode={detailMode}
        healthScoreMap={healthScoreMap}
        renderItemIcon={renderItemIcon}
        onSkillContextMenu={onSkillContextMenu}
        combineMode={combineMode}
        combineSelectedIds={combineSelectedIds}
        onCombineToggleSkill={onCombineToggleSkill}
      />
    </Profiler>
  )
}

function SkillCardViewImpl({
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
}: SkillCardViewProps) {
  const showDetails = detailMode === 'full'
  const rowCount = Math.ceil(skills.length / 2)
  const rowHeight = showDetails ? 132 : 72
  const { containerRef, totalHeight, virtualRows } = useVirtualRows({
    count: rowCount,
    rowHeight,
  })

  if (skills.length === 0) {
    return (
      <div className="flex items-center justify-center py-8 text-xs text-muted-foreground">
        No skills match your filters
      </div>
    )
  }

  return (
    <div
      ref={containerRef}
      className="h-full overflow-y-auto p-2 grid-cols-2"
      role="listbox"
      data-testid="skill-card-view"
    >
      <div className="relative" style={{ height: totalHeight }}>
        {virtualRows.map(({ index: rowIndex, offsetTop }) => (
          <div
            key={rowIndex}
            className="absolute left-0 right-0 grid grid-cols-2 gap-2"
            style={{ top: offsetTop, height: rowHeight }}
          >
            {[0, 1].map((column) => {
              const skill = skills[rowIndex * 2 + column]
              if (!skill) return <div key={`empty-${column}`} />

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
                    'flex flex-col gap-1 p-2 rounded-lg border text-left transition-colors relative',
                    combineMode
                      ? isCombineSelected
                        ? 'bg-primary/10 border-primary/40'
                        : 'border-border hover:bg-muted hover:border-muted-foreground/20'
                      : isSelected
                        ? 'bg-primary/20 border-primary/40'
                        : 'border-border hover:bg-muted hover:border-muted-foreground/20'
                  )}
                  data-testid="skill-card-item"
                  data-skill-id={skill.id}
                >
                  {combineMode && (
                    <span className={cn(
                      'absolute top-1.5 right-1.5 w-4 h-4 rounded border flex items-center justify-center transition-colors',
                      isCombineSelected
                        ? 'bg-primary border-primary'
                        : 'border-muted-foreground/40 bg-background'
                    )}>
                      {isCombineSelected && <Check className="h-3 w-3 text-primary-foreground" />}
                    </span>
                  )}

                  <div className="flex items-start gap-1.5 min-w-0">
                    {!combineMode && renderItemIcon && (
                      <span className="flex-shrink-0 mt-0.5">{renderItemIcon(skill)}</span>
                    )}
                    <span className="text-xs font-medium truncate flex-1 text-foreground">
                      {skill.name}
                    </span>
                    {isDirty && (
                      <span className="w-2 h-2 bg-amber-500 rounded-full flex-shrink-0 mt-1" />
                    )}
                  </div>

                  {showDetails && (
                    <>
                      {skill.description && (
                        <p className="text-[10px] text-muted-foreground line-clamp-2 leading-tight">
                          {skill.description}
                        </p>
                      )}

                      {skill.tags.length > 0 && (
                        <div className="flex items-center gap-0.5 flex-wrap">
                          {skill.tags.slice(0, 2).map((tag) => (
                            <span
                              key={tag}
                              className="px-1 py-0.5 text-[8px] bg-muted rounded text-muted-foreground"
                            >
                              {tag}
                            </span>
                          ))}
                          {skill.tags.length > 2 && (
                            <span className="text-[8px] text-muted-foreground">
                              +{skill.tags.length - 2}
                            </span>
                          )}
                        </div>
                      )}

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
        ))}
      </div>
    </div>
  )
}
