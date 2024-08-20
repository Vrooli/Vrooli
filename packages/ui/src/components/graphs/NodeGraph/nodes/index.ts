export * from "./DraggableNode/DraggableNode";
export * from "./EndNode/EndNode";
export * from "./RedirectNode/RedirectNode";
export * from "./RoutineListNode/RoutineListNode";
export * from "./StartNode/StartNode";
export * from "./SubroutineNode/SubroutineNode";

/**
 * Calculates the width/height of a node, using an exponential function.
 * @param initialSize - the initial size of the node
 * @param scale - the scale of the graph (initial = 0)
 * @param doubleEvery - the increase/decrease in scale required to double/halve the size
 * @returns the width/height of the node
 */
export function calculateNodeSize(initialSize: number, scale: number, doubleEvery = 1): number {
    return initialSize * Math.pow(2, scale / doubleEvery);
}

export const SHOW_TITLE_ABOVE_SCALE = -2;
/**
 * Distance before a click is considered a drag
 */
export const DRAG_THRESHOLD = 10;
