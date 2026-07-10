/**
 * @vrooliComponentSource react-component-library:Input
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption 3ba1f8b7-85f3-416d-b76c-d464874333cb
 * @vrooliComponentAppliedAt 2026-07-09T04:31:18Z
 * @vrooliComponentSourceSha256 e084a1555d945b936c05d213465e08b8dacc7d3b46f3afd8bf0cb730af7fb005
 * @vrooliComponentDriftHash e084a1555d945b936c05d213465e08b8dacc7d3b46f3afd8bf0cb730af7fb005
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
