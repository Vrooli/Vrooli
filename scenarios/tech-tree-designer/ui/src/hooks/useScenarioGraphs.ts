import { useMemo } from 'react'
import type {
  ScenarioCatalogSnapshot,
  ScenarioDependencyEdge,
  ScenarioCatalogEntry,
  Sector
} from '../types/techTree'
import type { DesignerEdge, DesignerNode, ScenarioEntryMap, StageNode } from '../types/graph'
import { EDGE_LABEL_STYLE } from '../utils/constants'

interface ScenarioGraphParams {
  scenarioOnlyMode: boolean
  showLiveScenarios: boolean
  scenarioCatalog: ScenarioCatalogSnapshot
  scenarioEntryMap: ScenarioEntryMap
  showHiddenScenarios: boolean
  showIsolatedScenarios: boolean // Show scenarios with no dependencies
  nodes: StageNode[]
  sectors: Sector[]
}

interface ScenarioGraphs {
  overlayNodes: DesignerNode[]
  overlayEdges: DesignerEdge[]
  scenarioOnlyNodes: DesignerNode[]
  scenarioOnlyEdges: DesignerEdge[]
}

const createScenarioDependencyEdges = (
  dependencyEdges: ScenarioDependencyEdge[],
  visibleNames: Set<string>,
  prefix: string
): DesignerEdge[] =>
  dependencyEdges
    .filter((edge) => visibleNames.has(edge.from) && visibleNames.has(edge.to))
    .map((edge) => ({
      id: `${prefix}::${edge.from}::${edge.to}`,
      source: `scenario::${edge.from}`,
      target: `scenario::${edge.to}`,
      animated: prefix === 'scenario-only-edge',
      style: {
        stroke: prefix === 'scenario-only-edge' ? 'rgba(59,130,246,0.6)' : 'rgba(94,234,212,0.75)',
        strokeWidth: 1.5,
        strokeDasharray: prefix === 'scenario-only-edge' ? undefined : '4 2'
      },
      label: edge.type || 'requires',
      labelStyle: EDGE_LABEL_STYLE,
      labelBgPadding: [4, 2]
    })) as DesignerEdge[]

