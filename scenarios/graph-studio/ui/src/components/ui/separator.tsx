import type { HTMLAttributes } from 'react'
import { cn } from '../../lib/utils'

function Separator({ className, orientation = 'horizontal', ...props }: HTMLAttributes<HTMLDivElement> & { orientation?: 'horizontal' | 'vertical' }) {
  return (
    <div
      className={cn(
        'shrink-0 bg-border',
        orientation === 'horizontal' ? 'h-[1px] w-full' : 'h-full w-[1px]',
        className,
      )}
      role="separator"
      aria-orientation={orientation}
      {...props}
    />
  )
}

export { Separator }
