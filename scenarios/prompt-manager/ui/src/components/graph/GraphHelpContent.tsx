/**
 * GraphHelpContent - Help body for the dependency graph view.
 *
 * Rendered inside OverlayModal by ViewOverlay. Contains no modal chrome.
 */

import { Network, MousePointer, Search } from 'lucide-react'

export function GraphHelpContent() {
  return (
    <div className="space-y-6">
      <section>
        <h3 className="text-sm font-medium text-indigo-400 mb-2">
          What is the Graph?
        </h3>
        <p className="text-sm text-slate-300 leading-relaxed">
          The dependency graph visualizes how teams, agents, skills, and CLIs relate to
          each other. Edges represent references, membership, and usage patterns detected
          in your project files.
        </p>
      </section>

      <section>
        <h3 className="text-sm font-medium text-indigo-400 mb-3">
          Node Types
        </h3>
        <div className="space-y-3">
          <HelpItem
            icon={<div className="w-4 h-4 rounded-sm bg-slate-500/20 border border-slate-300/70" />}
            title="Team"
            description="Rectangular nodes representing teams that organize agents."
          />
          <HelpItem
            icon={<div className="w-4 h-4 rounded-full bg-slate-500/20 border border-slate-300/70" />}
            title="Agent"
            description="Circular nodes representing agents that execute skills."
          />
          <HelpItem
            icon={<div className="w-4 h-4 rotate-45 rounded-sm bg-slate-500/20 border border-slate-300/70" />}
            title="Skill"
            description="Diamond nodes representing skills (markdown files) that define behavior."
          />
          <HelpItem
            icon={<div className="w-4 h-4 clip-hexagon bg-slate-500/20 border border-slate-300/70" />}
            title="CLI"
            description="Hexagonal nodes representing command-line tools linked to skills."
          />
        </div>
      </section>

      <section>
        <h3 className="text-sm font-medium text-indigo-400 mb-3">
          Health & Edges
        </h3>
        <div className="space-y-3">
          <HelpItem
            icon={<div className="w-4 h-4 rounded-sm bg-red-500/20 border-2 border-red-400/90" />}
            title="Health Coloring"
            description="Node fill and border color indicate health: red (critical), yellow (warning), green (healthy)."
          />
          <HelpItem
            icon={<div className="w-5 border-t-2 border-violet-500 border-dashed mt-2" />}
            title="Edge Meanings"
            description="Line style and color encode edge kind. See the legend for the full mapping."
          />
        </div>
      </section>

      <section>
        <h3 className="text-sm font-medium text-indigo-400 mb-3">
          Interactions
        </h3>
        <div className="space-y-3">
          <HelpItem
            icon={<MousePointer className="h-4 w-4" />}
            title="Click a Node"
            description="Navigates to the corresponding editor panel for that entity."
          />
          <HelpItem
            icon={<Network className="h-4 w-4" />}
            title="Hover a Node"
            description="Shows a tooltip with health score details and connection info."
          />
        </div>
      </section>

      <section>
        <h3 className="text-sm font-medium text-indigo-400 mb-3">
          Queries
        </h3>
        <div className="space-y-3">
          <HelpItem
            icon={<Search className="h-4 w-4" />}
            title="Query Panel"
            description="Use the query panel (top-left) to find orphaned skills, skillless agents, empty teams, and more. Matching nodes will be highlighted."
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
