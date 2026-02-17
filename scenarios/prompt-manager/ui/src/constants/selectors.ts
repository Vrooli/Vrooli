/**
 * Prompt Manager selector registry
 *
 * Single source of truth for UI selectors used by BAS workflows.
 * The selectors.manifest.json file is generated from this file.
 */

type LiteralSelectorTree = { readonly [key: string]: string | LiteralSelectorTree }

const literalSelectors = {
  sidebar: {
    container: 'skill-sidebar',
    tabSkills: 'skill-sidebar-tab-skills',
    tabAgents: 'skill-sidebar-tab-agents',
    searchInput: 'skill-sidebar-search-input',
    emptyState: 'skill-sidebar-empty-state',
    skillRow: 'skill-sidebar-skill-row',
    expandAllButton: 'skill-sidebar-expand-all',
    newSkillButton: 'skill-sidebar-new-skill',
  },
  editor: {
    header: 'skill-editor-header',
    nameDisplay: 'skill-editor-name-display',
    nameInput: 'skill-editor-name-input',
    unsavedIndicator: 'skill-editor-unsaved-indicator',
    actionsMenu: 'skill-editor-actions-menu',
    discardAction: 'skill-editor-discard-action',
    saveButton: 'skill-editor-save-button',
    saveAllButton: 'skill-editor-save-all-button',
    discardButton: 'skill-editor-discard-button',
  },
  agents: {
    list: 'agent-list',
    newButton: 'agent-new-button',
    row: 'agent-row',
    contextMenu: 'agent-context-menu',
  },
  agentEditor: {
    header: 'agent-editor-header',
    nameDisplay: 'agent-editor-name-display',
    nameInput: 'agent-editor-name-input',
  },
  teams: {
    list: 'team-list',
    newButton: 'team-new-button',
    importButton: 'team-import-button',
    row: 'team-row',
    exportButton: 'team-export-button',
    contextMenu: 'team-context-menu',
  },
  teamEditor: {
    header: 'team-editor-header',
    nameDisplay: 'team-editor-name-display',
    nameInput: 'team-editor-name-input',
    orgChart: 'team-editor-org-chart',
    memberDetail: 'team-editor-member-detail',
    addMemberButton: 'team-editor-add-member',
    spawnMode: 'team-editor-spawn-mode',
  },
  runs: {
    list: 'run-list',
    row: 'run-row',
  },
  runEditor: {
    header: 'run-editor-header',
    tabInfo: 'run-editor-tab-info',
    tabEvents: 'run-editor-tab-events',
    tabInvestigation: 'run-editor-tab-investigation',
  },
  world: {
    canvas: 'world-canvas',
    stats: 'world-stats',
  },
  settings: {
    button: 'world-settings-button',
    popup: 'world-settings-popup',
    camera: 'world-settings-camera',
    timeSlider: 'world-settings-time-slider',
    realTimeToggle: 'world-settings-realtime-toggle',
    scene: 'world-settings-scene',
    graphics: 'world-settings-graphics',
    customToggle: 'world-settings-custom-toggle',
    fpsOverlayToggle: 'world-settings-fps-overlay-toggle',
    fpsTraceToggle: 'world-settings-fps-trace-toggle',
  },
  viewOverlay: {
    stats: 'view-overlay-stats',
    viewToggle: 'view-overlay-view-toggle',
    settingsButton: 'view-overlay-settings-button',
    helpButton: 'view-overlay-help-button',
    mobileStatsButton: 'view-overlay-mobile-stats-button',
    mobileLeftPanelButton: 'view-overlay-mobile-left-panel-button',
    mobilePanelBackdrop: 'view-overlay-mobile-panel-backdrop',
    mobilePanelSheet: 'view-overlay-mobile-panel-sheet',
  },
  graph: {
    modeToggle: 'graph-mode-toggle',
    modeVisual: 'graph-mode-visual',
    modeJson: 'graph-mode-json',
    jsonCopyButton: 'graph-json-copy-button',
  },
  xrefs: {
    panel: 'xrefs-panel',
    toggle: 'xrefs-toggle',
    list: 'xrefs-list',
    item: 'xrefs-item',
  },
  environment: {
    controls: 'environment-controls',
    sceneSpace: 'environment-scene-space',
    scenePark: 'environment-scene-park',
    sceneOffice: 'environment-scene-office',
    timeSlider: 'environment-time-slider',
    realTimeToggle: 'environment-realtime-toggle',
    // Legacy discrete time selectors (deprecated)
    timeMorning: 'environment-time-morning',
    timeNoon: 'environment-time-noon',
    timeSunset: 'environment-time-sunset',
    timeNight: 'environment-time-night',
  },
} as const satisfies LiteralSelectorTree

const flattenLiteralSelectors = (
  tree: LiteralSelectorTree,
  prefix: string[] = [],
  target: Record<string, { testId: string; selector: string }> = {},
) => {
  for (const [key, value] of Object.entries(tree)) {
    const nextPath = [...prefix, key]
    if (typeof value === 'string') {
      const manifestKey = nextPath.join('.')
      target[manifestKey] = {
        testId: value,
        selector: `[data-testid="${value}"]`,
      }
      continue
    }
    flattenLiteralSelectors(value, nextPath, target)
  }
  return target
}

const mergeLiteralNodes = (
  literalNode: LiteralSelectorTree | undefined,
): Record<string, unknown> => {
  const merged: Record<string, unknown> = {}
  const keys = Object.keys(literalNode ?? {})

  keys.forEach((key) => {
    const literalValue = literalNode?.[key]

    if (typeof literalValue === 'string') {
      merged[key] = literalValue
      return
    }

    if (literalValue && typeof literalValue === 'object') {
      merged[key] = mergeLiteralNodes(literalValue)
    }
  })

  return merged
}

const createSelectorRegistry = <L extends LiteralSelectorTree>(literalTree: L) => {
  const selectors = mergeLiteralNodes(literalTree) as L
  const manifest = {
    selectors: flattenLiteralSelectors(literalTree),
    dynamicSelectors: {},
  }
  return { selectors, manifest }
}

const { selectors, manifest } = createSelectorRegistry(literalSelectors)

export { selectors }
export const selectorsManifest = manifest
