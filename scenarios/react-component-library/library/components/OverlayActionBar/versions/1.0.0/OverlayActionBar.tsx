/**
 * @libraryId react-component-library:OverlayActionBar
 * @displayName OverlayActionBar
 * @description One row, one primary action, shortened labels, and an overflow menu for additional overlay actions.
 * @version 1.0.0
 * @tags ["overlay","actions","responsive","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { useState, type CSSProperties } from "react";
import overlayActionBarContract from "./usage-contract.json";

const runtimeGlobal: typeof globalThis & { process?: { env?: { NODE_ENV?: string } } } = globalThis;
// Browser globals can omit Node's process/env despite Node typings.
// eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
const isDevelopment = runtimeGlobal.process?.env?.NODE_ENV !== "production";

export interface OverlayAction {
  id: string;
  label: string;
  onSelect: () => void;
  disabled?: boolean;
  primary?: boolean;
  testId?: string;
}

export type OverlayPrimaryAction = OverlayAction & { primary: true };
export type OverlaySecondaryAction = OverlayAction & { primary?: false };
export type OverlayActionList = readonly [OverlayPrimaryAction, ...OverlaySecondaryAction[]];

/** Single source of truth used by the runtime check and static contract rule. */
export const OVERLAY_ACTION_BAR_CONTRACT = overlayActionBarContract;

export interface OverlayActionBarProps {
  actions: OverlayActionList;
  label?: string;
  maxVisible?: number;
  maxLabelLength?: number;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-overlay-action-bar] { display:flex; align-items:center; justify-content:flex-end; gap:var(--space-xs); flex-wrap:nowrap; min-inline-size:0; inline-size:100%; }
