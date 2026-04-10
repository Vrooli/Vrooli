/**
 * AppearanceTab - Agent appearance customization tab.
 *
 * Features:
 * - Color pickers for body, head, and accent
 * - Uses centralized form state for dirty tracking
 */

import { useCallback } from 'react'
import { ColorPicker } from '@/components/shared/ColorPicker'
import type { NormalizedAgentFormState } from '@/stores/agentEditorStore'

interface AppearanceTabProps {
  /** Form state from the editor store */
  formState: NormalizedAgentFormState
  /** Update a single field */
  updateField: <K extends keyof NormalizedAgentFormState>(field: K, value: NormalizedAgentFormState[K]) => void
}

/**
 * Appearance customization tab component.
 */
export function AppearanceTab({ formState, updateField }: AppearanceTabProps) {
  // Extract colors (appearance is always defined in NormalizedAgentFormState)
  const bodyColor = formState.appearance.body
  const headColor = formState.appearance.head
  const accentColor = formState.appearance.accent

  // Handle color changes - update through form state
  const handleBodyColorChange = useCallback(
    (color: string) => {
      updateField('appearance', {
        ...formState.appearance,
        body: color,
      })
    },
    [formState.appearance, updateField]
  )

  const handleHeadColorChange = useCallback(
    (color: string) => {
      updateField('appearance', {
        ...formState.appearance,
        head: color,
      })
    },
    [formState.appearance, updateField]
  )

  const handleAccentColorChange = useCallback(
    (color: string) => {
      updateField('appearance', {
        ...formState.appearance,
        accent: color,
      })
    },
    [formState.appearance, updateField]
  )

  return (
    <div className="space-y-6">
      {/* Color Pickers */}
      <div className="space-y-4">
        <ColorPicker
          label="Body Color"
          value={bodyColor}
          onChange={handleBodyColorChange}
        />
        <ColorPicker
          label="Head Color"
          value={headColor}
          onChange={handleHeadColorChange}
        />
        <ColorPicker
          label="Accent Color"
          value={accentColor}
          onChange={handleAccentColorChange}
        />
      </div>
    </div>
  )
}
