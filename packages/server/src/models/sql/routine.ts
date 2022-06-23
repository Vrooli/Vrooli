import { Routine, RoutineCreateInput, RoutineUpdateInput, RoutineSearchInput, RoutineSortBy, Count, ResourceListUsedFor, NodeRoutineListItem, NodeCreateInput, NodeUpdateInput, NodeRoutineListCreateInput, NodeRoutineListUpdateInput, NodeRoutineListItemCreateInput } from "../../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { addCountFieldsHelper, addCreatorField, addJoinTablesHelper, addOwnerField, addSupplementalFields, CUDInput, CUDResult, deleteOneHelper, DuplicateInput, DuplicateResult, FormatConverter, GraphQLInfo, GraphQLModelType, modelToGraphQL, PartialGraphQLInfo, relationshipToPrisma, RelationshipTypes, removeCountFieldsHelper, removeCreatorField, removeJoinTablesHelper, removeOwnerField, Searcher, selectHelper, toPartialGraphQLInfo, ValidateMutationsInput } from "./base";
import { CustomError } from "../../error";
import { CODE, DeleteOneType, inputsCreate, inputsUpdate, inputTranslationCreate, inputTranslationUpdate, MemberRole, outputsCreate, outputsUpdate, outputTranslationCreate, outputTranslationUpdate, routinesCreate, routinesUpdate, routineTranslationCreate, routineTranslationUpdate } from "@local/shared";
import { hasProfanity } from "../../utils/censor";
import { OrganizationModel } from "./organization";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { NodeModel } from "./node";
import { StandardModel } from "./standard";
import _ from "lodash";
import { TranslationModel } from "./translation";
import { ResourceListModel } from "./resourceList";
import { genErrorCode } from "../../logger";
import { ViewModel } from "./view";
import { runFormatter } from "./run";
import { v4 as uuidv4 } from 'uuid';

