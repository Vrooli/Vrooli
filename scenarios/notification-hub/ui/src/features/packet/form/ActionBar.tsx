import type { ReactNode } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";

import { PinnedBar } from "./PinnedBar";

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
 * The main action of the decision, pinned to the bottom edge so a thumb reaches
 * it without scrolling. It is one row. While the action is unavailable the
 * reason sits beside a compact button instead of above a full-width one, and
 * the button takes the whole row once it can be pressed. The quieter actions
 * live in {@link SecondaryActions}, so the bar stays small on a phone.
 */
export function ActionBar({ primary, back, hint, disabled }: ActionBarProps) {
  return (
    <PinnedBar>
      <div className="flex items-center gap-2">
        {back && (
          <Button className="shrink-0 border border-app-border bg-app-surface" size="lg" type="button" variant="outline" disabled={disabled} onClick={back.onClick}>
            {back.label}
          </Button>
        )}
        {hint && <p id="packet-action-hint" role="status" className="min-w-0 flex-1 text-sm leading-snug text-app-muted-foreground">{hint}</p>}
        <Button
          data-testid={primary.testId}
          aria-describedby={hint ? "packet-action-hint" : undefined}
          className={hint ? "shrink-0" : "flex-1"}
          size="lg"
          type="button"
          variant="primary"
          disabled={disabled || primary.disabled}
          onClick={primary.onClick}
        >
          {primary.label}
        </Button>
      </div>
    </PinnedBar>
  );
}
