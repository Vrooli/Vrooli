import type { Ref } from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";

/** One line saying what is about to happen, and the button that does it. */
export function LaunchFooter({ summary, label, disabled, busy, busyLabel, onLaunch, testId, buttonRef }: {
  buttonRef?: Ref<HTMLButtonElement>;
  summary: string;
  label: string;
  disabled: boolean;
  busy: boolean;
  busyLabel: string;
  onLaunch: () => void;
  testId: string;
}) {
  return (
    <div className="grid w-full gap-1.5" data-testid="launcher-footer">
      <p data-testid="launcher-summary" title={summary} className="truncate text-center text-xs text-wc-text-muted">{summary}</p>
      <Button ref={buttonRef} type="button" variant="primary" size="lg" shape="square" data-testid={testId} disabled={disabled || busy} onClick={onLaunch} className="w-full font-semibold">
        {busy ? <span data-testid="launcher-creating" className="flex items-center justify-center gap-2"><Loader2 className="h-4 w-4 animate-spin" aria-hidden />{busyLabel}</span> : label}
      </Button>
    </div>
  );
}
