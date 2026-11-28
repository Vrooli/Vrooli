import { forwardRef, type InputHTMLAttributes } from "react";

import { cn } from "../../lib/utils";

const inputClasses =
  "min-h-11 w-full rounded-control border border-app-border bg-app-surface px-3 py-2 text-app-foreground placeholder:text-app-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary disabled:cursor-not-allowed disabled:opacity-50";

export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(
  function Input({ className, type = "text", ...props }, ref) {
    return <input ref={ref} type={type} className={cn(inputClasses, className)} {...props} />;
  },
);

Input.displayName = "Input";
