import { exists, uniqBy } from "@shared/utils";
import { uuid } from '@shared/uuid';
import { Status } from "./consts";
import { GqlModelType, Node, NodeLink, NodeType, ProjectVersion, RoutineVersion, RunRoutineInput, RunRoutineInputCreateInput, RunRoutineInputUpdateInput } from "@shared/consts";
import { NodeLinkShape, NodeShape, shapeRunRoutineInput, updateRel } from "./shape";

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
 * Determines if a routine version has subroutines. This may be simple, depending on 
 * how much information is provided
 * @param routine The routine version to check
 * @return True if the routine version has subroutines, false otherwise
 */
export const routineVersionHasSubroutines = (routineVersion: Partial<RoutineVersion>): boolean => {
    // If routineVersion has nodes or links, we know it has subroutines
    if (routineVersion.nodes && routineVersion.nodes.length > 0) return true;
    if (routineVersion.nodeLinks && routineVersion.nodeLinks.length > 0) return true;
    if ((routineVersion as any).nodesCount && (routineVersion as any).nodesCount > 0) return true;
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
    for (const runInput of runInputs) {
        result[`inputs-${runInput.input.id}`] = JSON.parse(runInput.data);
    }
    return result;
}

/**
 * Converts a run inputs object to a run input create object
 * @param runInputs The run inputs object
 * @returns The run input create input object
 */
export const runInputsCreate = (runInputs: { [x: string]: string }, runRoutineId: string): { inputsCreate: RunRoutineInputCreateInput[] } => {
    return {
        inputsCreate: Object.entries(runInputs).map(([inputId, data]) => ({
            id: uuid(),
            data,
            inputConnect: inputId,
            runRoutineConnect: runRoutineId
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
    original: RunRoutineInput[],
    updated: { [x: string]: string },
): {
    inputsCreate?: RunRoutineInputCreateInput[],
    inputsUpdate?: RunRoutineInputUpdateInput[],
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
    return updateRel({ inputs: original }, { inputs: updatedInputs }, 'inputs', ['Create', 'Update', 'Delete'], 'many', shapeRunRoutineInput);
}

type GetRoutineVersionStatusResult = {
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
export const getRoutineVersionStatus = (routineVersion?: Partial<RoutineVersion> | null): GetRoutineVersionStatusResult => {
    if (!routineVersion || !routineVersion.nodeLinks || !routineVersion.nodes) { 
        return { status: Status.Invalid, messages: ['No node or link data found'], nodesById: {}, nodesOffGraph: [], nodesOnGraph: [] };
    }
    const nodesOnGraph: Node[] = [];
    const nodesOffGraph: Node[] = [];
    const nodesById: { [id: string]: Node } = {};
    const statuses: [Status, string][] = []; // Holds all status messages, so multiple can be displayed
    // Loop through nodes and add to appropriate array (and also populate nodesById dictionary)
    for (const node of routineVersion.nodes) {
        if (exists(node.columnIndex) && exists(node.rowIndex)) {
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
        return { status: Status.Invalid, messages: ['Ran into error determining node positions'], nodesById, nodesOffGraph: routineVersion.nodes, nodesOnGraph: [] };
    }
    // Now perform checks to see if the routine can be run
    // 1. There is only one start node
    // 2. There is only one linked node which has no incoming edges, and it is the start node
    // 3. Every node that has no outgoing edges is an end node
    // 4. Validate loop TODO
    // 5. Validate redirects TODO
    // Check 1
    const startNodes = routineVersion.nodes.filter(node => node.nodeType === NodeType.Start);
    if (startNodes.length === 0) {
        statuses.push([Status.Invalid, 'No start node found']);
    }
    else if (startNodes.length > 1) {
        statuses.push([Status.Invalid, 'More than one start node found']);
    }
    // Check 2
    const nodesWithoutIncomingEdges = nodesOnGraph.filter(node => routineVersion.nodeLinks!.every(link => link.to.id !== node.id));
    if (nodesWithoutIncomingEdges.length === 0) {
        //TODO this would be fine with a redirect link
        statuses.push([Status.Invalid, 'Error determining start node']);
    }
    else if (nodesWithoutIncomingEdges.length > 1) {
        statuses.push([Status.Invalid, 'Nodes are not fully connected']);
    }
    // Check 3
    const nodesWithoutOutgoingEdges = nodesOnGraph.filter(node => routineVersion.nodeLinks!.every(link => link.from.id !== node.id));
    if (nodesWithoutOutgoingEdges.length >= 0) {
        // Check that every node without outgoing edges is an end node
        if (nodesWithoutOutgoingEdges.some(node => node.nodeType !== NodeType.End)) {
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
    if (nodesOnGraph.some(node => node.nodeType === NodeType.RoutineList && node.routineList?.items?.length === 0)) {
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

type GetProjectVersionStatusResult = {
    status: Status;
    messages: string[];
}

/**
 * Calculates the status of a project (anything that's not valid cannot be run). 
 * Also returns some other information which is useful for displaying projects
 * @param project The project to check
 */
export const getProjectVersionStatus = (projectVersion?: Partial<ProjectVersion> | null): GetProjectVersionStatusResult => {
    return {} as any;//TODO
}

/**
 * Multi-step routine initial data, if creating from scratch
 * @param language The language of the routine
 * @returns Initial data for a new routine
 */
export const initializeRoutineGraph = (language: string, routineVersionId: string): { nodes: NodeShape[], nodeLinks: NodeLinkShape[] } => {
    const startNode: NodeShape = {
        id: uuid(),
        nodeType: NodeType.Start,
        columnIndex: 0,
        rowIndex: 0,
        routineVersion: { id: routineVersionId },
        translations: [],
    };
    const routineListNodeId = uuid();
    const routineListNode: NodeShape = {
        id: routineListNodeId,
        nodeType: NodeType.RoutineList,
        columnIndex: 1,
        rowIndex: 0,
        routineList: {
            id: uuid(),
            isOptional: false,
            isOrdered: false,
            items: [],
            node: { id: routineListNodeId },
        },
        routineVersion: { id: routineVersionId },
        translations: [{
            id: uuid(),
            language,
            name: 'Subroutine 1',
        }] as Node['translations'],
    };
    const endNode: NodeShape = {
        type: GqlModelType.Node,
        id: uuid(),
        nodeType: NodeType.End,
        columnIndex: 2,
        rowIndex: 0,
        routineVersion: { id: routineVersionId },
    } as Node;
    const link1: NodeLinkShape = {
        id: uuid(),
        from: startNode,
        to: endNode,
        whens: [],
        operation: null,
        routineVersion: { id: routineVersionId }
    }
    const link2: NodeLinkShape = {
        id: uuid(),
        from: routineListNode,
        to: endNode,
        whens: [],
        operation: null,
        routineVersion: { id: routineVersionId }
    }
    return {
        nodes: [startNode, routineListNode, endNode],
        nodeLinks: [link1, link2],
    };
}