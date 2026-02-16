/**
 * CollapsibleSection - Reusable expandable/collapsible container.
 *
 * Supports both controlled (expanded prop) and uncontrolled (defaultExpanded) modes.
 * Optional dirty indicator dot and header-right slot for action buttons.
 */

import { useState } from 'react'
import { ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'

interface CollapsibleSectionProps {
  title: string
  defaultExpanded?: boolean
  /** Controlled mode — overrides internal state when provided */
  expanded?: boolean
  onExpandedChange?: (expanded: boolean) => void
  /** Show amber dot on header to indicate unsaved changes */
  isDirty?: boolean
  /** Slot rendered on the right side of the header (e.g. Save button) */
  headerRight?: React.ReactNode
  children: React.ReactNode
  /** HTML id for scrollIntoView targeting */
  id?: string
}

export function CollapsibleSection({
  title,
  defaultExpanded = false,
  expanded: controlledExpanded,
  onExpandedChange,
  isDirty,
  headerRight,
  children,
  id,
}: CollapsibleSectionProps) {
  const [internalExpanded, setInternalExpanded] = useState(defaultExpanded)
  const isControlled = controlledExpanded !== undefined
  const isExpanded = isControlled ? controlledExpanded : internalExpanded

  const toggle = () => {
    const next = !isExpanded
    if (!isControlled) setInternalExpanded(next)
    onExpandedChange?.(next)
  }

  return (
    <div id={id} className="rounded-lg border border-border bg-muted/40 px-3 py-3">
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={toggle}
          className="flex items-center gap-2 text-xs font-semibold text-muted-foreground hover:text-foreground transition-colors"
          aria-expanded={isExpanded}
        >
          <ChevronDown
            className={cn(
              'h-4 w-4 transition-transform',
              isExpanded ? 'rotate-0' : '-rotate-90'
            )}
          />
          {title}
          {isDirty && (
            <span className="w-2 h-2 bg-amber-500 rounded-full flex-shrink-0" />
          )}
        </button>
        {headerRight && <div className="ml-auto">{headerRight}</div>}
      </div>
      {isExpanded && <div className="mt-2">{children}</div>}
    </div>
  )
}
