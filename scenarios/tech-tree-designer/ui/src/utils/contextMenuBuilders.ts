import type { ContextMenuSection } from '../components/graph/ContextMenu'
import type { ProgressionStage } from '../types/techTree'
import type { DesignerNodeData, ScenarioNodeData, StageNodeData } from '../types/graph'

export interface MenuBuilderContext {
  isEditMode: boolean
  scenarioOnlyMode: boolean
  stageLookup?: Map<string, ProgressionStage & { sector: any }>
}

export interface MenuCallbacks {
  onAddChildStage?: (parentStageId: string, parentSectorId: string) => void
  onGenerateAISuggestions?: (stageId: string, sectorId: string) => void
  onLinkScenario?: (stage: ProgressionStage) => void
  onEditStage?: (stageId: string) => void
  onDeleteStage?: (stageId: string) => void
  onToggleScenarioVisibility?: (scenarioName: string, hidden: boolean) => void
  onUnlinkScenario?: (scenarioName: string) => void
  onExpandSector?: (sectorId: string) => void
  onAddStageToSector?: (sectorId: string) => void
  onAddSector?: () => void
}

/**
 * Builds context menu for empty pane (background click).
 */
export function buildPaneMenu(
  context: MenuBuilderContext,
  callbacks: MenuCallbacks
): ContextMenuSection[] {
  return [
    {
      items: [
        {
          id: 'add-sector',
          label: 'Add New Sector',
          icon: 'Plus',
          onClick: () => callbacks.onAddSector?.(),
          disabled: !callbacks.onAddSector || context.scenarioOnlyMode
        }
      ]
    }
  ]
}

/**
 * Builds context menu for collapsed sector group nodes.
 */
export function buildSectorGroupMenu(
  sectorId: string,
  context: MenuBuilderContext,
  callbacks: MenuCallbacks
): ContextMenuSection[] {
  return [
    {
      items: [
        {
          id: 'expand-sector',
          label: 'Expand Sector',
          icon: 'ChevronDown',
          onClick: () => callbacks.onExpandSector?.(sectorId),
          disabled: !callbacks.onExpandSector
        },
        {
          id: 'add-stage',
          label: 'Add Stage to Sector',
          icon: 'Plus',
          onClick: () => callbacks.onAddStageToSector?.(sectorId),
          disabled: !callbacks.onAddStageToSector || !context.isEditMode
        },
        {
          id: 'generate-ai-sector',
          label: 'Generate AI Suggestions',
          icon: 'Sparkles',
          onClick: () => callbacks.onGenerateAISuggestions?.(sectorId, sectorId),
          disabled: !callbacks.onGenerateAISuggestions || !context.isEditMode
        }
      ]
    }
  ]
}

/**
 * Builds context menu for scenario nodes.
 */
export function buildScenarioMenu(
  nodeData: ScenarioNodeData,
  context: MenuBuilderContext,
  callbacks: MenuCallbacks
): ContextMenuSection[] {
  const scenarioName = nodeData?.scenarioName || ''
  const isHidden = nodeData?.hidden || false

  return [
    {
      items: [
        {
          id: 'toggle-visibility',
          label: isHidden ? 'Show Scenario' : 'Hide Scenario',
          icon: isHidden ? 'Eye' : 'EyeOff',
          onClick: () => callbacks.onToggleScenarioVisibility?.(scenarioName, !isHidden),
          disabled: !callbacks.onToggleScenarioVisibility
        },
        {
          id: 'unlink-scenario',
          label: 'Unlink from Stage',
          icon: 'Trash2',
          onClick: () => callbacks.onUnlinkScenario?.(scenarioName),
          disabled: !callbacks.onUnlinkScenario || !context.isEditMode,
          danger: true,
          divider: true
        }
      ]
    }
  ]
}

/**
 * Builds context menu for stage nodes.
 */
export function buildStageMenu(
  nodeId: string,
  nodeData: StageNodeData,
  context: MenuBuilderContext,
  callbacks: MenuCallbacks
): ContextMenuSection[] {
  const stageId = nodeId
  const sectorId = nodeData?.sectorId || ''

  // Get full stage info from lookup if available
  const stageInfo = context.stageLookup?.get(stageId)

  const sections: ContextMenuSection[] = [
    {
      title: 'Stage Actions',
      items: [
        {
          id: 'add-child',
          label: 'Add Child Stage',
          icon: 'Plus',
          onClick: () => callbacks.onAddChildStage?.(stageId, sectorId),
          disabled: !callbacks.onAddChildStage || !context.isEditMode || context.scenarioOnlyMode
        },
        {
          id: 'generate-ai',
          label: 'Generate AI Suggestions',
          icon: 'Sparkles',
          onClick: () => callbacks.onGenerateAISuggestions?.(stageId, sectorId),
          disabled: !callbacks.onGenerateAISuggestions || !context.isEditMode || context.scenarioOnlyMode
        },
        {
          id: 'link-scenario',
          label: 'Link Scenario',
          icon: 'Link2',
          onClick: () => {
            if (stageInfo) {
              callbacks.onLinkScenario?.(stageInfo)
            }
          },
          disabled: !callbacks.onLinkScenario || !stageInfo || !context.isEditMode || context.scenarioOnlyMode
        }
      ]
    }
  ]

  // Add edit section only in edit mode and not in scenario-only mode
  if (context.isEditMode && !context.scenarioOnlyMode) {
    sections.push({
      title: 'Edit',
      items: [
        {
          id: 'edit-stage',
          label: 'Edit Stage',
          icon: 'PenSquare',
          onClick: () => callbacks.onEditStage?.(stageId),
          disabled: !callbacks.onEditStage
        },
        {
          id: 'delete-stage',
          label: 'Delete Stage',
          icon: 'Trash2',
          onClick: () => callbacks.onDeleteStage?.(stageId),
          disabled: !callbacks.onDeleteStage,
          danger: true,
          divider: true
        }
      ]
    })
  }

  return sections
}

/**
 * Routes to the appropriate menu builder based on node type.
 */
export function buildContextMenu(
  nodeType: 'stage' | 'scenario' | 'sectorGroup' | 'pane' | null,
  nodeId: string | null,
  nodeData: DesignerNodeData | null,
  context: MenuBuilderContext,
  callbacks: MenuCallbacks
): ContextMenuSection[] {
  if (nodeType === 'pane') {
    return buildPaneMenu(context, callbacks)
  }

  if (nodeType === 'sectorGroup' && nodeId) {
    const sectorId = nodeId.replace('sector-group::', '')
    return buildSectorGroupMenu(sectorId, context, callbacks)
  }

  if (nodeType === 'scenario' && nodeData) {
    return buildScenarioMenu(nodeData as ScenarioNodeData, context, callbacks)
  }

  if (nodeType === 'stage' && nodeId && nodeData) {
    return buildStageMenu(nodeId, nodeData as StageNodeData, context, callbacks)
  }

  return []
}
