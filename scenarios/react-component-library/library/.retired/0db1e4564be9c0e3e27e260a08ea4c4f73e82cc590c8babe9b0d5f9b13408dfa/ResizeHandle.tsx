/**
 * @libraryId react-component-library:ResizeHandle
 * @displayName Resize Handle
 * @description The drag affordance for a resizable region: a wide pointer target carrying a thin visual seam, operable by pointer and keyboard alike.
 * @version 1.0.2
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

import { resizeHandleStyles } from "./styles";

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

const cn = (...inputs: Array<string | undefined>) =>
  inputs.filter(Boolean).join(" ");

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
      fill(t("manipulation.resize-handle.label", "Resize {panel}"), {
        panel: panelName,
      }),
    valueText: ({ size, panelName, isSnapped, isCollapsed }) => {
      if (isCollapsed) {
        return fill(
          t(
            "manipulation.resize-handle.valuetext.collapsed",
            "{panel} collapsed",
          ),
          {
            panel: panelName,
          },
        );
      }
      const key = isSnapped
        ? "manipulation.resize-handle.valuetext.snapped"
        : "manipulation.resize-handle.valuetext";
      const fallback = isSnapped
        ? "{panel} {size} pixels, snapped"
        : "{panel} {size} pixels";
      return fill(t(key, fallback), { panel: panelName, size });
    },
  };
}

/**
 * Appearance only. The hit target is deliberately far wider than the seam it
 * draws — the size comes from one token that the resize arithmetic also reads,
 * so the target can never drift from the number the clamp reserves for it.
 */
export const ResizeHandle = forwardRef<HTMLDivElement, ResizeHandleProps>(
  function ResizeHandle(
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
  },
);
