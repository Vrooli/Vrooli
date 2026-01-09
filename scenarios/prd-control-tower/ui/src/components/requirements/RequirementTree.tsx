import { ChevronDown, ChevronRight, FileText, FolderOpen, Folder, Package } from 'lucide-react'
import { useState } from 'react'
import type { RequirementGroup, RequirementRecord } from '../../types'
import { Badge } from '../ui/badge'
import { cn } from '../../lib/utils'
import { selectors } from '../../consts/selectors'

interface RequirementTreeProps {
  groups: RequirementGroup[]
  selectedId: string | null
  onSelect: (req: RequirementRecord) => void
}

interface RequirementGroupNodeProps {
  group: RequirementGroup
  level: number
  selectedId: string | null
  onSelect: (req: RequirementRecord) => void
}

function RequirementGroupNode({ group, level, selectedId, onSelect }: RequirementGroupNodeProps) {
  const [expanded, setExpanded] = useState(true)
  const hasChildren = (group.children?.length ?? 0) > 0
  const hasRequirements = group.requirements.length > 0

  // Determine icon based on whether this is a module file or folder
  const getIcon = () => {
    if (group.is_module) {
      // Module files get a package icon
      return <Package size={16} className={expanded ? 'text-purple-500' : 'text-slate-400'} />
    }
    // Folders get folder icons
    return expanded ? <FolderOpen size={16} className="text-amber-500" /> : <Folder size={16} className="text-slate-400" />
  }

  return (
    <div className="select-none">
      <div
        className={cn(
          'flex items-center gap-2 rounded-lg px-3 py-2 hover:bg-slate-100 cursor-pointer transition-colors',
          level > 0 && 'ml-4',
        )}
        onClick={() => setExpanded(!expanded)}
      >
        {hasChildren || hasRequirements ? (
          expanded ? (
            <ChevronDown size={16} className="text-slate-400" />
          ) : (
            <ChevronRight size={16} className="text-slate-400" />
          )
        ) : (
          <div className="w-4" />
        )}
        {getIcon()}
        <span className="font-medium text-sm text-slate-700">{group.name}</span>
        {group.description && <span className="text-xs text-muted-foreground truncate">{group.description}</span>}
      </div>

      {expanded && (
        <div className="ml-4 border-l border-slate-200">
          {group.requirements.map((req) => (
            <div
              key={req.id}
              data-testid={selectors.requirements.requirementCard}
              className={cn(
                'flex items-center gap-2 rounded-lg px-3 py-2 ml-4 cursor-pointer transition-colors',
                selectedId === req.id ? 'bg-blue-50 border-l-2 border-blue-500' : 'hover:bg-slate-50',
              )}
              onClick={() => onSelect(req)}
            >
              <FileText size={14} className={selectedId === req.id ? 'text-blue-600' : 'text-slate-400'} />
              <span className={cn('text-sm flex-1', selectedId === req.id ? 'font-medium text-blue-900' : 'text-slate-700')}>
                {req.title}
              </span>
              {req.status === 'complete' && (
                <Badge variant="success" className="text-xs">
                  ✓
                </Badge>
              )}
              {req.criticality && (
                <Badge variant={req.criticality === 'P0' ? 'destructive' : req.criticality === 'P1' ? 'warning' : 'secondary'} className="text-xs">
                  {req.criticality}
                </Badge>
              )}
            </div>
          ))}

          {group.children?.map((child) => (
            <RequirementGroupNode key={child.id} group={child} level={level + 1} selectedId={selectedId} onSelect={onSelect} />
          ))}
        </div>
      )}
    </div>
  )
}

export function RequirementTree({ groups, selectedId, onSelect }: RequirementTreeProps) {
  if (groups.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center p-12 text-center text-muted-foreground">
        <Folder size={48} className="mb-4 text-slate-300" />
        <p>No requirements found</p>
      </div>
    )
  }

  return (
    <div className="space-y-1 overflow-y-auto max-h-[calc(100vh-200px)]">
      {groups.map((group) => (
        <RequirementGroupNode key={group.id} group={group} level={0} selectedId={selectedId} onSelect={onSelect} />
      ))}
    </div>
  )
}
