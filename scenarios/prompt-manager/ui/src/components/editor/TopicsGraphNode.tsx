/**
 * TopicsGraphNode - Custom React Flow node for the topics-mode team graph.
 *
 * Renders members with their team/role identity, plus boundary nodes
 * (external producers, decision queues, PoR sinks, capability-gap registry,
 * skill proposals, backlog) using kind-specific styling.
 *
 * DOC: docs/agent-system/drafts/topics-schema.md
 */

import { memo } from 'react'
import { Handle, Position, type NodeProps, type Node } from '@xyflow/react'
import {
  AlertCircle,
  AlertTriangle,
  FileText,
  Globe,
  Inbox,
  ListTodo,
  Sparkles,
  User,
  Vote,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TopicsFlowNodeData, TopicNodeKind } from '@/types/topicsGraph'

type TopicsGraphNodeProps = NodeProps<Node<TopicsFlowNodeData, 'topicsNode'>>

const KIND_ICON: Record<TopicNodeKind, typeof User> = {
  member: User,
  external: Globe,
  decision: Vote,
  por_file: FileText,
  capability_gap: AlertTriangle,
  skill_proposal: Sparkles,
  backlog: ListTodo,
  knowledge_sink: Inbox,
}

const KIND_LABEL: Record<TopicNodeKind, string> = {
  member: 'Member',
  external: 'External',
  decision: 'Decision',
  por_file: 'PoR',
  capability_gap: 'Capability gap',
  skill_proposal: 'Skill proposal',
  backlog: 'Backlog',
  knowledge_sink: 'Topic',
}

const KIND_STYLES: Record<TopicNodeKind, string> = {
  member: 'bg-card border-border',
  external: 'bg-amber-500/10 border-amber-500/40',
  decision: 'bg-purple-500/10 border-purple-500/40',
  por_file: 'bg-blue-500/10 border-blue-500/40',
  capability_gap: 'bg-rose-500/10 border-rose-500/40',
  skill_proposal: 'bg-emerald-500/10 border-emerald-500/40',
  backlog: 'bg-slate-500/10 border-slate-500/40',
  knowledge_sink: 'bg-cyan-500/10 border-cyan-500/40',
}

function nodeLabel(label: string | undefined, fallbackId: string): string {
  if (label && label.length > 0) return label
  return fallbackId
}

function TopicsGraphNodeComponent({ data }: TopicsGraphNodeProps) {
  const { graphNode, errorCount, warningCount, isSelected, onSelect } = data
  const Icon = KIND_ICON[graphNode.kind]
  const label = nodeLabel(graphNode.label, graphNode.id)

  // Trim long PoR paths for readability — keep last segment + parent.
  const displayLabel =
    graphNode.kind === 'por_file' && label.includes('/')
      ? label.split('/').slice(-2).join('/')
      : label

  const subtitle =
    graphNode.kind === 'member'
      ? graphNode.ref?.team
      : KIND_LABEL[graphNode.kind]

  return (
    <button
      type="button"
      onClick={() => onSelect?.(graphNode.id)}
      className={cn(
        'relative px-3 py-2 border rounded-lg text-left transition-all',
        'min-w-[160px] max-w-[220px]',
        'hover:bg-muted/50 hover:border-primary/50',
        KIND_STYLES[graphNode.kind],
        isSelected && 'ring-2 ring-primary/60',
        errorCount > 0 && 'ring-2 ring-rose-500/70'
      )}
      data-node-id={graphNode.id}
      data-node-kind={graphNode.kind}
    >
      <Handle
        type="target"
        position={Position.Left}
        className="!w-2 !h-2 !bg-primary/50 !border-2 !border-background"
      />

      <div className="flex items-start gap-2">
        <Icon className="h-4 w-4 mt-0.5 flex-shrink-0 text-muted-foreground" />
        <div className="min-w-0 flex-1">
          <p
            className="text-sm font-medium truncate"
            title={label}
          >
            {displayLabel}
          </p>
          {subtitle && (
            <p className="text-[10px] text-muted-foreground/80 truncate">
              {subtitle}
            </p>
          )}
        </div>
        {errorCount + warningCount > 0 && (
          <span
            className={cn(
              'flex items-center gap-0.5 text-[10px] font-mono px-1 py-0.5 rounded',
              errorCount > 0
                ? 'bg-rose-500/20 text-rose-300'
                : 'bg-amber-500/20 text-amber-300'
            )}
            title={`${errorCount} error(s), ${warningCount} warning(s)`}
          >
            <AlertCircle className="h-3 w-3" />
            {errorCount > 0 ? errorCount : warningCount}
          </span>
        )}
      </div>

      <Handle
        type="source"
        position={Position.Right}
        className="!w-2 !h-2 !bg-primary/50 !border-2 !border-background"
      />
    </button>
  )
}

export const TopicsGraphNode = memo(TopicsGraphNodeComponent)
