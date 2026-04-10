/**
 * Indoor-office scene generator.
 * Places desks, chairs, a bookshelf, and plants in an office layout.
 */

import type { SceneDefaults } from './types'

export function generateIndoorOffice(): SceneDefaults {
  return {
    decorations: [
      // Bookshelf against the back wall
      { type: 'bookshelf', position: [0, 0, -6], rotation: 0, scale: 1 },
      // Plants in corners
      { type: 'potted-plant', position: [-5, 0, -5], rotation: 0, scale: 1 },
      { type: 'tall-plant', position: [5, 0, -5], rotation: 0, scale: 1 },
      { type: 'potted-plant', position: [-5, 0, 5], rotation: 0, scale: 0.9 },
      // Desk lamps on each desk
      { type: 'desk-lamp', position: [-3.5, 0, -1], rotation: 0, scale: 1 },
      { type: 'desk-lamp', position: [3.5, 0, -1], rotation: Math.PI, scale: 1 },
      // Floor lamp near seating area
      { type: 'floor-lamp', position: [5, 0, 3], rotation: 0, scale: 1 },
      // Wall decoration
      { type: 'clock', position: [0, 0, -5.5], rotation: 0, scale: 1 },
    ],
    furniture: [
      // Two desks facing each other
      { type: 'desk', position: [-3, 0, -2], rotation: 0, occupiedBy: null },
      { type: 'desk', position: [3, 0, -2], rotation: Math.PI, occupiedBy: null },
      // Chairs at desks
      { type: 'chair', position: [-3, 0, -0.5], rotation: Math.PI, occupiedBy: null },
      { type: 'chair', position: [3, 0, -0.5], rotation: 0, occupiedBy: null },
      // Coffee table with armchairs for a lounge area
      { type: 'coffee-table', position: [0, 0, 4], rotation: 0, occupiedBy: null },
      { type: 'armchair', position: [-2, 0, 4], rotation: Math.PI / 2, occupiedBy: null },
      { type: 'armchair', position: [2, 0, 4], rotation: -Math.PI / 2, occupiedBy: null },
    ],
  }
}
