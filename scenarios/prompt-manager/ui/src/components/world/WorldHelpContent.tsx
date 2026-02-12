/**
 * WorldHelpContent - Help body for the 3D avatar environment.
 *
 * Rendered inside OverlayModal by ViewOverlay. Contains no modal chrome.
 */

import { User, Eye, Map, MousePointer, Move3D } from 'lucide-react'

export function WorldHelpContent() {
  return (
    <div className="space-y-6">
      <section>
        <h3 className="text-sm font-medium text-indigo-400 mb-2">
          What are Avatars?
        </h3>
        <p className="text-sm text-slate-300 leading-relaxed">
          Avatars are visual characters that represent agents. Their behavior is described
          in the agent's Markdown files (like SOUL.md), where you can reference skills
          and instructions in plain text.
        </p>
      </section>

      <section>
        <h3 className="text-sm font-medium text-indigo-400 mb-3">
          Interactions
        </h3>
        <div className="space-y-3">
          <HelpItem
            icon={<MousePointer className="h-4 w-4" />}
            title="Click an Avatar"
            description="Opens the avatar's menu where you can customize it, duplicate, or delete it."
          />
          <HelpItem
            icon={<Move3D className="h-4 w-4" />}
            title="Drag to Orbit"
            description="Click and drag anywhere in the environment to rotate the camera around the scene."
          />
        </div>
      </section>

      <section>
        <h3 className="text-sm font-medium text-indigo-400 mb-3">
          Camera Views
        </h3>
        <p className="text-sm text-slate-400 mb-3">
          Use the camera button in the top-right to cycle through views:
        </p>
        <div className="space-y-3">
          <HelpItem
            icon={<User className="h-4 w-4" />}
            title="Focus on Avatar"
            description="Zooms in on the selected avatar (or the first one if none selected)."
          />
          <HelpItem
            icon={<Eye className="h-4 w-4" />}
            title="Default View"
            description="Standard perspective view showing all avatars in the environment."
          />
          <HelpItem
            icon={<Map className="h-4 w-4" />}
            title="Aerial View"
            description="Top-down view of the entire environment, useful for seeing all avatars at once."
          />
        </div>
      </section>
    </div>
  )
}

interface HelpItemProps {
  icon: React.ReactNode
  title: string
  description: string
}

function HelpItem({ icon, title, description }: HelpItemProps) {
  return (
    <div className="flex gap-3">
      <div className="p-2 rounded-lg bg-slate-800 text-slate-400 shrink-0">
        {icon}
      </div>
      <div>
        <p className="text-sm font-medium text-white">{title}</p>
        <p className="text-xs text-slate-400 mt-0.5">{description}</p>
      </div>
    </div>
  )
}
