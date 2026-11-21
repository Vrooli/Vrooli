import type { InputHTMLAttributes, ReactNode } from "react";

import { Chip } from "../Chip";

interface ChoiceCardProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "type" | "children"> {
  type: "radio" | "checkbox";
  label: ReactNode;
  detail?: ReactNode;
  recommendedLabel?: string;
}

/**
 * One selectable answer, drawn as a card. The native input stays in the tree
 * (visually hidden) so a radio group or checkbox set keeps its keyboard and
 * screen-reader behaviour; the card is the whole tap target.
 *
 * Label and detail are requester-authored text in whatever script the requester
 * used, which need not match the page's direction. `dir="auto"` isolates each
 * run so an English option in an Arabic page (or the reverse) keeps its
 * punctuation and alignment.
 */
export function ChoiceCard({ type, label, detail, recommendedLabel, className = "", ...input }: ChoiceCardProps) {
  return (
    <label
      className={`group flex min-h-12 cursor-pointer items-start gap-3 rounded-panel border border-app-border bg-app-surface p-3 transition-colors has-[:checked]:border-app-primary has-[:checked]:bg-app-primary/10 has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-app-focus has-[:disabled]:cursor-default has-[:disabled]:opacity-70 ${className}`}
    >
      <input type={type} className="peer sr-only" {...input} />
      <span
        aria-hidden="true"
        className={`mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center border-2 border-app-muted-foreground bg-app-surface peer-checked:border-app-primary peer-checked:bg-app-primary ${type === "radio" ? "rounded-pill" : "rounded-sm"} after:block after:h-2 after:w-2 after:scale-0 after:bg-app-primary-foreground after:transition-transform peer-checked:after:scale-100 ${type === "radio" ? "after:rounded-pill" : "after:rounded-[1px]"}`}
      />
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="flex flex-wrap items-center gap-x-2 gap-y-1 font-medium leading-snug">
          <span dir="auto" className="min-w-0 break-words">{label}</span>
          {recommendedLabel && <Chip tone="attention">{recommendedLabel}</Chip>}
        </span>
        {detail && <span dir="auto" className="break-words text-sm text-app-muted-foreground">{detail}</span>}
      </span>
    </label>
  );
}
