import { cn } from "../lib/utils";

/** The look of one chip, shared by the link chips and the toggle chips so every filter reads as one family. */
export const chipClass = (selected: boolean) => cn(
  "flex min-h-11 items-center rounded-pill border px-4 text-sm font-medium transition-colors duration-quick focus-visible:outline focus-visible:outline-2",
  selected
    ? "border-app-primary bg-app-primary text-app-primary-foreground"
    : "border-app-border bg-app-surface text-app-foreground hover:bg-app-surface-muted",
);
