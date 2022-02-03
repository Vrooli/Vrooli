import { CODE, standardAdd, standardUpdate } from "@local/shared";
import { CustomError } from "../error";
import { PrismaType, RecursivePartial } from "types";
import { DeleteOneInput, FindByIdInput, Standard, StandardCountInput, StandardAddInput, StandardUpdateInput, StandardSearchInput, StandardSortBy, Success } from "../schema/types";
import { addCountQueries, addCreatorField, addJoinTables, counter, FormatConverter, InfoType, MODEL_TYPES, PaginatedSearchResult, removeCountQueries, removeCreatorField, removeJoinTables, searcher, selectHelper, Sortable } from "./base";
import { hasProfanity } from "../utils/censor";
import { OrganizationModel } from "./organization";
import { standard } from "@prisma/client";

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
const formatter = (): FormatConverter<Standard, standard> => {
    const joinMapper = {
        tags: 'tag',
        starredBy: 'user',
    };
    const countMapper = {
        stars: 'starredBy',
    }
    return {
        toDB: (obj: RecursivePartial<Standard>): RecursivePartial<standard> => {
            let modified = addJoinTables(obj, joinMapper);
            modified = addCountQueries(modified, countMapper);
            modified = removeCreatorField(modified);
            // Remove isUpvoted and isStarred, as these are calculated in their own queries
            if (modified.isUpvoted) delete modified.isUpvoted;
            if (modified.isStarred) delete modified.isStarred;
            return modified;
        },
        toGraphQL: (obj: RecursivePartial<standard>): RecursivePartial<Standard> => {
            let modified = removeJoinTables(obj, joinMapper);
            modified = removeCountQueries(modified, countMapper);
            modified = addCreatorField(modified);
            return modified;
        },
    }
}

/**
 * Component for search filters
 */
