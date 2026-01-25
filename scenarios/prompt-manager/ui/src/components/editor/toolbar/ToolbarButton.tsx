/**
 * ToolbarButton - Reusable button component for the editor toolbar.
 *
 * Features:
 * - Active state styling
 * - Disabled state handling
 * - Accessible title/tooltip
 * - Consistent styling
 */

import { cn } from '@/lib/utils'

export interface ToolbarButtonProps {
  /** Click handler */
  onClick: () => void
  /** Whether the button is in an active/pressed state */
  isActive?: boolean
  /** Whether the button is disabled */
  disabled?: boolean
  /** Accessible title/tooltip */
  title: string
  /** Button content (typically an icon) */
  children: React.ReactNode
  /** Additional CSS classes */
  className?: string
}

/**
 * A toolbar button with consistent styling and state handling.
 */
export function ToolbarButton({
  onClick,
  isActive = false,
  disabled = false,
  title,
  children,
  className,
}: ToolbarButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      title={title}
      className={cn(
        'p-1.5 rounded transition-colors',
        isActive
          ? 'bg-primary/30 text-primary'
          : 'text-muted-foreground hover:text-foreground hover:bg-muted',
        disabled && 'opacity-50 cursor-not-allowed',
        className
      )}
    >
      {children}
    </button>
  )
}

/**
 * Vertical divider for separating toolbar button groups.
 */
export function ToolbarDivider({ className }: { className?: string }) {
  return <div className={cn('w-px h-6 bg-border mx-1', className)} />
}
