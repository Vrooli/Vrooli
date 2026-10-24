/** SkillListView — collection-backed skill list with scenario-owned content. */
import { Profiler, type ReactNode } from 'react'
import { CollectionList } from '@vrooli/react-component-library/CollectionList/1.0.0'
import { type Skill } from '@/types'
import type { DetailMode } from '@/types/filterSort'
import { SkillMetadataBadges } from './SkillMetadataBadges'
import { onProfilerRender } from '@/lib/profiler'
import { skillActions, syncSkillSelection } from './skill-actions'
import type { RowAction } from '@vrooli/react-component-library/useCollection/1'

interface SkillListViewProps {
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

export function SkillListView(props: SkillListViewProps) {
  const selection = props.combineSelectedIds ? [...props.combineSelectedIds] : undefined
  return <Profiler id="SkillListView" onRender={onProfilerRender}><CollectionList
    items={props.skills}
    getKey={(skill) => skill.id}
    label="Skills"
    selection={{ mode: props.selectionMode ? 'multi' : 'none', selected: selection, onChange: (keys) => syncSkillSelection(keys, props.combineSelectedIds, props.onCombineToggleSkill) }}
    onOpen={(skill) => props.onSelectItem(skill.id)}
    actions={props.actions ?? skillActions({ onOpen: (skill) => props.onSelectItem(skill.id) })}
    empty="No skills match your filters"
    renderItem={(skill, state) => {
      const showDetails = props.detailMode === 'full'
      const selected = props.selectionMode ? state.selection.selected : skill.id === props.selectedItemId
      return <div aria-selected={selected || undefined} className={`flex w-full flex-col gap-0.5 border-b border-border/50 px-3 text-left transition-colors ${showDetails ? 'py-2' : 'py-1.5'} ${selected ? 'bg-primary/30 text-foreground' : 'text-muted-foreground hover:bg-muted hover:text-foreground'}`} data-testid="skill-list-item" data-skill-id={skill.id} data-cursor={state.isCursor || undefined}>
        <div className="flex min-w-0 items-center gap-2">{props.renderItemIcon ? <span className="shrink-0">{props.renderItemIcon(skill)}</span> : null}<span className="flex-1 truncate text-xs font-medium">{skill.name}</span>{props.dirtyItemIds.has(skill.id) ? <span className="h-2 w-2 shrink-0 rounded-full bg-amber-500" /> : null}</div>
        {showDetails && <><p className="truncate text-[10px] text-muted-foreground">{skill.description}</p>{skill.tags.length > 0 && <div className="flex flex-wrap items-center gap-1">{skill.tags.slice(0, 3).map(tag => <span key={tag} className="rounded-full bg-muted px-1.5 py-0.5 text-[9px] text-muted-foreground">{tag}</span>)}{skill.tags.length > 3 && <span className="text-[9px] text-muted-foreground">+{skill.tags.length - 3}</span>}</div>}<SkillMetadataBadges skill={skill} healthScore={props.healthScoreMap?.get(skill.id) ?? null} /></>}
      </div>
    }}
    className="w-full"
  /></Profiler>
}
