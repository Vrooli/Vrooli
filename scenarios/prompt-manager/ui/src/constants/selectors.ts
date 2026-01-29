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
    newSkillButton: 'skill-sidebar-new-skill',
  },
  editor: {
    header: 'skill-editor-header',
  },
  members: {
    list: 'member-list',
  },
  world: {
    canvas: 'world-canvas',
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
  const selectors = mergeLiteralNodes(literalTree) as any
  const manifest = {
    selectors: flattenLiteralSelectors(literalTree),
    dynamicSelectors: {},
  }
  return { selectors, manifest }
}

const { selectors, manifest } = createSelectorRegistry(literalSelectors)

export { selectors }
export const selectorsManifest = manifest
