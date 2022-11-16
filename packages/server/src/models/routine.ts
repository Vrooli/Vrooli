import { addCountFieldsHelper, addJoinTablesHelper, addSupplementalFields, combineQueries, exceptionsBuilder, getSearchStringQueryHelper, modelToGraphQL, relationshipBuilderHelper, removeCountFieldsHelper, removeJoinTablesHelper, selectHelper, toPartialGraphQLInfo, visibilityBuilder } from "./builder";
import { inputTranslationCreate, inputTranslationUpdate, outputTranslationCreate, outputTranslationUpdate, routinesCreate, routineTranslationCreate, routineTranslationUpdate, routinesUpdate } from "@shared/validation";
import { CODE, ResourceListUsedFor } from "@shared/consts";
import { organizationValidator } from "./organization";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { NodeModel } from "./node";
import { StandardModel } from "./standard";
import { TranslationModel } from "./translation";
import { ResourceListModel } from "./resourceList";
import { ViewModel } from "./view";
import { runFormatter } from "./run";
import { CustomError, genErrorCode, Trigger } from "../events";
import { Routine, RoutinePermission, RoutineSearchInput, RoutineCreateInput, RoutineUpdateInput, NodeCreateInput, NodeUpdateInput, NodeRoutineListItem, NodeRoutineListCreateInput, NodeRoutineListItemCreateInput, NodeRoutineListUpdateInput, RoutineSortBy } from "../schema/types";
import { PrismaType } from "../types";
import { cudHelper } from "./actions";
import { FormatConverter, PartialGraphQLInfo, Searcher, CUDInput, CUDResult, DuplicateInput, DuplicateResult, GraphQLModelType, Validator, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { getPermissions, oneIsPublic } from "./utils";

type NodeWeightData = {
    simplicity: number,
    complexity: number,
    optionalInputs: number,
    allInputs: number,
}

const joinMapper = { tags: 'tag', starredBy: 'user' };
const countMapper = { commentsCount: 'comments', nodesCount: 'nodes', reportsCount: 'reports' };
type SupplementalFields = 'isUpvoted' | 'isStarred' | 'isViewed' | 'permissionsRoutine' | 'runs' | 'versions';
export const routineFormatter = (): FormatConverter<Routine, SupplementalFields> => ({
    relationshipMap: {
        __typename: 'Routine',
        comments: 'Comment',
        creator: {
            root: {
                User: 'User',
                Organization: 'Organization',
            }
        },
        forks: 'Routine',
        inputs: 'InputItem',
        nodes: 'Node',
        outputs: 'OutputItem',
        owner: {
            root: {
                User: 'User',
                Organization: 'Organization',
            }
        },
        parent: 'Routine',
        project: 'Project',
        reports: 'Report',
        resourceLists: 'ResourceList',
        starredBy: 'User',
        tags: 'Tag',
    },
    rootFields: ['hasCompleteVersion', 'isDeleted', 'isInternal', 'isPrivate', 'votes', 'stars', 'views', 'permissions'],
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
    addCountFields: (partial) => addCountFieldsHelper(partial, countMapper),
    removeCountFields: (data) => removeCountFieldsHelper(data, countMapper),
    supplemental: {
        graphqlFields: ['isUpvoted', 'isStarred', 'isViewed', 'permissionsRoutine', 'runs', 'versions'],
        toGraphQL: ({ ids, objects, partial, prisma, userId }) => [
            ['isStarred', async () => await StarModel.query(prisma).getIsStarreds(userId, ids, 'Routine')],
            ['isUpvoted', async () => await VoteModel.query(prisma).getIsUpvoteds(userId, ids, 'Routine')],
            ['isViewed', async () => await ViewModel.query(prisma).getIsVieweds(userId, ids, 'Routine')],
            ['permissionsRoutine', async () => await getPermissions({ objectType: 'Routine', ids, prisma, userId })],
            ['runs', async () => {
                if (userId) {
                    // Find requested fields of runs. Also add routineId, so we 
                    // can associate runs with their routine
                    const runPartial = {
                        ...toPartialGraphQLInfo(partial.runs as PartialGraphQLInfo, runFormatter().relationshipMap),
                        routineId: true
                    }
                    if (runPartial === undefined) {
                        throw new CustomError(CODE.InternalError, 'Error converting query', { code: genErrorCode('0178') });
                    }
                    // Query runs made by user
                    let runs: any[] = await prisma.run.findMany({
                        where: {
                            AND: [
                                { routineVersion: { root: { id: { in: ids } } } },
                                { user: { id: userId } }
                            ]
                        },
                        ...selectHelper(runPartial)
                    });
                    // Format runs to GraphQL
                    runs = runs.map(r => modelToGraphQL(r, runPartial));
                    // Add supplemental fields
                    runs = await addSupplementalFields(prisma, userId, runs, runPartial);
                    // Split runs by id
                    const routineRuns = ids.map((id) => runs.filter(r => r.routineId === id));
                    return routineRuns;
                } else {
                    // Set all runs to empty array
                    return new Array(objects.length).fill([]);
                }
            }],
            ['versions', async () => {
                const groupData = await prisma.routine.findMany({
                    where: { id: { in: ids } },
                    select: { versions: { select: { id: true, versionLabel: true } } }
                });
                return groupData.map(g => g.versions);
            }],
        ],
    },
})

export const routineSearcher = (): Searcher<RoutineSearchInput> => ({
    defaultSort: RoutineSortBy.VotesDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [RoutineSortBy.CommentsAsc]: { comments: { _count: 'asc' } },
            [RoutineSortBy.CommentsDesc]: { comments: { _count: 'desc' } },
            [RoutineSortBy.ForksAsc]: { forks: { _count: 'asc' } },
            [RoutineSortBy.ForksDesc]: { forks: { _count: 'desc' } },
            [RoutineSortBy.DateCompletedAsc]: { completedAt: 'asc' },
            [RoutineSortBy.DateCompletedDesc]: { completedAt: 'desc' },
            [RoutineSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [RoutineSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [RoutineSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [RoutineSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [RoutineSortBy.StarsAsc]: { stars: 'asc' },
            [RoutineSortBy.StarsDesc]: { stars: 'desc' },
            [RoutineSortBy.VotesAsc]: { score: 'asc' },
            [RoutineSortBy.VotesDesc]: { score: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        return getSearchStringQueryHelper({
            searchString,
            resolver: ({ insensitive }) => ({
                OR: [
                    { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
                    { translations: { some: { language: languages ? { in: languages } : undefined, title: { ...insensitive } } } },
                    { tags: { some: { tag: { tag: { ...insensitive } } } } },
                ]
            })
        })
    },
    customQueries(input: RoutineSearchInput, userId: string | null | undefined): { [x: string]: any } {
        const isComplete = exceptionsBuilder({
            canQuery: ['createdByOrganization', 'createdByUser', 'organization.id', 'project.id', 'user.id'],
            exceptionField: 'isCompleteExceptions',
            input,
            mainField: 'isComplete',
        })
        const isInternal = exceptionsBuilder({
            canQuery: ['createdByOrganization', 'createdByUser', 'organization.id', 'project.id', 'user.id'],
            defaultValue: false,
            exceptionField: 'isInternalExceptions',
            input,
            mainField: 'isInternal',
        })
        console.log('before routine customqueries combinequeries', JSON.stringify(isComplete), '\n', JSON.stringify(isInternal), '\n');
        return combineQueries([
            isComplete,
            isInternal,
            visibilityBuilder({ model: RoutineModel, userId, visibility: input.visibility }),
            (input.excludeIds !== undefined ? { NOT: { id: { in: input.excludeIds } } } : {}),
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            (input.minComplexity !== undefined ? { complexity: { gte: input.minComplexity } } : {}),
            (input.maxComplexity !== undefined ? { complexity: { lte: input.maxComplexity } } : {}),
            (input.minSimplicity !== undefined ? { simplicity: { gte: input.minSimplicity } } : {}),
            (input.maxSimplicity !== undefined ? { simplicity: { lte: input.maxSimplicity } } : {}),
            (input.maxTimesCompleted !== undefined ? { timesCompleted: { lte: input.maxTimesCompleted } } : {}),
            (input.minScore !== undefined ? { score: { gte: input.minScore } } : {}),
            (input.minStars !== undefined ? { stars: { gte: input.minStars } } : {}),
            (input.minTimesCompleted !== undefined ? { timesCompleted: { gte: input.minTimesCompleted } } : {}),
            (input.minViews !== undefined ? { views: { gte: input.minViews } } : {}),
            (input.resourceLists !== undefined ? { resourceLists: { some: { translations: { some: { title: { in: input.resourceLists } } } } } } : {}),
            (input.resourceTypes !== undefined ? { resourceLists: { some: { usedFor: ResourceListUsedFor.Display as any, resources: { some: { usedFor: { in: input.resourceTypes } } } } } } : {}),
            (input.userId !== undefined ? { userId: input.userId } : {}),
            (input.organizationId !== undefined ? { organizationId: input.organizationId } : {}),
            (input.projectId !== undefined ? { projectId: input.projectId } : {}),
            (input.parentId !== undefined ? { parentId: input.parentId } : {}),
            (input.reportId !== undefined ? { reports: { some: { id: input.reportId } } } : {}),
            (input.tags !== undefined ? { tags: { some: { tag: { tag: { in: input.tags } } } } } : {}),
        ])
    },
})

export const routineValidator = (): Validator<
    RoutineCreateInput,
    RoutineUpdateInput,
    Routine,
    Prisma.routine_versionGetPayload<{ select: { [K in keyof Required<Prisma.routine_versionSelect>]: true } }>,
    RoutinePermission,
    Prisma.routine_versionSelect,
    Prisma.routine_versionWhereInput
> => ({
    validateMap: {
        __typename: 'Routine',
        fasdfasd
    },
    permissionsSelect: {
        id: true,
        isComplete: true,
        isPrivate: true,
        isDeleted: true,
        root: {
            select: {
                isDeleted: true,
                isPrivate: true,
                isInternal: true,
                permissions: true,
                user: { select: { id: true } },
                organization: { select: organizationValidator().permissionsSelect },
            }
        }
    },
    permissionsFromSelect: (select, userId) => asdf as any,
    isPublic: (data) => data.isPrivate === false &&
        data.isDeleted === false &&
        data.root?.isDeleted === false &&
        data.root?.isInternal === false &&
        data.root?.isPrivate === false && oneIsPublic<Prisma.routineSelect>(data.root, [
            ['organization', 'Organization'],
            ['user', 'User'],
        ]),
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
 * @returns [shortestPath, longestPath] The shortest and longest weighted distance
 */
export const calculateShortestLongestWeightedPath = (nodes: { [id: string]: NodeWeightData }, edges: { fromId: string, toId: string }[]): [number, number] => {
    // First, check that all edges point to valid nodes. 
    // If this isn't the case, this algorithm could run into an error
    for (const edge of edges) {
        if (!nodes[edge.toId] || !nodes[edge.fromId]) {
            throw new CustomError(CODE.InvalidArgs, 'Could not calculate complexity/simplicity: not all edges map to existing nodes', { code: genErrorCode('0237'), failedEdge: edge })
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
 * Handles authorized creates, updates, and deletes
 */
export const routineMutater = (prisma: PrismaType): Mutater<Routine> => ({
    /**
     * Calculates the maximum and minimum complexity of a routine based on the number of steps. 
     * Simplicity is a the minimum number of inputs and decisions required to complete the routine, while 
     * complexity is the maximum.
     * @param data The routine data, either a create or update
     * @param versionId If existing data, its version ID
     * @returns [simplicity, complexity] Numbers representing the shorted and longest weighted paths
     */
    async calculateComplexity(data: RoutineCreateInput | RoutineUpdateInput, versionId?: string | null): Promise<[number, number]> {
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
                                    routines: {
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
                ids = ids.concat(listCreate.routinesCreate?.map((item: NodeRoutineListItemCreateInput) => item.routineConnect) ?? []);
            }
            else if ((data as NodeUpdateInput).nodeRoutineListUpdate) {
                const listUpdate = (data as NodeUpdateInput).nodeRoutineListUpdate as NodeRoutineListUpdateInput;
                // Handle creates
                ids = ids.concat(listUpdate.routinesCreate?.map((item: NodeRoutineListItemCreateInput) => item.routineConnect) ?? []);
                // Handle deletes. No need to handle updates, as routine items cannot switch their routine associations
                if (listUpdate.routinesDelete) {
                    ids = ids.filter(id => !listUpdate.routinesDelete?.find(deletedId => deletedId === id));
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
        const [shortest, longest] = calculateShortestLongestWeightedPath(nodeWeightDataDict, nodeLinks);
        // return with +1, so that nesting routines has a small factor in determining weight
        return [shortest + 1, longest + 1];
    },
    /**
     * Validates node positions
     */
    validateNodePositions(input: RoutineCreateInput | RoutineUpdateInput): void {
        // // Check that node columnIndexes and rowIndexes are valid TODO query existing data to do this
        // let combinedNodes = [];
        // if (input.nodesCreate) combinedNodes.push(...input.nodesCreate);
        // if ((input as RoutineUpdateInput).nodesUpdate) combinedNodes.push(...(input as any).nodesUpdate);
        // if ((input as RoutineUpdateInput).nodesDelete) combinedNodes = combinedNodes.filter(node => !(input as any).nodesDelete.includes(node.id));
        // // Remove nodes that have duplicate rowIndexes and columnIndexes
        // console.log('unique nodes check', JSON.stringify(combinedNodes));
        // const uniqueNodes = uniqBy(combinedNodes, (n) => `${n.rowIndex}-${n.columnIndex}`);
        // if (uniqueNodes.length < combinedNodes.length) throw new CustomError(CODE.NodeDuplicatePosition);
        return;
    },
    async shapeBase(userId: string, data: RoutineCreateInput | RoutineUpdateInput, isAdd: boolean) {
        return {
            root: {
                isPrivate: data.isPrivate ?? undefined,
                hasCompleteVersion: (data.isComplete === true) ? true : (data.isComplete === false) ? false : undefined,
                completedAt: (data.isComplete === true) ? new Date().toISOString() : (data.isComplete === false) ? null : undefined,
                project: data.projectId ? { connect: { id: data.projectId } } : undefined,
                tags: await TagModel.mutate(prisma).relationshipBuilder(userId, data, 'Routine'),
                permissions: JSON.stringify({}),
            },
            isAutomatable: data.isAutomatable ?? undefined,
            isComplete: data.isComplete ?? undefined,
            isInternal: data.isInternal ?? undefined,
            versionLabel: data.versionLabel ?? undefined,
            resourceList: await ResourceListModel.mutate(prisma).relationshipBuilder(userId, data, isAdd),
            inputs: await this.relationshipBuilderInput(userId, data, isAdd),
            outputs: await this.relationshipBuilderOutput(userId, data, isAdd),
            nodes: await NodeModel.mutate(prisma).relationshipBuilder(userId, (data as RoutineUpdateInput)?.id ?? null, data, isAdd),
            nodeLinks: NodeModel.mutate(prisma).relationshipBuilderNodeLink(userId, data, isAdd),
            translations: TranslationModel.relationshipBuilder(userId, data, { create: routineTranslationCreate, update: routineTranslationUpdate }, isAdd),
        }
    },
    async shapeCreate(userId: string, data: RoutineCreateInput): Promise<Prisma.routine_versionUpsertArgs['create']> {
        const [simplicity, complexity] = await this.calculateComplexity(data);
        const base = await this.shapeBase(userId, data, true);
        return {
            ...base,
            root: {
                create: {
                    ...base.root,
                    parent: data.parentId ? { connect: { id: data.parentId } } : undefined,
                    createdByOrganization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
                    organization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
                    createdByUser: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
                    user: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
                }
            },
            id: data.id,
            complexity,
            simplicity,
        }
    },
    async shapeUpdate(userId: string, data: RoutineUpdateInput): Promise<Prisma.routine_versionUpsertArgs['update']> {
        const [simplicity, complexity] = await this.calculateComplexity(data, data.versionId);
        const base = await this.shapeBase(userId, data, false);
        return {
            ...base,
            root: {
                update: {
                    ...base.root,
                    // parent: data.parentId ? { connect: { id: data.parentId } } : undefined,
                    organization: data.organizationId ? { connect: { id: data.organizationId } } : data.userId ? { disconnect: true } : undefined,
                    user: data.userId ? { connect: { id: data.userId } } : data.organizationId ? { disconnect: true } : undefined,
                }
            },
            complexity: complexity,
            simplicity: simplicity,
        }
    },
    async relationshipBuilderInput(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        // If nodes relationship provided, calculate inputs from nodes. Otherwise, use given inputs
        //TODO
        return relationshipBuilderHelper({
            data,
            relationshipName: 'inputs',
            isAdd,
            isTransferable: false,
            shape: {
                shapeCreate: async (userId, cuData) => ({
                    id: cuData.id,
                    name: cuData.name,
                    standard: await StandardModel.mutate(prisma).relationshipBuilder(userId, {
                        ...cuData,
                        // If standard was not internal, then it would have been created 
                        // in its own mutation
                        isInternal: true,
                    }, true),
                    translations: TranslationModel.relationshipBuilder(userId, cuData, { create: inputTranslationCreate, update: inputTranslationUpdate }, true),
                }),
                shapeUpdate: async (userId, cuData) => ({
                    name: cuData.name,
                    standardId: await StandardModel.mutate(prisma).relationshipBuilder(userId, cuData, false),
                    translations: TranslationModel.relationshipBuilder(userId, cuData, { create: inputTranslationCreate, update: inputTranslationUpdate }, false),
                })
            },
            userId,
        });
    },
    async relationshipBuilderOutput(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        // If nodes relationship provided, calculate outputs from nodes. Otherwise, use given outputs
        //TODO
        return relationshipBuilderHelper({
            data,
            relationshipName: 'outputs',
            isAdd,
            isTransferable: false,
            shape: {
                shapeCreate: async (userId, cuData) => ({
                    id: cuData.id,
                    name: cuData.name,
                    standard: await StandardModel.mutate(prisma).relationshipBuilder(userId, {
                        ...cuData,
                        // If standard was not internal, then it would have been created 
                        // in its own mutation
                        isInternal: true,
                    }, true),
                    translations: TranslationModel.relationshipBuilder(userId, cuData, { create: outputTranslationCreate, update: outputTranslationUpdate }, true),
                }),
                shapeUpdate: async (userId, cuData) => ({
                    name: cuData.name,
                    standard: await StandardModel.mutate(prisma).relationshipBuilder(userId, cuData, false),
                    translations: TranslationModel.relationshipBuilder(userId, cuData, { create: outputTranslationCreate, update: outputTranslationUpdate }, false),
                })
            },
            userId,
        });
    },
    /**
     * Add, update, or remove a one-to-one routine relationship. 
     * Due to some unknown Prisma bug, it won't let us create/update a routine directly
     * in the main mutation query like most other relationship builders. Instead, we 
     * must do this separately, and return the routine's ID.
     */
    async relationshipBuilder(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        return relationshipBuilderHelper({
            data,
            relationshipName: 'routine',
            fieldExcludes: ['user', 'userId', 'organization', 'organizationId', 'createdByUser', 'createdByUserId', 'createdByOrganization', 'createdByOrganizationId'],
            isAdd,
            isOneToOne: true,
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
            userId,
        });
    },
    async cud(params: CUDInput<RoutineCreateInput, RoutineUpdateInput>): Promise<CUDResult<Routine>> {
        return cudHelper({
            ...params,
            objectType: 'Routine',
            prisma,
            yup: { yupCreate: routinesCreate, yupUpdate: routinesUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
            onCreated: (created) => {
                for (const c of created) {
                    Trigger(prisma).objectCreate('Routine', c.id as string, params.userId);
                }
            },
        })
    },
    /**
     * Duplicates a routine, along with its nodes and edges. 
     * If a fork, then the parent is set as the original routine. 
     * If a copy, there is no parent.
     */
    async duplicate({ userId, objectId, isFork, createCount = 0 }: DuplicateInput): Promise<DuplicateResult<Routine>> {
        throw new CustomError(CODE.NotImplemented);
        // let newCreateCount = createCount;
        // // Find routine, with fields we want to copy.
        // // I hope I discover a better way to do this.
        // // Notable fields not being copied are completedAt and isComplete (since the copy will default to not complete),
        // // score and views and other stats (since the copy will default to 0),
        // // createdByUserId and createdByOrganizationId and projectId (since the copy will default to your own),
        // // parentId (since the original will be the parent of the copy),
        // // version (since the copy will default to 1.0.0),
        // const routine = await prisma.routine.findFirst({
        //     where: { id: objectId },
        //     select: {
        //         id: true,
        //         complexity: true,
        //         isAutomatable: true,
        //         isInternal: true,
        //         simplicity: true,
        //         userId: true,
        //         organizationId: true,
        //         nodes: {
        //             select: {
        //                 id: true,
        //                 columnIndex: true,
        //                 rowIndex: true,
        //                 type: true,
        //                 nodeEnd: {
        //                     select: {
        //                         wasSuccessful: true
        //                     }
        //                 },
        //                 loop: {
        //                     select: {
        //                         loops: true,
        //                         maxLoops: true,
        //                         operation: true,
        //                         whiles: {
        //                             select: {
        //                                 condition: true,
        //                                 translations: {
        //                                     select: {
        //                                         description: true,
        //                                         title: true,
        //                                         language: true,
        //                                     }
        //                                 }
        //                             }
        //                         }
        //                     }
        //                 },
        //                 nodeRoutineList: {
        //                     select: {
        //                         isOrdered: true,
        //                         isOptional: true,
        //                         routines: {
        //                             select: {
        //                                 id: true,
        //                                 index: true,
        //                                 isOptional: true,
        //                                 routine: {
        //                                     select: {
        //                                         id: true,
        //                                         isInternal: true,
        //                                     }
        //                                 },
        //                                 translations: {
        //                                     select: {
        //                                         description: true,
        //                                         title: true,
        //                                         language: true,
        //                                     }
        //                                 }
        //                             }
        //                         }
        //                     }
        //                 },
        //                 translations: {
        //                     select: {
        //                         description: true,
        //                         title: true,
        //                         language: true,
        //                     }
        //                 }
        //             }
        //         },
        //         nodeLinks: {
        //             select: {
        //                 fromId: true,
        //                 toId: true,
        //                 operation: true,
        //                 whens: {
        //                     select: {
        //                         condition: true,
        //                         translations: {
        //                             select: {
        //                                 description: true,
        //                                 title: true,
        //                                 language: true,
        //                             }
        //                         }
        //                     }
        //                 }
        //             }
        //         },
        //         resourceLists: {
        //             select: {
        //                 index: true,
        //                 usedFor: true,
        //                 resources: {
        //                     select: {
        //                         index: true,
        //                         link: true,
        //                         usedFor: true,
        //                         translations: {
        //                             select: {
        //                                 description: true,
        //                                 title: true,
        //                                 language: true,
        //                             }
        //                         }
        //                     }
        //                 },
        //                 translations: {
        //                     select: {
        //                         description: true,
        //                         title: true,
        //                         language: true,
        //                     }
        //                 }
        //             }
        //         },
        //         inputs: {
        //             select: {
        //                 isRequired: true,
        //                 name: true,
        //                 standardId: true,
        //                 translations: {
        //                     select: {
        //                         description: true,
        //                         language: true,
        //                     }
        //                 }
        //             }
        //         },
        //         outputs: {
        //             select: {
        //                 name: true,
        //                 standardId: true,
        //                 translations: {
        //                     select: {
        //                         description: true,
        //                         language: true,
        //                     }
        //                 }
        //             }
        //         },
        //         tags: {
        //             select: {
        //                 id: true,
        //             }
        //         },
        //         translations: {
        //             select: {
        //                 description: true,
        //                 instructions: true,
        //                 title: true,
        //                 language: true,
        //             }
        //         }
        //     }
        // });
        // // If routine didn't exist
        // if (!routine) {
        //     throw new CustomError(CODE.NotFound, 'Routine not found', { code: genErrorCode('0225') });
        // }
        // // If routine is marked as internal and it doesn't belong to you
        // else if (routine.isInternal && routine.userId !== userId) {
        //     const roles = await OrganizationModel.query().hasRole(prisma, userId, [routine.organizationId]);
        //     if (roles.some(role => !role))
        //         throw new CustomError(CODE.Unauthorized, 'Not authorized to copy', { code: genErrorCode('0226') })
        // }
        // // Initialize new routine object
        // const newRoutine: any = routine;
        // // For every node in the routine (and every edge which references it), change the ID 
        // // to a new random ID.
        // for (const node of newRoutine.nodes) {
        //     const oldId = node.id;
        //     // Update ID
        //     node.id = uuid();
        //     // Update reference to node in nodeLinks
        //     for (const nodeLink of newRoutine.nodeLinks) {
        //         if (nodeLink.fromId === oldId) {
        //             nodeLink.fromId = node.id;
        //         }
        //         if (nodeLink.toId === oldId) {
        //             nodeLink.toId = node.id;
        //         }
        //     }
        // }
        // // If copying subroutines, call this function for each subroutine
        // if (createCount < 100 && routine.nodes && routine.nodes.length > 0) {
        //     const newSubroutineIds: string[] = [];
        //     // Find the IDs of all isInternal subroutines
        //     const oldSubroutineIds: string[] = [];
        //     for (const node of routine.nodes) {
        //         if (node.nodeRoutineList?.routines && node.nodeRoutineList?.routines.length > 0) {
        //             for (const subroutine of node.nodeRoutineList.routines) {
        //                 if (subroutine.routine?.id && subroutine.routine?.isInternal) {
        //                     oldSubroutineIds.push(subroutine.routine.id);
        //                 }
        //             }
        //         }
        //     }
        //     // Copy each subroutine
        //     for (const subroutineId of oldSubroutineIds) {
        //         const { object: copiedSubroutine, numCreated } = await this.duplicate({ userId, objectId: subroutineId, isFork: false, createCount: newCreateCount });
        //         newSubroutineIds.push(copiedSubroutine.id ?? '');
        //         newCreateCount += numCreated;
        //         if (newCreateCount >= 100) {
        //             break;
        //         }
        //     }
        //     // Change the IDs of all subroutines to the new IDs
        //     for (const node of newRoutine.nodes) {
        //         if (node.nodeRoutineList?.routines && node.nodeRoutineList?.routines.length > 0) {
        //             for (const subroutine of node.nodeRoutineList.routines) {
        //                 if (subroutine.routine?.id && newSubroutineIds.includes(subroutine.routine.id)) {
        //                     subroutine.routine.id = newSubroutineIds.find(id => id === subroutine.routine.id);
        //                 }
        //             }
        //         }
        //     }
        // }
        // // Create the new routine
        // const createdRoutine = await prisma.routine.create({
        //     data: {
        //         // Give ownership to user copying routine
        //         createdByUser: { connect: { id: userId } },
        //         user: { connect: { id: userId } },
        //         // Set parentId to original routine's id
        //         parent: isFork ? { connect: { id: routine.id } } : undefined,
        //         // Copy the rest of the fields
        //         complexity: newRoutine.complexity,
        //         isAutomatable: newRoutine.isAutomatable,
        //         isInternal: newRoutine.isInternal,
        //         simplicity: newRoutine.simplicity,
        //         versionGroupId: uuid(),
        //         nodes: newRoutine.nodes ? {
        //             create: newRoutine.nodes.map((node: any) => ({
        //                 id: node.id,
        //                 columnIndex: node.columnIndex,
        //                 rowIndex: node.rowIndex,
        //                 type: node.type,
        //                 nodeEnd: node.nodeEnd ? {
        //                     create: {
        //                         wasSuccessful: node.nodeEnd.wasSuccessful
        //                     }
        //                 } : undefined,
        //                 loop: node.loop ? {
        //                     create: {
        //                         loops: node.loop.loops,
        //                         maxLoops: node.loop.maxLoops,
        //                         operation: node.loop.operation,
        //                         whiles: node.loop.whiles ? {
        //                             create: node.loop.whiles.map((whileNode: any) => ({
        //                                 condition: whileNode.condition,
        //                                 translations: whileNode.translations ? {
        //                                     create: whileNode.translations.map((translation: any) => ({
        //                                         description: translation.description,
        //                                         title: translation.title,
        //                                         language: translation.language
        //                                     }))
        //                                 } : undefined
        //                             }))
        //                         } : undefined
        //                     }
        //                 } : undefined,
        //                 nodeRoutineList: node.nodeRoutineList ? {
        //                     create: {
        //                         isOrdered: node.nodeRoutineList.isOrdered,
        //                         isOptional: node.nodeRoutineList.isOptional,
        //                         routines: node.nodeRoutineList.routines ? {
        //                             create: [...node.nodeRoutineList.routines.map((routine: any) => {
        //                                 return ({
        //                                     index: routine.index,
        //                                     routine: { connect: { id: routine.routine.id } },
        //                                     translations: routine.translations ? {
        //                                         create: routine.translations.map((translation: any) => ({
        //                                             description: translation.description,
        //                                             title: translation.title,
        //                                             language: translation.language
        //                                         }))
        //                                     } : undefined
        //                                 })
        //                             })]
        //                         } : undefined
        //                     }
        //                 } : undefined,
        //                 translations: node.translations ? {
        //                     create: node.translations.map((translation: any) => ({
        //                         description: translation.description,
        //                         title: translation.title,
        //                         language: translation.language
        //                     }))
        //                 } : undefined,
        //             }))
        //         } : undefined,
        //         nodeLinks: newRoutine.nodeLinks ? {
        //             create: newRoutine.nodeLinks.map((nodeLink: any) => ({
        //                 from: { connect: { id: nodeLink.fromId } },
        //                 to: { connect: { id: nodeLink.toId } },
        //                 operation: nodeLink.operation,
        //                 whens: nodeLink.whens ? {
        //                     create: nodeLink.whens.map((when: any) => ({
        //                         condition: when.condition,
        //                         translations: when.translations ? {
        //                             create: when.translations.map((translation: any) => ({
        //                                 description: translation.description,
        //                                 title: translation.title,
        //                                 language: translation.language
        //                             }))
        //                         } : undefined
        //                     }))
        //                 } : undefined,
        //                 translations: nodeLink.translations ? {
        //                     create: nodeLink.translations.map((translation: any) => ({
        //                         description: translation.description,
        //                         title: translation.title,
        //                         language: translation.language
        //                     }))
        //                 } : undefined
        //             }))
        //         } : undefined,
        //         resourceLists: newRoutine.resourceLists ? {
        //             create: newRoutine.resourceLists.map((resourceList: any) => ({
        //                 index: resourceList.index,
        //                 usedFor: resourceList.usedFor,
        //                 resources: resourceList.resources ? {
        //                     create: resourceList.resources.map((resource: any) => ({
        //                         index: resource.index,
        //                         link: resource.link,
        //                         usedFor: resource.usedFor,
        //                         translations: resource.translations ? {
        //                             create: resource.translations.map((translation: any) => ({
        //                                 description: translation.description,
        //                                 title: translation.title,
        //                                 language: translation.language
        //                             }))
        //                         } : null
        //                     }))
        //                 } : null
        //             }))
        //         } : undefined,
        //         inputs: newRoutine.inputs ? {
        //             create: newRoutine.inputs.map((input: any) => ({
        //                 isRequired: input.isRequired,
        //                 name: input.name,
        //                 standard: { connect: { id: input.standardId } },
        //                 translations: input.translations ? {
        //                     create: input.translations.map((translation: any) => ({
        //                         description: translation.description,
        //                         title: translation.title,
        //                         language: translation.language
        //                     }))
        //                 } : undefined
        //             }))
        //         } : undefined,
        //         outputs: newRoutine.outputs ? {
        //             create: newRoutine.outputs.map((output: any) => ({
        //                 name: output.name,
        //                 standardId: output.standardId,
        //                 translations: output.translations ? {
        //                     create: output.translations.map((translation: any) => ({
        //                         description: translation.description,
        //                         title: translation.title,
        //                         language: translation.language
        //                     }))
        //                 } : null
        //             }))
        //         } : undefined,
        //         tags: newRoutine.tags ? {
        //             connect: newRoutine.tags.map((tag: any) => ({
        //                 id: tag.id
        //             }))
        //         } : undefined,
        //         translations: newRoutine.translations ? {
        //             create: newRoutine.translations.map((translation: any) => ({
        //                 description: translation.description,
        //                 instructions: translation.instructions,
        //                 // Add "Copy" to title
        //                 title: `${translation.title} (Copy)`,
        //                 language: translation.language
        //             }))
        //         } : undefined,
        //     }
        // });
        // return {
        //     object: createdRoutine as any,
        //     numCreated: newCreateCount + 1
        // }
    }
})

export const RoutineModel = ({
    prismaObject: (prisma: PrismaType) => prisma.routine_version,
    format: routineFormatter(),
    mutate: routineMutater,
    search: routineSearcher(),
    type: 'Routine' as GraphQLModelType,
    validate: routineValidator()
})