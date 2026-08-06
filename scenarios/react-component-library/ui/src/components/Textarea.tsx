/**
 * @vrooliComponentSource react-component-library:Textarea
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption a5fbb27a-b093-49b1-a813-b0240181e659
 * @vrooliComponentAppliedAt 2026-08-06T03:45:30Z
 * @vrooliComponentSourceSha256 cf1e68fe3800b454eb473bcd1c0c18046249bcfb9e4e94214b4cdd9ce1268d02
 * @vrooliComponentDriftHash 0455997ff9dae410c4b3c149c35dc500ee9595372d19eab62b0d0bde4e0658d6
 * @vrooliComponentTokenTranslation bg-app-surface->bg-app-surface,border-app-border->border-app-border,ring-app-primary/50->ring-app-primary/50,text-app-foreground->text-app-foreground,text-app-muted-foreground->text-app-muted-foreground
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { clsx, type ClassValue } from "clsx";
import * as React from "react";
import { twMerge } from "tailwind-merge";

export type TextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, ...props }, ref) => {
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
  },
);
Textarea.displayName = "Textarea";

export { Textarea };
