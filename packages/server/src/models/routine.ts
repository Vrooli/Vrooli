import { DeleteOneInput, FindByIdInput, Routine, RoutineCountInput, RoutineCreateInput, RoutineUpdateInput, RoutineSearchInput, RoutineSortBy, Success } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { addCreatorField, addJoinTables, addOwnerField, counter, FormatConverter, InfoType, MODEL_TYPES, PaginatedSearchResult, removeCreatorField, removeJoinTables, removeOwnerField, searcher, selectHelper, Sortable } from "./base";
import { CustomError } from "../error";
import { CODE, MemberRole, routineCreate, routineUpdate } from "@local/shared";
import { hasProfanity } from "../utils/censor";
import { OrganizationModel } from "./organization";
import { ResourceModel } from "./resource";
import { routine } from "@prisma/client";
import { TagModel } from "./tag";
import { StarModel } from "./star";
import { VoteModel } from "./vote";

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Handles the authorized adding, updating, and deleting of routines.
 */
const routiner = (format: FormatConverter<Routine, routine>, sort: Sortable<RoutineSortBy>, prisma: PrismaType) => ({
    async find(
        userId: string | null | undefined, // Of the user making the request, not the routine
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Routine> | null> {
        // Create selector
        const select = selectHelper<Routine, routine>(info, formatter().toDB);
        // Access database
        let routine = await prisma.routine.findUnique({ where: { id: input.id }, ...select });
        // Return routine with "isUpvoted" and "isStarred" fields. These must be queried separately.
        if (!routine) throw new CustomError(CODE.InternalError, 'Routine not found');
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [format.toGraphQL(routine)], {});
        return formatted[0];
    },
    async search(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: RoutineSearchInput,
        info: InfoType = null,
    ): Promise<PaginatedSearchResult> {
        // Create where clauses
        const userIdQuery = input.userId ? { userId: input.userId } : undefined;
        const organizationIdQuery = input.organizationId ? { organizationId: input.organizationId } : undefined;
        const parentIdQuery = input.parentId ? { parentId: input.parentId } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        // Search
        const search = searcher<RoutineSortBy, RoutineSearchInput, Routine, routine>(MODEL_TYPES.Routine, format.toDB, format.toGraphQL, sort, prisma);
        let searchResults = await search.search({ ...userIdQuery, ...organizationIdQuery, ...parentIdQuery, ...reportIdQuery, ...where }, input, info);
        // Format and add supplemental/calculated fields to each result node
        console.log('routine searchResults', searchResults);
        let formattedNodes = searchResults.edges.map(({ node }) => node);
        console.log('routine formatted nodes', formattedNodes);
        formattedNodes = await this.supplementalFields(userId, formattedNodes, {});
        return { pageInfo: searchResults.pageInfo, edges: searchResults.edges.map(({ node, ...rest }) => ({ node: formattedNodes.shift(), ...rest })) };
    },
    async create(
        userId: string,
        input: RoutineCreateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Routine>> {
        // Check for valid arguments
        routineCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.title, input.description)) throw new CustomError(CODE.BannedWord);
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
        };
        // Associate with either organization or user
        if (input.createdByOrganizationId) {
            // Make sure the user is an admin of the organization
            const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.createdByOrganizationId);
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
        // Handle inputs TODO
        // Handle outputs TODO
        // Handle nodes TODO
        // Create routine
        const routine = await prisma.routine.create({
            data: routineData as any,
            ...selectHelper<Routine, routine>(info, format.toDB)
        })
        // Return project with "role", "isUpvoted" and "isStarred" fields. These will be their default values.
        return { ...format.toGraphQL(routine), role: MemberRole.Owner as any, isUpvoted: null, isStarred: false };
    },
    async update(
        userId: string,
        input: RoutineUpdateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Routine>> {
        // Check for valid arguments
        routineUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.title, input.description)) throw new CustomError(CODE.BannedWord);
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
            }, true),
            externalResources: ResourceModel(prisma).relationshipBuilder(userId, {
                resourcesCreate: input.resourcesExternalCreate,
                resourcesUpdate: input.resourcesExternalUpdate,
                resourcesDelete: input.resourcesExternalDelete,
            }, true),
            // Handle tags
            tags: await TagModel(prisma).relationshipBuilder(userId, input, true),
        };
        // Associate with either organization or user. This will remove the association with the other.
        if (input.organizationId) {
            // Make sure the user is an admin of the organization
            const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.organizationId);
            if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
            routineData = { ...routineData, organization: { connect: { id: input.organizationId } } };
        } else {
            routineData = { ...routineData, user: { connect: { id: userId } } };
        }
        // TODO inputs
        // TODO outputs
        // Handle nodes TODO
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
            ...selectHelper<Routine, routine>(info, format.toDB)
        });
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [format.toGraphQL(routine)], {});
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
            authorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, routine.organizationId);
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
        objects: RecursivePartial<Routine>[],
        known: { [x: string]: any[] }, // Known values (i.e. don't need to query), in same order as objects
    ): Promise<RecursivePartial<Routine>[]> {
        // If userId not provided, return the input with isStarred false, isUpvoted null, and role null
        if (!userId) return objects.map(x => ({ ...x, isStarred: false, isUpvoted: null, role: null }));
        // Get all of the ids
        const ids = objects.map(x => x.id) as string[];
        // Check if isStarred is provided
        if (known.isStarred) objects = objects.map((x, i) => ({ ...x, isStarred: known.isStarred[i] }));
        // Otherwise, query for isStarred
        else {
            const isStarredArray = await StarModel(prisma).getIsStarreds(userId, ids, 'routine');
            objects = objects.map((x, i) => ({ ...x, isStarred: isStarredArray[i] }));
        }
        // Check if isUpvoted is provided
        if (known.isUpvoted) objects = objects.map((x, i) => ({ ...x, isUpvoted: known.isUpvoted[i] }));
        // Otherwise, query for isStarred
        else {
            const isUpvotedArray = await VoteModel(prisma).getIsUpvoteds(userId, ids, 'routine');
            objects = objects.map((x, i) => ({ ...x, isUpvoted: isUpvotedArray[i] }));
        }
        // Check is role is provided
        if (known.role) objects = objects.map((x, i) => ({ ...x, role: known.role[i] }));
        // Otherwise, query for role
        else {
            console.log('routine supplemental fields', objects)
            // If owned by user, set role to owner if userId matches
            // If owned by organization, set role user's role in organization
            const organizationIds = objects
                .filter(x => x.owner?.__typename === 'Organization')
                .map(x => x.id)
                .filter(x => Boolean(x)) as string[];
            const roles = await OrganizationModel(prisma).getRoles(userId, organizationIds);
            objects = objects.map((x) => {
                const orgRoleIndex = organizationIds.findIndex(id => id === x.id);
                if (orgRoleIndex >= 0) {
                    return { ...x, role: roles[orgRoleIndex] };
                }
                return { ...x, role: x.owner?.id === userId ? MemberRole.Owner : undefined };
            }) as any;
        }
        return objects;
    },
})

/**
 * Component for formatting between graphql and prisma types
 */
const formatter = (): FormatConverter<Routine, routine> => {
    const joinMapper = {
        tags: 'tag',
        starredBy: 'user',
    };
    return {
        toDB: (obj: RecursivePartial<Routine>): RecursivePartial<routine> => {
            let modified = addJoinTables(obj, joinMapper);
            modified = removeCreatorField(modified);
            modified = removeOwnerField(modified);
            // Remove calculated fields
            delete modified.isUpvoted;
            delete modified.isStarred;
            delete modified.role
            return modified;
        },
        toGraphQL: (obj: RecursivePartial<routine>): RecursivePartial<Routine> => {
            let modified = removeJoinTables(obj, joinMapper);
            modified = addCreatorField(modified);
            modified = addOwnerField(modified);
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
            [RoutineSortBy.StarsAsc]: { starredBy: { _count: 'asc' } },
            [RoutineSortBy.StarsDesc]: { starredBy: { _count: 'desc' } },
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

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function RoutineModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Routine;
    const format = formatter();
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