const sorter = (): Sortable<StandardSortBy> => ({
    defaultSort: StandardSortBy.AlphabeticalDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [StandardSortBy.AlphabeticalAsc]: { name: 'asc' },
            [StandardSortBy.AlphabeticalDesc]: { name: 'desc' },
            [StandardSortBy.CommentsAsc]: { comments: { _count: 'asc' } },
            [StandardSortBy.CommentsDesc]: { comments: { _count: 'desc' } },
            [StandardSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [StandardSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [StandardSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [StandardSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [StandardSortBy.StarsAsc]: { starredBy: { _count: 'asc' } },
            [StandardSortBy.StarsDesc]: { starredBy: { _count: 'desc' } },
            [StandardSortBy.VotesAsc]: { score: 'asc' },
            [StandardSortBy.VotesDesc]: { score: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { name: { ...insensitive } },
                { description: { ...insensitive } },
                { tags: { some: { tag: { tag: { ...insensitive } } } } },
            ]
        })
    }
})

/**
 * Handles the authorized adding, updating, and deleting of standards.
 */
const standarder = (format: FormatConverter<Standard, standard>, sort: Sortable<StandardSortBy>, prisma: PrismaType) => ({
    async findStandard(
        userId: string | null | undefined, // Of the user making the request, not the standard
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Standard> | null> {
        // Create selector
        const select = selectHelper<Standard, standard>(info, formatter().toDB);
        // Access database
        let standard = await prisma.standard.findUnique({ where: { id: input.id }, ...select });
        // Return standard with "isUpvoted" and "isStarred" fields. These must be queried separately.
        if (!standard) throw new CustomError(CODE.InternalError, 'Standard not found');
        if (!userId) return { ...format.toGraphQL(standard), isUpvoted: false, isStarred: false };
        const vote = await prisma.vote.findFirst({ where: { userId, standardId: standard.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        const star = await prisma.star.findFirst({ where: { byId: userId, standardId: standard.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...format.toGraphQL(standard), isUpvoted, isStarred };
    },
    async searchStandards(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: StandardSearchInput,
        info: InfoType = null,
    ): Promise<PaginatedSearchResult> {
        // Create where clauses
        const userIdQuery = input.userId ? { userId: input.userId } : {};
        const organizationIdQuery = input.organizationId ? { organizationId: input.organizationId } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const routineIdQuery = input.routineId ? {
            OR: [
                { routineInputs: { some: { routineId: input.routineId } } },
                { routineOutputs: { some: { routineId: input.routineId } } },
            ]
        } : {};
        // Search
        const search = searcher<StandardSortBy, StandardSearchInput, Standard, standard>(MODEL_TYPES.Standard, format.toDB, format.toGraphQL, sort, prisma);
        let searchResults = await search.search({ ...userIdQuery, ...organizationIdQuery, ...reportIdQuery, ...routineIdQuery, ...where }, input, info);
        // Compute "isUpvoted" and "isStarred" field for each standard
        // If userId not provided, then "isUpvoted" is null and "isStarred" is false
        if (!userId) {
            searchResults.edges = searchResults.edges.map(({ cursor, node }) => ({ cursor, node: { ...node, isUpvoted: null, isStarred: null } }));
            return searchResults;
        }
        // Otherwise, query votes for all search results in one query
        const resultIds = searchResults.edges.map(({ node }) => node.id).filter(id => Boolean(id));
        const isUpvotedArray = await prisma.vote.findMany({ where: { userId, standardId: { in: resultIds } } });
        const isStarredArray = await prisma.star.findMany({ where: { byId: userId, standardId: { in: resultIds } } });
        console.log('isUpvotedArray', isUpvotedArray)
        searchResults.edges = searchResults.edges.map(({ cursor, node }) => {
            console.log('ids', node.id, isUpvotedArray.map(({ standardId }) => standardId));
            const isUpvoted = isUpvotedArray.find(({ standardId }) => standardId === node.id)?.isUpvote ?? null;
            const isStarred = Boolean(isStarredArray.find(({ standardId }) => standardId === node.id));
            return { cursor, node: { ...node, isUpvoted, isStarred } };
        });
        return searchResults;
    },
    async addStandard(
        userId: string,
        input: StandardAddInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Standard>> {
        // Check for valid arguments
        standardAdd.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.name, input.description)) throw new CustomError(CODE.BannedWord);
        // Create standard data
        let standardData: { [x: string]: any } = {
            name: input.name,
            description: input.description,
            default: input.default,
            isFile: input.isFile,
            schema: input.schema,
            type: input.type
        };
        // Associate with either organization or user
        if (input.createdByOrganizationId) {
            // Make sure the user is an admin of the organization
            const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, input.createdByOrganizationId);
            if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
            standardData = { 
                ...standardData, 
                createdByOrganization: { connect: { id: input.createdByOrganizationId } },
            };
        } else {
            standardData = { 
                ...standardData, 
                createdByUser: { connect: { id: userId } },
            };
        }
        // Handle tags TODO
        // Create standard
        const standard = await prisma.standard.create({
            data: standardData as any,
            ...selectHelper<Standard, standard>(info, format.toDB)
        })
        // Return standard with "isUpvoted" and "isStarred" fields. These will be their default values.
        return { ...format.toGraphQL(standard), isUpvoted: null, isStarred: false };
    },
    async updateStandard(
        userId: string,
        input: StandardUpdateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Standard>> {
        // Check for valid arguments
        standardUpdate.validateSync(input, { abortEarly: false });
        if (!input.id) throw new CustomError(CODE.InternalError, 'No standard id provided');
        // Create standard data
        let standardData: { [x: string]: any } = { description: input.description };
        // Check if authorized to update
        let standard = await prisma.standard.findUnique({
            where: { id: input.id },
            select: {
                id: true,
                createdByUserId: true,
                createdByOrganizationId: true,
            }
        });
        if (!standard) throw new CustomError(CODE.NotFound, 'Standard not found');
        if (standard.createdByUserId && standard.createdByUserId !== userId) throw new CustomError(CODE.Unauthorized);
        if (standard.createdByOrganizationId) {
            const isAuthorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, standard.createdByOrganizationId);
            if (!isAuthorized) throw new CustomError(CODE.Unauthorized);
        }
        // Handle tags TODO
        // Update standard
        standard = await prisma.standard.update({
            where: { id: standard.id },
            data: standardData as any,
            ...selectHelper<Standard, standard>(info, format.toDB)
        });
        // Return standard with "isUpvoted" and "isStarred" fields. These must be queried separately.
        const vote = await prisma.vote.findFirst({ where: { userId, standardId: standard.id } });
        const isUpvoted = vote?.isUpvote ?? null; // Null means no vote, false means downvote, true means upvote
        const star = await prisma.star.findFirst({ where: { byId: userId, standardId: standard.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...standard, isUpvoted, isStarred };
    },
    async deleteStandard(userId: string, input: DeleteOneInput): Promise<Success> {
        // Find
        const standard = await prisma.standard.findFirst({
            where: { id: input.id },
            select: {
                id: true,
                createdByUserId: true,
                createdByOrganizationId: true,
            }
        })
        if (!standard) throw new CustomError(CODE.NotFound, "Standard not found");
        // Check if user is authorized
        let authorized = userId === standard.createdByUserId;
        if (!authorized && standard.createdByOrganizationId) {
            authorized = await OrganizationModel(prisma).isOwnerOrAdmin(userId, standard.createdByOrganizationId);
        }
        if (!authorized) throw new CustomError(CODE.Unauthorized);
        // Delete
        await prisma.standard.delete({
            where: { id: standard.id },
        });
        return { success: true };
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function StandardModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Standard;
    const format = formatter();
    const sort = sorter();

    return {
        prisma,
        model,
        ...format,
        ...sort,
        ...counter<StandardCountInput>(model, prisma),
        ...standarder(format, sort, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================