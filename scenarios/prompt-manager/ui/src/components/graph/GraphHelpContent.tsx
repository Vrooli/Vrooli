/**
 * GraphHelpContent - Help body for the dependency graph view.
 *
 * Rendered inside shared floating panel chrome by ViewOverlay.
 */

import { useCallback } from 'react'
import { Network, MousePointer, Search } from 'lucide-react'
import { GraphLegendSections } from './GraphLegend'

interface HelpSection {
  id: string
  label: string
}

const HELP_SECTIONS: HelpSection[] = [
  { id: 'overview', label: 'Overview' },
  { id: 'legend', label: 'Legend' },
  { id: 'queries', label: 'Queries' },
  { id: 'interactions', label: 'Interactions' },
]

const QUERY_HELP_ITEMS = [
  { label: 'Orphaned Skills', description: 'Skills not referenced by any agent' },
  { label: 'Skillless Agents', description: 'Agents with no skill references' },
  { label: 'Empty Teams', description: 'Teams with no members' },
  { label: 'Unaffiliated Agents', description: 'Agents not in any team' },
  { label: 'CLI-less Skills', description: 'Skills with no CLI tool' },
  { label: 'Circular Refs', description: 'Circular dependency chains' },
]

export function GraphHelpContent() {
  const scrollToSection = useCallback((id: string) => {
    const el = document.getElementById(`graph-help-${id}`)
    el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }, [])

  return (
    <div className="flex gap-4">
      <nav className="shrink-0 w-32 md:w-36 border-r border-white/10 pr-3">
        <p className="text-xs uppercase tracking-wide text-slate-500 mb-2">Sections</p>
        <div className="space-y-1">
          {HELP_SECTIONS.map((section) => (
            <button
              key={section.id}
              type="button"
              onClick={() => scrollToSection(section.id)}
              className="block w-full rounded px-2 py-1 text-left text-xs text-slate-300 hover:text-white hover:bg-white/10 transition-colors"
            >
              {section.label}
            </button>
          ))}
        </div>
      </nav>

      <div className="flex-1 space-y-6">
      <section id="graph-help-overview">
        <h3 className="text-sm font-medium text-indigo-400 mb-2">
          What is the Graph?
        </h3>
        <p className="text-sm text-slate-300 leading-relaxed">
          The dependency graph visualizes how teams, agents, skills, and CLIs relate to
          each other. Edges represent references, membership, and usage patterns detected
          in your project files.
        </p>
      </section>

      <section id="graph-help-legend">
        <h3 className="text-sm font-medium text-indigo-400 mb-3">
          Legend
        </h3>
        <div className="rounded-lg border border-white/10 bg-slate-900/50 p-3 text-xs text-slate-300">
          <GraphLegendSections />
        </div>
      </section>

      <section id="graph-help-queries">
        <h3 className="text-sm font-medium text-indigo-400 mb-3">
          Queries
        </h3>
        <div className="space-y-2">
          <HelpItem
            icon={<Search className="h-4 w-4" />}
            title="Query Panel"
            description="Use the query panel (top-left) to find structural issues. Matching nodes are highlighted in the graph."
          />
          <div className="grid gap-1.5 sm:grid-cols-2">
            {QUERY_HELP_ITEMS.map((query) => (
              <div key={query.label} className="rounded border border-white/10 bg-slate-900/40 p-2">
                <p className="text-xs font-medium text-white">{query.label}</p>
                <p className="text-[11px] text-slate-400 mt-0.5">{query.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section id="graph-help-interactions">
        <h3 className="text-sm font-medium text-indigo-400 mb-3">
          Interactions
        </h3>
        <div className="space-y-3">
          <HelpItem
            icon={<MousePointer className="h-4 w-4" />}
            title="Click a Node"
            description="Opens node details. Use Navigate in that popover to jump into the matching editor panel."
          />
          <HelpItem
            icon={<Network className="h-4 w-4" />}
            title="Hover and Pan"
            description="Hovering surfaces graph context while dragging the canvas pans around. Edges are visual only and do not capture clicks."
          />
        </div>
      </section>
      </div>
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
