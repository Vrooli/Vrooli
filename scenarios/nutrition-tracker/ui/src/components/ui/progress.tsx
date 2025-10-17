import * as React from 'react';

import { cn } from '../../lib/utils';

const Progress = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement> & { value?: number }>(
  ({ className, value = 0, ...props }, ref) => (
    <div
      ref={ref}
      className={cn('relative h-2 w-full overflow-hidden rounded-full bg-muted', className)}
      {...props}
    >
      <div
        className="h-full w-full flex-1 rounded-full bg-gradient-to-r from-brand-500 to-blue-500"
        style={{ transform: `translateX(-${100 - Math.min(100, Math.max(0, value))}%)` }}
      />
    </div>
  )
);
Progress.displayName = 'Progress';

export { Progress };
