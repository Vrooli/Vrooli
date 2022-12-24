import { RoutineVersionCreateInput, RoutineVersionUpdateInput } from '@shared/consts';
import { CustomError } from "../events";
import { PrismaType } from "../types";

/**
 * Weight data for a subroutine, or all subroutines in a node combined.
 */
type WeightData = {
    simplicity: number,
    complexity: number,
    optionalInputs: number,
    allInputs: number,
}

type LinkData = {
    fromId: string;
    toId: string;
};

/**
 * Calculates the shortest AND longest weighted path on a directed cyclic graph. (loops are actually not the cyclic part, but redirects)
 * A routine with no nodes has a complexity equal to the number of its inputs.
 * Each decision the user makes (i.e. multiple edges coming out of a node) has a weight of 1.
 * Each node has a weight that is the summation of its contained subroutines.
 * @param nodes A map of node IDs to their weight (simplicity/complexity)
 * @param edges The edges of the graph, with each object containing a fromId and toId
 * @param languages Preferred languages for error messages
 * @returns [shortestPath, longestPath] The shortest and longest weighted distance
 */
export const calculateShortestLongestWeightedPath = (
    nodes: { [id: string]: WeightData },
    edges: LinkData[],
    languages: string[]
): [number, number] => {
    // First, check that all edges point to valid nodes. 
    // If this isn't the case, this algorithm could run into an error
    for (const edge of edges) {
        if (!nodes[edge.toId] || !nodes[edge.fromId]) {
            throw new CustomError('0237', 'UnlinkedNodes', languages, { failedEdge: edge })
        }
    }
    // If no nodes or edges, return 1
    if (Object.keys(nodes).length === 0 || edges.length === 0) return [1, 1];
    // Create a dictionary of edges, where the key is a node ID and the value is an array of edges that END at that node
    const edgesByNode: { [id: string]: { fromId: string, toId: string }[] } = {};
    edges.forEach(edge => {
        edgesByNode[edge.toId] = edgesByNode[edge.toId] ? edgesByNode[edge.toId].concat(edge) : [edge];
    });
    /**
     * Calculates the shortest and longest weighted distance
     * @param currentNodeId The current node ID
     * @param visitedEdges The edges that have been visited so far
     * @param currShortest The current shortest distance
     * @param currLongest The current longest distance
     * @returns [shortest, longest] The shortest and longest distance. -1 if doesn't 
     * end with a start node (i.e. caught in a loop)
     */
    const getShortLong = (currentNodeId: string, visitedEdges: { fromId: string, toId: string }[], currShortest: number, currLongest: number): [number, number] => {
        const fromEdges = edgesByNode[currentNodeId];
        // If no from edges, must be start node. Return currShortest and currLongest unchanged
        if (!fromEdges || fromEdges.length === 0) return [currShortest, currLongest];
        // If edges but all have been visited, must be a loop. Return -1
        if (fromEdges.every(edge => visitedEdges.some(visitedEdge => visitedEdge.fromId === edge.fromId && visitedEdge.toId === edge.toId))) return [-1, -1];
        // Otherwise, calculate the shortest and longest distance
        let edgeShorts: number[] = [];
        let edgeLongs: number[] = [];
        for (const edge of fromEdges) {
            // Find the weight of the edge from the node's complexity. Add one if there are multiple edges,
            // since the user has to make a decision
            let weight = nodes[edge.fromId];
            if (fromEdges.length > 1) weight = { ...weight, complexity: weight.complexity + 1, simplicity: weight.simplicity + 1 };
            // Add edge to visited edges
            const newVisitedEdges = visitedEdges.concat([edge]);
            // Recurse on the next node 
            const [shortest, longest] = getShortLong(
                edge.fromId,
                newVisitedEdges,
                currShortest + weight.simplicity + weight.optionalInputs,
                currLongest + weight.complexity + weight.allInputs);
            // If shortest is not -1, add to edgeShorts
            if (shortest !== -1) edgeShorts.push(shortest);
            // If longest is not -1, add to edgeLongs
            if (longest !== -1) edgeLongs.push(longest);
        }
        // Calculate the shortest and longest distance
        const shortest = edgeShorts.length > 0 ? Math.min(...edgeShorts) : -1;
        const longest = edgeLongs.length > 0 ? Math.max(...edgeLongs) : -1;
        return [shortest, longest];
    }
    // Find all of the end nodes, by finding all nodes without any outgoing edges
    const endNodes = Object.keys(nodes).filter(nodeId => !edges.find(e => e.fromId === nodeId));
    // Calculate the shortest and longest for each end node
    const distances: [number, number][] = endNodes.map(nodeId => getShortLong(nodeId, [], 0, 0));
    // Return shortest short and longest long
    return [
        Math.min(...distances.map(d => d[0])),
        Math.max(...distances.map(d => d[1]))
    ]
}

