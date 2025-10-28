import { jsx as _jsx } from "react/jsx-runtime";
import * as CheckboxPrimitive from '@radix-ui/react-checkbox';
import { Check } from 'lucide-react';
import * as React from 'react';
import { cn } from '@/lib/utils';
const Checkbox = React.forwardRef(({ className, ...props }, ref) => (_jsx(CheckboxPrimitive.Root, { ref: ref, className: cn('peer h-4 w-4 shrink-0 rounded border border-slate-700 bg-slate-900/70 shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-slate-950 disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground', className), ...props, children: _jsx(CheckboxPrimitive.Indicator, { className: cn('flex items-center justify-center text-current'), children: _jsx(Check, { className: "h-3 w-3" }) }) })));
Checkbox.displayName = CheckboxPrimitive.Root.displayName;
export { Checkbox };
