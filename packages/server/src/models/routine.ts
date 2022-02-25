import { DeleteOneInput, FindByIdInput, Routine, RoutineCountInput, RoutineCreateInput, RoutineUpdateInput, RoutineSearchInput, RoutineSortBy, Success } from "../schema/types";
import { PartialSelectConvert, PrismaType, RecursivePartial } from "types";
import { addCreatorField, addJoinTables, addOwnerField, counter, FormatConverter, FormatterMap, infoToPartialSelect, InfoType, MODEL_TYPES, PaginatedSearchResult, relationshipFormatter, relationshipToPrisma, RelationshipTypes, removeCreatorField, removeJoinTables, removeOwnerField, searcher, selectHelper, Sortable } from "./base";
import { CustomError } from "../error";
import { CODE, inputCreate, inputUpdate, MemberRole, routineCreate, routineUpdate } from "@local/shared";
import { hasProfanity } from "../utils/censor";
import { OrganizationModel } from "./organization";
import { ResourceModel } from "./resource";
import { routine } from "@prisma/client";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { NodeModel } from "./node";
import { StandardModel } from "./standard";
import _ from "lodash";

export const routineDBFields = ['id', 'created_at', 'updated_at', 'description', 'instructions', 'isAutomatable', 'title', 'version', 'createdByUserId', 'createdByOrganizationId', 'userId', 'organizationId', 'parentId', 'projectId', 'score', 'stars']

//==============================================================
/* #region Custom Components */
//==============================================================

type RoutineFormatterType = FormatConverter<Routine, routine>;
/**
 * Component for formatting between graphql and prisma types
 */
export const routineFormatter = (): RoutineFormatterType => {
    const joinMapper = {
        tags: 'tag',
        starredBy: 'user',
    };
    return {
        dbShape: (partial: PartialSelectConvert<Routine>): PartialSelectConvert<routine> => {
            let modified = partial;
            console.log('in routine toDB', modified);
            // Deconstruct GraphQL unions
            modified = removeCreatorField(modified);
            modified = removeOwnerField(modified);
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['inputs.standard', FormatterMap.Standard.dbShape],
                ['outputs.standard', FormatterMap.Standard.dbShape],
                ['nodes', FormatterMap.Node.dbShape],
                // TODO nodeLists might be useful for metrics
                ['parent', FormatterMap.Routine.dbShape],
                ['project', FormatterMap.Project.dbShape],
                ['contextualResources', FormatterMap.Resource.dbShape],
                ['externalResources', FormatterMap.Resource.dbShape],
                ['forks', FormatterMap.Routine.dbShape],
                ['tags', FormatterMap.Tag.dbShape],
                ['starredBy', FormatterMap.User.dbShapeUser],
                ['reports', FormatterMap.Report.dbShape],
                ['comments', FormatterMap.Comment.dbShape],
            ]);
            // Add join tables not present in GraphQL type, but present in Prisma
            modified = addJoinTables(modified, joinMapper);
            return modified;
        },
        dbPrune: (info: InfoType): PartialSelectConvert<routine> => {
            // Convert GraphQL info object to a partial select object
            let modified = infoToPartialSelect(info);
            // Remove calculated fields
            let { isUpvoted, isStarred, role, ...rest } = modified;
            modified = rest;
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['inputs.standard', FormatterMap.Standard.dbPrune],
                ['outputs.standard', FormatterMap.Standard.dbPrune],
                ['nodes', FormatterMap.Node.dbPrune],
                // TODO nodeLists might be useful for metrics
                ['parent', FormatterMap.Routine.dbPrune],
                ['project', FormatterMap.Project.dbPrune],
                ['contextualResources', FormatterMap.Resource.dbPrune],
                ['externalResources', FormatterMap.Resource.dbPrune],
                ['forks', FormatterMap.Routine.dbPrune],
                ['tags', FormatterMap.Tag.dbPrune],
                ['starredBy', FormatterMap.User.dbPruneUser],
                ['reports', FormatterMap.Report.dbPrune],
                ['comments', FormatterMap.Comment.dbPrune],
            ]);
            return modified;
        },
        selectToDB: (info: InfoType): PartialSelectConvert<routine> => {
            return routineFormatter().dbShape(routineFormatter().dbPrune(info));
        },
        selectToGraphQL: (obj: RecursivePartial<routine>): RecursivePartial<Routine> => {
            if (!_.isObject(obj)) return obj;
            // Create unions
            let modified = addCreatorField(obj);
            modified = addOwnerField(modified);
            // Remove join tables that are not present in GraphQL type, but present in Prisma
            modified = removeJoinTables(modified, joinMapper);
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['inputs.standard', FormatterMap.Standard.selectToGraphQL],
                ['outputs.standard', FormatterMap.Standard.selectToGraphQL],
                ['nodes', FormatterMap.Node.selectToGraphQL],
                // TODO nodeLists might be useful for metrics
                ['parent', FormatterMap.Routine.selectToGraphQL],
                ['project', FormatterMap.Project.selectToGraphQL],
                ['contextualResources', FormatterMap.Resource.selectToGraphQL],
                ['externalResources', FormatterMap.Resource.selectToGraphQL],
                ['forks', FormatterMap.Routine.selectToGraphQL],
                ['tags', FormatterMap.Tag.selectToGraphQL],
                ['starredBy', FormatterMap.User.selectToGraphQLUser],
                ['reports', FormatterMap.Report.selectToGraphQL],
                ['comments', FormatterMap.Comment.selectToGraphQL],
            ]);
            // NOTE: "Add calculated fields" is done in the supplementalFields function. Allows results to batch their database queries.
            return modified;
        },
    }
}

