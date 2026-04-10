/**
 * GraphJsonView - Read-only Monaco JSON viewer for the filtered graph data.
 *
 * Shows the same data visible in the ReactFlow canvas (respects type/health filters)
 * as formatted JSON. Useful for debugging graph issues.
 */

import { useMemo } from 'react'
import Editor from '@monaco-editor/react'
import { Copy, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useGraphStore, selectFilteredNodes, selectEffectiveHealthScores } from '@/stores/graphStore'
import { useShallow } from 'zustand/react/shallow'
import { useCodeCopy } from '@/components/markdown/hooks/useCodeCopy'
import { useResolvedTheme } from '@/hooks/use-theme'
import { selectors } from '@/constants/selectors'

export function GraphJsonView() {
  const resolvedTheme = useResolvedTheme()
  const graph = useGraphStore((s) => s.graph)
  const filteredNodes = useGraphStore(useShallow(selectFilteredNodes))
  const effectiveHealthScores = useGraphStore(useShallow(selectEffectiveHealthScores))

  const filteredJson = useMemo(() => {
    if (!graph) return '{}'

    const nodeIds = new Set(filteredNodes.map((n) => n.id))
    const filteredEdges = graph.graph.edges.filter(
      (e) => nodeIds.has(e.from) && nodeIds.has(e.to),
    )
    const filteredHealthScores = effectiveHealthScores.filter(
      (hs) => nodeIds.has(hs.nodeId),
    )

    return JSON.stringify(
      {
        generatedAt: graph.generatedAt,
        graph: {
          nodes: filteredNodes,
          edges: filteredEdges,
          healthScores: filteredHealthScores,
        },
      },
      null,
      2,
    )
  }, [graph, filteredNodes, effectiveHealthScores])

  const { copied, copyCode } = useCodeCopy(filteredJson)

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="flex-shrink-0 flex items-center gap-2 px-3 py-1.5 bg-card border-b border-border">
        <span className="text-xs text-muted-foreground font-medium">graph.json</span>
        <span className="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
          Read-only
        </span>
        <button
          type="button"
          data-testid={selectors.graph.jsonCopyButton}
          onClick={copyCode}
          className={cn(
            'ml-auto flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-md',
            'bg-muted text-foreground border border-border',
            'hover:bg-muted/80 transition-colors',
          )}
          title="Copy JSON to clipboard"
        >
          {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
          {copied ? 'Copied' : 'Copy'}
        </button>
      </div>

      {/* Editor */}
      <div className="flex-1">
        <Editor
          height="100%"
          defaultLanguage="json"
          value={filteredJson}
          theme={resolvedTheme === 'dark' ? 'vs-dark' : 'vs-light'}
          options={{
            readOnly: true,
            minimap: { enabled: true },
            wordWrap: 'off',
            lineNumbers: 'on',
            fontSize: 13,
            fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
            tabSize: 2,
            scrollBeyondLastLine: false,
            padding: { top: 12, bottom: 12 },
            renderLineHighlight: 'line',
            smoothScrolling: true,
            scrollbar: {
              vertical: 'auto',
              horizontal: 'auto',
              verticalScrollbarSize: 8,
              horizontalScrollbarSize: 8,
            },
            overviewRulerBorder: false,
            hideCursorInOverviewRuler: true,
            folding: true,
            automaticLayout: true,
          }}
        />
      </div>
    </div>
  )
}
