import { uniqBy } from "@local/shared";
import { NodeType, RunInputCreateInput, RunInputUpdateInput } from "graphql/generated/globalTypes";
import { Node, NodeDataRoutineList, NodeLink, Routine, RunInput } from "types";
import { v4 as uuid } from "uuid";
import { Status } from "./consts";
import { shapeRunInputCreate, shapeRunInputUpdate } from "./shape";
import { shapeUpdateList } from "./shape/shapeTools";

/**
 * Calculates the percentage of the run that has been completed.
 * @param completedComplexity The number of completed steps. Ideally shouldn't exceed the 
 * totalComplexity, since steps shouldn't be counted multiple times.
 * @param totalComplexity The total number of steps.
 * @returns The percentage of the run that has been completed, 0-100.
 */
export const getRunPercentComplete = (
    completedComplexity: number | null | undefined,
    totalComplexity: number | null | undefined,
) => {
    if (!completedComplexity || !totalComplexity || totalComplexity === 0) return 0;
    const percentage = Math.round(completedComplexity as number / totalComplexity * 100);
    return Math.min(percentage, 100);
}

/**
 * Determines if two location arrays match, where a location array is an array of step indices
 * @param locationA The first location array 
 * @param locationB The second location array
 * @return True if the location arrays match, false otherwise
 */
export const locationArraysMatch = (locationA: number[], locationB: number[]): boolean => {
    if (locationA.length !== locationB.length) return false;
    for (let i = 0; i < locationA.length; i++) {
        if (locationA[i] !== locationB[i]) return false;
    }
    return true;
}

/**
 * Determines if a routine has subroutines. This may be simple, depending on 
 * how much information is provided
 * @param routine The routine to check
 * @return True if the routine has subroutines, false otherwise
 */
export const routineHasSubroutines = (routine: Partial<Routine>): boolean => {
    // If routine has nodes or links, we know it has subroutines
    if (routine.nodes && routine.nodes.length > 0) return true;
    if (routine.nodeLinks && routine.nodeLinks.length > 0) return true;
    if ((routine as any).nodesCount && (routine as any).nodesCount > 0) return true;
    return false;
}

/**
 * Converts formik into object with run input data
 * @param values The formik values object
 * @returns object where keys are inputIds, and values are the run input data
 */
export const formikToRunInputs = (values: { [x: string]: string }): { [x: string]: string } => {
    const result: { [x: string]: string } = {};
    // Get user inputs, and ignore empty values and blank strings.
    const inputValues = Object.entries(values).filter(([key, value]) =>
        key.startsWith('inputs-') &&
        typeof value === 'string' &&
        value.length > 0);
    // Input keys are in the form of inputs-<inputId>. We need the inputId
    for (const [key, value] of inputValues) {
        const inputId = key.substring(key.indexOf('-') + 1)
        result[inputId] = JSON.stringify(value);
    }
    return result;
}

/**
 * Updates formik values with run input data
 * @param runInputs The run input data
 * @returns Object to pass into formik setValues function
 */
export const runInputsToFormik = (runInputs: { input: { id: string }, data: string }[]): { [x: string]: string } => {
    const result: { [x: string]: string } = {};
    console.log('in runinputstoformik');
    for (const runInput of runInputs) {
        result[`inputs-${runInput.input.id}`] = JSON.parse(runInput.data);
    }
    console.log('returning formik inputs result', result);
    return result;
}

/**
 * Converts a run inputs object to a run input create object
 * @param runInputs The run inputs object
 * @returns The run input create input object
 */
export const runInputsCreate = (runInputs: { [x: string]: string }): { inputsCreate: RunInputCreateInput[] } => {
    return {
        inputsCreate: Object.entries(runInputs).map(([inputId, data]) => ({
            id: uuid(),
            data,
            inputId
        }))
    }
}

/**
 * Converts a run inputs object to a run input update object
 * @param original RunInputs[] array of existing run inputs data
 * @param updated Run input object with updated data
 * @returns The run input update input object
 */
export const runInputsUpdate = (
    original: RunInput[],
    updated: { [x: string]: string },
): {
    inputsCreate?: RunInputCreateInput[],
    inputsUpdate?: RunInputUpdateInput[],
    inputsDelete?: string[],
} => {
    // Convert user inputs object to RunInput[]
    const updatedInputs = Object.entries(updated).map(([inputId, data]) => ({
        id: uuid(),
        data,
        input: { id: inputId }
    }))
    // Loop through original run inputs. Where input id is in updated inputs, update id
    for (const currOriginal of original) {
        const currUpdated = updatedInputs.findIndex((input) => input.input.id === currOriginal.input.id);
        if (currUpdated !== -1) {
            updatedInputs[currUpdated].id = currOriginal.id;
        }
    }
    console.log('yeeeeeet', original, updatedInputs)
    return shapeUpdateList(
        { inputs: original },
        { inputs: updatedInputs },
        'inputs',
        (o, u) => o.data !== u.data,
        shapeRunInputCreate,
        shapeRunInputUpdate,
        'id'
    );
}

type GetRoutineStatusResult = {
    status: Status;
    messages: string[];
    nodesById: { [id: string]: Node };
    nodesOnGraph: Node[];
    nodesOffGraph: Node[];
}

/**
 * Calculates the status of a routine (anything that's not valid cannot be run). 
 * Also returns some other information which is useful for displaying routines
 * @param routine The routine to check
 */
