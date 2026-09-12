/**
 * @vrooliComponentSource react-component-library:Input
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption template:react-vite:input
 * @vrooliComponentAppliedAt 2026-07-14T20:14:31Z
 * @vrooliComponentSourceSha256 460078e2e5c34ee506c7e70d3f0ee91625736eb1c43f1c98be7b7238b3903c30
 * @vrooliComponentDriftHash 460078e2e5c34ee506c7e70d3f0ee91625736eb1c43f1c98be7b7238b3903c30
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import { forwardRef, type InputHTMLAttributes } from "react";

export type InputProps = InputHTMLAttributes<HTMLInputElement>;

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

export const Input = forwardRef<HTMLInputElement, InputProps>(
  function Input({ className, type, ...props }, ref) {
    return (
      <input
        ref={ref}
        type={type}
        className={cn(
          "flex min-h-11 w-full rounded-control border border-app-border bg-app-surface px-3 py-2 text-base text-app-foreground placeholder:text-app-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50 disabled:cursor-not-allowed disabled:opacity-60 md:text-sm",
          className,
        )}
        {...props}
      />
    );
  },
);
