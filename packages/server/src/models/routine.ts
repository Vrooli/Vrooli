import { DeleteOneInput, FindByIdInput, Routine, RoutineCountInput, RoutineAddInput, RoutineUpdateInput, RoutineSearchInput, RoutineSortBy, Success } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { addCreatorField, addJoinTables, addOwnerField, counter, FormatConverter, InfoType, MODEL_TYPES, PaginatedSearchResult, removeCreatorField, removeJoinTables, removeOwnerField, searcher, selectHelper, Sortable } from "./base";
import { CustomError } from "../error";
import { CODE, routineAdd, routineUpdate } from "@local/shared";
import { hasProfanity } from "../utils/censor";
import { OrganizationModel } from "./organization";
import { ResourceModel } from "./resource";
import { routine } from "@prisma/client";
import { TagModel } from "./tag";

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Handles the authorized adding, updating, and deleting of routines.
 */
const routiner = (format: FormatConverter<Routine, routine>, sort: Sortable<RoutineSortBy>, prisma: PrismaType) => ({
    async findRoutine(
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
        if (!userId) return { ...format.toGraphQL(routine), isUpvoted: false, isStarred: false };
        const vote = await prisma.vote.findFirst({ where: { userId, routineId: routine.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        const star = await prisma.star.findFirst({ where: { byId: userId, routineId: routine.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...format.toGraphQL(routine), isUpvoted, isStarred };
    },
    async searchRoutines(
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
        // Compute "isUpvoted" and "isStarred" field for each routine
        // If userId not provided, then "isUpvoted" is null and "isStarred" is false
        if (!userId) {
            searchResults.edges = searchResults.edges.map(({ cursor, node }) => ({ cursor, node: { ...node, isUpvoted: null, isStarred: false } }));
            return searchResults;
        }
        // Otherwise, query votes for all search results in one query
        const resultIds = searchResults.edges.map(({ node }) => node.id).filter(id => Boolean(id));
        const isUpvotedArray = await prisma.vote.findMany({ where: { userId, routineId: { in: resultIds } } });
        const isStarredArray = await prisma.star.findMany({ where: { byId: userId, routineId: { in: resultIds } } });
        console.log('isUpvotedArray', isUpvotedArray)
        console.log('isStarredArray', isStarredArray)
        searchResults.edges = searchResults.edges.map(({ cursor, node }) => {
            console.log('ids', node.id, isUpvotedArray.map(({ routineId }) => routineId));
            const isUpvoted = isUpvotedArray.find(({ routineId }) => routineId === node.id)?.isUpvote ?? null;
            const isStarred = Boolean(isStarredArray.find(({ routineId }) => routineId === node.id));
            return { cursor, node: { ...node, isUpvoted, isStarred } };
        });
        return searchResults;
    },
    async addRoutine(
        userId: string,
        input: RoutineAddInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Routine>> {
        // Check for valid arguments
        routineAdd.validateSync(input, { abortEarly: false });
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
            contextualResources: ResourceModel(prisma).relationshipBuilder(userId, { resourcesAdd: input.resourcesContextualAdd }, true),
            externalResources: ResourceModel(prisma).relationshipBuilder(userId, { resourcesAdd: input.resourcesExternalAdd }, true),
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
        // Return routine with "isUpvoted" and "isStarred" fields. These will be their default values.
        return { ...format.toGraphQL(routine), isUpvoted: null, isStarred: false };
    },
    async updateRoutine(
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
                resourcesAdd: input.resourcesContextualAdd,
                resourcesUpdate: input.resourcesContextualUpdate,
                resourcesDelete: input.resourcesContextualDelete, 
            }, true),
            externalResources: ResourceModel(prisma).relationshipBuilder(userId, { 
                resourcesAdd: input.resourcesExternalAdd,
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
        // Return routine with "isUpvoted" and "isStarred" field. These must be queried separately.
        const vote = await prisma.vote.findFirst({ where: { userId, routineId: routine.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        const star = await prisma.star.findFirst({ where: { byId: userId, routineId: routine.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...format.toGraphQL(routine), isUpvoted, isStarred };
    },
    async deleteRoutine(userId: string, input: DeleteOneInput): Promise<Success> {
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
    }
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
            // Remove isUpvoted and isStarred, as they are calculated in their own queries
            if (modified.isUpvoted) delete modified.isUpvoted;
            if (modified.isStarred) delete modified.isStarred;
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
    defaultSort: RoutineSortBy.AlphabeticalDesc,
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