import { forwardRef } from 'react'
import { cn } from '../../lib/utils'

export interface LabelProps extends React.LabelHTMLAttributes<HTMLLabelElement> {}

const Label = forwardRef<HTMLLabelElement, LabelProps>(({ className, ...props }, ref) => (
  <label
    ref={ref}
    className={cn('text-sm font-medium text-muted-foreground', className)}
    {...props}
  />
))

Label.displayName = 'Label'

export { Label }
