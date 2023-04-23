import { NodeType } from "@local/consts";
import { exists, uniqBy } from "@local/utils";
import { uuid } from "@local/uuid";
import { Status } from "./consts";
import { shapeRunRoutineInput } from "./shape/models/runRoutineInput";
import { updateRel } from "./shape/models/tools";
export const getRunPercentComplete = (completedComplexity, totalComplexity) => {
    if (!completedComplexity || !totalComplexity || totalComplexity === 0)
        return 0;
    const percentage = Math.round(completedComplexity / totalComplexity * 100);
    return Math.min(percentage, 100);
};
export const locationArraysMatch = (locationA, locationB) => {
    if (locationA.length !== locationB.length)
        return false;
    for (let i = 0; i < locationA.length; i++) {
        if (locationA[i] !== locationB[i])
            return false;
    }
    return true;
};
export const routineVersionHasSubroutines = (routineVersion) => {
    if (routineVersion.nodes && routineVersion.nodes.length > 0)
        return true;
    if (routineVersion.nodeLinks && routineVersion.nodeLinks.length > 0)
        return true;
    if (routineVersion.nodesCount && routineVersion.nodesCount > 0)
        return true;
    return false;
};
export const formikToRunInputs = (values) => {
    const result = {};
    const inputValues = Object.entries(values).filter(([key, value]) => key.startsWith("inputs-") &&
        typeof value === "string" &&
        value.length > 0);
    for (const [key, value] of inputValues) {
        const inputId = key.substring(key.indexOf("-") + 1);
        result[inputId] = JSON.stringify(value);
    }
    return result;
};
export const runInputsToFormik = (runInputs) => {
    const result = {};
    for (const runInput of runInputs) {
        result[`inputs-${runInput.input.id}`] = JSON.parse(runInput.data);
    }
    return result;
};
export const runInputsCreate = (runInputs, runRoutineId) => {
    return {
        inputsCreate: Object.entries(runInputs).map(([inputId, data]) => ({
            id: uuid(),
            data,
            inputConnect: inputId,
            runRoutineConnect: runRoutineId,
        })),
    };
};
export const runInputsUpdate = (original, updated) => {
    const updatedInputs = Object.entries(updated).map(([inputId, data]) => ({
        id: uuid(),
        data,
        input: { id: inputId },
    }));
    for (const currOriginal of original) {
        const currUpdated = updatedInputs.findIndex((input) => input.input.id === currOriginal.input.id);
        if (currUpdated !== -1) {
            updatedInputs[currUpdated].id = currOriginal.id;
        }
    }
    return updateRel({ inputs: original }, { inputs: updatedInputs }, "inputs", ["Create", "Update", "Delete"], "many", shapeRunRoutineInput);
};
export const getRoutineVersionStatus = (routineVersion) => {
    if (!routineVersion || !routineVersion.nodeLinks || !routineVersion.nodes) {
        return { status: Status.Invalid, messages: ["No node or link data found"], nodesById: {}, nodesOffGraph: [], nodesOnGraph: [] };
    }
    const nodesOnGraph = [];
    const nodesOffGraph = [];
    const nodesById = {};
    const statuses = [];
    for (const node of routineVersion.nodes) {
        if (exists(node.columnIndex) && exists(node.rowIndex)) {
            nodesOnGraph.push(node);
        }
        else {
            nodesOffGraph.push(node);
        }
        nodesById[node.id] = node;
    }
    const uniqueDict = uniqBy(nodesOnGraph, (n) => `${n.columnIndex}-${n.rowIndex}`);
    if (uniqueDict.length !== Object.values(nodesOnGraph).length) {
        return { status: Status.Invalid, messages: ["Ran into error determining node positions"], nodesById, nodesOffGraph: routineVersion.nodes, nodesOnGraph: [] };
    }
    const startNodes = routineVersion.nodes.filter(node => node.nodeType === NodeType.Start);
    if (startNodes.length === 0) {
        statuses.push([Status.Invalid, "No start node found"]);
    }
    else if (startNodes.length > 1) {
        statuses.push([Status.Invalid, "More than one start node found"]);
    }
    const nodesWithoutIncomingEdges = nodesOnGraph.filter(node => routineVersion.nodeLinks.every(link => link.to.id !== node.id));
    if (nodesWithoutIncomingEdges.length === 0) {
        statuses.push([Status.Invalid, "Error determining start node"]);
    }
    else if (nodesWithoutIncomingEdges.length > 1) {
        statuses.push([Status.Invalid, "Nodes are not fully connected"]);
    }
    const nodesWithoutOutgoingEdges = nodesOnGraph.filter(node => routineVersion.nodeLinks.every(link => link.from.id !== node.id));
    if (nodesWithoutOutgoingEdges.length >= 0) {
        if (nodesWithoutOutgoingEdges.some(node => node.nodeType !== NodeType.End)) {
            statuses.push([Status.Invalid, "Not all paths end with an end node"]);
        }
    }
    if (nodesOffGraph.length > 0) {
        statuses.push([Status.Incomplete, "Some nodes are not linked"]);
    }
    if (nodesOnGraph.some(node => node.nodeType === NodeType.RoutineList && node.routineList?.items?.length === 0)) {
        statuses.push([Status.Incomplete, "At least one routine list is empty"]);
    }
    if (statuses.length > 0) {
        let status = Status.Incomplete;
        if (statuses.some(status => status[0] === Status.Invalid))
            status = Status.Invalid;
        return { status, messages: statuses.map(status => status[1]), nodesById, nodesOffGraph, nodesOnGraph };
    }
    else {
        return { status: Status.Valid, messages: ["Routine is fully connected"], nodesById, nodesOffGraph, nodesOnGraph };
    }
};
export const getProjectVersionStatus = (projectVersion) => {
    return {};
};
export const initializeRoutineGraph = (language, routineVersionId) => {
    const startNode = {
        id: uuid(),
        nodeType: NodeType.Start,
        columnIndex: 0,
        rowIndex: 0,
        routineVersion: { id: routineVersionId },
        translations: [],
    };
    const routineListNodeId = uuid();
    const routineListNode = {
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
                name: "Subroutine 1",
            }],
    };
    const endNode = {
        __typename: "Node",
        id: uuid(),
        nodeType: NodeType.End,
        columnIndex: 2,
        rowIndex: 0,
        routineVersion: { id: routineVersionId },
        translations: [],
    };
    const link1 = {
        id: uuid(),
        from: startNode,
        to: routineListNode,
        whens: [],
        operation: null,
        routineVersion: { id: routineVersionId },
    };
    const link2 = {
        id: uuid(),
        from: routineListNode,
        to: endNode,
        whens: [],
        operation: null,
        routineVersion: { id: routineVersionId },
    };
    return {
        nodes: [startNode, routineListNode, endNode],
        nodeLinks: [link1, link2],
    };
};
//# sourceMappingURL=runUtils.js.map