export const useScenarioGraphs = ({
  scenarioOnlyMode,
  showLiveScenarios,
  scenarioCatalog,
  scenarioEntryMap,
  showHiddenScenarios,
  showIsolatedScenarios,
  nodes,
  sectors
}: ScenarioGraphParams): ScenarioGraphs => {
  const overlayGraphs = useMemo(() => {
    if (scenarioOnlyMode || !showLiveScenarios || !scenarioCatalog.scenarios || !scenarioCatalog.scenarios.length) {
      return { overlayNodes: [], overlayEdges: [] }
    }

    const overlayNodes: DesignerNode[] = []
    const overlayEdges: DesignerEdge[] = []
    const placementUsage = new Map<string, number>()
    const stagePositionMap = new Map<string, { x: number; y: number }>()
    const visibleScenarioNames = new Set<string>()
    const renderedScenarioNames = new Set<string>()

    nodes.forEach((node) => {
      stagePositionMap.set(node.id, node.position || { x: 0, y: 0 })
    })

    sectors.forEach((sector) => {
      ;(sector.stages || []).forEach((stage) => {
        const stageMappings = stage.scenario_mappings || []
        if (!stageMappings.length) {
          return
        }
        const stagePosition = stagePositionMap.get(stage.id)
        if (!stagePosition) {
          return
        }

        stageMappings.forEach((mapping) => {
          const scenarioName = mapping.scenario_name
          if (!scenarioName) {
            return
          }
          const entry = scenarioEntryMap.get(scenarioName) || scenarioEntryMap.get(scenarioName.toLowerCase())
          const isHidden = entry?.hidden && !showHiddenScenarios
          if (isHidden) {
            return
          }

          if (!renderedScenarioNames.has(scenarioName)) {
            const count = placementUsage.get(stage.id) || 0
            placementUsage.set(stage.id, count + 1)

            const offsetX = 48 + (count % 3) * 42
            const offsetY = 64 + Math.floor(count / 3) * 52
            const nodeId = `scenario::${scenarioName}`

            overlayNodes.push({
              id: nodeId,
              position: {
                x: stagePosition.x + offsetX,
                y: stagePosition.y + offsetY
              },
              data: {
                label: entry?.display_name || scenarioName,
                scenarioName,
                description: entry?.description || '',
                tags: entry?.tags || [],
                hidden: entry?.hidden || false
              },
              draggable: false,
              type: 'default',
              style: {
                background: entry?.hidden ? 'rgba(148,163,184,0.15)' : 'rgba(14,165,233,0.12)',
                border: `1px solid ${entry?.hidden ? 'rgba(148,163,184,0.45)' : 'rgba(14,165,233,0.8)'}`,
                borderRadius: 12,
                padding: 10,
                fontSize: 11,
                color: '#f8fafc',
                minWidth: 170,
                boxShadow: '0 8px 20px rgba(8, 47, 73, 0.35)'
              }
            })

            renderedScenarioNames.add(scenarioName)
          }

          overlayEdges.push({
            id: `stage-link::${stage.id}::${scenarioName}`,
            source: stage.id,
            target: `scenario::${scenarioName}`,
            animated: true,
            style: {
              strokeDasharray: '6 4',
              stroke: 'rgba(14,165,233,0.8)'
            },
            label: 'scenario'
          } as DesignerEdge)

          visibleScenarioNames.add(scenarioName)
        })
      })
    })

    const dependencyEdges = createScenarioDependencyEdges(
      scenarioCatalog.edges || [],
      visibleScenarioNames,
      'scenario-dependency'
    )

    return { overlayNodes, overlayEdges: [...overlayEdges, ...dependencyEdges] }
  }, [
    nodes,
    scenarioCatalog.edges,
    scenarioCatalog.scenarios,
    scenarioEntryMap,
    scenarioOnlyMode,
    sectors,
    showHiddenScenarios,
    showLiveScenarios
  ])

  const scenarioOnlyGraphs = useMemo(() => {
    if (!scenarioOnlyMode || !scenarioCatalog.scenarios) {
      return { scenarioOnlyNodes: [], scenarioOnlyEdges: [] }
    }

    // Build set of scenarios that have at least one dependency
    const connectedScenarios = new Set<string>()
    if (!showIsolatedScenarios) {
      ;(scenarioCatalog.edges || []).forEach((edge) => {
        connectedScenarios.add(edge.from)
        connectedScenarios.add(edge.to)
      })
    }

    const visibleEntries = scenarioCatalog.scenarios.filter((entry) => {
      // Filter by hidden status
      if (!showHiddenScenarios && entry.hidden) {
        return false
      }
      // Filter by connectivity if showIsolatedScenarios is false
      if (!showIsolatedScenarios && !connectedScenarios.has(entry.name)) {
        return false
      }
      return true
    })
    const columns = Math.max(1, Math.ceil(Math.sqrt(Math.max(visibleEntries.length, 1))))
    const spacingX = 260
    const spacingY = 170
    const scenarioOnlyNodes: DesignerNode[] = visibleEntries.map((entry, index) => {
      const column = index % columns
      const row = Math.floor(index / columns)
      return {
        id: `scenario::${entry.name}`,
        position: {
          x: column * spacingX,
          y: row * spacingY
        },
        data: {
          label: entry.display_name || entry.name,
          scenarioName: entry.name,
          description: entry.description || '',
          tags: entry.tags || [],
          hidden: entry.hidden || false
        },
        draggable: false,
        type: 'default',
        style: {
          background: entry.hidden ? 'rgba(148,163,184,0.15)' : 'rgba(59,130,246,0.12)',
          border: `1px solid ${entry.hidden ? 'rgba(148,163,184,0.4)' : 'rgba(59,130,246,0.75)'}`,
          borderRadius: 12,
          padding: 12,
          fontSize: 12,
          color: '#f8fafc',
          minWidth: 200
        }
      }
    })

    const visibleNames = new Set(visibleEntries.map((entry) => entry.name))
    const scenarioOnlyEdges = createScenarioDependencyEdges(
      scenarioCatalog.edges || [],
      visibleNames,
      'scenario-only-edge'
    )

    return { scenarioOnlyNodes, scenarioOnlyEdges }
  }, [scenarioCatalog.edges, scenarioCatalog.scenarios, scenarioOnlyMode, showHiddenScenarios, showIsolatedScenarios])

  return {
    overlayNodes: overlayGraphs.overlayNodes,
    overlayEdges: overlayGraphs.overlayEdges,
    scenarioOnlyNodes: scenarioOnlyGraphs.scenarioOnlyNodes,
    scenarioOnlyEdges: scenarioOnlyGraphs.scenarioOnlyEdges
  }
}
