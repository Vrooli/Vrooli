/**
 * GraphSettingsContent - Settings body for graph visualization controls.
 *
 * Rendered inside shared floating panel by ViewOverlay. Contains no panel chrome.
 */

import { useEffect, useMemo, useState } from 'react'
import {
  Users,
  Bot,
  Sparkles,
  PlaySquare,
  Terminal,
  LayoutGrid,
  RefreshCw,
  Maximize2,
  Link2Off,
  FoldVertical,
  SlidersHorizontal,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useGraphStore } from '@/stores/graphStore'
import { useGraphHealthConfigStore } from '@/stores/graphHealthConfigStore'
import { buildPreviewHealthScores } from '@/services/graphHealthPreview'

const ENTITY_META = [
  { key: 'team' as const, label: 'Team', icon: Users },
  { key: 'agent' as const, label: 'Agent', icon: Bot },
  { key: 'skill' as const, label: 'Skill', icon: Sparkles },
  { key: 'action' as const, label: 'Action', icon: PlaySquare },
]

const WEIGHT_FIELDS = [
  { key: 'outgoingEdges' as const, label: 'Outgoing edges' },
  { key: 'incomingEdges' as const, label: 'Incoming edges' },
  { key: 'codeUsage' as const, label: 'Code usage' },
  { key: 'recentActivity' as const, label: 'Recent activity' },
]

const ENTITY_WEIGHT_FIELDS = {
  team: [
    ...WEIGHT_FIELDS,
    { key: 'teamMemberCountBalance' as const, label: 'Member count balance' },
    { key: 'teamRoleCoverage' as const, label: 'Role coverage' },
  ],
  agent: [
    ...WEIGHT_FIELDS,
    { key: 'agentContextLoad' as const, label: 'Context load' },
  ],
  skill: [
    ...WEIGHT_FIELDS,
    { key: 'skillContentLength' as const, label: 'Content length' },
  ],
  action: [
    ...WEIGHT_FIELDS,
    { key: 'actionContract' as const, label: 'Contract' },
    { key: 'actionCommand' as const, label: 'Command' },
    { key: 'actionExamples' as const, label: 'Examples' },
    { key: 'actionOwner' as const, label: 'Owner' },
  ],
}

