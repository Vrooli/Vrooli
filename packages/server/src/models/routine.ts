import { Routine, RoutineCreateInput, RoutineUpdateInput, RoutineSearchInput, RoutineSortBy, Count, ResourceListUsedFor, NodeLink, NodeRoutineList, NodeRoutineListItem, Node, NodeCreateInput, NodeUpdateInput, NodeRoutineListCreateInput, NodeRoutineListUpdateInput, NodeRoutineListItemUpdateInput, NodeRoutineListItemCreateInput } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { addCountFieldsHelper, addCreatorField, addJoinTablesHelper, addOwnerField, CUDInput, CUDResult, FormatConverter, GraphQLModelType, modelToGraphQL, PartialInfo, relationshipToPrisma, RelationshipTypes, removeCountFieldsHelper, removeCreatorField, removeJoinTablesHelper, removeOwnerField, Searcher, selectHelper, ValidateMutationsInput } from "./base";
import { CustomError } from "../error";
import { CODE, inputCreate, inputUpdate, MemberRole, routineCreate, routineTranslationCreate, routineTranslationUpdate, routineUpdate } from "@local/shared";
import { hasProfanity } from "../utils/censor";
import { OrganizationModel } from "./organization";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { NodeModel } from "./node";
import { StandardModel } from "./standard";
import _ from "lodash";
import { TranslationModel } from "./translation";
import { ResourceListModel } from "./resourceList";

//==============================================================
/* #region Custom Components */
//==============================================================

const joinMapper = { tags: 'tag', starredBy: 'user' };
const countMapper = { nodesCount: 'nodes' }
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
        let { isUpvoted, isStarred, role, ...rest } = partial;
        return rest;
    },
    constructUnions: (data) => {
        // console.log('CONSTRUCT UNIONS routine', data);
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
        // console.log('fhdkasfdas routine add join tables', partial);
        return addJoinTablesHelper(partial, joinMapper);
    },
    removeJoinTables: (data) => {
        return removeJoinTablesHelper(data, joinMapper);
    },
    addCountFields: (partial) => {
        // console.log('fhdkasfdas routine add count fields', partial);
        return addCountFieldsHelper(partial, countMapper);
    },
    removeCountFields: (data) => {
        return removeCountFieldsHelper(data, countMapper);
    },
    async addSupplementalFields(
        prisma: PrismaType,
        userId: string | null, // Of the user making the request
        objects: RecursivePartial<any>[],
        partial: PartialInfo,
    ): Promise<RecursivePartial<Routine>[]> {
        console.log('ROUTINE ADD SUPPL FILEDS', objects.length);
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
        // Query for isStarred
        if (partial.isStarred) {
            const isStarredArray = userId
                ? await StarModel(prisma).getIsStarreds(userId, ids, 'routine')
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
        }
        // Query for isUpvoted
        if (partial.isUpvoted) {
            const isUpvotedArray = userId
                ? await VoteModel(prisma).getIsUpvoteds(userId, ids, 'routine')
                : Array(ids.length).fill(false);
            objects = objects.map((x, i) => ({ ...x, isUpvoted: isUpvotedArray[i] }));
        }
        // Query for role
        if (partial.role) {
            console.log('ROUTINE QUERYING FOR ROLE');
            // console.log('routine supplemental fields', objects)
            // If owned by user, set role to owner if userId matches
            // If owned by organization, set role user's role in organization
            const organizationIds = objects
                .filter(x => Array.isArray(x.owner?.translations) && x.owner.translations.length > 0 && x.owner.translations[0].name)
                .map(x => x.owner.id)
                .filter(x => Boolean(x)) as string[];
            console.log('routine query organizationIds', organizationIds);
            const roles = userId
                ? await OrganizationModel(prisma).getRoles(userId, organizationIds)
                : [];
            console.log('routine query got roles', roles);
            objects = objects.map((x) => {
                console.log('routien role objects map x.id', x.id, organizationIds)
                const orgRoleIndex = organizationIds.findIndex(id => id === x.owner?.id);
                console.log('bbop', orgRoleIndex);
                if (orgRoleIndex >= 0) {
                    return { ...x, role: roles[orgRoleIndex] };
                }
                return { ...x, role: (Boolean(x.owner?.id) && x.owner?.id === userId) ? MemberRole.Owner : undefined };
            }) as any;
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
        const isCompleteQuery = input.isComplete ? { isComplete: true } : {};
        const languagesQuery = input.languages ? { translations: { some: { language: { in: input.languages } } } } : {};
        const minComplexity = input.minComplexity ? { complexity: { gte: input.minComplexity } } : {};
        const maxComplexity = input.maxComplexity ? { complexity: { lte: input.maxComplexity } } : {};
        const minSimplicity = input.minSimplicity ? { simplicity: { gte: input.minSimplicity } } : {};
        const maxSimplicity = input.maxSimplicity ? { simplicity: { lte: input.maxSimplicity } } : {};
        const minScoreQuery = input.minScore ? { score: { gte: input.minScore } } : {};
        const minStarsQuery = input.minStars ? { stars: { gte: input.minStars } } : {};
        const resourceListsQuery = input.resourceLists ? { resourceLists: { some: { translations: { some: { title: { in: input.resourceLists } } } } } } : {};
        const resourceTypesQuery = input.resourceTypes ? { resourceLists: { some: { usedFor: ResourceListUsedFor.Display as any, resources: { some: { usedFor: { in: input.resourceTypes } } } } } } : {}; const userIdQuery = input.userId ? { userId: input.userId } : undefined;
        const organizationIdQuery = input.organizationId ? { organizationId: input.organizationId } : undefined;
        const projectIdQuery = input.projectId ? { projectId: input.projectId } : {};
        const parentIdQuery = input.parentId ? { parentId: input.parentId } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const tagsQuery = input.tags ? { tags: { some: { tag: { tag: { in: input.tags } } } } } : {};
        return { isInternal: false, ...isCompleteQuery, ...languagesQuery, ...minComplexity, ...maxComplexity, ...minSimplicity, ...maxSimplicity, ...minScoreQuery, ...minStarsQuery, ...resourceListsQuery, ...resourceTypesQuery, ...userIdQuery, ...organizationIdQuery, ...parentIdQuery, ...projectIdQuery, ...reportIdQuery, ...tagsQuery };
    },
})

