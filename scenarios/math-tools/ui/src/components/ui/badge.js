import { jsx as _jsx } from "react/jsx-runtime";
import * as React from 'react';
import { cva } from 'class-variance-authority';
import { cn } from '@/lib/utils';
const badgeVariants = cva('inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 focus:ring-offset-slate-950', {
    variants: {
        variant: {
            default: 'border-transparent bg-primary/20 text-primary',
            secondary: 'border-transparent bg-secondary/20 text-secondary-foreground',
            outline: 'border-slate-700 text-foreground',
            success: 'border-transparent bg-emerald-500/20 text-emerald-400',
            warning: 'border-transparent bg-amber-500/20 text-amber-300',
            danger: 'border-transparent bg-rose-500/20 text-rose-300',
        },
    },
    defaultVariants: {
        variant: 'default',
    },
});
const Badge = React.forwardRef(({ className, variant, ...props }, ref) => (_jsx("div", { ref: ref, className: cn(badgeVariants({ variant }), className), ...props })));
Badge.displayName = 'Badge';
export { Badge, badgeVariants };
