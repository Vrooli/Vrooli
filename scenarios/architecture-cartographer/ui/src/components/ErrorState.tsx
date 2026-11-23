import { cn } from "../lib/utils";
import { selectors } from "../consts/selectors";
import { Button } from "@vrooli/react-component-library/Button/2.2.14";

export interface ErrorStateProps {
  /** Pre-translated title. */
  title: string;
  /** Pre-translated message body (e.g. via decodeApiError). */
  message: string;
  /** Pre-translated retry label. If undefined, no retry affordance renders. */
  retryLabel?: string;
  /** Retry handler; required when retryLabel is set. */
  onRetry?: () => void;
  className?: string;
}

export function ErrorState({ title, message, retryLabel, onRetry, className }: ErrorStateProps) {
  return (
    <div
      data-testid={selectors.shared.errorState.root}
      role="alert"
      className={cn(
        "flex flex-col gap-2 rounded-panel border border-app-danger/40 bg-app-danger/10 p-4 text-app-foreground backdrop-blur-sm",
        className,
      )}
    >
      <p
        data-testid={selectors.shared.errorState.title}
        className="text-sm font-semibold text-app-danger"
      >
        {title}
      </p>
      <p data-testid={selectors.shared.errorState.message} className="text-sm">
        {message}
      </p>
      {retryLabel && onRetry ? (
        <div>
          <Button
            data-testid={selectors.shared.errorState.retryButton}
            variant="ghost"
            size="sm"
            onClick={onRetry}
            shape="square"
            className={cn("border border-app-border text-app-foreground hover:bg-app-surface-muted")}
          >
            {retryLabel}
          </Button>
        </div>
      ) : null}
    </div>
  );
}
