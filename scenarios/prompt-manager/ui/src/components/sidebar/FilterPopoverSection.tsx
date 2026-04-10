/**
 * FilterPopoverSection — Reusable collapsible section within the filter popover.
 */

import { useState } from 'react'
import { ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'

interface FilterPopoverSectionProps {
  title: string
  /** Active filter count shown as a badge. */
  count?: number
  /** Whether section starts expanded. */
  defaultOpen?: boolean
  children: React.ReactNode
}

export function FilterPopoverSection({
  title,
  count,
  defaultOpen = true,
  children,
}: FilterPopoverSectionProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen)

  return (
    <div className="border-b border-border last:border-b-0">
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center justify-between w-full px-3 py-2 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors"
        aria-expanded={isOpen}
      >
        <span className="flex items-center gap-1.5">
          {title}
          {count != null && count > 0 && (
            <span className="px-1.5 py-0.5 text-[10px] font-semibold bg-primary/20 text-primary rounded-full leading-none">
              {count}
            </span>
          )}
        </span>
        <ChevronDown
          className={cn(
            'h-3 w-3 transition-transform',
            isOpen ? 'rotate-0' : '-rotate-90'
          )}
        />
      </button>
      {isOpen && <div className="px-3 pb-2">{children}</div>}
    </div>
  )
}
