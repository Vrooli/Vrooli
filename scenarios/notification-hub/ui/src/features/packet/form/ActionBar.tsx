import type { ReactNode } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";

export interface ActionBarProps {
  /** The one main action: Send answer, Next, or Answer. */
  primary: { label: string; onClick: () => void; disabled?: boolean; testId?: string };
  /** A step back (previous question). */
  back?: { label: string; onClick: () => void };
  /** Why the main action is unavailable, said where the operator is looking. */
  hint?: ReactNode;
  disabled: boolean;
}

/**
 * The main action of the decision, pinned to the bottom edge with the
 * safe-area inset so a thumb reaches it without scrolling. It must be a direct
 * child of the tall content it pins within (a sticky bar cannot leave its
 * parent), and it holds only the one filled button: the quieter actions live
 * in {@link SecondaryActions}, so the pinned bar stays small on a phone.
 */
export function ActionBar({ primary, back, hint, disabled }: ActionBarProps) {
  return (
    <footer className="sticky bottom-0 z-10 -mx-4 border-t border-app-border bg-app-surface/95 px-4 pt-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] backdrop-blur sm:-mx-8 sm:px-8 lg:mx-0 lg:rounded-b-panel lg:px-4">
      <div className="flex flex-col gap-2">
        {hint && <p id="packet-action-hint" role="status" className="text-sm text-app-muted-foreground">{hint}</p>}
        <div className="flex gap-2">
          {back && (
            <Button className="min-h-12 shrink-0 border border-app-border bg-app-surface" type="button" variant="outline" disabled={disabled} onClick={back.onClick}>
              {back.label}
            </Button>
          )}
          <Button
            data-testid={primary.testId}
            aria-describedby={hint ? "packet-action-hint" : undefined}
            className="min-h-12 flex-1"
            type="button"
            variant="primary"
            disabled={disabled || primary.disabled}
            onClick={primary.onClick}
          >
            {primary.label}
          </Button>
        </div>
      </div>
    </footer>
  );
}
