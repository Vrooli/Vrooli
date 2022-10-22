export * from './DraggableNode/DraggableNode';
export * from './EndNode/EndNode';
export * from './RedirectNode/RedirectNode';
export * from './RoutineListNode/RoutineListNode';
export * from './SubroutineNode/SubroutineNode';
export * from './StartNode/StartNode';

/**
 * Calculates the width/height of a node, using an exponential function.
 * @param initialSize - the initial size of the node
 * @param scale - the scale of the graph (initial = 0)
 * @param doubleEvery - the increase/decrease in scale required to double/halve the size
 * @returns the width/height of the node
 */
export const calculateNodeSize = (initialSize: number, scale: number, doubleEvery: number = 1): number => {
    return initialSize * Math.pow(2, scale / doubleEvery);
}