/**
 * Select query for calculating the complexity of a routine version
 */
const routineVersionSelect = ({
    id: true,
    nodeLinks: { select: { id: true, fromId: true, toId: true } },
    nodes: {
        select: {
            id: true,
            nodeRoutineList: {
                select: {
                    id: true,
                    items: {
                        select: {
                            id: true,
                            routineVersion: {
                                select: {
                                    id: true,
                                    complexity: true,
                                    simplicity: true,
                                    inputs: {
                                        select: {
                                            isRequired: true
                                        }
                                    }
                                }
                            }
                        }
                    },
                }
            },
        }
    },
    inputs: {
        select: {
            id: true,
            isRequired: true
        }
    }
})

type GroupRoutineVersionDataResult = {
    linkData: { [linkId: string]: LinkData },
    nodeData: { [nodeId: string]: { subroutines: WeightData[] } },
    subroutineItemData: { [itemId: string]: string },
    optionalRootRutineInputCounts: { [routineId: string]: number }
    allRootRoutineInputCounts: { [routineId: string]: number }
}

/**
 * Queries and groups data into links, nodes, and a map of nodeRoutineListItems to subroutines
 * @param data The queried data
 * @param prisma The prisma client
 * @returns Object with linkData, nodeData, subroutineItemData, and input counts by root routine
 */
const groupRoutineVersionData = async (ids: string[], prisma: PrismaType): Promise<GroupRoutineVersionDataResult> => {
    // Initialize data
    const linkData: { [id: string]: LinkData } = {};
    const nodeData: { [id: string]: { subroutines: WeightData[] } } = {};
    const subroutineItemData: { [id: string]: string } = {};
    const optionalRootRutineInputCounts: { [routineId: string]: number } = {};
    const allRootRoutineInputCounts: { [routineId: string]: number } = {};
    // Query database
    let data = await prisma.routine_version.findMany({
        where: { id: { in: ids } },
        select: routineVersionSelect,
    });
    // Add existing links, nodes data, subroutineItemData, and input counts
    for (const routine of data) {
        // Links
        for (const link of routine.nodeLinks) {
            linkData[link.id] = {
                fromId: link.fromId,
                toId: link.toId,
            }
        }
        // Nodes and subroutineItemData
        for (const node of routine.nodes) {
            if (node.nodeRoutineList) {
                nodeData[node.id] = {
                    subroutines: node.nodeRoutineList.items.map(item => {
                        subroutineItemData[item.id] = item.routineVersion.id;
                        return {
                            id: item.routineVersion.id,
                            complexity: item.routineVersion.complexity,
                            simplicity: item.routineVersion.simplicity,
                            allInputs: item.routineVersion.inputs.length,
                            optionalInputs: item.routineVersion.inputs.filter(input => !input.isRequired).length,
                        }
                    })
                }
            } else {
                nodeData[node.id] = {
                    subroutines: []
                }
            }
        }
    }
    // Add input counts for root routines
    for (const routine of data) {
        optionalRootRutineInputCounts[routine.id] = routine.inputs.filter(input => !input.isRequired).length;
        allRootRoutineInputCounts[routine.id] = routine.inputs.length;
    }
    // Return data
    return {
        linkData,
        nodeData,
        subroutineItemData,
        optionalRootRutineInputCounts,
        allRootRoutineInputCounts,
    };
}

type CalculateComplexityResult = {
    updatingSubroutineIds: string[],
    dataWeights: (WeightData & { id: string })[],
}

/**
 * Calculates the weight data (complexity, simplicity, and num all/required inputs) of a list of routines based on the number of steps. 
 * Simplicity is a the minimum number of inputs and decisions required to complete the routine, while 
 * complexity is the maximum. Inputs are used to 
 * @param prisma The prisma client
 * @param languages Preferred languages for error messages
 * @param inputs The routine version's create or update input
 * @param disallowIds IDs of routines that are not allowed to be used. This is used to prevent multiple updates of the same routine.
 * @returns Data used for recursion, as well as an array of weightData (in same order as inputs)
 */
