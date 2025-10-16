import { forwardRef } from 'react'
import { cn } from '../../lib/utils'

export type SelectProps = React.SelectHTMLAttributes<HTMLSelectElement>

const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ className, children, ...props }, ref) => (
    <select
      ref={ref}
      className={cn(
        'flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm transition focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-60',
        className,
      )}
      {...props}
    >
      {children}
    </select>
  ),
)

Select.displayName = 'Select'

export { Select }
