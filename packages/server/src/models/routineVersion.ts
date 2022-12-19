import { routineVersionValidation } from "@shared/validation";
import { CustomError, Trigger } from "../events";
import { RoutineCreateInput, RoutineUpdateInput, NodeCreateInput, NodeUpdateInput, NodeRoutineListItem, NodeRoutineListCreateInput, NodeRoutineListItemCreateInput, NodeRoutineListUpdateInput, RoutineVersionSortBy, SessionUser, RoutineVersionSearchInput, RoutineVersionCreateInput, RoutineVersion, RoutineVersionPermission, RoutineVersionUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, Validator, Mutater, Displayer, ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { RunRoutineModel } from "./runRoutine";
import { PartialGraphQLInfo, SelectWrap } from "../builders/types";
import { addSupplementalFields, modelToGraphQL, padSelect, permissionsSelectHelper, selectHelper, toPartialGraphQLInfo } from "../builders";
import { bestLabel, oneIsPublic } from "../utils";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RoutineVersionCreateInput,
    GqlUpdate: RoutineVersionUpdateInput,
    GqlModel: RoutineVersion,
    GqlSearch: RoutineVersionSearchInput,
    GqlSort: RoutineVersionSortBy,
    GqlPermission: RoutineVersionPermission,
    PrismaCreate: Prisma.routine_versionUpsertArgs['create'],
    PrismaUpdate: Prisma.routine_versionUpsertArgs['update'],
    PrismaModel: Prisma.routine_versionGetPayload<SelectWrap<Prisma.routine_versionSelect>>,
    PrismaSelect: Prisma.routine_versionSelect,
    PrismaWhere: Prisma.routine_versionWhereInput,
}

type NodeWeightData = {
    simplicity: number,
    complexity: number,
    optionalInputs: number,
    allInputs: number,
}

const __typename = 'RoutineVersion' as const;

const suppFields = ['runs'] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    gqlRelMap: {
        __typename,
        comments: 'Comment',
        createdBy: 'User',
        forks: 'Routine',
        inputs: 'RoutineVersionInput',
        nodes: 'Node',
        outputs: 'RoutineVersionOutput',
        parent: 'Routine',
        project: 'Project',
        reports: 'Report',
        resourceLists: 'ResourceList',
        root: 'Routine',
        suggestedNextByRoutineVersion: 'RoutineVersion',
        starredBy: 'User',
        tags: 'Tag',
    },
    prismaRelMap: {
        __typename,
        apiVersion: 'ApiVersion',
        comments: 'Comment',
        reports: 'Report',
        smartContractVersion: 'SmartContractVersion',
        nodes: 'Node',
        nodeLinks: 'NodeLink',
        resourceList: 'ResourceList',
        root: 'Routine',
        forks: 'Routine',
        inputs: 'RoutineVersionInput',
        outputs: 'RoutineVersionOutput',
        pullRequest: 'PullRequest',
        runRoutines: 'RunRoutine',
        runSteps: 'RunRoutineStep',
        suggestedNextByRoutineVersion: 'RoutineVersion',
    },
    joinMap: {
        tags: 'tag',
        starredBy: 'user',
        suggestedNextByRoutineVersion: 'toRoutineVersion',
    },
    countFields: ['commentsCount', 'nodesCount', 'reportsCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, objects, partial, prisma, userData }) => ({
            runs: async () => {
                if (!userData) return new Array(objects.length).fill([]);
                // Find requested fields of runs. Also add routineId, so we 
                // can associate runs with their routine
                const runPartial: PartialGraphQLInfo = {
                    ...toPartialGraphQLInfo(partial.runs as PartialGraphQLInfo, RunRoutineModel.format.gqlRelMap, userData.languages, true),
                    routineVersionId: true
                }
                // Query runs made by user
                let runs: any[] = await prisma.run_routine.findMany({
                    where: {
                        AND: [
                            { routineVersion: { root: { id: { in: ids } } } },
                            { user: { id: userData.id } }
                        ]
                    },
                    ...selectHelper(runPartial)
                });
                // Format runs to GraphQL
                runs = runs.map(r => modelToGraphQL(r, runPartial));
                // Add supplemental fields
                runs = await addSupplementalFields(prisma, userData, runs, runPartial);
                // Split runs by id
                const routineRuns = ids.map((id) => runs.filter(r => r.routineId === id));
                return routineRuns;
            },
        }),
    },
})

