import type { HTMLAttributes, ElementType, ReactNode } from "react";
import { forwardRef } from "react";
import { cn } from "../../utils/tailwind-theme.js";

// Export types for external use
export type GridColumns = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | "none";
export type GridRows = 1 | 2 | 3 | 4 | 5 | 6 | "none";
export type GridGap = "none" | "xs" | "sm" | "md" | "lg" | "xl";
export type GridAlign = "start" | "center" | "end" | "stretch";
export type GridJustify = "start" | "center" | "end" | "between" | "around" | "evenly";
export type GridFlow = "row" | "col" | "row-dense" | "col-dense";

// Grid item types
export type GridItemSpan = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | "auto" | "full";
export type GridItemStart = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | 13 | "auto";
export type GridItemEnd = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | 13 | "auto";
export type GridItemAlign = "start" | "center" | "end" | "stretch";
export type GridItemJustify = "start" | "center" | "end" | "stretch";

export interface GridProps extends HTMLAttributes<HTMLElement> {
    /** HTML element type to render */
    component?: ElementType;
    /** Whether this is a grid item instead of container */
    item?: boolean;
    /** Number of columns in the grid (container only) */
    columns?: GridColumns;
    /** Number of rows in the grid (container only) */
    rows?: GridRows;
    /** Gap between grid items (container only) */
    gap?: GridGap;
    /** Column gap between grid items (container only) */
    columnGap?: GridGap;
    /** Row gap between grid items (container only) */
    rowGap?: GridGap;
    /** Align items along the cross axis (container only) */
    alignItems?: GridAlign;
    /** Justify content along the main axis (container only) */
    justifyContent?: GridJustify;
    /** Grid auto flow direction (container only) */
    flow?: GridFlow;
    /** Whether the grid should take full width */
    fullWidth?: boolean;
    /** Whether the grid should take full height */
    fullHeight?: boolean;
    /** Number of columns to span (item only) */
    colSpan?: GridItemSpan;
    /** Number of rows to span (item only) */
    rowSpan?: GridItemSpan;
    /** Column start position (item only) */
    colStart?: GridItemStart;
    /** Column end position (item only) */
    colEnd?: GridItemEnd;
    /** Row start position (item only) */
    rowStart?: GridItemStart;
    /** Row end position (item only) */
    rowEnd?: GridItemEnd;
    /** Align self along the cross axis (item only) */
    alignSelf?: GridItemAlign;
    /** Justify self along the main axis (item only) */
    justifySelf?: GridItemJustify;
    /** Grid content */
    children?: ReactNode;
}

/**
 * Grid columns configuration
 */
const GRID_COLUMNS = {
    none: "",
    1: "tw-grid-cols-1",
    2: "tw-grid-cols-2",
    3: "tw-grid-cols-3",
    4: "tw-grid-cols-4",
    5: "tw-grid-cols-5",
    6: "tw-grid-cols-6",
    7: "tw-grid-cols-7",
    8: "tw-grid-cols-8",
    9: "tw-grid-cols-9",
    10: "tw-grid-cols-10",
    11: "tw-grid-cols-11",
    12: "tw-grid-cols-12",
} as const;

/**
 * Grid rows configuration
 */
const GRID_ROWS = {
    none: "",
    1: "tw-grid-rows-1",
    2: "tw-grid-rows-2",
    3: "tw-grid-rows-3",
    4: "tw-grid-rows-4",
    5: "tw-grid-rows-5",
    6: "tw-grid-rows-6",
} as const;

/**
 * Grid gap configuration
 */
const GRID_GAP = {
    none: "",
    xs: "tw-gap-1",
    sm: "tw-gap-2",
    md: "tw-gap-4",
    lg: "tw-gap-6",
    xl: "tw-gap-8",
} as const;

/**
 * Grid column gap configuration
 */
const GRID_COLUMN_GAP = {
    none: "",
    xs: "tw-gap-x-1",
    sm: "tw-gap-x-2",
    md: "tw-gap-x-4",
    lg: "tw-gap-x-6",
    xl: "tw-gap-x-8",
} as const;

/**
 * Grid row gap configuration
 */
const GRID_ROW_GAP = {
    none: "",
    xs: "tw-gap-y-1",
    sm: "tw-gap-y-2",
    md: "tw-gap-y-4",
    lg: "tw-gap-y-6",
    xl: "tw-gap-y-8",
} as const;

