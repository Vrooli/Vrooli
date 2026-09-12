import { Circle, CheckCircle2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Skeleton } from '@/components/ui/skeleton'

interface DraftStatusChipProps {
  isDraft: boolean
  onChange: (next: boolean) => void
  disabled?: boolean
  isLoading?: boolean
  className?: string
}

export function DraftStatusChip({
  isDraft,
  onChange,
  disabled,
  isLoading,
  className,
}: DraftStatusChipProps) {
  if (isLoading) {
    return <Skeleton className={cn('h-7 w-20 rounded-md', className)} />
  }

  const label = isDraft ? 'Draft' : 'Published'
  const Icon = isDraft ? Circle : CheckCircle2
  const toggleTitle = isDraft ? 'Click to publish' : 'Click to mark as draft'

  return (
    <button
      type="button"
      onClick={() => !disabled && onChange(!isDraft)}
      disabled={disabled}
      aria-pressed={!isDraft}
      aria-label={`Status: ${label}. ${toggleTitle}.`}
      title={toggleTitle}
      data-testid="draft-status-chip"
      className={cn(
        'h-7 px-2 flex items-center gap-1.5 rounded-md text-xs font-medium transition-colors',
        'focus:outline-none focus:ring-2 focus:ring-offset-1',
        isDraft
          ? 'bg-amber-500/20 text-amber-300 hover:bg-amber-500/30 focus:ring-amber-500/50'
          : 'bg-emerald-500/20 text-emerald-300 hover:bg-emerald-500/30 focus:ring-emerald-500/50',
        disabled && 'opacity-50 cursor-not-allowed',
        className
      )}
    >
      <Icon className={cn('h-3 w-3', isDraft && 'fill-current')} />
      <span>{label}</span>
    </button>
  )
}
