import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import * as React from 'react';
import * as ScrollAreaPrimitive from '@radix-ui/react-scroll-area';
import { cn } from '@/lib/utils';
const ScrollArea = React.forwardRef(({ className, children, ...props }, ref) => (_jsxs(ScrollAreaPrimitive.Root, { ref: ref, className: cn('relative overflow-hidden rounded-lg border border-slate-800 bg-slate-900/50', className), ...props, children: [_jsx(ScrollAreaPrimitive.Viewport, { className: "h-full w-full rounded-[inherit]", children: children }), _jsx(ScrollBar, {}), _jsx(ScrollAreaPrimitive.Corner, { className: "bg-slate-800" })] })));
ScrollArea.displayName = ScrollAreaPrimitive.Root.displayName;
const ScrollBar = React.forwardRef(({ className, orientation = 'vertical', ...props }, ref) => (_jsx(ScrollAreaPrimitive.ScrollAreaScrollbar, { ref: ref, orientation: orientation, className: cn('flex touch-none select-none transition-colors', orientation === 'vertical' && 'h-full w-2.5 border-l border-slate-800', orientation === 'horizontal' && 'h-2.5 border-t border-slate-800', className), ...props, children: _jsx(ScrollAreaPrimitive.ScrollAreaThumb, { className: "relative flex-1 rounded-full bg-slate-700" }) })));
ScrollBar.displayName = ScrollAreaPrimitive.ScrollAreaScrollbar.displayName;
export { ScrollArea, ScrollBar };