export const calculateWeightData = async (
    prisma: PrismaType,
    languages: string[],
    inputs: (RoutineVersionUpdateInput | RoutineVersionCreateInput)[],
    disallowIds: string[],
): Promise<CalculateComplexityResult> => {
    // Make sure inputs do not contain disallowed IDs
    const inputIds = inputs.map(i => i.id);
    if (inputIds.some(id => disallowIds.includes(id))) {
        throw new CustomError('0370', 'InvalidArgs', languages);
    }
    // Initialize data used to calculate complexity/simplicity
    const linkData: { [id: string]: { rootRoutineId: string, link: LinkData } } = {};
    const nodeData: { [id: string]: { rootRoutineId: string, subroutines: (WeightData & { id: string })[] } } = {};
    const subroutineItemData: { [id: string]: string } = {}; // Routine list item ID to subroutine ID
    const optionalRootRutineInputCounts: { [routineId: string]: number } = {}; // Excludes subroutine inputs
    const allRootRoutineInputCounts: { [routineId: string]: number } = {}; // Includes subroutine inputs
    // Initialize data used to query/recurse nested complexities/simplicities
    const connectingSubroutineDataIds: string[] = []; // Subroutines that we need to query data for
    const updatingSubroutineData: (RoutineVersionUpdateInput | RoutineVersionCreateInput)[] = []; // Subroutines that are being updated (we will recurse on these)
    const updatingSubroutineIds: string[] = inputIds; // All recursed disallowIds, with the current inputs' IDs added
    // Add existing links, nodes data, and subroutineItemData
    const {
        linkData: existingLinkData,
        nodeData: existingNodeData,
        subroutineItemData: existingSubroutineItemData,
        optionalRootRutineInputCounts: existingOptionalRootRutineInputCounts,
        allRootRoutineInputCounts: existingAllRootRoutineInputCounts,
    } = await groupRoutineVersionData(inputs.map(i => i.id), prisma);
    Object.assign(linkData, existingLinkData);
    Object.assign(nodeData, existingNodeData);
    Object.assign(subroutineItemData, existingSubroutineItemData);
    Object.assign(optionalRootRutineInputCounts, existingOptionalRootRutineInputCounts);
    Object.assign(allRootRoutineInputCounts, existingAllRootRoutineInputCounts);
    // Add links and nodes data from inputs
    for (const input of inputs) {
        // Adding links
        for (const link of input.nodeLinksCreate ?? []) {
            linkData[link.id] = {
                rootRoutineId: input.id,
                link: {
                    fromId: link.fromConnect,
                    toId: link.toConnect,
                }
            }
        }
        // Updating links
        const linksUpdate = (input as RoutineVersionUpdateInput).nodeLinksUpdate ?? [];
        for (const link of linksUpdate) {
            if (link.fromConnect) linkData[link.id].link.fromId = link.fromConnect;
            if (link.toConnect) linkData[link.id].link.toId = link.toConnect;
        }
        // Removing links
        const linksDelete = (input as RoutineVersionUpdateInput).nodeLinksDelete ?? [];
        for (const linkId of linksDelete) {
            delete linkData[linkId];
        }
        // Adding nodes
        for (const node of input.nodesCreate ?? []) {
            if (node.nodeRoutineListCreate) {
                // When adding nodes, subroutines can only be connected
                const subroutineIds = (node.nodeRoutineListCreate.itemsCreate ?? []).map(item => item.routineVersionConnect);
                connectingSubroutineDataIds.push(...subroutineIds); // This will add the full data to nodeData later
            } else {
                nodeData[node.id] = { rootRoutineId: input.id, subroutines: [] }
            }
        }
        // Updating nodes
        const nodesUpdate = (input as RoutineVersionUpdateInput).nodesUpdate ?? [];
        for (const node of nodesUpdate) {
            // Ignore if routine list is not being updated
            if (!node.nodeRoutineListUpdate) continue;
            // Handle items being added. Only supports connecting existing subroutines
            const itemsAdding = node.nodeRoutineListUpdate.itemsCreate ?? [];
            for (const item of itemsAdding) {
                // Add to connecting subroutines so we can query data for them
                connectingSubroutineDataIds.push(item.routineVersionConnect);
            }
            // Handle items being updated
            const itemsUpdating = node.nodeRoutineListUpdate.itemsUpdate ?? [];
            for (const item of itemsUpdating) {
                // Ignore if routine is not being updated
                if (!item.routineVersionUpdate) continue;
                // Add to list of subroutines being updated
                updatingSubroutineData.push(item.routineVersionUpdate);
            }
            // Handle items being removed
            const itemsRemoving = node.nodeRoutineListUpdate.itemsDelete ?? [];
            for (const itemId of itemsRemoving) {
                // Find the subroutine being removed
                const subroutineId = subroutineItemData[itemId];
                // Remove subroutine from node
                nodeData[node.id].subroutines = nodeData[node.id].subroutines.filter(subroutine => subroutine.id !== subroutineId);
            }
        }
        // Removing nodes
        const nodesDelete = (input as RoutineVersionUpdateInput).nodesDelete ?? [];
        for (const nodeId of nodesDelete) {
            delete nodeData[nodeId];
        }
    }
    // Query for missing subroutine data
    if (connectingSubroutineDataIds.length > 0) {
        const {
            linkData: connectingSubroutineLinkData,
            nodeData: connectingSubroutineNodeData,
            subroutineItemData: connectingSubroutineItemData,
        } = await groupRoutineVersionData(connectingSubroutineDataIds, prisma);
        updatingSubroutineIds.push(...connectingSubroutineDataIds);
        Object.assign(linkData, connectingSubroutineLinkData);
        Object.assign(nodeData, connectingSubroutineNodeData);
        Object.assign(subroutineItemData, connectingSubroutineItemData);
    }
    // Recurse on subroutines being updated
    if (updatingSubroutineData.length > 0) {
        const {
            updatingSubroutineIds: recursedUpdatingSubroutineIds,
            dataWeights: recursedDataWeights,
        } = await calculateWeightData(prisma, languages, updatingSubroutineData, [...disallowIds, ...inputIds]);
        updatingSubroutineIds.push(...recursedUpdatingSubroutineIds);
        for (let i = 0; i < recursedDataWeights.length; i++) {
            let currWeight = recursedDataWeights[i];
            // Find nodes that contain the subroutine
            const currNodes = Object.values(nodeData).filter(node => node.subroutines.some(subroutine => subroutine.id === updatingSubroutineData[i].id));
            // Add or replace the weight data to each node's subroutines array
            for (const node of currNodes) {
                const subroutineIndex = node.subroutines.findIndex(subroutine => subroutine.id === updatingSubroutineData[i].id);
                if (subroutineIndex === -1) {
                    node.subroutines.push(currWeight);
                } else {
                    node.subroutines[subroutineIndex] = currWeight;
                }
            }
        }
    }
    // Calculate weights for each root routine
    // Map rootRoutineId to nodes and links
    const nodesByRootRoutineId: { [rootRoutineId: string]: { nodeId: string, subroutines: (WeightData & { id: string })[] }[] } = {};
    const linksByRootRoutineId: { [rootRoutineId: string]: LinkData[] } = {};
    for (const nodeId in nodeData) {
        const node = nodeData[nodeId];
        if (!nodesByRootRoutineId[node.rootRoutineId]) nodesByRootRoutineId[node.rootRoutineId] = [];
        nodesByRootRoutineId[node.rootRoutineId].push({ nodeId, subroutines: node.subroutines });
    }
    for (const linkId in linkData) {
        const link = linkData[linkId];
        if (!linksByRootRoutineId[link.rootRoutineId]) linksByRootRoutineId[link.rootRoutineId] = [];
        linksByRootRoutineId[link.rootRoutineId].push(link.link);
    }
    // For each root routine, calculate the weights using its nodes and links
    const dataWeights: (WeightData & { id: string })[] = [];
    for (const rootRoutineId in nodesByRootRoutineId) {
        const nodes = nodesByRootRoutineId[rootRoutineId];
        const links = linksByRootRoutineId[rootRoutineId];
        // Combine the weights of all subroutines in each node
        const squishedNodes: { [nodeId: string]: WeightData } = {};
        for (const node of nodes) {
            const squishedNode = node.subroutines.reduce((acc, { complexity, simplicity, optionalInputs, allInputs }) => {
                return {
                    complexity: acc.complexity + complexity,
                    simplicity: acc.simplicity + simplicity,
                    optionalInputs: acc.optionalInputs + optionalInputs,
                    allInputs: acc.allInputs + allInputs,
                };
            }, { complexity: 0, simplicity: 0, optionalInputs: 0, allInputs: 0 });
            squishedNodes[node.nodeId] = squishedNode;
        }
        // Calculate shortest and longest weighted paths
        const [shortest, longest] = calculateShortestLongestWeightedPath(squishedNodes, links, languages);
        // Add with +1, so that nesting routines has a small (but not zero) factor in determining weight
        dataWeights.push({
            id: rootRoutineId,
            complexity: longest + 1,
            simplicity: shortest + 1,
            // Use the inputs for the root routine (i.e. ignore inputs for subroutines)
            optionalInputs: optionalRootRutineInputCounts[rootRoutineId] ?? 0,
            allInputs: allRootRoutineInputCounts[rootRoutineId] ?? 0,
        });
    }
    return { updatingSubroutineIds, dataWeights };
}