const searcher = (): Searcher<Model> => ({
    defaultSort: RoutineVersionSortBy.DateCompletedDesc,
    sortBy: RoutineVersionSortBy,
    searchFields: [
        'createdById',
        'createdTimeFrame',
        'directoryListingsId',
        'excludeIds',
        'isCompleteWithRoot',
        'isCompleteWithRootExcludeOwnedByOrganizationId',
        'isCompleteWithRootExcludeOwnedByUserId',
        'isInternalWithRoot',
        'isInternalWithRootExcludeOwnedByOrganizationId',
        'isInternalWithRootExcludeOwnedByUserId',
        'translationLanguages',
        'maxComplexity',
        'maxSimplicity',
        'maxTimesCompleted',
        'minComplexity',
        'minScoreRoot',
        'minSimplicity',
        'minStarsRoot',
        'minTimesCompleted',
        'minViewsRoot',
        'ownedByOrganizationId',
        'ownedByUserId',
        'parentId',
        'reportId',
        'rootId',
        'tags',
        'updatedTimeFrame',
        'visibility',
    ],
    searchStringQuery: () => ({
        OR: [
            'transDescriptionWrapped',
            'transNameWrapped',
            { root: 'tagsWrapped' },
            { root: 'labelsWrapped' },
        ]
    }),
})