/**
 * Grid align items configuration
 */
const GRID_ALIGN_ITEMS = {
    start: "tw-items-start",
    center: "tw-items-center",
    end: "tw-items-end",
    stretch: "tw-items-stretch",
} as const;

/**
 * Grid justify content configuration
 */
const GRID_JUSTIFY_CONTENT = {
    start: "tw-justify-start",
    center: "tw-justify-center",
    end: "tw-justify-end",
    between: "tw-justify-between",
    around: "tw-justify-around",
    evenly: "tw-justify-evenly",
} as const;

/**
 * Grid flow configuration
 */
const GRID_FLOW = {
    row: "tw-grid-flow-row",
    col: "tw-grid-flow-col",
    "row-dense": "tw-grid-flow-row-dense",
    "col-dense": "tw-grid-flow-col-dense",
} as const;

/**
 * Grid item column span configuration
 */
const GRID_COL_SPAN = {
    auto: "tw-col-auto",
    full: "tw-col-span-full",
    1: "tw-col-span-1",
    2: "tw-col-span-2",
    3: "tw-col-span-3",
    4: "tw-col-span-4",
    5: "tw-col-span-5",
    6: "tw-col-span-6",
    7: "tw-col-span-7",
    8: "tw-col-span-8",
    9: "tw-col-span-9",
    10: "tw-col-span-10",
    11: "tw-col-span-11",
    12: "tw-col-span-12",
} as const;

/**
 * Grid item row span configuration
 */
const GRID_ROW_SPAN = {
    auto: "tw-row-auto",
    full: "tw-row-span-full",
    1: "tw-row-span-1",
    2: "tw-row-span-2",
    3: "tw-row-span-3",
    4: "tw-row-span-4",
    5: "tw-row-span-5",
    6: "tw-row-span-6",
} as const;

/**
 * Grid item column start configuration
 */
const GRID_COL_START = {
    auto: "tw-col-start-auto",
    1: "tw-col-start-1",
    2: "tw-col-start-2",
    3: "tw-col-start-3",
    4: "tw-col-start-4",
    5: "tw-col-start-5",
    6: "tw-col-start-6",
    7: "tw-col-start-7",
    8: "tw-col-start-8",
    9: "tw-col-start-9",
    10: "tw-col-start-10",
    11: "tw-col-start-11",
    12: "tw-col-start-12",
    13: "tw-col-start-13",
} as const;

/**
 * Grid item column end configuration
 */
const GRID_COL_END = {
    auto: "tw-col-end-auto",
    1: "tw-col-end-1",
    2: "tw-col-end-2",
    3: "tw-col-end-3",
    4: "tw-col-end-4",
    5: "tw-col-end-5",
    6: "tw-col-end-6",
    7: "tw-col-end-7",
    8: "tw-col-end-8",
    9: "tw-col-end-9",
    10: "tw-col-end-10",
    11: "tw-col-end-11",
    12: "tw-col-end-12",
    13: "tw-col-end-13",
} as const;

/**
 * Grid item row start configuration
 */
const GRID_ROW_START = {
    auto: "tw-row-start-auto",
    1: "tw-row-start-1",
    2: "tw-row-start-2",
    3: "tw-row-start-3",
    4: "tw-row-start-4",
    5: "tw-row-start-5",
    6: "tw-row-start-6",
    7: "tw-row-start-7",
} as const;

/**
 * Grid item row end configuration
 */
const GRID_ROW_END = {
    auto: "tw-row-end-auto",
    1: "tw-row-end-1",
    2: "tw-row-end-2",
    3: "tw-row-end-3",
    4: "tw-row-end-4",
    5: "tw-row-end-5",
    6: "tw-row-end-6",
    7: "tw-row-end-7",
} as const;

/**
 * Grid item align self configuration
 */
const GRID_ALIGN_SELF = {
    start: "tw-self-start",
    center: "tw-self-center",
    end: "tw-self-end",
    stretch: "tw-self-stretch",
} as const;

/**
 * Grid item justify self configuration
 */
const GRID_JUSTIFY_SELF = {
    start: "tw-justify-self-start",
    center: "tw-justify-self-center",
    end: "tw-justify-self-end",
    stretch: "tw-justify-self-stretch",
} as const;

/**
 * Builds the complete class string for the Grid container
 */
