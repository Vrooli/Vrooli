/**
 * Types for the scene generator system.
 *
 * Each scene type has a generator that produces default decorations and
 * furniture. Generators return items without IDs — the stores assign IDs
 * when the items are added.
 *
 * ## Adding a new generator
 * 1. Create a new file in this directory (e.g. `myScene.ts`).
 * 2. Export a function matching the `SceneGenerator` signature.
 * 3. Register it in `index.ts` under the corresponding `SceneType` key.
 */

import type { DecorationInstance } from '@/types/decoration'
import type { FurnitureInstance } from '@/types/furniture'

/** The default objects for a scene, without IDs (assigned by stores). */
export interface SceneDefaults {
  decorations: Omit<DecorationInstance, 'id'>[]
  furniture: Omit<FurnitureInstance, 'id'>[]
}

/** Context passed to scene generators so they can adapt to the current state. */
export interface SceneGeneratorContext {
  numAgents: number
}

/** A function that produces the default objects for a scene type. */
export type SceneGenerator = (ctx?: SceneGeneratorContext) => SceneDefaults
