import { jsx as _jsx } from "react/jsx-runtime";
import * as React from 'react';
import { cn } from '@/lib/utils';
const Input = React.forwardRef(({ className, type, ...props }, ref) => {
    return (_jsx("input", { type: type, className: cn('flex h-10 w-full rounded-md border border-slate-800 bg-slate-900/70 px-3 py-2 text-sm text-foreground shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-slate-950 disabled:cursor-not-allowed disabled:opacity-50', className), ref: ref, ...props }));
});
Input.displayName = 'Input';
export { Input };
