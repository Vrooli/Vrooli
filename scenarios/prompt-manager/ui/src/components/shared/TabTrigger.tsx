/**
 * Shared TabTrigger and TabList components for consistent tab UX.
 *
 * - TabList: horizontally scrollable container with hidden scrollbar
 * - TabTrigger: icon + label trigger that hides labels on small screens
 */

import { forwardRef } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import { cn } from '@/lib/utils'

// ============================================================================
// TabList — scrollable container
// ============================================================================

interface TabListProps {
  children: React.ReactNode
  className?: string
}

/**
 * Scrollable tab list that works on all screen sizes.
 * Hides scrollbar but allows horizontal scroll/swipe on touch.
 */
export const TabList = forwardRef<HTMLDivElement, TabListProps>(
  function TabList({ children, className }, ref) {
    return (
      <Tabs.List
        ref={ref}
        className={cn(
          'flex-shrink-0 flex flex-nowrap overflow-x-auto border-b border-border px-4',
          '[&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]',
          className
        )}
      >
        {children}
      </Tabs.List>
    )
  }
)

// ============================================================================
// TabTrigger — icon + responsive label
// ============================================================================

interface TabTriggerProps {
  value: string
  icon: React.ReactNode
  label: string
  /** Optional live indicator dot */
  live?: boolean
  /** Optional data-testid */
  testId?: string
  /** Use smaller sizing (for nested sub-tabs) */
  compact?: boolean
  /** Always show label regardless of screen width (for sidebars) */
  alwaysShowLabel?: boolean
  className?: string
}

/**
 * Tab trigger with icon and label.
 * Label is hidden on small screens (< sm breakpoint) to save space.
 */
export function TabTrigger({ value, icon, label, live, testId, compact, alwaysShowLabel, className }: TabTriggerProps) {
  return (
    <Tabs.Trigger
      value={value}
      className={cn(
        'flex-shrink-0 min-w-fit flex items-center gap-1.5 transition-colors',
        'border-b-2',
        'data-[state=active]:border-primary data-[state=active]:text-primary',
        'data-[state=inactive]:border-transparent data-[state=inactive]:text-muted-foreground',
        'hover:text-foreground',
        compact
          ? 'px-2.5 py-1.5 text-xs font-medium'
          : 'px-3 py-2 text-sm font-medium',
        className
      )}
      data-testid={testId}
      title={label}
    >
      {icon}
      <span className={alwaysShowLabel ? undefined : 'hidden sm:inline'}>{label}</span>
      {live && (
        <span className="relative flex h-2 w-2 ml-1">
          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
          <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500" />
        </span>
      )}
    </Tabs.Trigger>
  )
}
