import * as SwitchPrimitives from '@radix-ui/react-switch'
import { forwardRef } from 'react'
import { cn } from '../../lib/utils'

const Switch = forwardRef<
  React.ElementRef<typeof SwitchPrimitives.Root>,
  React.ComponentPropsWithoutRef<typeof SwitchPrimitives.Root>
>(({ className, ...props }, ref) => {
  return (
    <SwitchPrimitives.Root
      ref={ref}
      className={cn(
        'peer inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent bg-muted transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-60 data-[state=checked]:bg-primary',
        className,
      )}
      {...props}
    >
      <SwitchPrimitives.Thumb
        className={cn(
          'pointer-events-none block h-5 w-5 translate-x-0 rounded-full bg-background shadow transition-transform data-[state=checked]:translate-x-5',
        )}
      />
    </SwitchPrimitives.Root>
  )
})

Switch.displayName = SwitchPrimitives.Root.displayName

export { Switch }