function buildGridContainerClasses({
    columns,
    rows,
    gap,
    columnGap,
    rowGap,
    alignItems,
    justifyContent,
    flow,
    fullWidth,
    fullHeight,
    className,
}: {
    columns: GridColumns;
    rows: GridRows;
    gap?: GridGap;
    columnGap?: GridGap;
    rowGap?: GridGap;
    alignItems: GridAlign;
    justifyContent: GridJustify;
    flow: GridFlow;
    fullWidth: boolean;
    fullHeight: boolean;
    className?: string;
}) {
    return cn(
        // Base grid styles
        "tw-grid",
        
        // Grid template
        GRID_COLUMNS[columns],
        GRID_ROWS[rows],
        
        // Grid flow
        GRID_FLOW[flow],
        
        // Gaps - If gap is specified, it overrides columnGap and rowGap
        gap ? GRID_GAP[gap] : [
            columnGap && GRID_COLUMN_GAP[columnGap],
            rowGap && GRID_ROW_GAP[rowGap],
        ],
        
        // Alignment
        GRID_ALIGN_ITEMS[alignItems],
        GRID_JUSTIFY_CONTENT[justifyContent],
        
        // Size modifiers
        fullWidth && "tw-w-full",
        fullHeight && "tw-h-full",
        
        // Custom className
        className,
    );
}

/**
 * Builds the complete class string for Grid items
 */
function buildGridItemClasses({
    colSpan,
    rowSpan,
    colStart,
    colEnd,
    rowStart,
    rowEnd,
    alignSelf,
    justifySelf,
    fullWidth,
    fullHeight,
    className,
}: {
    colSpan?: GridItemSpan;
    rowSpan?: GridItemSpan;
    colStart?: GridItemStart;
    colEnd?: GridItemEnd;
    rowStart?: GridItemStart;
    rowEnd?: GridItemEnd;
    alignSelf?: GridItemAlign;
    justifySelf?: GridItemJustify;
    fullWidth: boolean;
    fullHeight: boolean;
    className?: string;
}) {
    return cn(
        // Column span
        colSpan && GRID_COL_SPAN[colSpan],
        
        // Row span
        rowSpan && rowSpan !== "auto" && rowSpan !== "full" && GRID_ROW_SPAN[rowSpan],
        
        // Column positioning
        colStart && GRID_COL_START[colStart],
        colEnd && GRID_COL_END[colEnd],
        
        // Row positioning
        rowStart && rowStart !== "auto" && (rowStart <= 7 ? GRID_ROW_START[rowStart] : undefined),
        rowEnd && rowEnd !== "auto" && (rowEnd <= 7 ? GRID_ROW_END[rowEnd] : undefined),
        
        // Self alignment
        alignSelf && GRID_ALIGN_SELF[alignSelf],
        justifySelf && GRID_JUSTIFY_SELF[justifySelf],
        
        // Size modifiers
        fullWidth && "tw-w-full",
        fullHeight && "tw-h-full",
        
        // Custom className
        className,
    );
}

/**
 * A flexible CSS Grid component for creating responsive layouts.
 * Can be used as both a grid container and a grid item.
 * 
 * Features:
 * - Container mode: Creates a CSS grid layout
 *   - 1-12 columns grid template
 *   - 1-6 rows grid template
 *   - 6 gap sizes from none to xl
 *   - Separate column and row gap control
 *   - Align and justify content options
 *   - Grid flow control for auto-placement
 * - Item mode: Controls grid item placement and sizing
 *   - Column and row spanning
 *   - Precise positioning with start/end
 *   - Self alignment options
 * - Customizable HTML element type
 * - Full width and height options
 * - Responsive grid support via className
 * 
 * @example
 * ```tsx
 * // Basic 3-column grid
 * <Grid columns={3} gap="md">
 *   <Grid item>Item 1</Grid>
 *   <Grid item>Item 2</Grid>
 *   <Grid item>Item 3</Grid>
 * </Grid>
 * 
 * // Grid with spanning items
 * <Grid columns={4} gap="md">
 *   <Grid item colSpan={2}>Wide item</Grid>
 *   <Grid item>Normal item</Grid>
 *   <Grid item>Normal item</Grid>
 * </Grid>
 * 
 * // Complex grid with positioning
 * <Grid columns={12} gap="lg">
 *   <Grid item colSpan={8}>Main content</Grid>
 *   <Grid item colSpan={4}>Sidebar</Grid>
 *   <Grid item colSpan={12}>Footer</Grid>
 * </Grid>
 * ```
 */