const validator = (): Validator<Model> => ({
    isTransferable: false,
    maxObjects: 1000000,
    permissionsSelect: (...params) => ({
        id: true,
        hasCompleteVersion: true,
        isDeleted: true,
        isPrivate: true,
        isInternal: true,
        permissions: true,
        createdBy: padSelect({ id: true }),
        ...permissionsSelectHelper({
            root: 'Routine',
        }, ...params),
    }),
    permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ({
        // canComment: async () => !isDeleted && (isAdmin || isPublic),
        // canDelete: async () => isAdmin && !isDeleted,
        // canEdit: async () => isAdmin && !isDeleted,
        // canReport: async () => !isAdmin && !isDeleted && isPublic,
        // canRun: async () => !isDeleted && (isAdmin || isPublic),
        // canStar: async () => !isDeleted && (isAdmin || isPublic),
        // canView: async () => !isDeleted && (isAdmin || isPublic),
        // canVote: async () => !isDeleted && (isAdmin || isPublic),
    } as any),
    owner: (data) => ({
        // Organization: data.ownedByOrganization,
        // User: data.ownedByUser,
    } as any),
    isDeleted: (data) => data.isDeleted,// || latest(data.versions)?.isDeleted,
    isPublic: (data, languages) => data.isPrivate === false &&
        data.isDeleted === false &&
        // data.isInternal === false &&
        //latest(data.versions)?.isPrivate === false &&
        //latest(data.versions)?.isDeleted === false &&
        oneIsPublic<Prisma.routineSelect>(data, [
            ['ownedByOrganization', 'Organization'],
            ['ownedByUser', 'User'],
        ], languages),
    visibility: {
        private: {
            isPrivate: true,
            // OR: [
            //     { isPrivate: true },
            //     { root: { isPrivate: true } },
            // ]
        },
        public: {
            isPrivate: false,
            // AND: [
            //     { isPrivate: false },
            //     { root: { isPrivate: false } },
            // ]
        },
        owner: (userId) => ({
            root: RoutineVersionModel.validate!.visibility.owner(userId),
        }),
    },
    // if (createMany) {
    //     createMany.forEach(input => this.validateNodePositions(input));
    // }
    // if (updateMany) {
    //     // Query version numbers and isCompletes of existing routines. 
    //     // Can only update if version number is greater, or if version number is the same and isComplete is false
    //     //TODO
    //     updateMany.forEach(input => this.validateNodePositions(input.data));
    // }

    // Also check profanity on input/output's name
})

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
const calculateShortestLongestWeightedPath = (
    nodes: { [id: string]: NodeWeightData },
    edges: { fromId: string, toId: string }[],
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
 * Calculates the maximum and minimum complexity of a routine based on the number of steps. 
 * Simplicity is a the minimum number of inputs and decisions required to complete the routine, while 
 * complexity is the maximum.
 * @param prisma The prisma client
 * @param languages Preferred languages for error messages
 * @param data The routine data, either a create or update
 * @param versionId If existing data, its version ID
 * @returns [simplicity, complexity] Numbers representing the shorted and longest weighted paths
 */
const calculateComplexity = async (
    prisma: PrismaType,
    languages: string[],
    data: RoutineCreateInput | RoutineUpdateInput,
    versionId?: string | null
): Promise<[number, number]> => {
    // If the routine is being updated, Find the complexity of existing subroutines
    let existingRoutine;
    if (versionId) {
        existingRoutine = await prisma.routine_version.findUnique({
            where: { id: versionId },
            select: {
                nodeLinks: { select: { id: true, fromId: true, toId: true } },
                nodes: {
                    select: {
                        id: true, // Needed to associate with links
                        nodeRoutineList: {
                            select: {
                                items: {
                                    select: {
                                        routineVersion: { select: { id: true, complexity: true, simplicity: true } }
                                    }
                                },
                            }
                        },
                    }
                },
            }
        })
    }
    // Calculate the list of links after mutations are applied
    let nodeLinks: any[] = existingRoutine?.nodeLinks || [];
    if (data.nodeLinksCreate) nodeLinks = nodeLinks.concat(data.nodeLinksCreate);
    if ((data as RoutineUpdateInput).nodeLinksUpdate) {
        nodeLinks = nodeLinks.map(link => {
            const updatedLink = (data as RoutineUpdateInput).nodeLinksUpdate?.find(updatedLink => link.id && updatedLink.id === link.id);
            return updatedLink ? { ...link, ...updatedLink } : link;
        })
    }
    if ((data as RoutineUpdateInput).nodeLinksDelete) {
        nodeLinks = nodeLinks.filter(link => !(data as RoutineUpdateInput).nodeLinksDelete?.find(deletedLink => deletedLink === link.id));
    }
    // Calculate the list of nodes after mutations are applied
    let nodes: any[] = existingRoutine?.nodes || [];
    if (data.nodesCreate) nodes = nodes.concat(data.nodesCreate);
    if ((data as RoutineUpdateInput).nodesUpdate) {
        nodes = nodes.map(node => {
            const updatedNode = (data as RoutineUpdateInput).nodesUpdate?.find(updatedNode => node.id && updatedNode.id === node.id);
            return updatedNode ? { ...node, ...updatedNode } : node;
        })
    }
    if ((data as RoutineUpdateInput).nodesDelete) {
        nodes = nodes.filter(node => !(data as RoutineUpdateInput).nodesDelete?.find(deletedNode => deletedNode === node.id));
    }
    // Initialize node dictionary to map node IDs to their subroutine IDs
    const subroutineIdsByNode: { [id: string]: string[] } = {};
    // Find the ID of every subroutine
    const subroutineIds: string[] = nodes.map((node: any | NodeCreateInput | NodeUpdateInput) => {
        // Calculate the list of subroutines after mutations are applied
        let ids: string[] = node.nodeRoutineList?.routines?.map((item: NodeRoutineListItem) => item.routineVersion.id) ?? [];
        if ((data as NodeCreateInput).nodeRoutineListCreate) {
            const listCreate = (data as NodeCreateInput).nodeRoutineListCreate as NodeRoutineListCreateInput;
            // Handle creates
            ids = ids.concat(listCreate.itemsCreate?.map((item: NodeRoutineListItemCreateInput) => item.routineVersionConnect) ?? []);
        }
        else if ((data as NodeUpdateInput).nodeRoutineListUpdate) {
            const listUpdate = (data as NodeUpdateInput).nodeRoutineListUpdate as NodeRoutineListUpdateInput;
            // Handle creates
            ids = ids.concat(listUpdate.itemsCreate?.map((item: NodeRoutineListItemCreateInput) => item.routineVersionConnect) ?? []);
            // Handle deletes. No need to handle updates, as routine items cannot switch their routine associations
            if (listUpdate.itemsDelete) {
                ids = ids.filter(id => !listUpdate.itemsDelete?.find(deletedId => deletedId === id));
            }
        }
        subroutineIdsByNode[node.id] = ids;
        return ids
    }).flat();
    // Query every subroutine's complexity, simplicity, and number of inputs
    const subroutineWeightData = await prisma.routine_version.findMany({
        where: { id: { in: subroutineIds } },
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
    })
    // Convert compexity/simplicity to a map for easy lookup
    let subroutineWeightDataDict: { [routineId: string]: NodeWeightData } = {};
    for (const sub of subroutineWeightData) {
        subroutineWeightDataDict[sub.id] = {
            complexity: sub.complexity,
            simplicity: sub.simplicity,
            optionalInputs: sub.inputs.filter(input => !input.isRequired).length,
            allInputs: sub.inputs.length,
        };
    }
    // Calculate the complexity/simplicity of each node. If node has no subroutines, its complexity is 0 (e.g. start node, end node)
    let nodeWeightDataDict: { [nodeId: string]: NodeWeightData } = {};
    for (const node of nodes) {
        nodeWeightDataDict[node.id] = {
            complexity: subroutineIdsByNode[node.id]?.reduce((acc, cur) => acc + subroutineWeightDataDict[cur].complexity, 0) || 0,
            simplicity: subroutineIdsByNode[node.id]?.reduce((acc, cur) => acc + subroutineWeightDataDict[cur].simplicity, 0) || 0,
            optionalInputs: subroutineIdsByNode[node.id]?.reduce((acc, cur) => acc + subroutineWeightDataDict[cur].optionalInputs, 0) || 0,
            allInputs: subroutineIdsByNode[node.id]?.reduce((acc, cur) => acc + subroutineWeightDataDict[cur].allInputs, 0) || 0,
        }
    }
    // Using the node links, determine the most complex path through the routine
    const [shortest, longest] = calculateShortestLongestWeightedPath(nodeWeightDataDict, nodeLinks, languages);
    // return with +1, so that nesting routines has a small factor in determining weight
    return [shortest + 1, longest + 1];
}

/**
 * Validates node positions
 */
const validateNodePositions = (input: RoutineCreateInput | RoutineUpdateInput): void => {
    // // Check that node columnIndexes and rowIndexes are valid TODO query existing data to do this
    // let combinedNodes = [];
    // if (input.nodesCreate) combinedNodes.push(...input.nodesCreate);
    // if ((input as RoutineUpdateInput).nodesUpdate) combinedNodes.push(...(input as any).nodesUpdate);
    // if ((input as RoutineUpdateInput).nodesDelete) combinedNodes = combinedNodes.filter(node => !(input as any).nodesDelete.includes(node.id));
    // // Remove nodes that have duplicate rowIndexes and columnIndexes
    // console.log('unique nodes check', JSON.stringify(combinedNodes));
    // const uniqueNodes = uniqBy(combinedNodes, (n) => `${n.rowIndex}-${n.columnIndex}`);
    // if (uniqueNodes.length < combinedNodes.length) throw new CustomError('NodeDuplicatePosition', {});
    return;
}

/**
 * Handles authorized creates, updates, and deletes
 */
const mutater = (): Mutater<Model> => ({
    shape: {
        create: async ({ data, prisma, userData }) => {
            return {} as any;
        },
        update: async ({ data, prisma, userData, where }) => {
            return {} as any;
        },
    },
    trigger: {
        onCreated: ({ created, prisma, userData }) => {
            for (const c of created) {
                Trigger(prisma, userData.languages).createRoutine(userData.id, c.id as string);
            }
        },
        onUpdated: ({ authData, prisma, updated, updateInput, userData }) => {
            // // Initialize transfers, if any
            // asdfasdfasfd
            // // Handle new version triggers, if any versions have been created
            // // Loop through updated items
            // for (let i = 0; i < updated.length; i++) {
            //     const u = updated[i];
            //     const input = updateInput[i];
            //     const permissionsData = authData[u.id];
            //     const { Organization, User } = validator().owner(permissionsData as any);
            //     const owner: { __typename: 'Organization' | 'User', id: string } | null = Organization ?
            //         { __typename: 'Organization', id: Organization.id } :
            //         User ? { __typename: 'User', id: User.id } : null;
            //     const hasOriginalOwner = validator().hasOriginalOwner(permissionsData as any);
            //     const wasPublic = validator().isPublic(permissionsData as any, userData.languages);
            //     const hadCompletedVersion = validator().hasCompletedVersion(permissionsData as any);
            //     const isPublic = input.isPrivate !== undefined ? !input.isPrivate : wasPublic;
            //     const hasCompletedVersion = asdfasdfasdf
            //     // Check if new version was created
            //     if (input.versionLabel) {
            //         Trigger(prisma, userData.languages).objectNewVersion(
            //             userData.id,
            //             'Routine',
            //             u.id,
            //             owner,
            //             hasOriginalOwner,
            //             hadCompletedVersion && wasPublic,
            //             hasCompletedVersion && isPublic
            //         );
            //     }
            // }
        },
    },
    yup: routineVersionValidation,
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const RoutineVersionModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.routine_version,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,//mutater(),
    search: searcher(),
    validate: validator(),
    calculateComplexity,
    validateNodePositions,
})