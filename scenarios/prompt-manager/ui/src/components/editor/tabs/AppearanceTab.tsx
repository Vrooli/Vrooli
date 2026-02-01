/**
 * AppearanceTab - Agent appearance customization tab.
 *
 * Features:
 * - Live 3D preview of agent
 * - Color pickers for body, head, and accent
 */

import { useCallback } from 'react'
import { ColorPicker } from '@/components/shared/ColorPicker'
import type { Agent, UpdateAgentRequest } from '@/types/agent'
import { DEFAULT_AGENT_COLORS } from '@/types/agent'

interface AppearanceTabProps {
  agent: Agent
  onUpdate: (updates: UpdateAgentRequest) => Promise<void>
}

/**
 * Appearance customization tab component.
 */
export function AppearanceTab({ agent, onUpdate }: AppearanceTabProps) {
  // Extract colors with defaults
  const bodyColor = agent.appearance?.body ?? DEFAULT_AGENT_COLORS.body
  const headColor = agent.appearance?.head ?? DEFAULT_AGENT_COLORS.head
  const accentColor = agent.appearance?.accent ?? DEFAULT_AGENT_COLORS.accent

  // Handle color changes
  const handleBodyColorChange = useCallback(
    async (color: string) => {
      await onUpdate({
        appearance: {
          body: color,
          head: headColor,
          accent: accentColor,
        },
      })
    },
    [headColor, accentColor, onUpdate]
  )

  const handleHeadColorChange = useCallback(
    async (color: string) => {
      await onUpdate({
        appearance: {
          body: bodyColor,
          head: color,
          accent: accentColor,
        },
      })
    },
    [bodyColor, accentColor, onUpdate]
  )

  const handleAccentColorChange = useCallback(
    async (color: string) => {
      await onUpdate({
        appearance: {
          body: bodyColor,
          head: headColor,
          accent: color,
        },
      })
    },
    [bodyColor, headColor, onUpdate]
  )

  return (
    <div className="space-y-6">
      {/* Live 3D Preview */}
      <div className="flex justify-center p-6 bg-muted/30 rounded-lg">
        <div className="relative">
          {/* Accent (antenna) */}
          <div
            className="absolute -top-3 left-1/2 -translate-x-1/2 w-4 h-4 rounded-full"
            style={{ backgroundColor: accentColor }}
          />
          {/* Body */}
          <div
            className="w-24 h-32 rounded-full flex items-start justify-center pt-6"
            style={{ backgroundColor: bodyColor }}
          >
            {/* Head */}
            <div
              className="w-14 h-14 rounded-full"
              style={{ backgroundColor: headColor }}
            />
          </div>
          {/* Arms */}
          <div
            className="absolute top-10 -left-3 w-4 h-14 rounded-full"
            style={{ backgroundColor: bodyColor }}
          />
          <div
            className="absolute top-10 -right-3 w-4 h-14 rounded-full"
            style={{ backgroundColor: bodyColor }}
          />
        </div>
      </div>

      {/* Color Pickers */}
      <div className="space-y-4">
        <ColorPicker
          label="Body Color"
          value={bodyColor}
          onChange={(color) => void handleBodyColorChange(color)}
        />
        <ColorPicker
          label="Head Color"
          value={headColor}
          onChange={(color) => void handleHeadColorChange(color)}
        />
        <ColorPicker
          label="Accent Color"
          value={accentColor}
          onChange={(color) => void handleAccentColorChange(color)}
        />
      </div>
    </div>
  )
}
