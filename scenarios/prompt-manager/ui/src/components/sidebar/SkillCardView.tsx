/** SkillCardView — collection-backed responsive card presentation. */
import { Profiler, type ReactNode } from 'react'
import { CollectionList } from '@vrooli/react-component-library/CollectionList/1.0.0'
import type { Skill } from '@/types'
import type { DetailMode } from '@/types/filterSort'
import { SkillMetadataBadges } from './SkillMetadataBadges'
import { onProfilerRender } from '@/lib/profiler'
import { skillActions, syncSkillSelection } from './skill-actions'
import type { RowAction } from '@vrooli/react-component-library/useCollection/1'

interface SkillCardViewProps {
  skills: Skill[]
  selectedItemId: string | null
  onSelectItem: (id: string) => void
  dirtyItemIds: Set<string>
  detailMode: DetailMode
  healthScoreMap?: Map<string, number>
  renderItemIcon?: (skill: Skill) => ReactNode
  actions?: readonly RowAction<Skill>[]
  selectionMode?: boolean
  combineSelectedIds?: Set<string>
  onCombineToggleSkill?: (skillId: string) => void
}

export function SkillCardView(props: SkillCardViewProps) {
  const selection = props.combineSelectedIds ? [...props.combineSelectedIds] : undefined

  return <Profiler id="SkillCardView" onRender={onProfilerRender}>
    <div data-testid="skill-card-view" className="grid grid-cols-2 gap-2 p-2">
      <CollectionList
        items={props.skills}
        getKey={(skill) => skill.id}
        label="Skills"
        selection={{
          mode: props.selectionMode ? 'multi' : 'none',
          selected: selection,
          onChange: (keys) => syncSkillSelection(keys, props.combineSelectedIds, props.onCombineToggleSkill),
        }}
        onOpen={(skill) => props.onSelectItem(skill.id)}
        actions={props.actions ?? skillActions({ onOpen: (skill) => props.onSelectItem(skill.id) })}
        empty="No skills match your filters"
        renderItem={(skill, state) => {
          const selected = props.selectionMode ? state.selection.selected : skill.id === props.selectedItemId
          return <div aria-selected={selected || undefined} className={`relative flex w-full flex-col gap-1 rounded-lg border p-2 text-left ${selected ? 'border-primary/40 bg-primary/20' : 'border-border hover:bg-muted'}`} data-testid="skill-card-item" data-skill-id={skill.id} data-cursor={state.isCursor || undefined}>
            <div className="flex items-center gap-2">
              {props.renderItemIcon ? <span className="shrink-0">{props.renderItemIcon(skill)}</span> : null}
              <span className="truncate text-xs font-medium text-foreground">{skill.name}</span>
              {props.dirtyItemIds.has(skill.id) ? <span className="h-2 w-2 shrink-0 rounded-full bg-amber-500" /> : null}
            </div>
            {props.detailMode === 'full' && <>
              <p className="line-clamp-2 text-[10px] text-muted-foreground">{skill.description}</p>
              <div className="flex flex-wrap gap-0.5">{skill.tags.slice(0, 2).map((tag) => <span key={tag} className="rounded bg-muted px-1 py-0.5 text-[8px]">{tag}</span>)}{skill.tags.length > 2 && <span className="text-[8px]">+{skill.tags.length - 2}</span>}</div>
              <SkillMetadataBadges skill={skill} healthScore={props.healthScoreMap?.get(skill.id) ?? null} />
            </>}
          </div>
        }}
        className="contents"
      />
    </div>
  </Profiler>
}
