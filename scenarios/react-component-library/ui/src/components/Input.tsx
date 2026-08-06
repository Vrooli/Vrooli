/**
 * @vrooliComponentSource react-component-library:Input
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption dace2f14-dee9-4c31-932b-476c0c005efb
 * @vrooliComponentAppliedAt 2026-08-06T03:45:31Z
 * @vrooliComponentSourceSha256 64bb0d9e93f1c07d17a1cb03820f9bf5e13459a54fed9773f161006cc332e9a7
 * @vrooliComponentDriftHash 6a3d3ecbdc68b00343a5dc149717c054aaf5d7f04e56414750afdfdf259c9513
 * @vrooliComponentTokenTranslation bg-app-surface->bg-app-surface,border-app-border->border-app-border,ring-app-primary/50->ring-app-primary/50,text-app-foreground->text-app-foreground,text-app-muted-foreground->text-app-muted-foreground
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import { forwardRef, type InputHTMLAttributes } from "react";
export const INPUT_MODES = ["controlled", "uncontrolled"] as const;
export const INPUT_SIZES = ["sm", "md", "lg"] as const;
export const INPUT_TONES = ["default", "invalid"] as const;
export const INPUT_PARTS = ["prefix", "control", "suffix"] as const;

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