function GraphDisplaySettings() {
  const filters = useGraphStore((s) => s.filters)
  const setFilter = useGraphStore((s) => s.setFilter)
  const layoutDirection = useGraphStore((s) => s.layoutDirection)
  const setLayoutDirection = useGraphStore((s) => s.setLayoutDirection)
  const layoutMode = useGraphStore((s) => s.layoutMode)
  const setLayoutMode = useGraphStore((s) => s.setLayoutMode)
  const regenerateGraph = useGraphStore((s) => s.regenerateGraph)
  const requestFitView = useGraphStore((s) => s.requestFitView)
  const loading = useGraphStore((s) => s.loading)

  const toggleButtons = [
    { key: 'showTeams' as const, label: 'Teams', icon: Users, active: filters.showTeams },
    { key: 'showAgents' as const, label: 'Agents', icon: Bot, active: filters.showAgents },
    { key: 'showSkills' as const, label: 'Skills', icon: Sparkles, active: filters.showSkills },
    { key: 'showActions' as const, label: 'Actions', icon: PlaySquare, active: filters.showActions ?? true },
    { key: 'showCLIs' as const, label: 'CLIs', icon: Terminal, active: filters.showCLIs },
  ]

  return (
    <div className="space-y-6">
      <section>
        <h3 className="text-sm font-medium text-primary mb-3">Visibility</h3>
        <div className="grid grid-cols-2 gap-2 mb-4">
          {toggleButtons.map((btn) => {
            const Icon = btn.icon
            return (
              <button
                key={btn.key}
                type="button"
                onClick={() => setFilter(btn.key, !btn.active)}
                className={cn(
                  'flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-lg border transition-colors',
                  btn.active
                    ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-700 dark:text-indigo-300'
                    : 'bg-muted border-border text-muted-foreground hover:text-foreground hover:border-border/80',
                )}
                title={`${btn.active ? 'Hide' : 'Show'} ${btn.label}`}
              >
                <Icon className="h-4 w-4" />
                {btn.label}
              </button>
            )
          })}
        </div>

        <div className="flex items-center gap-3">
          <span className="text-xs text-muted-foreground whitespace-nowrap">
            Health &ge; {Math.round(filters.healthThreshold * 100)}%
          </span>
          <input
            type="range"
            min={0}
            max={1}
            step={0.05}
            value={filters.healthThreshold}
            onChange={(e) => setFilter('healthThreshold', parseFloat(e.target.value))}
            className="flex-1 h-1 accent-indigo-500"
          />
        </div>

        <div className="grid grid-cols-1 gap-2 mt-3">
          <button
            type="button"
            onClick={() => setFilter('collapseCLIs', !filters.collapseCLIs)}
            className={cn(
              'flex items-center justify-between gap-2 px-3 py-2 text-xs font-medium rounded-lg border transition-colors',
              filters.collapseCLIs
                ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-700 dark:text-indigo-300'
                : 'bg-muted border-border text-muted-foreground hover:text-foreground hover:border-border/80',
            )}
            title={filters.collapseCLIs ? 'Expand individual CLI nodes' : 'Collapse CLI nodes into one cluster'}
          >
            <span className="flex items-center gap-2">
              <FoldVertical className="h-3.5 w-3.5" />
              Collapse CLI Nodes
            </span>
          </button>

          <button
            type="button"
            onClick={() => setFilter('showLowSignalEdges', !filters.showLowSignalEdges)}
            className={cn(
              'flex items-center justify-between gap-2 px-3 py-2 text-xs font-medium rounded-lg border transition-colors',
              filters.showLowSignalEdges
                ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-700 dark:text-indigo-300'
                : 'bg-muted border-border text-muted-foreground hover:text-foreground hover:border-border/80',
            )}
            title={filters.showLowSignalEdges ? 'Hide low-signal edges' : 'Show low-signal edges'}
          >
            <span className="flex items-center gap-2">
              <Link2Off className="h-3.5 w-3.5" />
              Low-Signal Edges
            </span>
          </button>

          <button
            type="button"
            onClick={() => setFilter('autoFitOnChange', !filters.autoFitOnChange)}
            className={cn(
              'flex items-center justify-between gap-2 px-3 py-2 text-xs font-medium rounded-lg border transition-colors',
              filters.autoFitOnChange
                ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-700 dark:text-indigo-300'
                : 'bg-muted border-border text-muted-foreground hover:text-foreground hover:border-border/80',
            )}
            title={filters.autoFitOnChange ? 'Disable auto-fit after changes' : 'Enable auto-fit after changes'}
          >
            Auto-fit after changes
          </button>
        </div>
      </section>

      <div className="border-t border-border" />

      <section>
        <h3 className="text-sm font-medium text-primary mb-3">Layout</h3>
        <div className="grid grid-cols-3 gap-2 mb-3">
          {[
            { id: 'hierarchical' as const, label: 'Hier' },
            { id: 'compact' as const, label: 'Compact' },
            { id: 'grouped' as const, label: 'Grouped' },
          ].map((option) => (
            <button
              key={option.id}
              type="button"
              onClick={() => setLayoutMode(option.id)}
              className={cn(
                'px-2 py-1.5 text-xs font-medium rounded-lg border transition-colors',
                layoutMode === option.id
                  ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-700 dark:text-indigo-300'
                  : 'bg-muted border-border text-muted-foreground hover:text-foreground hover:border-border/80',
              )}
            >
              {option.label}
            </button>
          ))}
        </div>

        <div className="flex gap-2 mb-3">
          <button
            type="button"
            onClick={() => setLayoutDirection('LR')}
            className={cn(
              'flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm font-medium rounded-lg border transition-all',
              layoutDirection === 'LR'
                ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-700 dark:text-indigo-300'
                : 'bg-muted border-border text-muted-foreground hover:text-foreground hover:border-border/80',
            )}
          >
            <LayoutGrid className="h-4 w-4" />
            Vertical
          </button>
          <button
            type="button"
            onClick={() => setLayoutDirection('TB')}
            className={cn(
              'flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm font-medium rounded-lg border transition-all',
              layoutDirection === 'TB'
                ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-700 dark:text-indigo-300'
                : 'bg-muted border-border text-muted-foreground hover:text-foreground hover:border-border/80',
            )}
          >
            <LayoutGrid className="h-4 w-4" />
            Horizontal
          </button>
        </div>

        <div className="flex gap-2">
          <button
            type="button"
            onClick={requestFitView}
            className={cn(
              'flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm font-medium rounded-lg',
              'bg-muted border border-border text-foreground hover:bg-muted/80 transition-colors',
            )}
            title="Fit to view"
          >
            <Maximize2 className="h-4 w-4" />
            Fit to View
          </button>
          <button
            type="button"
            onClick={() => void regenerateGraph()}
            disabled={loading}
            className={cn(
              'flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm font-medium rounded-lg',
              'bg-muted border border-border text-foreground hover:bg-muted/80 transition-colors',
              loading && 'opacity-50 cursor-not-allowed',
            )}
            title="Regenerate graph"
          >
            <RefreshCw className={cn('h-4 w-4', loading && 'animate-spin')} />
            Regenerate
          </button>
        </div>
      </section>
    </div>
  )
}

