/**
 * MaterialContext - Shared material cache context.
 */

import { createContext } from 'react'
import type * as THREE from 'three'
import type { StandardPresetName } from './presets'

/** Cached material instances */
export interface MaterialCache {
  /** Get or create a material for the given preset and color */
  getMaterial: (preset: StandardPresetName, color: string) => THREE.Material
  /** Clear all cached materials */
  clear: () => void
  /** Get cache statistics */
  stats: () => { count: number; presets: string[] }
}

export const MaterialContext = createContext<MaterialCache | null>(null)
