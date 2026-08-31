/**
 * @libraryId react-component-library:ResizeHandle
 * @displayName Resize Handle
 * @version 1.0.3
 * @tags ["manipulation","interaction","token-bound","accessibility"]
 * @deps {"react":"^18","react-component-library:useResizablePanel":"^1.0.0","react-component-library:useLocale":"^1.1.0","react-component-library:StyleSheet":"^1.0.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:ResizeHandle */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import type {
  ResizeSeparatorProps,
  ResizeValueTextContext,
} from "@vrooli/react-component-library/useResizablePanel/1";
import { forwardRef } from "react";

export const resizeHandleStyles = `
[data-rcl-resize-handle] {
  --rcl-resize-handle-size: var(--space-xs, 12px);
  position: absolute;
  z-index: var(--layer-sticky, 100);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 0;
  background: transparent;
  forced-color-adjust: none;
}
[data-rcl-resize-handle][data-axis="inline"] { inset-block: 0; inline-size: var(--rcl-resize-handle-size); cursor: col-resize; }
[data-rcl-resize-handle][data-axis="inline"][data-edge="end"] { inset-inline-end: calc(var(--rcl-resize-handle-size) / -2); }
[data-rcl-resize-handle][data-axis="inline"][data-edge="start"] { inset-inline-start: calc(var(--rcl-resize-handle-size) / -2); }
[data-rcl-resize-handle][data-axis="block"] { inset-inline: 0; block-size: var(--rcl-resize-handle-size); cursor: row-resize; }
[data-rcl-resize-handle][data-axis="block"][data-edge="end"] { inset-block-end: calc(var(--rcl-resize-handle-size) / -2); }
[data-rcl-resize-handle][data-axis="block"][data-edge="start"] { inset-block-start: calc(var(--rcl-resize-handle-size) / -2); }
[data-rcl-resize-handle][aria-disabled="true"] { cursor: default; }

[data-rcl-resize-handle] .rcl-resize-handle__bar {
  display: block;
  background: transparent;
  border-radius: var(--radius-control, 0.375rem);
  transition: background-color var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)),
              transform var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1));
}
[data-rcl-resize-handle][data-axis="inline"] .rcl-resize-handle__bar { inline-size: 1px; block-size: 100%; }
[data-rcl-resize-handle][data-axis="block"] .rcl-resize-handle__bar { block-size: 1px; inline-size: 100%; }

[data-rcl-resize-handle]:hover .rcl-resize-handle__bar,
[data-rcl-resize-handle][data-dragging="true"] .rcl-resize-handle__bar { background: var(--color-primary, #2563eb); }
[data-rcl-resize-handle][data-axis="inline"]:hover .rcl-resize-handle__bar,
[data-rcl-resize-handle][data-axis="inline"][data-dragging="true"] .rcl-resize-handle__bar { transform: scaleX(2); }
[data-rcl-resize-handle][data-axis="block"]:hover .rcl-resize-handle__bar,
[data-rcl-resize-handle][data-axis="block"][data-dragging="true"] .rcl-resize-handle__bar { transform: scaleY(2); }

[data-rcl-resize-handle][data-snapped="true"] .rcl-resize-handle__bar { background: var(--color-success, #16a34a); }
[data-rcl-resize-handle][data-collapsed="true"] .rcl-resize-handle__bar { background: var(--color-border, #cbd5e1); }
[data-rcl-resize-handle][aria-disabled="true"] .rcl-resize-handle__bar { background: transparent; }

`;
export interface ResizeHandleProps {
  /** The separator bag returned by `useResizablePanel`. */
  separatorProps: ResizeSeparatorProps;
  className?: string;
  /**
   * Overrides the test id so an adopting scenario can keep the selector it
   * already registered instead of re-implementing the element.
   */
  testId?: string;
}

const cn = (...inputs: Array<string | undefined>) => inputs.filter(Boolean).join(" ");

const fill = (template: string, values: Record<string, string | number>) =>
  template.replace(/\{(\w+)\}/g, (match, key: string) =>
    key in values ? String(values[key]) : match,
  );

export interface ResizeStrings {
  label: (panelName: string) => string;
  valueText: (context: ResizeValueTextContext) => string;
}

/**
 * The accessible sentences a resizable region needs, resolved through the
 * library's translation bridge. Pass the results to `useResizablePanel` as
 * `label` and `formatValueText` — the hook stays free of string state, and the
 * consumer never has to invent the wording.
 */
export function useResizeStrings(): ResizeStrings {
  const t = useStrings();
  return {
    label: (panelName) =>
      fill(t("manipulation.resize-handle.label", "Resize {panel}"), { panel: panelName }),
    valueText: ({ size, panelName, isSnapped, isCollapsed }) => {
      if (isCollapsed) {
        return fill(t("manipulation.resize-handle.valuetext.collapsed", "{panel} collapsed"), {
          panel: panelName,
        });
      }
      const key = isSnapped
        ? "manipulation.resize-handle.valuetext.snapped"
        : "manipulation.resize-handle.valuetext";
      const fallback = isSnapped ? "{panel} {size} pixels, snapped" : "{panel} {size} pixels";
      return fill(t(key, fallback), { panel: panelName, size });
    },
  };
}

/**
 * Appearance only. The hit target is deliberately far wider than the seam it
 * draws — the size comes from one token that the resize arithmetic also reads,
 * so the target can never drift from the number the clamp reserves for it.
 */
export const ResizeHandle = forwardRef<HTMLDivElement, ResizeHandleProps>(function ResizeHandle(
  { separatorProps, className, testId = "manipulation.resize-handle" },
  ref,
) {
  return (
    <>
      <StyleSheet name="resize-handle-1-0-0" css={resizeHandleStyles} />
      <div
        {...separatorProps}
        ref={ref}
        data-testid={testId}
        data-rcl-resize-handle=""
        className={cn("rcl-resize-handle", className)}
      >
        <span aria-hidden="true" className="rcl-resize-handle__bar" />
      </div>
    </>
  );
});
