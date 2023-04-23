import { CustomError } from "../events";
export const calculateShortestLongestWeightedPath = (nodes, edges, languages) => {
    for (const edge of edges) {
        if (!nodes[edge.toId] || !nodes[edge.fromId]) {
            throw new CustomError("0237", "UnlinkedNodes", languages, { failedEdge: edge });
        }
    }
    if (Object.keys(nodes).length === 0 || edges.length === 0)
        return [1, 1];
    const edgesByNode = {};
    edges.forEach(edge => {
        edgesByNode[edge.toId] = edgesByNode[edge.toId] ? edgesByNode[edge.toId].concat(edge) : [edge];
    });
    const getShortLong = (currentNodeId, visitedEdges, currShortest, currLongest) => {
        const fromEdges = edgesByNode[currentNodeId];
        if (!fromEdges || fromEdges.length === 0)
            return [currShortest, currLongest];
        if (fromEdges.every(edge => visitedEdges.some(visitedEdge => visitedEdge.fromId === edge.fromId && visitedEdge.toId === edge.toId)))
            return [-1, -1];
        const edgeShorts = [];
        const edgeLongs = [];
        for (const edge of fromEdges) {
            let weight = nodes[edge.fromId];
            if (fromEdges.length > 1)
                weight = { ...weight, complexity: weight.complexity + 1, simplicity: weight.simplicity + 1 };
            const newVisitedEdges = visitedEdges.concat([edge]);
            const [shortest, longest] = getShortLong(edge.fromId, newVisitedEdges, currShortest + weight.simplicity + weight.optionalInputs, currLongest + weight.complexity + weight.allInputs);
            if (shortest !== -1)
                edgeShorts.push(shortest);
            if (longest !== -1)
                edgeLongs.push(longest);
        }
        const shortest = edgeShorts.length > 0 ? Math.min(...edgeShorts) : -1;
        const longest = edgeLongs.length > 0 ? Math.max(...edgeLongs) : -1;
        return [shortest, longest];
    };
    const endNodes = Object.keys(nodes).filter(nodeId => !edges.find(e => e.fromId === nodeId));
    const distances = endNodes.map(nodeId => getShortLong(nodeId, [], 0, 0));
    return [
        Math.min(...distances.map(d => d[0])),
        Math.max(...distances.map(d => d[1])),
    ];
};
const routineVersionSelect = ({
    id: true,
    nodeLinks: { select: { id: true, fromId: true, toId: true } },
    nodes: {
        select: {
            id: true,
            routineList: {
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
                                            isRequired: true,
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
    },
    inputs: {
        select: {
            id: true,
            isRequired: true,
        },
    },
});
const groupRoutineVersionData = async (ids, prisma) => {
    const linkData = {};
    const nodeData = {};
    const subroutineItemData = {};
    const optionalRoutineVersionInputCounts = {};
    const allRoutineVersionInputCounts = {};
    const data = await prisma.routine_version.findMany({
        where: { id: { in: ids } },
        select: routineVersionSelect,
    });
    for (const routineVersion of data) {
        for (const link of routineVersion.nodeLinks) {
            linkData[link.id] = {
                fromId: link.fromId,
                toId: link.toId,
            };
        }
        for (const node of routineVersion.nodes) {
            if (node.routineList) {
                nodeData[node.id] = {
                    subroutines: node.routineList.items.map(item => {
                        subroutineItemData[item.id] = item.routineVersion.id;
                        return {
                            id: item.routineVersion.id,
                            complexity: item.routineVersion.complexity,
                            simplicity: item.routineVersion.simplicity,
                            allInputs: item.routineVersion.inputs.length,
                            optionalInputs: item.routineVersion.inputs.filter(input => !input.isRequired).length,
                        };
                    }),
                };
            }
            else {
                nodeData[node.id] = {
                    subroutines: [],
                };
            }
        }
    }
    for (const routineVersion of data) {
        optionalRoutineVersionInputCounts[routineVersion.id] = routineVersion.inputs.filter(input => !input.isRequired).length;
        allRoutineVersionInputCounts[routineVersion.id] = routineVersion.inputs.length;
    }
    return {
        linkData,
        nodeData,
        subroutineItemData,
        optionalRoutineVersionInputCounts,
        allRoutineVersionInputCounts,
    };
};
export const calculateWeightData = async (prisma, languages, inputs, disallowIds) => {
    const inputIds = inputs.map(i => i.id);
    if (inputIds.some(id => disallowIds.includes(id))) {
        throw new CustomError("0370", "InvalidArgs", languages);
    }
    const linkData = {};
    const nodeData = {};
    const subroutineItemData = {};
    const optionalRoutineVersionInputCounts = {};
    const allRoutineVersionInputCounts = {};
    const connectingSubroutineDataIds = [];
    const updatingSubroutineData = [];
    const updatingSubroutineIds = inputIds;
    const { linkData: existingLinkData, nodeData: existingNodeData, subroutineItemData: existingSubroutineItemData, optionalRoutineVersionInputCounts: existingOptionalRoutineVersionInputCounts, allRoutineVersionInputCounts: existingAllRoutineVersionInputCounts, } = await groupRoutineVersionData(inputs.map(i => i.id), prisma);
    Object.assign(linkData, existingLinkData);
    Object.assign(nodeData, existingNodeData);
    Object.assign(subroutineItemData, existingSubroutineItemData);
    Object.assign(optionalRoutineVersionInputCounts, existingOptionalRoutineVersionInputCounts);
    Object.assign(allRoutineVersionInputCounts, existingAllRoutineVersionInputCounts);
    for (const rVerCreateOrUpdate of inputs) {
        for (const link of rVerCreateOrUpdate.nodeLinksCreate ?? []) {
            linkData[link.id] = {
                routineVersionId: rVerCreateOrUpdate.id,
                link: {
                    fromId: link.fromConnect,
                    toId: link.toConnect,
                },
            };
        }
        const linksUpdate = rVerCreateOrUpdate.nodeLinksUpdate ?? [];
        for (const link of linksUpdate) {
            if (link.fromConnect)
                linkData[link.id].link.fromId = link.fromConnect;
            if (link.toConnect)
                linkData[link.id].link.toId = link.toConnect;
        }
        const linksDelete = rVerCreateOrUpdate.nodeLinksDelete ?? [];
        for (const linkId of linksDelete) {
            delete linkData[linkId];
        }
        for (const node of rVerCreateOrUpdate.nodesCreate ?? []) {
            if (node.routineListCreate) {
                const subroutineIds = (node.routineListCreate.itemsCreate ?? []).map(item => item.routineVersionConnect);
                connectingSubroutineDataIds.push(...subroutineIds);
            }
            else {
                nodeData[node.id] = { routineVersionId: rVerCreateOrUpdate.id, subroutines: [] };
            }
        }
        const nodesUpdate = rVerCreateOrUpdate.nodesUpdate ?? [];
        for (const node of nodesUpdate) {
            if (!node.routineListUpdate)
                continue;
            const itemsAdding = node.routineListUpdate.itemsCreate ?? [];
            for (const item of itemsAdding) {
                connectingSubroutineDataIds.push(item.routineVersionConnect);
            }
            const itemsUpdating = node.routineListUpdate.itemsUpdate ?? [];
            for (const item of itemsUpdating) {
                if (!item.routineVersionUpdate)
                    continue;
                updatingSubroutineData.push(item.routineVersionUpdate);
            }
            const itemsRemoving = node.routineListUpdate.itemsDelete ?? [];
            for (const itemId of itemsRemoving) {
                const subroutineId = subroutineItemData[itemId];
                nodeData[node.id].subroutines = nodeData[node.id].subroutines.filter(subroutine => subroutine.id !== subroutineId);
            }
        }
        const nodesDelete = rVerCreateOrUpdate.nodesDelete ?? [];
        for (const nodeId of nodesDelete) {
            delete nodeData[nodeId];
        }
    }
    if (connectingSubroutineDataIds.length > 0) {
        const { linkData: connectingSubroutineLinkData, nodeData: connectingSubroutineNodeData, subroutineItemData: connectingSubroutineItemData, } = await groupRoutineVersionData(connectingSubroutineDataIds, prisma);
        updatingSubroutineIds.push(...connectingSubroutineDataIds);
        Object.assign(linkData, connectingSubroutineLinkData);
        Object.assign(nodeData, connectingSubroutineNodeData);
        Object.assign(subroutineItemData, connectingSubroutineItemData);
    }
    if (updatingSubroutineData.length > 0) {
        const { updatingSubroutineIds: recursedUpdatingSubroutineIds, dataWeights: recursedDataWeights, } = await calculateWeightData(prisma, languages, updatingSubroutineData, [...disallowIds, ...inputIds]);
        updatingSubroutineIds.push(...recursedUpdatingSubroutineIds);
        for (let i = 0; i < recursedDataWeights.length; i++) {
            const currWeight = recursedDataWeights[i];
            const currNodes = Object.values(nodeData).filter(node => node.subroutines.some(subroutine => subroutine.id === updatingSubroutineData[i].id));
            for (const node of currNodes) {
                const subroutineIndex = node.subroutines.findIndex(subroutine => subroutine.id === updatingSubroutineData[i].id);
                if (subroutineIndex === -1) {
                    node.subroutines.push(currWeight);
                }
                else {
                    node.subroutines[subroutineIndex] = currWeight;
                }
            }
        }
    }
    const nodesByRVerId = {};
    const linksByRVerId = {};
    for (const nodeId in nodeData) {
        const node = nodeData[nodeId];
        if (!nodesByRVerId[node.routineVersionId])
            nodesByRVerId[node.routineVersionId] = [];
        nodesByRVerId[node.routineVersionId].push({ nodeId, subroutines: node.subroutines });
    }
    for (const linkId in linkData) {
        const link = linkData[linkId];
        if (!linksByRVerId[link.routineVersionId])
            linksByRVerId[link.routineVersionId] = [];
        linksByRVerId[link.routineVersionId].push(link.link);
    }
    const dataWeights = [];
    for (const versionId in nodesByRVerId) {
        const nodes = nodesByRVerId[versionId];
        const links = linksByRVerId[versionId];
        const squishedNodes = {};
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
        const [shortest, longest] = calculateShortestLongestWeightedPath(squishedNodes, links, languages);
        dataWeights.push({
            id: versionId,
            complexity: longest + 1,
            simplicity: shortest + 1,
            optionalInputs: optionalRoutineVersionInputCounts[versionId] ?? 0,
            allInputs: allRoutineVersionInputCounts[versionId] ?? 0,
        });
    }
    return { updatingSubroutineIds, dataWeights };
};
//# sourceMappingURL=routineComplexity.js.map