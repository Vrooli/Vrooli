import React from 'react'
import { Eye } from 'lucide-react'
import type { ScenarioCatalogEntry } from '../../types/techTree'

interface HiddenScenariosBannerProps {
  hidden: string[]
  scenarioEntryMap: Map<string, ScenarioCatalogEntry>
  onToggleVisibility: (name: string, hidden: boolean) => void
}

const HiddenScenariosBanner: React.FC<HiddenScenariosBannerProps> = ({ hidden, scenarioEntryMap, onToggleVisibility }) => {
  if (!hidden.length) {
    return null
  }

  return (
    <div className="hidden-scenarios-banner">
      <p>Hidden scenarios ({hidden.length})</p>
      <div className="hidden-scenarios-list">
        {hidden.map((name) => {
          const entry = scenarioEntryMap.get(name) || scenarioEntryMap.get(name?.toLowerCase())
          return (
            <button
              key={name}
              type="button"
              className="button button--ghost"
              onClick={() => onToggleVisibility(name, false)}
            >
              <Eye className="w-4 h-4" aria-hidden="true" />
              {entry?.display_name || name}
            </button>
          )
        })}
      </div>
    </div>
  )
}

export default HiddenScenariosBanner