type NodeWeightData = {
    simplicity: number,
    complexity: number,
    optionalInputs: number,
    allInputs: number,
}

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { tags: 'tag', starredBy: 'user' };
const countMapper = { commentsCount: 'comments', nodesCount: 'nodes', reportsCount: 'reports' };
const calculatedFields = ['inProgressCompletedSteps', 'inProgressCompletedComplexity', 'inProgressVersion', 'isUpvoted', 'isStarred', 'role', 'runs'];
export const routineFormatter = (): FormatConverter<Routine> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.Routine,
        'comments': GraphQLModelType.Comment,
        'creator': {
            'User': GraphQLModelType.User,
            'Organization': GraphQLModelType.Organization,
        },
        'forks': GraphQLModelType.Routine,
        'inputs': GraphQLModelType.InputItem,
        'nodes': GraphQLModelType.Node,
        'outputs': GraphQLModelType.OutputItem,
        'owner': {
            'User': GraphQLModelType.User,
            'Organization': GraphQLModelType.Organization,
        },
        'parent': GraphQLModelType.Routine,
        'project': GraphQLModelType.Project,
        'reports': GraphQLModelType.Report,
        'resourceLists': GraphQLModelType.ResourceList,
        'starredBy': GraphQLModelType.User,
        'tags': GraphQLModelType.Tag,
    },
    removeCalculatedFields: (partial) => {
        return _.omit(partial, calculatedFields);
    },
    constructUnions: (data) => {
        let modified = addCreatorField(data);
        modified = addOwnerField(modified);
        return modified;
    },
    deconstructUnions: (partial) => {
        let modified = removeCreatorField(partial);
        modified = removeOwnerField(modified);
        return modified;
    },
    addJoinTables: (partial) => {
        return addJoinTablesHelper(partial, joinMapper);
    },
    removeJoinTables: (data) => {
        return removeJoinTablesHelper(data, joinMapper);
    },
    addCountFields: (partial) => {
        return addCountFieldsHelper(partial, countMapper);
    },
    removeCountFields: (data) => {
        return removeCountFieldsHelper(data, countMapper);
    },
    async addSupplementalFields(
        prisma: PrismaType,
        userId: string | null, // Of the user making the request
        objects: RecursivePartial<any>[],
        partial: PartialGraphQLInfo,
    ): Promise<RecursivePartial<Routine>[]> {
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
        // Query for isStarred
        if (partial.isStarred) {
            const isStarredArray = userId
                ? await StarModel(prisma).getIsStarreds(userId, ids, GraphQLModelType.Routine)
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
        }
        // Query for isUpvoted
        if (partial.isUpvoted) {
            const isUpvotedArray = userId
                ? await VoteModel(prisma).getIsUpvoteds(userId, ids, GraphQLModelType.Routine)
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isUpvoted: isUpvotedArray[i] }));
        }
        // Query for isViewed
        if (partial.isViewed) {
            const isViewedArray = userId
                ? await ViewModel(prisma).getIsVieweds(userId, ids, GraphQLModelType.Routine)
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isViewed: isViewedArray[i] }));
        }
        // Query for role
        if (partial.role) {
            let organizationIds: string[] = [];
            // Collect owner data
            let ownerData: any = objects.map(x => x.owner).filter(x => x);
            // If no owner data was found, then owner data was not queried. In this case, query for owner data.
            if (ownerData.length === 0) {
                const ownerDataUnformatted = await prisma.routine.findMany({
                    where: { id: { in: ids } },
                    select: {
                        id: true,
                        user: { select: { id: true } },
                        organization: { select: { id: true } },
                    },
                });
                organizationIds = ownerDataUnformatted.map(x => x.organization?.id).filter(x => Boolean(x)) as string[];
                // Inject owner data into "objects"
                objects = objects.map((x, i) => { 
                    const unformatted = ownerDataUnformatted.find(y => y.id === x.id);
                    return ({ ...x, owner: unformatted?.user || unformatted?.organization })
                });            } else {
                organizationIds = objects
                    .filter(x => Array.isArray(x.owner?.translations) && x.owner.translations.length > 0 && x.owner.translations[0].name)
                    .map(x => x.owner.id)
                    .filter(x => Boolean(x)) as string[];
            }
            // If owned by user, set role to owner if userId matches
            // If owned by organization, set role user's role in organization
            const roles = userId
                ? await OrganizationModel(prisma).getRoles(userId, organizationIds)
                : [];
            objects = objects.map((x) => {
                const orgRoleIndex = organizationIds.findIndex(id => id === x.owner?.id);
                if (orgRoleIndex >= 0) {
                    return { ...x, role: roles[orgRoleIndex] };
                }
                return { ...x, role: (Boolean(x.owner?.id) && x.owner?.id === userId) ? MemberRole.Owner : undefined };
            }) as any;
        }
        // Query for run data. This must be done here because we are not querying all runs - just ones made by the user
        if (_.isObject(partial.runs)) {
            if (userId) {
                // Find requested fields of runs. Also add routineId, so we 
                // can associate runs with their routine
                const runPartial = {
                    ...toPartialGraphQLInfo(partial.runs, runFormatter().relationshipMap),
                    routineId: true
                }
                if (runPartial === undefined) {
                    throw new CustomError(CODE.InternalError, 'Error converting query', { code: genErrorCode('0178') });
                }
                // Query runs made by user
                let runs: any[] = await prisma.run.findMany({
                    where: {
                        AND: [
                            { routineId: { in: ids } },
                            { userId },
                        ]
                    },
                    ...selectHelper(runPartial)
                });
                // Format runs to GraphQL
                runs = runs.map(r => modelToGraphQL(r, runPartial));
                // Add supplemental fields
                runs = await addSupplementalFields(prisma, userId, runs, runPartial);
                // Apply data fields to objects
                objects = objects.map((x) => ({ ...x, runs: runs.filter(r => r.routineId === x.id) }));
            } else {
                // Set all runs to empty array
                objects = objects.map(x => ({ ...x, runs: [] }));
            }
        }
        // Convert Prisma objects to GraphQL objects
        return objects as RecursivePartial<Routine>[];
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
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
                { translations: { some: { language: languages ? { in: languages } : undefined, title: { ...insensitive } } } },
                { tags: { some: { tag: { tag: { ...insensitive } } } } },
            ]
        })
    },
    customQueries(input: RoutineSearchInput): { [x: string]: any } {
        // isComplete routines may be set to true or false generally, and also set exceptions
        let isComplete: any;
        if (!!input.isCompleteExceptions) {
            isComplete = { OR: [{ isComplete: input.isComplete }] };
            for (const exception of input.isCompleteExceptions) {
                if (['createdByOrganization', 'createdByUser', 'organization', 'parent', 'project', 'user'].includes(exception.relation)) {
                    isComplete.OR.push({ [exception.relation]: { id: exception.id } });
                }
            }
        } else {
            isComplete = { isComplete: input.isComplete };
        }
        // isInternal routines may be set to true or false generally, and also set exceptions
        let isInternal: any;
        if (!!input.isInternalExceptions) {
            isInternal = { OR: [{ isInternal: input.isInternal }] };
            for (const exception of input.isInternalExceptions) {
                if (['createdByOrganization', 'createdByUser', 'organization', 'parent', 'project', 'user'].includes(exception.relation)) {
                    isInternal.OR.push({ [exception.relation]: { id: exception.id } });
                }
            }
        } else {
            isInternal = { isInternal: input.isInternal };
        }
        return {
            ...(input.excludeIds !== undefined ? { NOT: { id: { in: input.excludeIds } } } : {}),
            ...(input.excludeIds !== undefined ? { NOT: { id: { in: input.excludeIds } } } : {}),
            ...isComplete,
            ...isInternal,
            ...(input.isInternal !== undefined ? { isInternal: input.isInternal } : {}),
            ...(input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            ...(input.minComplexity !== undefined ? { complexity: { gte: input.minComplexity } } : {}),
            ...(input.maxComplexity !== undefined ? { complexity: { lte: input.maxComplexity } } : {}),
            ...(input.minSimplicity !== undefined ? { simplicity: { gte: input.minSimplicity } } : {}),
            ...(input.maxSimplicity !== undefined ? { simplicity: { lte: input.maxSimplicity } } : {}),
            ...(input.maxTimesCompleted !== undefined ? { timesCompleted: { lte: input.maxTimesCompleted } } : {}),
            ...(input.minScore !== undefined ? { score: { gte: input.minScore } } : {}),
            ...(input.minStars !== undefined ? { stars: { gte: input.minStars } } : {}),
            ...(input.minTimesCompleted !== undefined ? { timesCompleted: { gte: input.minTimesCompleted } } : {}),
            ...(input.minViews !== undefined ? { views: { gte: input.minViews } } : {}),
            ...(input.resourceLists !== undefined ? { resourceLists: { some: { translations: { some: { title: { in: input.resourceLists } } } } } } : {}),
            ...(input.resourceTypes !== undefined ? { resourceLists: { some: { usedFor: ResourceListUsedFor.Display as any, resources: { some: { usedFor: { in: input.resourceTypes } } } } } } : {}),
            ...(input.userId !== undefined ? { userId: input.userId } : {}),
            ...(input.organizationId !== undefined ? { organizationId: input.organizationId } : {}),
            ...(input.projectId !== undefined ? { projectId: input.projectId } : {}),
            ...(input.parentId !== undefined ? { parentId: input.parentId } : {}),
            ...(input.reportId !== undefined ? { reports: { some: { id: input.reportId } } } : {}),
            ...(input.tags !== undefined ? { tags: { some: { tag: { tag: { in: input.tags } } } } } : {}),
        }
    },
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
export const routineMutater = (prisma: PrismaType) => ({
    /**
     * Calculates the maximum and minimum complexity of a routine based on the number of steps. 
     * Simplicity is a the minimum number of inputs and decisions required to complete the routine, while 
     * complexity is the maximum.
     * @param data The routine data, either a create or update
     * @returns [simplicity, complexity] Numbers representing the shorted and longest weighted paths
     */
    async calculateComplexity(data: RoutineCreateInput | RoutineUpdateInput): Promise<[number, number]> {
        // If the routine is being updated, Find the complexity of existing subroutines
        let existingRoutine;
        if ((data as RoutineUpdateInput).id) {
            existingRoutine = await prisma.routine.findUnique({
                where: { id: (data as RoutineUpdateInput).id },
                select: {
                    nodeLinks: { select: { fromId: true, toId: true } },
                    nodes: {
                        select: {
                            id: true, // Needed to associate with links
                            nodeRoutineList: {
                                select: {
                                    routines: {
                                        select: {
                                            routine: { select: { id: true, complexity: true, simplicity: true } }
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
            let ids: string[] = node.nodeRoutineList?.routines?.map((item: NodeRoutineListItem) => item.routine.id) ?? [];
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
        const subroutineWeightData = await prisma.routine.findMany({
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
        // const uniqueNodes = _.uniqBy(combinedNodes, (n) => `${n.rowIndex}-${n.columnIndex}`);
        // if (uniqueNodes.length < combinedNodes.length) throw new CustomError(CODE.NodeDuplicatePosition);
        return;
    },
    async toDBShape(userId: string | null, data: RoutineCreateInput | RoutineUpdateInput, isAdd: boolean): Promise<any> {
        const [simplicity, complexity] = await this.calculateComplexity(data);
        return {
            id: data.id ?? undefined,
            isAutomatable: data.isAutomatable,
            isComplete: data.isComplete,
            completedAt: data.isComplete ? new Date().toISOString() : null,
            complexity: complexity,
            simplicity: simplicity,
            isInternal: data.isInternal,
            parentId: data.parentId,
            version: data.version,
            resourceLists: await ResourceListModel(prisma).relationshipBuilder(userId, data, isAdd),
            tags: await TagModel(prisma).relationshipBuilder(userId, data, GraphQLModelType.Routine),
            inputs: await this.relationshipBuilderInput(userId, data, isAdd),
            outputs: await this.relationshipBuilderOutput(userId, data, isAdd),
            nodes: await NodeModel(prisma).relationshipBuilder(userId, (data as RoutineUpdateInput)?.id ?? null, data, isAdd),
            nodeLinks: NodeModel(prisma).relationshipBuilderNodeLink(userId, data, isAdd),
            translations: TranslationModel().relationshipBuilder(userId, data, { create: routineTranslationCreate, update: routineTranslationUpdate }, isAdd),
        }
    },
    /**
    * Add, update, or remove routine inputs from a routine.
    * NOTE: Input is whole routine data, not just the inputs. 
    * This is because we may need the node data to calculate inputs
    */
    async relationshipBuilderInput(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported in this case (since they can only be applied to one routine)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'inputs', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        let { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        const standardModel = StandardModel(prisma);
        // If nodes relationship provided, calculate inputs and outputs from nodes. Otherwise, use given inputs 
        //TODO
        // Validate create
        if (Array.isArray(createMany)) {
            inputsCreate.validateSync(createMany, { abortEarly: false });
            let result = [];
            for (let data of createMany) {
                // Check for censored words
                if (hasProfanity(data.name, data.description))
                    throw new CustomError(CODE.BannedWord, 'Name or description includes bad word', { code: genErrorCode('0091') });
                // Convert nested relationships
                result.push({
                    id: data.id ?? undefined,
                    name: data.name,
                    standardId: await standardModel.relationshipBuilder(userId, data, isAdd),
                    translations: TranslationModel().relationshipBuilder(userId, data, { create: inputTranslationCreate, update: inputTranslationUpdate }, false),
                })
            }
            createMany = result;
        }
        // Validate update
        if (Array.isArray(updateMany)) {
            inputsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            let result = [];
            for (let update of updateMany) {
                // Check for censored words
                if (hasProfanity(update.data.name, update.data.description))
                    throw new CustomError(CODE.BannedWord, 'Name or description contains banned word', { code: genErrorCode('0092') });
                // Convert nested relationships
                result.push({
                    where: update.where,
                    data: {
                        name: update.data.name,
                        standardId: await standardModel.relationshipBuilder(userId, update.data, isAdd),
                        translations: TranslationModel().relationshipBuilder(userId, update.data, { create: inputTranslationCreate, update: inputTranslationUpdate }, false),
                    }
                })
            }
            updateMany = result;
        }
        return Object.keys(formattedInput).length > 0 ? {
            create: createMany,
            update: updateMany,
            delete: deleteMany
        } : undefined;
    },
    /**
     * Add, update, or remove routine outputs from a routine
     * NOTE: Input is whole routine data, not just the outputs. 
     * This is because we may need the node data to calculate outputs
     */
    async relationshipBuilderOutput(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported in this case (since they can only be applied to one routine)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'outputs', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        let { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        const standardModel = StandardModel(prisma);
        // If nodes relationship provided, calculate inputs and outputs from nodes. Otherwise, use given inputs
        //TODO
        // Validate create
        if (Array.isArray(createMany)) {
            outputsCreate.validateSync(createMany, { abortEarly: false });
            TranslationModel().profanityCheck(createMany);
            let result = [];
            for (let data of createMany) {
                // Convert nested relationships
                result.push({
                    id: data.id ?? undefined,
                    name: data.name,
                    standard: await standardModel.relationshipBuilder(userId, data, isAdd),
                    translations: TranslationModel().relationshipBuilder(userId, data, { create: outputTranslationCreate, update: outputTranslationUpdate }, false),
                })
            }
            createMany = result;
        }
        // Validate update
        if (Array.isArray(updateMany)) {
            outputsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            TranslationModel().profanityCheck(updateMany.map(u => u.data));
            let result = [];
            for (let update of updateMany) {
                // Convert nested relationships
                result.push({
                    where: update.where,
                    data: {
                        name: update.data.name,
                        standard: await standardModel.relationshipBuilder(userId, update.data, isAdd),
                        translations: TranslationModel().relationshipBuilder(userId, update.data, { create: outputTranslationCreate, update: outputTranslationUpdate }, false),
                    }
                })
            }
            updateMany = result;
        }
        return Object.keys(formattedInput).length > 0 ? {
            create: createMany,
            update: updateMany,
            delete: deleteMany
        } : undefined;
    },
    /**
     * Add, update, or remove a one-to-one routine relationship. 
     * Due to some unknown Prisma bug, it won't let us create/update a routine directly
     * in the main mutation query like most other relationship builders. Instead, we 
     * must do this separately, and return the routine's ID.
     */
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<string | null> {
        // Convert input to Prisma shape
        const fieldExcludes = ['node', 'user', 'userId', 'organization', 'organizationId', 'createdByUser', 'createdByUserId', 'createdByOrganization', 'createdByOrganizationId'];
        let formattedInput: any = relationshipToPrisma({ data: input, relationshipName: 'routine', isAdd, fieldExcludes })
        // Validate
        const { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        await this.validateMutations({
            userId,
            createMany: createMany as RoutineCreateInput[],
            updateMany: updateMany as { where: { id: string }, data: RoutineUpdateInput }[],
            deleteMany: deleteMany?.map((d: any) => d.id)
        });
        // Shape
        if (Array.isArray(formattedInput.create) && formattedInput.create.length > 0) {
            const create = await this.toDBShape(userId, formattedInput.create[0], true);
            // Create routine
            const routine = await prisma.routine.create({
                data: create,
            })
            return routine?.id ?? null;
        }
        if (Array.isArray(formattedInput.connect) && formattedInput.connect.length > 0) {
            return formattedInput.connect[0].id;
        }
        if (Array.isArray(formattedInput.disconnect) && formattedInput.disconnect.length > 0) {
            return null;
        }
        if (Array.isArray(formattedInput.update) && formattedInput.update.length > 0) {
            const update = await this.toDBShape(userId, formattedInput.update[0].data, false);
            // Update routine
            const routine = await prisma.routine.update({
                where: { id: update.id },
                data: update
            })
            return routine?.id ?? null;
        }
        if (Array.isArray(formattedInput.delete) && formattedInput.delete.length > 0) {
            const deleteId = formattedInput.delete[0].id;
            // Delete routine
            await deleteOneHelper(userId, { id: deleteId, objectType: DeleteOneType.Routine }, RoutineModel(prisma));
            return deleteId;
        }
        return null;
    },
    /**
     * Validate adds, updates, and deletes
     */
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<RoutineCreateInput, RoutineUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        if (!userId)
            throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0093') });
        // Collect organizationIds from each object, and check if the user is an admin/owner of every organization
        const organizationIds: (string | null | undefined)[] = [];
        if (createMany) {
            routinesCreate.validateSync(createMany, { abortEarly: false });
            TranslationModel().profanityCheck(createMany);
            // Add createdByOrganizationIds to organizationIds array, if they are set
            organizationIds.push(...createMany.map(input => input.createdByOrganizationId).filter(id => id));
            createMany.forEach(input => this.validateNodePositions(input));
            // Check if user will pass max routines limit
            const existingCount = await prisma.routine.count({
                where: {
                    OR: [
                        { user: { id: userId } },
                        { organization: { members: { some: { userId: userId, role: MemberRole.Owner as any } } } },
                    ]
                }
            })
            if (existingCount + (createMany?.length ?? 0) - (deleteMany?.length ?? 0) > 100) {
                throw new CustomError(CODE.MaxRoutinesReached, 'Maximum routines reached on this account', { code: genErrorCode('0094') });
            }
        }
        if (updateMany) {
            routinesUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            TranslationModel().profanityCheck(updateMany.map(u => u.data));
            // Add new organizationIds to organizationIds array, if they are set
            organizationIds.push(...updateMany.map(input => input.data.organizationId).filter(id => id))
            // Add existing organizationIds to organizationIds array, if userId does not match the object's userId
            const objects = await prisma.routine.findMany({
                where: { id: { in: updateMany.map(input => input.where.id) } },
                select: { id: true, userId: true, organizationId: true },
            });
            organizationIds.push(...objects.filter(object => object.userId !== userId).map(object => object.organizationId));
            updateMany.forEach(input => this.validateNodePositions(input.data));
        }
        if (deleteMany) {
            // Add organizationIds to organizationIds array, if userId does not match the object's userId
            const objects = await prisma.routine.findMany({
                where: { id: { in: deleteMany } },
                select: { id: true, userId: true, organizationId: true },
            });
            organizationIds.push(...objects.filter(object => object.userId !== userId).map(object => object.organizationId));
        }
        // Find admin/owner member data for every organization
        const memberData = await OrganizationModel(prisma).isOwnerOrAdmin(userId ?? '', organizationIds);
        // If any member data is undefined, the user is not authorized to delete one or more objects
        if (memberData.some(member => !member))
            throw new CustomError(CODE.Unauthorized, 'Not authorized to delete.', { code: genErrorCode('0095') })
    },
    /**
     * Performs adds, updates, and deletes of routines. First validates that every action is allowed.
     */
    async cud({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<RoutineCreateInput, RoutineUpdateInput>): Promise<CUDResult<Routine>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                let data = await this.toDBShape(userId, input, true);
                // Associate with either organization or user
                if (input.createdByOrganizationId) {
                    data = {
                        ...data,
                        organization: { connect: { id: input.createdByOrganizationId } },
                        createdByOrganization: { connect: { id: input.createdByOrganizationId } },
                    };
                } else {
                    data = {
                        ...data,
                        user: { connect: { id: userId } },
                        createdByUser: { connect: { id: userId } },
                    };
                }
                // Create object
                const currCreated = await prisma.routine.create({ data, ...selectHelper(partialInfo) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partialInfo);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Call createData helper function
                let data = await this.toDBShape(userId, input.data, false);
                // Find object
                let object = await prisma.routine.findFirst({ where: input.where })
                if (!object)
                    throw new CustomError(CODE.NotFound, 'Routine not found', { code: genErrorCode('0098') });
                // Associate with either organization or user. This will remove the association with the other.
                if (input.data.organizationId) {
                    data = {
                        ...data,
                        organization: { connect: { id: input.data.organizationId } },
                        user: { disconnect: true },
                    };
                } else {
                    data = {
                        ...data,
                        user: { connect: { id: userId } },
                        organization: { disconnect: true },
                    };
                }
                // Update object
                const currUpdated = await prisma.routine.update({
                    where: input.where,
                    data,
                    ...selectHelper(partialInfo)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partialInfo);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.organization.deleteMany({
                where: { id: { in: deleteMany } }
            })
        }
        return {
            created: createMany ? created : undefined,
            updated: updateMany ? updated : undefined,
            deleted: deleteMany ? deleted : undefined,
        };
    },
    /**
     * Duplicates a routine, along with its nodes and edges. 
     * If a fork, then the parent is set as the original routine. 
     * If a copy, there is no parent.
     */
    async duplicate({ userId, objectId, isFork, createCount = 0 }: DuplicateInput): Promise<DuplicateResult<Routine>> {
        let newCreateCount = createCount;
        // Find routine, with fields we want to copy.
        // I hope I discover a better way to do this.
        // Notable fields not being copied are completedAt and isComplete (since the copy will default to not complete),
        // score and views and other stats (since the copy will default to 0),
        // createdByUserId and createdByOrganizationId and projectId (since the copy will default to your own),
        // parentId (since the original will be the parent of the copy),
        // version (since the copy will default to 1.0.0),
        const routine = await prisma.routine.findFirst({
            where: { id: objectId },
            select: {
                id: true,
                complexity: true,
                isAutomatable: true,
                isInternal: true,
                simplicity: true,
                userId: true,
                organizationId: true,
                nodes: {
                    select: {
                        id: true,
                        columnIndex: true,
                        rowIndex: true,
                        type: true,
                        nodeEnd: {
                            select: {
                                wasSuccessful: true
                            }
                        },
                        loop: {
                            select: {
                                loops: true,
                                maxLoops: true,
                                operation: true,
                                whiles: {
                                    select: {
                                        condition: true,
                                        translations: {
                                            select: {
                                                description: true,
                                                title: true,
                                                language: true,
                                            }
                                        }
                                    }
                                }
                            }
                        },
                        nodeRoutineList: {
                            select: {
                                isOrdered: true,
                                isOptional: true,
                                routines: {
                                    select: {
                                        id: true,
                                        index: true,
                                        isOptional: true,
                                        routine: {
                                            select: {
                                                id: true,
                                                isInternal: true,
                                            }
                                        },
                                        translations: {
                                            select: {
                                                description: true,
                                                title: true,
                                                language: true,
                                            }
                                        }
                                    }
                                }
                            }
                        },
                        translations: {
                            select: {
                                description: true,
                                title: true,
                                language: true,
                            }
                        }
                    }
                },
                nodeLinks: {
                    select: {
                        fromId: true,
                        toId: true,
                        operation: true,
                        whens: {
                            select: {
                                condition: true,
                                translations: {
                                    select: {
                                        description: true,
                                        title: true,
                                        language: true,
                                    }
                                }
                            }
                        }
                    }
                },
                resourceLists: {
                    select: {
                        index: true,
                        usedFor: true,
                        resources: {
                            select: {
                                index: true,
                                link: true,
                                usedFor: true,
                                translations: {
                                    select: {
                                        description: true,
                                        title: true,
                                        language: true,
                                    }
                                }
                            }
                        },
                        translations: {
                            select: {
                                description: true,
                                title: true,
                                language: true,
                            }
                        }
                    }
                },
                inputs: {
                    select: {
                        isRequired: true,
                        name: true,
                        standardId: true,
                        translations: {
                            select: {
                                description: true,
                                language: true,
                            }
                        }
                    }
                },
                outputs: {
                    select: {
                        name: true,
                        standardId: true,
                        translations: {
                            select: {
                                description: true,
                                language: true,
                            }
                        }
                    }
                },
                tags: {
                    select: {
                        id: true,
                    }
                },
                translations: {
                    select: {
                        description: true,
                        instructions: true,
                        title: true,
                        language: true,
                    }
                }
            }
        });
        // If routine didn't exist
        if (!routine) {
            throw new CustomError(CODE.NotFound, 'Routine not found', { code: genErrorCode('0225') });
        }
        // If routine is marked as internal and it doesn't belong to you
        else if (routine.isInternal && routine.userId !== userId) {
            const memberData = await OrganizationModel(prisma).isOwnerOrAdmin(userId, [routine.organizationId]);
            if (!memberData.some(member => member))
                throw new CustomError(CODE.Unauthorized, 'Not authorized to copy', { code: genErrorCode('0226') })
        }
        // Initialize new routine object
        const newRoutine: any = routine;
        // For every node in the routine (and every edge which references it), change the ID 
        // to a new random ID.
        for (const node of newRoutine.nodes) {
            const oldId = node.id;
            // Update ID
            node.id = uuidv4();
            // Update reference to node in nodeLinks
            for (const nodeLink of newRoutine.nodeLinks) {
                if (nodeLink.fromId === oldId) {
                    nodeLink.fromId = node.id;
                }
                if (nodeLink.toId === oldId) {
                    nodeLink.toId = node.id;
                }
            }
        }
        // If copying subroutines, call this function for each subroutine
        if (createCount < 100 && routine.nodes && routine.nodes.length > 0) {
            const newSubroutineIds: string[] = [];
            // Find the IDs of all isInternal subroutines
            const oldSubroutineIds: string[] = [];
            for (const node of routine.nodes) {
                if (node.nodeRoutineList?.routines && node.nodeRoutineList?.routines.length > 0) {
                    for (const subroutine of node.nodeRoutineList.routines) {
                        if (subroutine.routine?.id && subroutine.routine?.isInternal) {
                            oldSubroutineIds.push(subroutine.routine.id);
                        }
                    }
                }
            }
            // Copy each subroutine
            for (const subroutineId of oldSubroutineIds) {
                const { object: copiedSubroutine, numCreated } = await this.duplicate({ userId, objectId: subroutineId, isFork: false, createCount: newCreateCount });
                newSubroutineIds.push(copiedSubroutine.id ?? '');
                newCreateCount += numCreated;
                if (newCreateCount >= 100) {
                    break;
                }
            }
            // Change the IDs of all subroutines to the new IDs
            for (const node of newRoutine.nodes) {
                if (node.nodeRoutineList?.routines && node.nodeRoutineList?.routines.length > 0) {
                    for (const subroutine of node.nodeRoutineList.routines) {
                        if (subroutine.routine?.id && newSubroutineIds.includes(subroutine.routine.id)) {
                            subroutine.routine.id = newSubroutineIds.find(id => id === subroutine.routine.id);
                        }
                    }
                }
            }
        }
        // Create the new routine
        const createdRoutine = await prisma.routine.create({
            data: {
                // Give ownership to user copying routine
                createdByUser: { connect: { id: userId } },
                user: { connect: { id: userId } },
                // Set parentId to original routine's id
                parent: isFork ? { connect: { id: routine.id } } : undefined,
                // Copy the rest of the fields
                complexity: newRoutine.complexity,
                isAutomatable: newRoutine.isAutomatable,
                isInternal: newRoutine.isInternal,
                simplicity: newRoutine.simplicity,
                nodes: newRoutine.nodes ? {
                    create: newRoutine.nodes.map((node: any) => ({
                        id: node.id,
                        columnIndex: node.columnIndex,
                        rowIndex: node.rowIndex,
                        type: node.type,
                        nodeEnd: node.nodeEnd ? {
                            create: {
                                wasSuccessful: node.nodeEnd.wasSuccessful
                            }
                        } : undefined,
                        loop: node.loop ? {
                            create: {
                                loops: node.loop.loops,
                                maxLoops: node.loop.maxLoops,
                                operation: node.loop.operation,
                                whiles: node.loop.whiles ? {
                                    create: node.loop.whiles.map((whileNode: any) => ({
                                        condition: whileNode.condition,
                                        translations: whileNode.translations ? {
                                            create: whileNode.translations.map((translation: any) => ({
                                                description: translation.description,
                                                title: translation.title,
                                                language: translation.language
                                            }))
                                        } : undefined
                                    }))
                                } : undefined
                            }
                        } : undefined,
                        nodeRoutineList: node.nodeRoutineList ? {
                            create: {
                                isOrdered: node.nodeRoutineList.isOrdered,
                                isOptional: node.nodeRoutineList.isOptional,
                                routines: node.nodeRoutineList.routines ? {
                                    create: [...node.nodeRoutineList.routines.map((routine: any) => {
                                        return ({
                                            index: routine.index,
                                            routine: { connect: { id: routine.routine.id } },
                                            translations: routine.translations ? {
                                                create: routine.translations.map((translation: any) => ({
                                                    description: translation.description,
                                                    title: translation.title,
                                                    language: translation.language
                                                }))
                                            } : undefined
                                        })
                                    })]
                                } : undefined
                            }
                        } : undefined,
                        translations: node.translations ? {
                            create: node.translations.map((translation: any) => ({
                                description: translation.description,
                                title: translation.title,
                                language: translation.language
                            }))
                        } : undefined,
                    }))
                } : undefined,
                nodeLinks: newRoutine.nodeLinks ? {
                    create: newRoutine.nodeLinks.map((nodeLink: any) => ({
                        from: { connect: { id: nodeLink.fromId } },
                        to: { connect: { id: nodeLink.toId } },
                        operation: nodeLink.operation,
                        whens: nodeLink.whens ? {
                            create: nodeLink.whens.map((when: any) => ({
                                condition: when.condition,
                                translations: when.translations ? {
                                    create: when.translations.map((translation: any) => ({
                                        description: translation.description,
                                        title: translation.title,
                                        language: translation.language
                                    }))
                                } : undefined
                            }))
                        } : undefined,
                        translations: nodeLink.translations ? {
                            create: nodeLink.translations.map((translation: any) => ({
                                description: translation.description,
                                title: translation.title,
                                language: translation.language
                            }))
                        } : undefined
                    }))
                } : undefined,
                resourceLists: newRoutine.resourceLists ? {
                    create: newRoutine.resourceLists.map((resourceList: any) => ({
                        index: resourceList.index,
                        usedFor: resourceList.usedFor,
                        resources: resourceList.resources ? {
                            create: resourceList.resources.map((resource: any) => ({
                                index: resource.index,
                                link: resource.link,
                                usedFor: resource.usedFor,
                                translations: resource.translations ? {
                                    create: resource.translations.map((translation: any) => ({
                                        description: translation.description,
                                        title: translation.title,
                                        language: translation.language
                                    }))
                                } : null
                            }))
                        } : null
                    }))
                } : undefined,
                inputs: newRoutine.inputs ? {
                    create: newRoutine.inputs.map((input: any) => ({
                        isRequired: input.isRequired,
                        name: input.name,
                        standard: { connect: { id: input.standardId } },
                        translations: input.translations ? {
                            create: input.translations.map((translation: any) => ({
                                description: translation.description,
                                title: translation.title,
                                language: translation.language
                            }))
                        } : undefined
                    }))
                } : undefined,
                outputs: newRoutine.outputs ? {
                    create: newRoutine.outputs.map((output: any) => ({
                        name: output.name,
                        standardId: output.standardId,
                        translations: output.translations ? {
                            create: output.translations.map((translation: any) => ({
                                description: translation.description,
                                title: translation.title,
                                language: translation.language
                            }))
                        } : null
                    }))
                } : undefined,
                tags: newRoutine.tags ? {
                    connect: newRoutine.tags.map((tag: any) => ({
                        id: tag.id
                    }))
                } : undefined,
                translations: newRoutine.translations ? {
                    create: newRoutine.translations.map((translation: any) => ({
                        description: translation.description,
                        instructions: translation.instructions,
                        // Add "Copy" to title
                        title: `${translation.title} (Copy)`,
                        language: translation.language
                    }))
                } : undefined,
            }
        });
        return {
            object: createdRoutine as any,
            numCreated: newCreateCount + 1
        }
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function RoutineModel(prisma: PrismaType) {
    const prismaObject = prisma.routine;
    const format = routineFormatter();
    const search = routineSearcher();
    const mutate = routineMutater(prisma);

    return {
        prisma,
        prismaObject,
        ...format,
        ...search,
        ...mutate,
    }
}
//==============================================================
/* #endregion Model */
//==============================================================