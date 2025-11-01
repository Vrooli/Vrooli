import * as React from 'react';

import { cn } from '@/lib/utils';

export interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {}

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>((props, ref) => {
  const { className, ...rest } = props;
  return (
    <textarea
      ref={ref}
      className={cn(
        'input-chrome min-h-[120px] w-full resize-y rounded-2xl px-4 py-3 text-base leading-relaxed transition-colors placeholder:text-foreground/45 focus-visible:outline-none',
        className
      )}
      {...rest}
    />
  );
});

Textarea.displayName = 'Textarea';

export { Textarea };
