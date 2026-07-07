/**
 * @libraryId react-component-library:Input
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18"}
 */
import { forwardRef, type InputHTMLAttributes } from "react";

export type InputProps = InputHTMLAttributes<HTMLInputElement>;

const joinClasses = (...classes: Array<string | undefined | false>) =>
  classes.filter(Boolean).join(" ");

export const Input = forwardRef<HTMLInputElement, InputProps>(
  function Input({ className, type, ...props }, ref) {
    return (
      <input
        ref={ref}
        type={type}
        className={joinClasses(
          "flex min-h-11 w-full rounded-control border border-app-border bg-app-surface px-3 py-2 text-base text-app-foreground placeholder:text-app-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50 disabled:cursor-not-allowed disabled:opacity-60 md:text-sm",
          className,
        )}
        {...props}
      />
    );
  },
);