export const Grid = forwardRef<HTMLElement, GridProps>(
    (
        {
            component: Component = "div",
            item = false,
            // Container props
            columns = "none",
            rows = "none",
            gap,
            columnGap,
            rowGap,
            alignItems = "stretch",
            justifyContent = "start",
            flow = "row",
            // Item props
            colSpan,
            rowSpan,
            colStart,
            colEnd,
            rowStart,
            rowEnd,
            alignSelf,
            justifySelf,
            // Common props
            fullWidth = false,
            fullHeight = false,
            className,
            children,
            ...props
        },
        ref,
    ) => {
        const classes = item
            ? buildGridItemClasses({
                colSpan,
                rowSpan,
                colStart,
                colEnd,
                rowStart,
                rowEnd,
                alignSelf,
                justifySelf,
                fullWidth,
                fullHeight,
                className,
            })
            : buildGridContainerClasses({
                columns,
                rows,
                gap,
                columnGap,
                rowGap,
                alignItems,
                justifyContent,
                flow,
                fullWidth,
                fullHeight,
                className,
            });

        return (
            <Component
                ref={ref}
                className={classes}
                {...props}
            >
                {children}
            </Component>
        );
    },
);

Grid.displayName = "Grid";

/**
 * Pre-configured grid components for common use cases
 */
export const GridFactory = {
    // Container presets
    /** Two column grid with equal spacing */
    TwoColumn: (props: Omit<GridProps, "columns">) => (
        <Grid columns={2} gap="md" {...props} />
    ),
    /** Three column grid with equal spacing */
    ThreeColumn: (props: Omit<GridProps, "columns">) => (
        <Grid columns={3} gap="md" {...props} />
    ),
    /** Four column grid with equal spacing */
    FourColumn: (props: Omit<GridProps, "columns">) => (
        <Grid columns={4} gap="md" {...props} />
    ),
    /** Twelve column grid system */
    TwelveColumn: (props: Omit<GridProps, "columns">) => (
        <Grid columns={12} gap="md" {...props} />
    ),
    /** Centered grid with items in the middle */
    Centered: (props: GridProps) => (
        <Grid alignItems="center" justifyContent="center" {...props} />
    ),
    /** Grid with items spaced evenly */
    SpacedEvenly: (props: GridProps) => (
        <Grid justifyContent="evenly" {...props} />
    ),
    /** Full width responsive grid */
    Responsive: (props: GridProps) => (
        <Grid fullWidth className={cn("md:tw-grid-cols-2 lg:tw-grid-cols-3 xl:tw-grid-cols-4", props.className)} {...props} />
    ),
    
    // Item presets
    /** Grid item that spans half the grid (6 columns in 12-column grid) */
    ItemHalf: (props: Omit<GridProps, "item" | "colSpan">) => (
        <Grid item colSpan={6} {...props} />
    ),
    /** Grid item that spans a third of the grid (4 columns in 12-column grid) */
    ItemThird: (props: Omit<GridProps, "item" | "colSpan">) => (
        <Grid item colSpan={4} {...props} />
    ),
    /** Grid item that spans a quarter of the grid (3 columns in 12-column grid) */
    ItemQuarter: (props: Omit<GridProps, "item" | "colSpan">) => (
        <Grid item colSpan={3} {...props} />
    ),
    /** Grid item that spans two thirds of the grid (8 columns in 12-column grid) */
    ItemTwoThirds: (props: Omit<GridProps, "item" | "colSpan">) => (
        <Grid item colSpan={8} {...props} />
    ),
    /** Grid item that spans the full width */
    ItemFull: (props: Omit<GridProps, "item" | "colSpan">) => (
        <Grid item colSpan="full" {...props} />
    ),
    /** Grid item centered within its cell */
    ItemCentered: (props: Omit<GridProps, "item">) => (
        <Grid item alignSelf="center" justifySelf="center" {...props} />
    ),
    /** Grid item for sidebar layouts (spans 3 columns) */
    ItemSidebar: (props: Omit<GridProps, "item" | "colSpan">) => (
        <Grid item colSpan={3} {...props} />
    ),
    /** Grid item for main content (spans 9 columns) */
    ItemMainContent: (props: Omit<GridProps, "item" | "colSpan">) => (
        <Grid item colSpan={9} {...props} />
    ),
} as const;
