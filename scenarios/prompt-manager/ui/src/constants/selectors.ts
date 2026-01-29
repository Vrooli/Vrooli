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
    tabMembers: 'skill-sidebar-tab-members',
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
  members: {
    list: 'member-list',
    newButton: 'member-new-button',
    row: 'member-row',
  },
  world: {
    canvas: 'world-canvas',
    stats: 'world-stats',
  },
  environment: {
    controls: 'environment-controls',
    sceneSpace: 'environment-scene-space',
    scenePark: 'environment-scene-park',
    sceneOffice: 'environment-scene-office',
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
      merged[key] = mergeLiteralNodes(
        literalValue as LiteralSelectorTree,
      )
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
