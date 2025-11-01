import * as React from 'react';
import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';

import { cn } from '@/lib/utils';

const buttonVariants = cva(
  'inline-flex items-center justify-center whitespace-nowrap rounded-full text-sm font-semibold uppercase tracking-[0.24em] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/70 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default:
          'bg-accent text-black hover:bg-accent/90 shadow-[0_0_18px_rgba(94,255,196,0.38)]',
        secondary:
          'border border-panelBorder/60 bg-black/40 text-foreground hover:bg-black/50 shadow-[0_0_20px_rgba(94,255,196,0.12)]',
        outline:
          'border border-panelBorder/60 bg-transparent text-foreground hover:bg-panel/70',
        ghost: 'hover:bg-panel/60 hover:text-foreground text-foreground/70',
        destructive:
          'bg-danger text-white shadow-[0_0_18px_rgba(255,107,147,0.35)] hover:bg-[#ff84a7]'
      },
      size: {
        sm: 'h-9 px-5 text-xs',
        md: 'h-11 px-7 text-sm',
        lg: 'h-12 px-9 text-sm',
        icon: 'h-11 w-11 rounded-full'
      }
    },
    defaultVariants: {
      variant: 'default',
      size: 'md'
    }
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>((props, ref) => {
  const { className, variant, size, asChild = false, ...rest } = props;
  const Comp = asChild ? Slot : 'button';
  return (
    <Comp className={cn(buttonVariants({ variant, size }), className)} ref={ref} {...rest} />
  );
});

Button.displayName = 'Button';

export { Button, buttonVariants };
