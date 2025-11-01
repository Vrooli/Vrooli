import * as React from 'react';
import * as LabelPrimitive from '@radix-ui/react-label';

import { cn } from '@/lib/utils';

const Label = React.forwardRef<
  React.ElementRef<typeof LabelPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof LabelPrimitive.Root>
>(({ className, ...props }, ref) => (
  <LabelPrimitive.Root
    ref={ref}
    className={cn(
      'mb-2 inline-flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.26em] text-foreground/70',
      className
    )}
    {...props}
  />
));

Label.displayName = LabelPrimitive.Root.displayName;

export { Label };