[data-rcl-overlay-action-bar] > button, [data-rcl-overlay-action-bar] [data-overflow-trigger] { box-sizing:border-box; min-block-size:var(--tap-target-min); max-inline-size:100%; padding-inline:var(--space-sm); border:1px solid var(--color-border-strong); border-radius:var(--radius-control); background:var(--color-surface); color:var(--color-foreground); font:var(--text-label); white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
[data-rcl-overlay-action-bar] > [data-primary="true"] { border-color:var(--color-primary); background:var(--color-primary); color:var(--color-primary-foreground); }
[data-rcl-overlay-action-bar] [data-overflow] { position:relative; flex:0 0 auto; }
[data-rcl-overlay-action-bar] [data-overflow-menu] { position:absolute; inset-block-end:calc(100% + var(--space-2xs)); inset-inline-end:0; z-index:var(--layer-popover); display:grid; min-inline-size:12rem; max-inline-size:min(20rem, 90vw); gap:var(--space-3xs); padding:var(--space-2xs); border:1px solid var(--color-border); border-radius:var(--radius-control); background:var(--color-surface-raised); box-shadow:var(--elev-floating); }
[data-rcl-overlay-action-bar] [data-overflow-menu] button { min-block-size:var(--tap-target-min); max-inline-size:100%; padding:var(--space-xs); border:0; border-radius:var(--radius-control); background:transparent; color:var(--color-foreground); text-align:start; white-space:normal; overflow-wrap:anywhere; font:var(--text-label); }
[data-rcl-overlay-action-bar] [data-overflow-menu] button:hover { background:var(--color-surface-muted); }
[data-rcl-overlay-action-bar] button:focus-visible { outline:3px solid var(--color-focus); outline-offset:2px; }
@media (max-width:34rem) { [data-rcl-overlay-action-bar] { gap:var(--space-2xs); } [data-rcl-overlay-action-bar] > button { flex:1 1 auto; min-inline-size:0; } }
`;

function shortened(label: string, max: number) {
  const chars = Array.from(label.trim());
  return chars.length > max ? `${chars.slice(0, Math.max(1, max - 1)).join("")}…` : label;
}

function warnContract(actions: readonly OverlayAction[], maxVisible: number, maxLabelLength: number) {
  if (!isDevelopment) return;
  const violations: string[] = [];
  if (actions.length > OVERLAY_ACTION_BAR_CONTRACT.items.maximumCount) violations.push(`received ${actions.length} actions; maximum is ${OVERLAY_ACTION_BAR_CONTRACT.items.maximumCount} (typical ${OVERLAY_ACTION_BAR_CONTRACT.items.typicalCount})`);
  if (maxVisible > OVERLAY_ACTION_BAR_CONTRACT.layout.overflowAfterVisibleCount) violations.push(`maxVisible ${maxVisible} exceeds overflow limit ${OVERLAY_ACTION_BAR_CONTRACT.layout.overflowAfterVisibleCount}`);
  if (maxLabelLength > OVERLAY_ACTION_BAR_CONTRACT.items.maximumLabelCharacters) violations.push(`maxLabelLength ${maxLabelLength} exceeds ${OVERLAY_ACTION_BAR_CONTRACT.items.maximumLabelCharacters}`);
  const ids = new Set<string>();
  for (const [index, action] of actions.entries()) {
    const fields = action as unknown as Record<string, unknown>;
    for (const field of OVERLAY_ACTION_BAR_CONTRACT.items.requiredFields) {
      const value = fields[field];
      const expectedType = OVERLAY_ACTION_BAR_CONTRACT.items.requiredFieldTypes[field as "id" | "label" | "onSelect"];
      const valid = expectedType === "function"
        ? typeof value === "function"
        : expectedType === "string" && typeof value === "string" && value.trim().length > 0;
      if (!valid) {
        violations.push(`action ${index + 1} has no ${field}`);
      }
    }
    if (typeof action.id === "string" && action.id.trim()) {
      if (ids.has(action.id)) violations.push(`duplicate action id ${action.id}`);
      ids.add(action.id);
    }
    const label = fields.label;
    if (typeof label === "string" && Array.from(label).length > OVERLAY_ACTION_BAR_CONTRACT.items.maximumLabelCharacters) violations.push(`action ${action.id || index + 1} label exceeds ${OVERLAY_ACTION_BAR_CONTRACT.items.maximumLabelCharacters} characters`);
  }
  if (actions.filter((action) => action[OVERLAY_ACTION_BAR_CONTRACT.items.primaryField as "primary"]).length !== OVERLAY_ACTION_BAR_CONTRACT.items.primaryCount) violations.push(`exactly ${OVERLAY_ACTION_BAR_CONTRACT.items.primaryCount} action must be primary`);
  if (violations.length) console.warn(`[RCL usage contract] OverlayActionBar: ${violations.join("; ")}`, { component: "OverlayActionBar", count: actions.length, maxVisible });
}

export function OverlayActionBar({ actions, label = "Actions", maxVisible = OVERLAY_ACTION_BAR_CONTRACT.layout.overflowAfterVisibleCount, maxLabelLength = OVERLAY_ACTION_BAR_CONTRACT.items.typicalLabelCharacters, className, style }: OverlayActionBarProps) {
  // Public tuning props may make the component more conservative, but may not
  // weaken the contract and let extra buttons or overlong labels escape.
  const visibleLimit = Math.max(1, Math.min(maxVisible, OVERLAY_ACTION_BAR_CONTRACT.layout.overflowAfterVisibleCount));
  const labelLimit = Math.max(1, Math.min(maxLabelLength, OVERLAY_ACTION_BAR_CONTRACT.items.maximumLabelCharacters));
  warnContract(actions, maxVisible, maxLabelLength);
  const [open, setOpen] = useState(false);
  const primary = (actions.find((action) => action.primary) ?? actions[0]) as OverlayAction | undefined;
  const visible = primary ? [primary, ...actions.filter((action) => action.id !== primary.id)].slice(0, visibleLimit) : [];
  const overflow = actions.filter((action) => !visible.some((item) => item.id === action.id));
  return (
    <>
      <StyleSheet libraryId="react-component-library:OverlayActionBar" version="1.0.0" css={styles} />
      <div className={className} style={style} data-rcl-overlay-action-bar role="group" aria-label={label}>
        {visible.map((action) => {
          const fields = action as unknown as Record<string, unknown>;
          const actionLabel = typeof fields.label === "string" ? fields.label : "";
          return (
            <button key={action.id} type="button" data-testid={action.testId} data-action-id={action.id} data-primary={action.primary || (!actions.some((item) => item.primary) && action.id === primary?.id) ? "true" : "false"} aria-label={actionLabel} title={actionLabel} disabled={action.disabled} onClick={action.onSelect}>
              {shortened(actionLabel, labelLimit)}
            </button>
          );
        })}
        {overflow.length > 0 && (
          <div data-overflow>
            <button type="button" data-overflow-trigger aria-expanded={open} aria-haspopup="menu" onClick={() => setOpen((value) => !value)}>
              More <span className="sr-only">{overflow.length} actions</span>
            </button>
            {open && <div data-overflow-menu role="menu">{overflow.map((action) => <button key={action.id} type="button" role="menuitem" data-action-id={action.id} disabled={action.disabled} onClick={() => { setOpen(false); action.onSelect(); }}>{action.label}</button>)}</div>}
          </div>
        )}
      </div>
    </>
  );
}