/**
 * Calculates the shortest AND longest weighted path on a directed cyclic graph. (loops are actually not the cyclic part, but redirects)
 * A routine with no nodes has a complexity of 1.
 * Each decision the user makes (i.e. multiple edges coming out of a node) has a weight of 1.
 * Each node has a weight that is the summation of its contained subroutines.
 * @param nodes A map of node IDs to their weight (simplicity/complexity)
 * @param edges The edges of the graph, with each object containing a fromId and toId
 * @returns [shortestPath, longestPath] The shortest and longest weighted distance
 */
export const calculateShortestLongestWeightedPath = (nodes: { [id: string]: { simplicity: number, complexity: number } }, edges: { fromId: string, toId: string }[]): [number, number] => {
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
            if (fromEdges.length > 1) weight = { complexity: weight.complexity + 1, simplicity: weight.simplicity + 1 };
            // Add edge to visited edges
            const newVisitedEdges = visitedEdges.concat([edge]);
            // Recurse on the next node
            const [shortest, longest] = getShortLong(edge.fromId, newVisitedEdges, currShortest + weight.simplicity, currLongest + weight.complexity);
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
     * @param data The routine data, either a create or update
     * @returns [simplicity, complexity] Numbers representing the shorted and longest weighted paths
     */
    async calculateComplexity(data: RoutineCreateInput | RoutineUpdateInput): Promise<[number, number]> {
        console.log('calculate complexity start', JSON.stringify(data))
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
                                    }
                                }
                            }
                        }
                    }
                }
            })
        }
        console.log('Existing routine', JSON.stringify(existingRoutine))
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
        const nodeDictionary: { [id: string]: string[] } = {};
        // Find the ID of every subroutine
        const subroutineIds: string[] = nodes.map((node: any | NodeCreateInput | NodeUpdateInput) => {
            console.log('in subroutineIds loop node', JSON.stringify(node));
            // Calculate the list of subroutines after mutations are applied
            let ids: string[] = node.nodeRoutineList?.routines?.map((item: any) => item.routine.id) ?? [];
            console.log('got idsss', ids);
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
            nodeDictionary[node.id] = ids;
            return ids
        }).flat();
        // Query every subroutine's complexity/simplicity in one go
        const complicities = await prisma.routine.findMany({
            where: { id: { in: subroutineIds } },
            select: {
                id: true,
                complexity: true,
                simplicity: true
            }
        })
        // Convert compexity/simplicity to a map for easy lookup
        let routineComplicityDict: { [routineId: string]: { complexity: number, simplicity: number } } = {};
        for (const sub of complicities) {
            routineComplicityDict[sub.id] = { complexity: sub.complexity, simplicity: sub.simplicity };
        }
        // Calculate the complexity/simplicity of each node. If node has no subroutines, its complexity is 0 (e.g. start node, end node)
        let nodeComplicityDict: { [nodeId: string]: { complexity: number, simplicity: number } } = {};
        for (const node of nodes) {
            nodeComplicityDict[node.id] = {
                complexity: nodeDictionary[node.id]?.reduce((acc, cur) => acc + routineComplicityDict[cur].complexity, 0) || 0,
                simplicity: nodeDictionary[node.id]?.reduce((acc, cur) => acc + routineComplicityDict[cur].simplicity, 0) || 0,
            }
        }
        // Using the node links, determine the most complex path through the routine
        const [shortest, longest] = calculateShortestLongestWeightedPath(nodeComplicityDict, nodeLinks);
        // return with +1 
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
    async toDBShape(userId: string | null, data: RoutineCreateInput | RoutineUpdateInput): Promise<any> {
        const [simplicity, complexity] = await this.calculateComplexity(data);
        console.log('complexity calculated', complexity, simplicity);
        return {
            id: (data as RoutineUpdateInput)?.id ?? undefined,
            isAutomatable: data.isAutomatable,
            isComplete: data.isComplete,
            completedAt: data.isComplete ? new Date().toISOString() : null,
            complexity: complexity,
            simplicity: simplicity,
            isInternal: data.isInternal,
            parentId: data.parentId,
            version: data.version,
            resourceLists: await ResourceListModel(prisma).relationshipBuilder(userId, data, false),
            tags: await TagModel(prisma).relationshipBuilder(userId, data, false),
            inputs: this.relationshipBuilderInput(userId, data, false),
            outpus: this.relationshipBuilderOutput(userId, data, false),
            nodes: await NodeModel(prisma).relationshipBuilder(userId, (data as RoutineUpdateInput)?.id ?? null, data, false),
            nodeLinks: NodeModel(prisma).relationshipBuilderNodeLink(userId, data, false),
            translations: TranslationModel().relationshipBuilder(userId, data, { create: routineTranslationCreate, update: routineTranslationUpdate }, false),
        }
    },
    /**
    * Add, update, or remove routine inputs from a routine.
    * NOTE: Input is whole routine data, not just the inputs. 
    * This is because we may need the node data to calculate inputs
    */
    relationshipBuilderInput(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        console.log('in relationshipBuilderInput');
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported in this case (since they can only be applied to one routine)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'inputs', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        const standardModel = StandardModel(prisma);
        console.log('relationshipBuilderInput formattedInput', formattedInput);
        // If nodes relationship provided, calculate inputs and outputs from nodes. Otherwise, use given inputs 
        //TODO
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const data of formattedInput.create) {
                // Check for valid arguments
                inputCreate.validateSync(data, { abortEarly: false });
                // Check for censored words
                if (hasProfanity(data.name, data.description)) throw new CustomError(CODE.BannedWord);
                // Convert nested relationships
                data.standard = standardModel.relationshipBuilder(userId, data, isAdd);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments
                inputUpdate.validateSync(data, { abortEarly: false });
                // Check for censored words
                if (hasProfanity(data.name, data.description)) throw new CustomError(CODE.BannedWord);
                // Convert nested relationships
                data.standard = standardModel.relationshipBuilder(userId, data, isAdd);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove routine outputs from a routine
     * NOTE: Input is whole routine data, not just the outputs. 
     * This is because we may need the node data to calculate outputs
     */
    relationshipBuilderOutput(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported in this case (since they can only be applied to one routine)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'outputs', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        const standardModel = StandardModel(prisma);
        // If nodes relationship provided, calculate inputs and outputs from nodes. Otherwise, use given inputs
        //TODO
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const data of formattedInput.create) {
                // Check for valid arguments
                inputCreate.validateSync(data, { abortEarly: false });
                // Check for censored words
                TranslationModel().profanityCheck(data);
                // Convert nested relationships
                data.standard = standardModel.relationshipBuilder(userId, data, isAdd);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments
                inputUpdate.validateSync(data, { abortEarly: false });
                // Check for censored words
                TranslationModel().profanityCheck(data);
                // Convert nested relationships
                data.standard = standardModel.relationshipBuilder(userId, data, isAdd);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove a routine relationship from a routine list
     */
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'routines',
    ): Promise<{ [x: string]: any } | undefined> {
        const fieldExcludes = ['node', 'user', 'userId', 'organization', 'organizationId', 'createdByUser', 'createdByUserId', 'createdByOrganization', 'createdByOrganizationId'];
        let formattedInput = relationshipToPrisma({ data: input, relationshipName, isAdd, fieldExcludes })
        // Validate
        const { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        await this.validateMutations({
            userId,
            createMany: createMany as RoutineCreateInput[],
            updateMany: updateMany?.map(d => d.data) as RoutineUpdateInput[],
            deleteMany: deleteMany?.map(d => d.id)
        });
        // Shape
        if (Array.isArray(formattedInput.create)) {
            formattedInput.create = formattedInput.create.map(async (data) => await this.toDBShape(userId, data as any));
        }
        if (Array.isArray(formattedInput.update)) {
            formattedInput.update = formattedInput.update.map(async (data) => ({
                where: data.where,
                data: await this.toDBShape(userId, data.data as any)
            }))
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Validate adds, updates, and deletes
     */
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<RoutineCreateInput, RoutineUpdateInput>): Promise<void> {
        if ((createMany || updateMany || deleteMany) && !userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations');
        if (createMany) {
            createMany.forEach(input => routineCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => TranslationModel().profanityCheck(input));
            createMany.forEach(input => this.validateNodePositions(input));
            // Check if user will pass max routines limit
            const existingCount = await prisma.routine.count({
                where: {
                    OR: [
                        { user: { id: userId ?? '' } },
                        { organization: { members: { some: { userId: userId ?? '', role: MemberRole.Owner as any } } } },
                    ]
                }
            })
            if (existingCount + (createMany?.length ?? 0) - (deleteMany?.length ?? 0) > 100) {
                throw new CustomError(CODE.MaxRoutinesReached);
            }
        }
        if (updateMany) {
            console.log('ROUTINE VALIDATE MUTATIONS UPDATEMANY', JSON.stringify(updateMany));
            updateMany.forEach(input => routineUpdate.validateSync(input, { abortEarly: false }));
            updateMany.forEach(input => TranslationModel().profanityCheck(input));
            updateMany.forEach(input => this.validateNodePositions(input));
        }
        if (deleteMany) {
            // Check if user is authorized to delete
            const objects = await prisma.routine.findMany({
                where: { id: { in: deleteMany } },
                select: { id: true, userId: true, organizationId: true },
            });
            // Filter out objects that match the user's Id, since we know those are authorized
            const objectsToCheck = objects.filter(object => object.userId !== userId);
            if (objectsToCheck.length > 0) {
                for (const check of objectsToCheck) {
                    // Check if user is authorized to delete
                    if (!check.organizationId) throw new CustomError(CODE.Unauthorized, 'Not authorized to delete');
                    const [authorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId ?? '', check.organizationId);
                    if (!authorized) throw new CustomError(CODE.Unauthorized, 'Not authorized to delete.');
                }
            }
        }
    },
    /**
     * Performs adds, updates, and deletes of organizations. First validates that every action is allowed.
     */
    async cud({ partial, userId, createMany, updateMany, deleteMany }: CUDInput<RoutineCreateInput, RoutineUpdateInput>): Promise<CUDResult<Routine>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                let data = await this.toDBShape(userId, input);
                // Associate with either organization or user
                if (input.createdByOrganizationId) {
                    // Make sure the user is an admin of the organization
                    const [isAuthorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId ?? '', input.createdByOrganizationId);
                    if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
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
                const currCreated = await prisma.routine.create({ data, ...selectHelper(partial) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partial);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Call createData helper function
                let data = await this.toDBShape(userId, input);
                console.log('ROUTINE UPDATEMANY TODBSHAPE AFTER', JSON.stringify(data));
                // Associate with either organization or user. This will remove the association with the other.
                if (input.organizationId) {
                    // Make sure the user is an admin of the organization
                    console.log('checking isauthorized org', userId, input);
                    const [isAuthorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId ?? '', input.organizationId);
                    if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
                    data = {
                        ...data,
                        organization: { connect: { id: input.organizationId } },
                        user: { disconnect: true },
                    };
                } else {
                    console.log('checking isauthorized user', userId, input);
                    data = {
                        ...data,
                        user: { connect: { id: userId } },
                        organization: { disconnect: true },
                    };
                }
                let object = await prisma.routine.findFirst({
                    where: {
                        AND: [
                            { id: input.id },
                            {
                                OR: [
                                    { organizationId: input.organizationId },
                                    { userId },
                                ]
                            }
                        ]
                    }
                })
                console.log('routine found: ', input.id, userId, input.organizationId, JSON.stringify(object));
                if (!object) throw new CustomError(CODE.ErrorUnknown);
                // Update object
                const currUpdated = await prisma.routine.update({
                    where: { id: object.id },
                    data,
                    ...selectHelper(partial)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partial);
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