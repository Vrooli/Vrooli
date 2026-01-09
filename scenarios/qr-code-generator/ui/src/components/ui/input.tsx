import * as React from 'react';

import { cn } from '@/lib/utils';

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {}

const Input = React.forwardRef<HTMLInputElement, InputProps>((props, ref) => {
  const { className, type = 'text', ...rest } = props;
  return (
    <input
      type={type}
      className={cn(
        'input-chrome flex w-full rounded-2xl px-4 py-3 text-base transition-colors placeholder:text-foreground/45 focus-visible:outline-none',
        className
      )}
      ref={ref}
      {...rest}
    />
  );
});

Input.displayName = 'Input';

export { Input };
