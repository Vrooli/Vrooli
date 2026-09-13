/** Shared settings information-architecture metadata (no components here). */

/** The four groups of the shared settings overlay. */
export type WorldSettingsGroup = 'world' | 'appearance' | 'management' | 'advanced'
/** A specific group or the union used when content renders standalone. */
export type WorldSettingsGroupSelection = WorldSettingsGroup | 'all'

export interface WorldSettingsSection {
  id: WorldSettingsGroup
  label: string
}

/** Canonical group order shared by the desktop and narrow navigation. */
export const WORLD_SETTINGS_SECTIONS: ReadonlyArray<WorldSettingsSection> = [
  { id: 'world', label: 'World' },
  { id: 'appearance', label: 'Appearance' },
  { id: 'management', label: 'Management' },
  { id: 'advanced', label: 'Advanced' },
]
