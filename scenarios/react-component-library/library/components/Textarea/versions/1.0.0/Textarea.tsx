/**
 * @libraryId react-component-library:Textarea
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18","clsx":"^2.1.1","tailwind-merge":"^2.3.0"}
 * @category controls
 */
import { clsx, type ClassValue } from "clsx";
import * as React from "react";
import { twMerge } from "tailwind-merge";

export type TextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(({ className, ...props }, ref) => {
  return (
    <textarea
      className={cn(
        "flex min-h-[80px] w-full rounded-control border border-app-border bg-app-surface px-3 py-2 text-base text-app-foreground placeholder:text-app-muted-foreground focus:outline-none focus:ring-2 focus:ring-app-primary/50 disabled:cursor-not-allowed disabled:opacity-60 md:text-sm",
        className,
      )}
      ref={ref}
      {...props}
    />
  );
});
Textarea.displayName = "Textarea";

export { Textarea };