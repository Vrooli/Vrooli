import { OverlayActionBar } from "@vrooli/react-component-library/OverlayActionBar/1";

/** One line saying what is about to happen, and its shared overlay action. */
export function LaunchFooter({ summary, label, buttonLabel, disabled, busy, busyLabel, reason, onLaunch, testId }: {
  summary: string;
  label: string;
  buttonLabel: string;
  disabled: boolean;
  busy: boolean;
  busyLabel: string;
  /** Why the button is disabled, shown beside it. Never leave a dead button unexplained. */
  reason?: string;
  onLaunch: () => void;
  testId: string;
}) {
  const showReason = !busy && disabled && Boolean(reason);
  return (
    <div className="grid w-full gap-1.5" data-testid="launcher-footer">
      <p data-testid="launcher-summary" title={summary} className="truncate text-center text-xs text-wc-text-muted">{summary}</p>
      {showReason && <p data-testid="launcher-disabled-reason" role="status" className="break-words text-center text-xs leading-[18px] text-wc-warning">{reason}</p>}
      <OverlayActionBar
        label={showReason ? `${label}. ${reason}` : label}
        maxVisible={1}
        maxLabelLength={28}
        actions={[{
          id: "launch",
          label: busy ? busyLabel : buttonLabel,
          onSelect: onLaunch,
          disabled: disabled || busy,
          primary: true,
          testId,
        }]}
      />
    </div>
  );
}