export const getRoutineStatus = (routine?: Partial<Routine> | null): GetRoutineStatusResult => {
    if (!routine || !routine.nodeLinks || !routine.nodes) { 
        return { status: Status.Invalid, messages: ['No node or link data found'], nodesById: {}, nodesOffGraph: [], nodesOnGraph: [] };
    }
    const nodesOnGraph: Node[] = [];
    const nodesOffGraph: Node[] = [];
    const nodesById: { [id: string]: Node } = {};
    const statuses: [Status, string][] = []; // Holds all status messages, so multiple can be displayed
    // Loop through nodes and add to appropriate array (and also populate nodesById dictionary)
    for (const node of routine.nodes) {
        if ((node.columnIndex !== null && node.columnIndex !== undefined) && (node.rowIndex !== null && node.rowIndex !== undefined)) {
            nodesOnGraph.push(node);
        } else {
            nodesOffGraph.push(node);
        }
        nodesById[node.id] = node;
    }
    // Now, perform a few checks to make sure that the columnIndexes and rowIndexes are valid
    // 1. Check that (columnIndex, rowIndex) pairs are all unique
    // First check
    // Remove duplicate values from positions dictionary
    const uniqueDict = uniqBy(nodesOnGraph, (n) => `${n.columnIndex}-${n.rowIndex}`);
    // Check if length of removed duplicates is equal to the length of the original positions dictionary
    if (uniqueDict.length !== Object.values(nodesOnGraph).length) {
        return { status: Status.Invalid, messages: ['Ran into error determining node positions'], nodesById, nodesOffGraph: routine.nodes, nodesOnGraph: [] };
    }
    // Now perform checks to see if the routine can be run
    // 1. There is only one start node
    // 2. There is only one linked node which has no incoming edges, and it is the start node
    // 3. Every node that has no outgoing edges is an end node
    // 4. Validate loop TODO
    // 5. Validate redirects TODO
    // Check 1
    const startNodes = routine.nodes.filter(node => node.type === NodeType.Start);
    if (startNodes.length === 0) {
        statuses.push([Status.Invalid, 'No start node found']);
    }
    else if (startNodes.length > 1) {
        statuses.push([Status.Invalid, 'More than one start node found']);
    }
    // Check 2
    const nodesWithoutIncomingEdges = nodesOnGraph.filter(node => routine.nodeLinks!.every(link => link.toId !== node.id));
    if (nodesWithoutIncomingEdges.length === 0) {
        //TODO this would be fine with a redirect link
        statuses.push([Status.Invalid, 'Error determining start node']);
    }
    else if (nodesWithoutIncomingEdges.length > 1) {
        statuses.push([Status.Invalid, 'Nodes are not fully connected']);
    }
    // Check 3
    const nodesWithoutOutgoingEdges = nodesOnGraph.filter(node => routine.nodeLinks!.every(link => link.fromId !== node.id));
    if (nodesWithoutOutgoingEdges.length >= 0) {
        // Check that every node without outgoing edges is an end node
        if (nodesWithoutOutgoingEdges.some(node => node.type !== NodeType.End)) {
            statuses.push([Status.Invalid, 'Not all paths end with an end node']);
        }
    }
    // Performs checks which make the routine incomplete, but not invalid
    // 1. There are unpositioned nodes
    // 2. Every routine list has at least one subroutine
    // Check 1
    if (nodesOffGraph.length > 0) {
        statuses.push([Status.Incomplete, 'Some nodes are not linked']);
    }
    // Check 2
    if (nodesOnGraph.some(node => node.type === NodeType.RoutineList && (node.data as NodeDataRoutineList)?.routines?.length === 0)) {
        statuses.push([Status.Incomplete, 'At least one routine list is empty']);
    }
    // Return statuses, or valid if no statuses
    if (statuses.length > 0) {
        // Status sent is the worst status
        let status = Status.Incomplete;
        if (statuses.some(status => status[0] === Status.Invalid)) status = Status.Invalid;
        return { status, messages: statuses.map(status => status[1]), nodesById, nodesOffGraph, nodesOnGraph };
    } else {
        return { status: Status.Valid, messages: ['Routine is fully connected'], nodesById, nodesOffGraph, nodesOnGraph };
    }
}

/**
 * Multi-step routine initial data, if creating from scratch
 * @param language The language of the routine
 * @returns Initial data for a new routine
 */
export const initializeRoutine = (language: string): Routine => {
    const startNode: Node = {
        id: uuid(),
        type: NodeType.Start,
        columnIndex: 0,
        rowIndex: 0,
    } as Node;
    const routineListNode: Node = {
        __typename: 'Node',
        id: uuid(),
        type: NodeType.RoutineList,
        columnIndex: 1,
        rowIndex: 0,
        data: {
            __typename: 'NodeRoutineList',
            id: uuid(),
            isOptional: false,
            isOrdered: false,
            routines: [],
        } as Node['data'],
        translations: [{
            id: uuid(),
            language,
            title: 'Subroutine 1',
        }] as Node['translations'],
    } as Node;
    const endNode: Node = {
        __typename: 'Node',
        id: uuid(),
        type: NodeType.End,
        columnIndex: 2,
        rowIndex: 0,
    } as Node;
    const link1: NodeLink = {
        __typename: 'NodeLink',
        id: uuid(),
        fromId: startNode.id,
        toId: routineListNode.id,
        whens: [],
        operation: null,
    }
    const link2: NodeLink = {
        __typename: 'NodeLink',
        id: uuid(),
        fromId: routineListNode.id,
        toId: endNode.id,
        whens: [],
        operation: null,
    }
    return {
        inputs: [],
        outputs: [],
        nodes: [startNode, routineListNode, endNode],
        nodeLinks: [link1, link2],
        translations: [{
            id: uuid(),
            language,
            title: 'New Routine',
            instructions: 'Enter instructions here',
            description: '',
        }]
    } as any
}