import { jsx as _jsx } from "react/jsx-runtime";
import * as React from 'react';
import * as LabelPrimitive from '@radix-ui/react-label';
import { cva } from 'class-variance-authority';
import { cn } from '@/lib/utils';
const labelVariants = cva('text-sm font-medium leading-none text-muted-foreground peer-disabled:cursor-not-allowed peer-disabled:opacity-70', {
    variants: {
        tone: {
            default: 'text-slate-200',
            muted: 'text-slate-400',
        },
    },
    defaultVariants: {
        tone: 'default',
    },
});
const Label = React.forwardRef(({ className, tone, ...props }, ref) => (_jsx(LabelPrimitive.Root, { ref: ref, className: cn(labelVariants({ tone, className })), ...props })));
Label.displayName = LabelPrimitive.Root.displayName;
export { Label };
