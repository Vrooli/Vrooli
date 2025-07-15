import { type SxProps, type Theme } from "@mui/material/styles";

// External apps configuration
export const findRoutineLimitTo = ["RoutineMultiStep", "RoutineSingleStep"] as const;

// Dimensions
export const iconHeight = 32;
export const iconWidth = 32;
export const MAX_ROWS_EXPANDED = 50;
export const MIN_ROWS_EXPANDED = 5;
export const MAX_ROWS_COLLAPSED = 6;
export const MIN_ROWS_COLLAPSED = 1;

// Trigger characters
export const TRIGGER_CHARS = {
    AT: "@",
    SLASH: "/",
} as const;

// Class names
export const toolChipIconButtonClassName = "tw-p-0 tw-pr-0.5" as const;
export const toolbarIconButtonClassName = "tw-p-1 tw-opacity-50" as const;

// Selector for non-focusable elements
export const NON_FOCUSABLE_SELECTOR = [
    ".toolbar-button",
    ".MuiTab-root",
    ".AdvancedInputToolbar",  // The toolbar itself
    "[aria-haspopup=\"true\"]", // Elements with popups/dropdowns
    // Context elements
    ".context-item",
    ".task-item",
    // File dropzone elements
    ".dropzone",
    // Specific elements we know about
    "[data-toolbar-button=\"true\"]",
].join(",");

// Styles
export const toolbarRowStyles: SxProps<Theme> = {
    display: "flex",
    flexDirection: "row",
    alignItems: "center",
    gap: 1,
    flexWrap: "wrap",
    paddingTop: 1,
    paddingBottom: 0.5,
};

export const bottomRowStyles: SxProps<Theme> = {
    display: "flex",
    alignItems: "center",
};

export const popoverAnchorOrigin = { vertical: "top", horizontal: "left" } as const;
export const popoverTransformOrigin = { vertical: "bottom", horizontal: "left" } as const;

export const verticalMiddleStyle = { verticalAlign: "middle" } as const;
export const dividerStyle = { my: 1, opacity: 0.2 } as const;