function GraphHealthSettings() {
  const loadConfig = useGraphHealthConfigStore((s) => s.loadConfig)
  const config = useGraphHealthConfigStore((s) => s.config)
  const dirty = useGraphHealthConfigStore((s) => s.dirty)
  const loading = useGraphHealthConfigStore((s) => s.loading)
  const saving = useGraphHealthConfigStore((s) => s.saving)
  const error = useGraphHealthConfigStore((s) => s.error)
  const setEntityWeight = useGraphHealthConfigStore((s) => s.setEntityWeight)
  const setCLIField = useGraphHealthConfigStore((s) => s.setCLIField)
  const setNeutralCommandsText = useGraphHealthConfigStore((s) => s.setNeutralCommandsText)
  const saveConfig = useGraphHealthConfigStore((s) => s.saveConfig)
  const resetToDefault = useGraphHealthConfigStore((s) => s.resetToDefault)
  const fetchGraph = useGraphStore((s) => s.fetchGraph)
  const graph = useGraphStore((s) => s.graph)
  const setHealthScoreOverride = useGraphStore((s) => s.setHealthScoreOverride)
  const clearHealthScoreOverride = useGraphStore((s) => s.clearHealthScoreOverride)

  useEffect(() => {
    void loadConfig()
  }, [loadConfig])

  const neutralCommandText = useMemo(() => config.cli.neutralCommands.join(', '), [config.cli.neutralCommands])

  useEffect(() => {
    if (!graph || !dirty) {
      clearHealthScoreOverride()
      return
    }
    const preview = buildPreviewHealthScores(graph.graph, config, graph.graph.healthScores)
    setHealthScoreOverride(preview)
  }, [graph, config, dirty, setHealthScoreOverride, clearHealthScoreOverride])

  const onSave = async () => {
    const ok = await saveConfig()
    if (ok) {
      clearHealthScoreOverride()
      await fetchGraph(true)
    }
  }

  return (
    <div className="space-y-4">
      <section className="space-y-2">
        <h3 className="text-sm font-medium text-primary">Entity Scoring Weights</h3>
        <p className="text-xs text-muted-foreground">
          Tune weighted health factors per entity type. Changes are stored in
          <code className="ml-1">store/config/graph-health.json</code>.
        </p>
        <p className={cn(
          'text-xs',
          dirty ? 'text-amber-600 dark:text-amber-300' : 'text-muted-foreground',
        )}>
          {dirty ? 'Unsaved preview active (graph is using draft health config)' : 'No unsaved health changes'}
        </p>
      </section>

      {ENTITY_META.map((entity) => {
        const Icon = entity.icon
        const weights = config[entity.key] ?? config.skill
        return (
          <section key={entity.key} className="rounded-lg border border-border bg-card/40 p-3 space-y-2">
            <div className="flex items-center gap-2 text-foreground">
              <Icon className="h-4 w-4" />
              <span className="text-sm font-medium">{entity.label}</span>
            </div>
            {ENTITY_WEIGHT_FIELDS[entity.key].map((field) => (
              <label key={field.key} className="grid grid-cols-[1fr_auto] items-center gap-2 text-xs text-muted-foreground">
                <span>{field.label}</span>
                <span className="tabular-nums text-foreground">{Math.round((weights[field.key] ?? 0) * 100)}%</span>
                <input
                  type="range"
                  min={0}
                  max={1}
                  step={0.05}
                  value={weights[field.key] ?? 0}
                  onChange={(e) => setEntityWeight(entity.key, field.key, parseFloat(e.target.value))}
                  className="col-span-2 h-1 accent-indigo-500"
                />
              </label>
            ))}
          </section>
        )
      })}

      <section className="rounded-lg border border-border bg-card/40 p-3 space-y-2">
        <div className="flex items-center gap-2 text-foreground">
          <Terminal className="h-4 w-4" />
          <span className="text-sm font-medium">CLI Policy</span>
        </div>

        <label className="block text-xs text-muted-foreground">
          Neutral Commands (comma-separated)
          <input
            type="text"
            value={neutralCommandText}
            onChange={(e) => setNeutralCommandsText(e.target.value)}
            className="mt-1 w-full rounded border border-border bg-card px-2 py-1"
          />
        </label>

        <label className="grid grid-cols-[1fr_auto] items-center gap-2 text-xs text-muted-foreground">
          <span>External tool score</span>
          <span className="tabular-nums text-foreground">{Math.round(config.cli.externalToolScore * 100)}%</span>
          <input
            type="range"
            min={0}
            max={1}
            step={0.05}
            value={config.cli.externalToolScore}
            onChange={(e) => setCLIField('externalToolScore', parseFloat(e.target.value))}
            className="col-span-2 h-1 accent-indigo-500"
          />
        </label>

        <label className="grid grid-cols-[1fr_auto] items-center gap-2 text-xs text-muted-foreground">
          <span>Scenario fallback score</span>
          <span className="tabular-nums text-foreground">{Math.round(config.cli.scenarioFallbackScore * 100)}%</span>
          <input
            type="range"
            min={0}
            max={1}
            step={0.05}
            value={config.cli.scenarioFallbackScore}
            onChange={(e) => setCLIField('scenarioFallbackScore', parseFloat(e.target.value))}
            className="col-span-2 h-1 accent-indigo-500"
          />
        </label>
      </section>

      {error && (
        <p className="text-xs text-red-600 dark:text-red-300">{error}</p>
      )}

      <div className="flex gap-2">
        <button
          type="button"
          onClick={resetToDefault}
          disabled={saving || loading}
          className={cn(
            'flex-1 px-3 py-2 text-xs font-medium rounded-lg border border-border bg-muted text-foreground',
            (saving || loading) && 'opacity-60 cursor-not-allowed',
          )}
        >
          Reset Defaults
        </button>
        <button
          type="button"
          onClick={() => void onSave()}
          disabled={saving || loading}
          className={cn(
            'flex-1 px-3 py-2 text-xs font-medium rounded-lg border border-indigo-500/50 bg-indigo-500/30 text-indigo-700 dark:text-indigo-100',
            (saving || loading) && 'opacity-60 cursor-not-allowed',
          )}
        >
          {saving ? 'Saving...' : 'Save + Recompute'}
        </button>
      </div>
    </div>
  )
}

export function GraphSettingsContent() {
  const [tab, setTab] = useState<'display' | 'health'>('display')

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-2">
        <button
          type="button"
          onClick={() => setTab('display')}
          className={cn(
            'px-3 py-2 text-xs font-medium rounded-lg border transition-colors',
            tab === 'display'
              ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-700 dark:text-indigo-300'
              : 'bg-muted border-border text-muted-foreground hover:text-foreground hover:border-border/80',
          )}
        >
          Display
        </button>
        <button
          type="button"
          onClick={() => setTab('health')}
          className={cn(
            'px-3 py-2 text-xs font-medium rounded-lg border transition-colors flex items-center justify-center gap-2',
            tab === 'health'
              ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-700 dark:text-indigo-300'
              : 'bg-muted border-border text-muted-foreground hover:text-foreground hover:border-border/80',
          )}
        >
          <SlidersHorizontal className="h-3.5 w-3.5" />
          Health
        </button>
      </div>

      {tab === 'display' ? <GraphDisplaySettings /> : <GraphHealthSettings />}
    </div>
  )
}
