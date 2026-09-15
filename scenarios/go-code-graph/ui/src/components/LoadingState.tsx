import { cn } from "../lib/utils";
import { selectors } from "../consts/selectors";

export interface LoadingStateProps {
  /** Pre-translated aria-label. Use `t(strings.shared.loading.label)`. */
  label: string;
  className?: string;
}

export function LoadingState({ label, className }: LoadingStateProps) {
  return (
    <div
      data-testid={selectors.shared.loadingState.root}
      role="status"
      aria-live="polite"
      aria-label={label}
      className={cn(
        "flex items-center justify-center gap-2 rounded-panel border border-app-border bg-app-surface p-6 text-app-muted-foreground backdrop-blur-sm",
        className,
      )}
    >
      <span
        aria-hidden="true"
        className="h-4 w-4 animate-spin rounded-full border-2 border-app-border border-t-app-primary"
      />
      <span className="text-sm">{label}</span>
    </div>
  );
}
