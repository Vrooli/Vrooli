/**
 * Scene defaults registry.
 *
 * Maps each `SceneType` to a generator that produces the default decorations
 * and furniture for that scene. Call `getSceneDefaults(sceneType)` to obtain
 * the defaults — the stores assign unique IDs when items are added.
 *
 * ## Adding a new scene generator
 * 1. Create a file in this directory exporting a `SceneGenerator` function.
 * 2. Import it here and add an entry to `SCENE_GENERATORS`.
 */

import type { SceneType } from '@/types/environment'
import type { SceneDefaults, SceneGenerator } from './types'
import { generateOutdoorPark } from './outdoorPark'
import { generateIndoorOffice } from './indoorOffice'
import { generateAbstractSpace } from './abstractSpace'
import { generateCustom } from './custom'

const SCENE_GENERATORS: Record<SceneType, SceneGenerator> = {
  'outdoor-park': generateOutdoorPark,
  'indoor-office': generateIndoorOffice,
  'abstract-space': generateAbstractSpace,
  'custom': generateCustom,
}

/** Get the default decorations and furniture for a scene type. */
export function getSceneDefaults(sceneType: SceneType): SceneDefaults {
  return SCENE_GENERATORS[sceneType]()
}

export type { SceneDefaults, SceneGenerator } from './types'
