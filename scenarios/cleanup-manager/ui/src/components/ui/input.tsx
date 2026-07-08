/**
 * @vrooliComponentSource react-component-library:Input
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption template:react-vite:input
 * @vrooliComponentAppliedAt 2026-07-07T00:00:00Z
 * @vrooliComponentSourceSha256 cb55efab31f6c36a87f614f54b33ae58abcdd3bc24726cbf0f88f411fb034468
 * @vrooliComponentDriftHash cb55efab31f6c36a87f614f54b33ae58abcdd3bc24726cbf0f88f411fb034468
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
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
