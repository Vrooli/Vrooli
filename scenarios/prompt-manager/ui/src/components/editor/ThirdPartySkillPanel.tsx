/**
 * ThirdPartySkillPanel - Attribution for a skill imported from an external source.
 *
 * States that the skill is not Vrooli-authored, credits its source, and lists the
 * external tools it needs that Vrooli does not provide or manage.
 */

import { ExternalLink, PackageOpen, Wrench } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Skill } from '@/types'
import { isThirdPartySkill } from '@/lib/skillOrigin'

interface ThirdPartySkillPanelProps {
  skill: Pick<Skill, 'id' | 'origin' | 'externalTools'>
  className?: string
}

function sourceLabel(url: string): string {
  const match = url.match(/^https?:\/\/github\.com\/([^/]+\/[^/#?]+)/)
  return match?.[1] ?? url
}

export function ThirdPartySkillPanel({ skill, className }: ThirdPartySkillPanelProps) {
  if (!isThirdPartySkill(skill) || !skill.origin) return null
  const { origin } = skill
  const verdict = origin.review?.verdict ?? 'pending'
  const tools = skill.externalTools ?? []

  return (
    <section
      aria-label="Third-party skill attribution"
      data-testid="third-party-skill-panel"
      className={cn('rounded-lg border border-violet-500/30 bg-violet-500/5 px-3 py-2 text-xs', className)}
    >
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <span className="flex items-center gap-1.5 font-medium text-violet-300">
          <PackageOpen className="h-3.5 w-3.5" />
          Third-party skill
        </span>
        <a
          href={origin.sourceUrl}
          target="_blank"
          rel="noreferrer"
          className="flex items-center gap-1 text-foreground underline-offset-2 hover:underline"
        >
          {sourceLabel(origin.sourceUrl)}
          <ExternalLink className="h-3 w-3" />
        </a>
        <span className="text-muted-foreground">License {origin.license}</span>
        <span className="font-mono text-muted-foreground" title={origin.commit}>
          @{origin.commit.slice(0, 8)}
        </span>
        <span className={cn('text-muted-foreground', verdict !== 'passed' && 'text-amber-400')}>
          Review: {verdict}
          {origin.review?.reviewer ? ` by ${origin.review.reviewer}` : ''}
        </span>
      </div>
      <p className="mt-1 text-muted-foreground">
        Not authored or maintained by Vrooli. The imported content is read-only; local changes belong in the skill's
        overlays.
      </p>
      {tools.length > 0 && (
        <div className="mt-1.5">
          <span className="flex items-center gap-1 text-muted-foreground">
            <Wrench className="h-3 w-3" />
            Requires external tools that Vrooli does not provide or manage:
          </span>
          <ul className="mt-1 flex flex-wrap gap-1.5">
            {tools.map((tool) => (
              <li key={tool.name} className="rounded border border-border/60 px-1.5 py-0.5" title={tool.purpose}>
                {tool.url ? (
                  <a href={tool.url} target="_blank" rel="noreferrer" className="hover:underline">
                    {tool.name}
                  </a>
                ) : (
                  tool.name
                )}
                {tool.purpose && <span className="text-muted-foreground">: {tool.purpose}</span>}
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  )
}
