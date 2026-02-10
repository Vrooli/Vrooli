/**
 * Abstract-space scene generator.
 * Matches the previous hardcoded defaults from useWorldDefaults.ts.
 */

import type { SceneDefaults } from './types'

export function generateAbstractSpace(): SceneDefaults {
  return {
    decorations: [
      { type: 'potted-plant', position: [-4, 0, -4], rotation: 0, scale: 1 },
      { type: 'tall-plant', position: [4, 0, -3], rotation: 0, scale: 1 },
      { type: 'floor-lamp', position: [-4, 0, 3], rotation: 0, scale: 1 },
      { type: 'globe', position: [3, 0, 4], rotation: 0, scale: 1 },
    ],
    furniture: [
      { type: 'bench', position: [3, 0, -3], rotation: Math.PI / 4, occupiedBy: null },
      { type: 'chair', position: [-3, 0, 2], rotation: -Math.PI / 6, occupiedBy: null },
      { type: 'coffee-table', position: [0, 0, -4], rotation: 0, occupiedBy: null },
    ],
  }
}
