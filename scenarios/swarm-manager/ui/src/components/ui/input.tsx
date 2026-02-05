/**
 * Input Component
 *
 * A styled input component with CVA variants for consistent form inputs
 * across the application. Supports left icons, right-side slots, and sizes.
 */

import { forwardRef } from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const inputVariants = cva(
  "w-full rounded-lg border bg-slate-800/50 text-slate-50 placeholder:text-slate-400 focus:outline-none focus:ring-1 transition-colors disabled:pointer-events-none disabled:opacity-60",
  {
    variants: {
      variant: {
        default: "border-white/10 focus:border-cyan-500 focus:ring-cyan-500",
        error: "border-red-500/50 focus:border-red-500 focus:ring-red-500",
      },
      size: {
        default: "h-10 px-4 text-sm",
        sm: "h-8 px-3 text-sm",
        lg: "h-12 px-5 text-base",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

export interface InputProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "size">,
    VariantProps<typeof inputVariants> {
  /** Optional icon to display on the left side */
  leftIcon?: React.ReactNode;
  /** Optional element to render on the right side (e.g., clear button) */
  rightSlot?: React.ReactNode;
}

/**
 * Input component with consistent styling and optional left icon support.
 *
 * Usage:
 * ```tsx
 * // Basic input
 * <Input placeholder="Enter text..." />
 *
 * // With left icon
 * <Input leftIcon={<Search className="h-4 w-4" />} placeholder="Search..." />
 *
 * // With right slot (e.g., clear button)
 * <Input rightSlot={<button>Clear</button>} placeholder="Search..." />
 *
 * // Error variant
 * <Input variant="error" placeholder="Invalid input" />
 * ```
 */
export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, variant, size, leftIcon, rightSlot, ...props }, ref) => {
    if (leftIcon || rightSlot) {
      return (
        <div className="relative">
          {leftIcon && (
            <div className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">
              {leftIcon}
            </div>
          )}
          {rightSlot && (
            <div className="absolute right-3 top-1/2 -translate-y-1/2">
              {rightSlot}
            </div>
          )}
          <input
            ref={ref}
            className={cn(
              inputVariants({ variant, size, className }),
              leftIcon && "pl-10",
              rightSlot && "pr-10"
            )}
            {...props}
          />
        </div>
      );
    }

    return (
      <input
        ref={ref}
        className={cn(inputVariants({ variant, size, className }))}
        {...props}
      />
    );
  }
);
Input.displayName = "Input";
