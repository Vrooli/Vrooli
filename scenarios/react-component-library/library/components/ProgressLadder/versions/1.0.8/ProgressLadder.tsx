/**
 * @libraryId react-component-library:ProgressLadder
 * @displayName ProgressLadder
 * @description A rung-by-rung maturity ladder showing achieved and target states.
 * @version 1.0.8
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:ProgressLadder */
import type { HTMLAttributes } from "react";
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

export type ProgressRungState = "pending" | "active" | "complete" | "failed";
export interface ProgressRung {
  id: string;
  label: string;
  complete: boolean;
  current?: boolean;
  state?: ProgressRungState;
  detail?: string;
  durationMs?: number;
}
export interface ProgressLadderProps extends HTMLAttributes<HTMLOListElement> {
  rungs?: ProgressRung[];
  orientation?: "vertical" | "horizontal";
}
export const progressLadderStyles = `
[data-rcl-progress-ladder] { display: grid; gap: var(--space-2xs); margin: 0; padding: 0; list-style: none; }
[data-rcl-progress-ladder][data-orientation="vertical"] [data-rcl-progress-rung] { position: relative; display: grid; grid-template-columns: var(--control-size-md) minmax(0, 1fr); gap: var(--space-xs); align-items: start; min-block-size: var(--control-size-md); }
[data-rcl-progress-ladder][data-orientation="vertical"] [data-rcl-progress-rung]:not(:last-child)::before { content: ""; position: absolute; inset-block-start: var(--control-size-md); inset-inline-start: calc(var(--control-size-md) / 2); inline-size: var(--border-hairline); block-size: calc(100% + var(--space-2xs)); background: var(--rcl-progress-rail, var(--color-border)); }
[data-rcl-progress-ladder][data-orientation="horizontal"] { display: flex; align-items: center; gap: 0; }
[data-rcl-progress-ladder][data-orientation="horizontal"] [data-rcl-progress-rung] { position: relative; display: grid; flex: 1 1 0; justify-items: center; min-inline-size: var(--control-size-md); }
[data-rcl-progress-ladder][data-orientation="horizontal"] [data-rcl-progress-rung]:not(:last-child)::before { content: ""; position: absolute; inset-block-start: calc(var(--control-size-md) / 2); inset-inline-start: 50%; inline-size: 100%; block-size: var(--border-hairline); background: var(--rcl-progress-rail, var(--color-border)); }
[data-rcl-progress-marker] { position: relative; z-index: 1; display: grid; place-items: center; inline-size: var(--control-size-md); block-size: var(--control-size-md); box-sizing: border-box; border: var(--border-medium) solid var(--rcl-progress-tone, var(--color-border)); border-radius: var(--radius-pill); background: var(--color-surface); color: var(--rcl-progress-tone, var(--color-border)); font: var(--text-label); transition: box-shadow var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard), background var(--dur-quick) var(--ease-standard); }
[data-rcl-progress-rung][data-state="complete"] { --rcl-progress-tone: var(--color-success); --rcl-progress-rail: var(--color-success); }
[data-rcl-progress-rung][data-state="active"] { --rcl-progress-tone: var(--color-primary); }
[data-rcl-progress-rung][data-state="failed"] { --rcl-progress-tone: var(--color-danger); }
[data-rcl-progress-rung][data-state="active"] [data-rcl-progress-marker] { box-shadow: 0 0 0 var(--space-3xs) color-mix(in srgb, var(--color-primary) 22%, transparent); }
[data-rcl-progress-rung][data-state="complete"] [data-rcl-progress-marker] { background: var(--color-success); color: var(--color-surface); }
[data-rcl-progress-rung][data-state="failed"] [data-rcl-progress-marker] { background: var(--color-danger); color: var(--color-surface); }
[data-rcl-progress-mark] { inline-size: .45em; block-size: .25em; border-inline-start: 2px solid currentColor; border-block-end: 2px solid currentColor; transform: rotate(-45deg) translateY(-.08em); }
[data-rcl-progress-fail] { inline-size: .55em; block-size: .55em; position: relative; }
[data-rcl-progress-fail]::before, [data-rcl-progress-fail]::after { content: ""; position: absolute; inset: 50% auto auto 0; inline-size: 100%; block-size: 2px; background: currentColor; transform: rotate(45deg); }
[data-rcl-progress-fail]::after { transform: rotate(-45deg); }
[data-rcl-progress-label] { display: grid; gap: var(--space-3xs); min-inline-size: 0; color: var(--color-foreground); }
[data-rcl-progress-detail] { color: var(--color-muted-foreground); font-size: var(--font-size-sm); }
[data-rcl-progress-duration] { color: var(--color-muted-foreground); font-variant-numeric: tabular-nums; justify-self: end; }
[data-rcl-progress-ladder][data-orientation="horizontal"] [data-rcl-progress-label], [data-rcl-progress-ladder][data-orientation="horizontal"] [data-rcl-progress-duration] { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
@media (prefers-reduced-motion: reduce) { [data-rcl-progress-marker] { transition: none; } }
`;
export const ProgressLadder = withClassName(function ProgressLadder({
  rungs = [],
  orientation = "vertical",
  className,
  ...props
}: ProgressLadderProps) {
  const strings = useStrings();
  useLibraryStyleSheet("progress-ladder", progressLadderStyles);
  return (
    <ol
      {...props}
      className={className}
      data-rcl-progress-ladder
      data-orientation={orientation}
      data-testid="data-display.progress-ladder"
      aria-label={strings("data-display.progress-ladder.progress-ladder", "Progress ladder")}
    >
      {rungs.map((rung, index) => {
        const state =
          rung.state ?? (rung.complete ? "complete" : rung.current ? "active" : "pending");
        return (
          <li
            key={rung.id}
            data-rcl-progress-rung
            data-state={state}
            data-complete={state === "complete" || undefined}
            aria-current={rung.current || state === "active" ? "step" : undefined}
          >
            <span data-rcl-progress-marker aria-hidden="true">
              {state === "complete" ? (
                <span data-rcl-progress-mark />
              ) : state === "failed" ? (
                <span data-rcl-progress-fail />
              ) : (
                index + 1
              )}
            </span>
            <span data-rcl-progress-label>
              <span>{rung.label}</span>
              {rung.detail && <span data-rcl-progress-detail>{rung.detail}</span>}
            </span>
            {rung.durationMs !== undefined && (
              <span data-rcl-progress-duration>{rung.durationMs}ms</span>
            )}
          </li>
        );
      })}
    </ol>
  );
});
