/**
 * Custom scene generator — returns empty arrays.
 * Users populate custom scenes manually via the editor.
 */

import type { SceneDefaults } from './types'

export function generateCustom(): SceneDefaults {
  return { decorations: [], furniture: [] }
}