/**
 * Component for search filters
 */
const sorter = (): Sortable<RoutineSortBy> => ({
    defaultSort: RoutineSortBy.AlphabeticalAsc,
    getSortQuery: (sortBy: string): any => {
        return {
            [RoutineSortBy.AlphabeticalAsc]: { title: 'asc' },
            [RoutineSortBy.AlphabeticalDesc]: { title: 'desc' },
            [RoutineSortBy.CommentsAsc]: { comments: { _count: 'asc' } },
            [RoutineSortBy.CommentsDesc]: { comments: { _count: 'desc' } },
            [RoutineSortBy.ForksAsc]: { forks: { _count: 'asc' } },
            [RoutineSortBy.ForksDesc]: { forks: { _count: 'desc' } },
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
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { title: { ...insensitive } },
                { description: { ...insensitive } },
                { instructions: { ...insensitive } },
                { tags: { some: { tag: { tag: { ...insensitive } } } } },
            ]
        })
    }
})

/**
 * Handles the authorized adding, updating, and deleting of routines.
 */
const routiner = (format: RoutineFormatterType, sort: Sortable<RoutineSortBy>, prisma: PrismaType) => ({
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
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const data of formattedInput.create) {
                // Check for valid arguments
                inputCreate.validateSync(data, { abortEarly: false });
                // Check for censored words
                this.profanityCheck(data as RoutineCreateInput);
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
                this.profanityCheck(data as RoutineUpdateInput);
                // Convert nested relationships
                data.standard = standardModel.relationshipBuilder(userId, data, isAdd);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove a routine relationship from a routine list
     */
    relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        const fieldExcludes = ['node', 'user', 'userId', 'organization', 'organizationId', 'createdByUser', 'createdByUserId', 'createdByOrganization', 'createdByOrganizationId'];
        // Convert input to Prisma shape, excluding nodes and auth fields
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'routine', isAdd, fieldExcludes })
        const resourceModel = ResourceModel(prisma);
        const tagModel = TagModel(prisma);
        // If nodes relationship provided, calculate inputs and outputs from nodes. Otherwise, use given inputs
        // Also calculate each node's previous and next nodes, if those fields are using a UUID 
        // that refers to a new node (i.e. not added to the database before). These UUIDs will
        // look something like 'new-uuid-12345' instead of 'f5f5f5f5-f5f5-f5f5-f5f5-f5f5f5f5f5f5' TODO
        if (Object.keys(input).some(key => key.startsWith('nodes'))) {
            // NOTE: nodes cannot be connected/disconnected from a routine, so we don't need to check for that here
            const creates = Array.isArray(input.nodesCreate) ? input.nodesCreate : [];
            const updates = Array.isArray(input.nodesUpdate) ? input.nodesUpdate : [];
            const deletes = Array.isArray(input.nodesDelete) ? input.nodesDelete : [];
            // Make sure not passing node limit
            // if (creates.length - )
            // If node links relationship provided, replace ID
            if (Object.keys(input).some(key => key.startsWith('nodeLinks'))) {
            }
        }
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const data of formattedInput.create) {
                // Check for valid arguments
                routineCreate.omit(fieldExcludes).validateSync(data, { abortEarly: false });
                // Check for censored words
                this.profanityCheck(data as RoutineCreateInput);
                // Handle resources
                data.contextualResources = resourceModel.relationshipBuilder(userId, { resourcesCreate: data.resourcesContextualCreate }, isAdd);
                data.externalResources = resourceModel.relationshipBuilder(userId, { resourcesCreate: data.resourcesExternalCreate }, isAdd);
                // Handle tags
                data.tags = tagModel.relationshipBuilder(userId, data, isAdd);
                // Handle input/outputs
                data.inputs = this.relationshipBuilderInput(userId, data, isAdd);
                data.outpus = this.relationshipBuilderOutput(userId, data, isAdd);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const data of formattedInput.update) {
                // Check for valid arguments
                routineUpdate.omit(fieldExcludes).validateSync(data, { abortEarly: false });
                // Check for censored words
                this.profanityCheck(data as RoutineUpdateInput)
                // Handle resources
                data.contextualResources = resourceModel.relationshipBuilder(userId, { resourcesCreate: data.resourcesContextualCreate }, true);
                data.externalResources = resourceModel.relationshipBuilder(userId, { resourcesCreate: data.resourcesExternalCreate }, true);
                // Handle tags
                data.tags = tagModel.relationshipBuilder(userId, data, true);
                // Handle input/outputs
                data.inputs = this.relationshipBuilderInput(userId, data, isAdd);
                data.outpus = this.relationshipBuilderOutput(userId, data, isAdd);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async find(
        userId: string | null | undefined, // Of the user making the request, not the routine
        input: FindByIdInput,
        info: InfoType,
    ): Promise<RecursivePartial<Routine> | null> {
        console.log('routine find');
        // Create selector
        const select = selectHelper(info, format.selectToDB);
        console.log('routine find select', select);
        // Access database
        let routine = await prisma.routine.findUnique({ where: { id: input.id }, ...select });
        console.log('routine find routine', routine);
        // Return routine with "isUpvoted" and "isStarred" fields. These must be queried separately.
        if (!routine) throw new CustomError(CODE.InternalError, 'Routine not found');
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [routine], info);
        console.log('routine find formatted[0]', formatted[0]);
        return formatted[0];
    },
    async search(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: RoutineSearchInput,
        info: InfoType,
    ): Promise<PaginatedSearchResult> {
        // Create where clauses
        const userIdQuery = input.userId ? { userId: input.userId } : undefined;
        const organizationIdQuery = input.organizationId ? { organizationId: input.organizationId } : undefined;
        const parentIdQuery = input.parentId ? { parentId: input.parentId } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        // Search
        const search = searcher<RoutineSortBy, RoutineSearchInput, Routine, routine>(MODEL_TYPES.Routine, format.selectToDB, sort, prisma);
        console.log('calling routine search...')
        let searchResults = await search.search({ ...userIdQuery, ...organizationIdQuery, ...parentIdQuery, ...reportIdQuery, ...where }, input, info);
        // Format and add supplemental/calculated fields to each result node
        console.log('routine searchResults', searchResults);
        let formattedNodes = searchResults.edges.map(({ node }) => node);
        console.log('routine formatted nodes', formattedNodes);
        formattedNodes = await this.supplementalFields(userId, formattedNodes, info);
        return { pageInfo: searchResults.pageInfo, edges: searchResults.edges.map(({ node, ...rest }) => ({ node: formattedNodes.shift(), ...rest })) };
    },
    async create(
        userId: string,
        input: RoutineCreateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Routine>> {
        // Check for valid arguments
        routineCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input)
        // Create routine data
        let routineData: { [x: string]: any } = {
            description: input.description,
            name: input.title,
            instructions: input.instructions,
            isAutomatable: input.isAutomatable,
            parentId: input.parentId,
            version: input.version,
            // Handle resources
            contextualResources: ResourceModel(prisma).relationshipBuilder(userId, { resourcesCreate: input.resourcesContextualCreate }, true),
            externalResources: ResourceModel(prisma).relationshipBuilder(userId, { resourcesCreate: input.resourcesExternalCreate }, true),
            // Handle tags
            tags: await TagModel(prisma).relationshipBuilder(userId, input, true),
            // Handle input/outputs
            inputs: this.relationshipBuilderInput(userId, input, true),
            outpus: this.relationshipBuilderOutput(userId, input, true),
        };
        // Associate with either organization or user
        if (input.createdByOrganizationId) {
            // Make sure the user is an admin of the organization
            const [isAuthorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.createdByOrganizationId);
            if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
            routineData = {
                ...routineData,
                organization: { connect: { id: input.createdByOrganizationId } },
                createdByOrganization: { connect: { id: input.createdByOrganizationId } },
            };
        } else {
            routineData = {
                ...routineData,
                user: { connect: { id: userId } },
                createdByUser: { connect: { id: userId } },
            };
        }
        // Create routine
        const routine = await prisma.routine.create({
            data: routineData as any,
            ...selectHelper(info, format.selectToDB)
        })
        // Handle nodes TODO
        if (input.nodesCreate) {
            // Check if routine will pass max nodes
            const createCount = Array.isArray(input.nodesCreate) ? input.nodesCreate.length : 0;
            NodeModel(prisma).nodeCountCheck(routine.id, createCount);
        }
        // Return project with "role", "isUpvoted" and "isStarred" fields. These will be their default values.
        return { ...format.selectToGraphQL(routine), role: MemberRole.Owner as any, isUpvoted: null, isStarred: false };
    },
    async update(
        userId: string,
        input: RoutineUpdateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Routine>> {
        // Check for valid arguments
        routineUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input)
        // Create routine data
        let routineData: { [x: string]: any } = {
            description: input.description,
            name: input.title,
            instructions: input.instructions,
            isAutomatable: input.isAutomatable,
            parentId: input.parentId,
            version: input.version,
            // Handle resources
            contextualResources: ResourceModel(prisma).relationshipBuilder(userId, {
                resourcesCreate: input.resourcesContextualCreate,
                resourcesUpdate: input.resourcesContextualUpdate,
                resourcesDelete: input.resourcesContextualDelete,
            }, false),
            externalResources: ResourceModel(prisma).relationshipBuilder(userId, {
                resourcesCreate: input.resourcesExternalCreate,
                resourcesUpdate: input.resourcesExternalUpdate,
                resourcesDelete: input.resourcesExternalDelete,
            }, false),
            // Handle tags
            tags: await TagModel(prisma).relationshipBuilder(userId, input, false),
            // Handle input/outputs
            inputs: this.relationshipBuilderInput(userId, input, false),
            outpus: this.relationshipBuilderOutput(userId, input, false),
        };
        // Associate with either organization or user. This will remove the association with the other.
        if (input.organizationId) {
            // Make sure the user is an admin of the organization
            const [isAuthorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.organizationId);
            if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
            routineData = {
                ...routineData,
                organization: { connect: { id: input.organizationId } },
                user: { disconnect: true },
            };
        } else {
            routineData = {
                ...routineData,
                user: { connect: { id: userId } },
                organization: { disconnect: true },
            };
        }
        // Find routine
        let routine = await prisma.routine.findFirst({
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
        if (!routine) throw new CustomError(CODE.ErrorUnknown);
        // Update routine
        routine = await prisma.routine.update({
            where: { id: routine.id },
            data: routine as any,
            ...selectHelper(info, format.selectToDB)
        });
        // Handle nodes TODO
        if (input.nodesCreate) {
            // Check if routine will pass max nodes
            const createCount = Array.isArray(input.nodesCreate) ? input.nodesCreate.length : 0;
            const deleteCount = Array.isArray(input.nodesDelete) ? input.nodesDelete.length : 0;
            NodeModel(prisma).nodeCountCheck(routine.id, createCount - deleteCount);
        }
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [routine], info);
        return formatted[0];
    },
    async delete(userId: string, input: DeleteOneInput): Promise<Success> {
        // Find
        const routine = await prisma.routine.findFirst({
            where: { id: input.id },
            select: {
                id: true,
                userId: true,
                organizationId: true,
            }
        })
        if (!routine) throw new CustomError(CODE.NotFound, "Routine not found");
        // Check if user is authorized
        let authorized = userId === routine.userId;
        if (!authorized && routine.organizationId) {
            [authorized] = await OrganizationModel(prisma).isOwnerOrAdmin(userId, routine.organizationId);
        }
        if (!authorized) throw new CustomError(CODE.Unauthorized);
        // Delete
        await prisma.routine.delete({
            where: { id: routine.id },
        });
        return { success: true };
    },
    /**
     * Supplemental fields are role, isUpvoted, and isStarred
     */
    async supplementalFields(
        userId: string | null | undefined, // Of the user making the request
        objects: RecursivePartial<any>[],
        info: InfoType, // GraphQL select info
    ): Promise<RecursivePartial<Routine>[]> {
        // Convert GraphQL info object to a partial select object
        const partial = infoToPartialSelect(info);
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
            console.log('routine supplemental fields', objects)
            // If owned by user, set role to owner if userId matches
            // If owned by organization, set role user's role in organization
            const organizationIds = objects
                .filter(x => x.owner?.__typename === 'Organization')
                .map(x => x.id)
                .filter(x => Boolean(x)) as string[];
            const roles = userId
                ? await OrganizationModel(prisma).getRoles(userId, organizationIds)
                : [];
            objects = objects.map((x) => {
                const orgRoleIndex = organizationIds.findIndex(id => id === x.id);
                if (orgRoleIndex >= 0) {
                    return { ...x, role: roles[orgRoleIndex] };
                }
                return { ...x, role: x.owner?.id === userId ? MemberRole.Owner : undefined };
            }) as any;
        }
        // Convert Prisma objects to GraphQL objects
        return objects.map(format.selectToGraphQL);
    },
    profanityCheck(data: RoutineCreateInput | RoutineUpdateInput): void {
        if (hasProfanity(data.title, data.description, data.instructions)) throw new CustomError(CODE.BannedWord);
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function RoutineModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Routine;
    const format = routineFormatter();
    const sort = sorter();

    return {
        prisma,
        model,
        ...format,
        ...sort,
        ...counter<RoutineCountInput>(model, prisma),
        ...routiner(format, sort, prisma),
    }
}
//==============================================================
/* #endregion Model */
//==============================================================