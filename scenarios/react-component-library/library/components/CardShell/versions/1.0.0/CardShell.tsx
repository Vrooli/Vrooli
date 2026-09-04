/** @libraryId react-component-library:CardShell */
/* eslint-disable jsx-a11y/no-static-element-interactions, jsx-a11y/no-noninteractive-tabindex */
/** @vrooliComponentSource data-display.card-shell */
import { type KeyboardEvent, type ReactNode } from "react";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { SelectionControl } from "@vrooli/react-component-library/SelectionControl/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { useStrings } from "@vrooli/react-component-library/useLocale/1";

export interface RowSelection {
  selectionMode: boolean;
  selected: boolean;
  disabled?: boolean;
  disabledReason?: string;
  onToggleSelect?: () => void;
}

export interface CardShellProps {
  children: ReactNode;
  selection?: RowSelection;
  isCursor?: boolean;
  onPress?: () => void;
  actionsSlot?: ReactNode;
  className?: string;
  testId?: string;
  interactive?: boolean;
  selectLabel?: string;
}

const css = `
[data-rcl-card-shell] { position: relative; display: grid; inline-size: 100%; min-inline-size: 0; grid-template-columns: minmax(0, 1fr); align-items: stretch; border: var(--border-hairline) solid transparent; border-radius: var(--radius-card); transition: box-shadow var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
[data-rcl-card-shell][data-selection-mode=true] { grid-template-columns: auto minmax(0, 1fr); }
[data-rcl-card-shell][data-has-actions=true] { grid-template-columns: minmax(0, 1fr) auto; }
[data-rcl-card-shell][data-selection-mode=true][data-has-actions=true] { grid-template-columns: auto minmax(0, 1fr) auto; }
[data-rcl-card-shell][data-selected=true] { border-color: var(--color-primary); box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-primary) 20%, transparent); }
[data-rcl-card-shell][data-cursor=true] { outline: 2px solid var(--color-focus); outline-offset: 2px; }
[data-rcl-card-shell][data-interactive=true]:active { transform: scale(.995); }
[data-rcl-card-shell][data-disabled=true] { opacity: .68; }
[data-rcl-card-shell] [data-rcl-card-content] { inline-size: 100%; min-inline-size: 0; }
[data-rcl-card-shell] [data-rcl-card-actions] { display: flex; align-items: flex-start; justify-content: center; min-inline-size: var(--tap-target-min); }
[data-rcl-card-shell] [data-rcl-card-actions] button { min-block-size: var(--tap-target-min); min-inline-size: var(--tap-target-min); border: 0; background: transparent; color: var(--color-muted-foreground); cursor: pointer; }
[data-rcl-card-shell] [data-rcl-card-actions] button:hover { color: var(--color-foreground); background: var(--color-surface-muted); }
[data-rcl-card-shell] [data-rcl-card-veil] { position: absolute; inset: 0; display: grid; place-items: center; padding: var(--space-sm); background: color-mix(in srgb, var(--color-surface) 78%, transparent); color: var(--color-muted-foreground); pointer-events: none; }
[data-rcl-card-shell] [data-rcl-card-reason] { font: var(--text-caption); text-align: center; }
`;

export const CardShell = withClassName(function CardShell({
  children,
  selection,
  isCursor = false,
  onPress,
  actionsSlot,
  className,
  testId = "data-display.card-shell",
  interactive = true,
  selectLabel,
}: CardShellProps) {
  const strings = useStrings();
  const selectionMode = selection?.selectionMode === true;
  const toggleSelection = selection?.onToggleSelect;
  const handleKeyboard = (event: KeyboardEvent<HTMLDivElement>) => {
    if (!interactive || (event.key !== "Enter" && event.key !== " ")) return;
    event.preventDefault();
    if (selectionMode) toggleSelection?.();
    else onPress?.();
  };

  return (
    <>
      <StyleSheet name="card-shell-1-0-0" css={css} />
      <div
        data-rcl-card-shell
        data-selection-mode={selectionMode || undefined}
        data-has-actions={actionsSlot ? true : undefined}
        data-selected={selection?.selected || undefined}
        data-disabled={selection?.disabled || undefined}
        data-cursor={isCursor || undefined}
        data-interactive={interactive || undefined}
        aria-disabled={selection?.disabled || undefined}
        role={interactive ? "button" : undefined}
        className={className}
        data-testid={testId}
        tabIndex={interactive ? 0 : undefined}
        onClick={interactive ? () => (selectionMode ? toggleSelection?.() : onPress?.()) : undefined}
        onKeyDown={handleKeyboard}
      >
        {selectionMode ? (
          <div data-rcl-card-rail>
            <SelectionControl
              kind="checkbox"
              aria-label={selectLabel ?? strings("data-display.card-shell.select-row", "Select row")}
              checked={selection.selected}
              disabled={selection.disabled}
              onCheckedChange={() => toggleSelection?.()}
            />
          </div>
        ) : null}
        <div data-rcl-card-content>{children}</div>
        {actionsSlot ? <div data-rcl-card-actions>{actionsSlot}</div> : null}
        {selection?.disabled ? (
          <div data-rcl-card-veil>
            {selection.disabledReason ? (
              <span data-rcl-card-reason>{selection.disabledReason}</span>
            ) : null}
          </div>
        ) : null}
      </div>
    </>
  );
});
