import { CODE, CommentSortBy } from "@shared/consts";
import { commentsCreate, commentsUpdate, commentTranslationCreate, commentTranslationUpdate } from "@shared/validation";
import { omit } from '@shared/utils';
import { Comment, CommentCreateInput, CommentFor, CommentPermission, CommentSearchInput, CommentSearchResult, CommentThread, CommentUpdateInput } from "../schema/types";
import { PrismaType, RecursivePartial, ReqForUserAuth } from "../types";
import { addJoinTablesHelper, removeJoinTablesHelper, selectHelper, modelToGraphQL, toPartialGraphQLInfo, timeFrameToPrisma, addSupplementalFields, addCountFieldsHelper, removeCountFieldsHelper, addSupplementalFieldsHelper, getSearchStringQueryHelper, onlyValidIds, combineQueries, getUserId } from "./builder";
import { TranslationModel } from "./translation";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { organizationQuerier } from "./organization";
import { projectPermissioner } from "./project";
import { routinePermissioner } from "./routine";
import { standardPermissioner } from "./standard";
import { CustomError, genErrorCode, Trigger } from "../events";
import { FormatConverter, Searcher, Permissioner, Querier, GraphQLInfo, PartialGraphQLInfo, CUDInput, CUDResult, GraphQLModelType, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { cudHelper } from "./actions";

const joinMapper = { starredBy: 'user' };
const countMapper = { reportsCount: 'reports' };
const supplementalFields = ['isStarred', 'isUpvoted', 'permissionsComment'];
export const commentFormatter = (): FormatConverter<Comment, CommentPermission> => ({
    relationshipMap: {
        '__typename': 'Comment',
        'creator': {
            'User': 'User',
            'Organization': 'Organization',
        },
        'commentedOn': {
            'Project': 'Project',
            'Routine': 'Routine',
            'Standard': 'Standard',
        },
        'reports': 'Report',
        'starredBy': 'User',
    },
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
    addCountFields: (partial) => addCountFieldsHelper(partial, countMapper),
    removeCountFields: (data) => removeCountFieldsHelper(data, countMapper),
    removeSupplementalFields: (partial) => omit(partial, supplementalFields),
    async addSupplementalFields({ objects, partial, permissions, prisma, userId }): Promise<RecursivePartial<Comment>[]> {
        return addSupplementalFieldsHelper({
            objects,
            partial,
            resolvers: [
                ['isStarred', async (ids) => await StarModel.query(prisma).getIsStarreds(userId, ids, 'Comment')],
                ['isUpvoted', async (ids) => await VoteModel.query(prisma).getIsUpvoteds(userId, ids, 'Routine')],
                ['permissionsComment', async () => await CommentModel.permissions().get({ objects, permissions, prisma, userId })],
            ]
        });
    },
})

export const commentSearcher = (): Searcher<CommentSearchInput> => ({
    defaultSort: CommentSortBy.VotesDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [CommentSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [CommentSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [CommentSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [CommentSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [CommentSortBy.StarsAsc]: { stars: 'asc' },
            [CommentSortBy.StarsDesc]: { stars: 'desc' },
            [CommentSortBy.VotesAsc]: { score: 'asc' },
            [CommentSortBy.VotesDesc]: { score: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        return getSearchStringQueryHelper({
            searchString,
            resolver: ({ insensitive }) => ({
                translations: { some: { language: languages ? { in: languages } : undefined, text: { ...insensitive } } }
            })
        })
    },
    customQueries(input: CommentSearchInput): { [x: string]: any } {
        return combineQueries([
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            (input.minScore !== undefined ? { score: { gte: input.minScore } } : {}),
            (input.minStars !== undefined ? { stars: { gte: input.minStars } } : {}),
            (input.userId !== undefined ? { userId: input.userId } : {}),
            (input.organizationId !== undefined ? { organizationId: input.organizationId } : {}),
            (input.projectId !== undefined ? { projectId: input.projectId } : {}),
            (input.routineId !== undefined ? { routineId: input.routineId } : {}),
            (input.standardId !== undefined ? { standardId: input.standardId } : {}),
        ])
    },
})

export const commentPermissioner = (): Permissioner<CommentPermission, CommentSearchInput> => ({
    async get({
        objects,
        permissions,
        prisma,
        userId,
    }) {
        // Initialize result with default permissions
        const result: (CommentPermission & { id?: string })[] = objects.map((o) => ({
            id: o.id,
            canDelete: false, // own
            canEdit: false, // own 
            canReply: true, // own or associated project, routine, or standard is public
            canReport: true, // !own and associated project, routine, or standard is public
            canStar: false, // own or associated project, routine, or standard is public
            canView: false, // own or associated project, routine, or standard is public
            canVote: false, // own or associated project, routine, or standard is public
        }));
        // Check ownership
        if (userId) {
            // Query for objects owned by user, or an organization they have an admin role in
            const owned = await prisma.comment.findMany({
                where: {
                    id: { in: onlyValidIds(objects.map(o => o.id)) },
                    ...this.ownershipQuery(userId),
                },
                select: { id: true },
            })
            // Set permissions for owned objects
            owned.forEach((o) => {
                const index = objects.findIndex((r) => r.id === o.id);
                result[index] = {
                    ...result[index],
                    canDelete: true,
                    canEdit: true,
                    canReply: true,
                    canReport: false,
                    canStar: true,
                    canView: true,
                    canVote: true,
                }
            });
        }
        // Query all public comments, or comments where user owns associated project, routine, or standard
        const all = await prisma.comment.findMany({
            where: {
                id: { in: onlyValidIds(objects.map(o => o.id)) },
                AND: [
                    // projectId is null, project is public, or user owns project
                    {
                        OR: [
                            { projectId: null },
                            { project: { isPrivate: false } },
                            { project: projectPermissioner().ownershipQuery(userId ?? '') },
                        ]
                    },
                    // routineId is null, routine is public, or user owns routine
                    {
                        OR: [
                            { routineId: null },
                            { routine: { isPrivate: false } },
                            { routine: routinePermissioner().ownershipQuery(userId ?? '') },
                        ]
                    },
                    // standardId is null, standard is public, or user owns standard
                    {
                        OR: [
                            { standardId: null },
                            { standard: { isPrivate: false } },
                            { standard: standardPermissioner().ownershipQuery(userId ?? '') },
                        ]
                    },
                ]
            },
            select: { id: true },
        })
        // Set permissions for all objects
        all.forEach((o) => {
            const index = objects.findIndex((r) => r.id === o.id);
            result[index] = {
                ...result[index],
                canReply: true,
                canReport: result[index].canReport === false ? false : true,
                canStar: true,
                canView: true,
                canVote: true,
            }
        });
        // Return result with IDs removed
        result.forEach((r) => delete r.id);
        return result as CommentPermission[];
    },
    ownershipQuery: (userId) => ({
        OR: [
            organizationQuerier().hasRoleInOrganizationQuery(userId),
            { user: { id: userId } },
        ]
    }),
})

export const commentQuerier = (prisma: PrismaType): Querier => ({
    /**
     * Custom search query for querying comment threads
     */
    async searchThreads(
        userId: string | null,
        input: { ids: string[], take: number, sortBy: CommentSortBy },
        info: GraphQLInfo | PartialGraphQLInfo,
        nestLimit: number = 2,
    ): Promise<CommentThread[]> {
        // Partially convert info type
        let partialInfo = toPartialGraphQLInfo(info, commentFormatter().relationshipMap);
        if (!partialInfo)
            throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0023') });
        // Create query for specified ids
        const idQuery = (Array.isArray(input.ids)) ? ({ id: { in: input.ids } }) : undefined;
        // Combine queries
        const where = { ...idQuery };
        // Determine sort order
        const orderBy = (commentSearcher() as any).getSortQuery(input.sortBy ?? commentSearcher().defaultSort);
        // Find requested search array
        const searchResults = await prisma.comment.findMany({
            where,
            orderBy,
            take: input.take ?? 10,
            ...selectHelper(partialInfo)
        });
        // If there are no results
        if (searchResults.length === 0) return [];
        // Initialize result 
        const threads: CommentThread[] = [];
        // For each result
        for (const result of searchResults) {
            // Find total in thread
            const totalInThread = await prisma.comment.count({
                where: {
                    ...where,
                    parentId: result.id,
                }
            });
            // Query for nested threads
            const nestedThreads = nestLimit > 0 ? await prisma.comment.findMany({
                where: {
                    ...where,
                    parentId: result.id,
                },
                take: input.take ?? 10,
                ...selectHelper(partialInfo)
            }) : [];
            // Find end cursor of nested threads
            const endCursor = nestedThreads.length > 0 ? nestedThreads[nestedThreads.length - 1].id : undefined;
            // For nested threads, recursively call this function
            const childThreads = nestLimit > 0 ? await this.searchThreads(userId, {
                ids: nestedThreads.map(n => n.id),
                take: input.take ?? 10,
                sortBy: input.sortBy
            }, info, nestLimit - 1) : [];
            // Add thread to result
            threads.push({
                childThreads,
                comment: result as any,
                endCursor,
                totalInThread,
            });
        }
        // Return result
        return threads;
    },
    /**
     * Custom search query for comments. Searches n top-level comments 
     * (i.e. no parentId), n second-level comments (i.e. parentId equal to 
     * one of the top-level comments), and n third-level comments (i.e. 
     * parentId equal to one of the second-level comments).
     */
    async searchNested(
        req: ReqForUserAuth,
        input: CommentSearchInput,
        info: GraphQLInfo | PartialGraphQLInfo,
        nestLimit: number = 2,
    ): Promise<CommentSearchResult> {
        // Partially convert info type
        let partialInfo = toPartialGraphQLInfo(info, commentFormatter().relationshipMap);
        if (!partialInfo)
            throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0023') });
        // Determine text search query
        const searchQuery = input.searchString ? (commentSearcher() as any).getSearchStringQuery(input.searchString) : undefined;
        // Determine createdTimeFrame query
        const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
        // Determine updatedTimeFrame query
        const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
        // Create type-specific queries
        let typeQuery = (commentSearcher() as any).customQueries(input);
        // Combine queries
        const where = { ...searchQuery, ...createdQuery, ...updatedQuery, ...typeQuery };
        // Determine sort order
        const orderBy = (commentSearcher() as any).getSortQuery(input.sortBy ?? commentSearcher().defaultSort);
        // Find requested search array
        const searchResults = await prisma.comment.findMany({
            where,
            orderBy,
            take: input.take ?? 10,
            skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
            cursor: input.after ? {
                id: input.after
            } : undefined,
            ...selectHelper(partialInfo)
        });
        // If there are no results
        if (searchResults.length === 0) return {
            totalThreads: 0,
            threads: [],
        }
        // Query total in thread, if cursor is not provided (since this means this data was already given to the user earlier)
        const totalInThread = input.after ? undefined : await prisma.comment.count({
            where: { ...where }
        });
        // Calculate end cursor
        const endCursor = searchResults[searchResults.length - 1].id;
        // If not as nestLimit, recurse with all result IDs
        const childThreads = nestLimit > 0 ? await this.searchThreads(getUserId(req), {
            ids: searchResults.map(r => r.id),
            take: input.take ?? 10,
            sortBy: input.sortBy ?? commentSearcher().defaultSort,
        }, info, nestLimit) : [];
        // Find every comment in "childThreads", and put into 1D array. This uses a helper function to handle recursion
        const flattenThreads = (threads: CommentThread[]) => {
            const result: Comment[] = [];
            for (const thread of threads) {
                result.push(thread.comment);
                result.push(...flattenThreads(thread.childThreads));
            }
            return result;
        }
        let comments: any = flattenThreads(childThreads);
        // Shape comments and add supplemental fields
        comments = comments.map((c: any) => modelToGraphQL(c, partialInfo as PartialGraphQLInfo));
        comments = await addSupplementalFields(prisma, getUserId(req), comments, partialInfo);
        // Put comments back into "threads" object, using another helper function. 
        // Comments can be matched by their ID
        const shapeThreads = (threads: CommentThread[]) => {
            const result: CommentThread[] = [];
            for (const thread of threads) {
                // Find current-level comment
                const comment = comments.find((c: any) => c.id === thread.comment.id);
                // Recurse
                const children = shapeThreads(thread.childThreads);
                // Add thread to result
                result.push({
                    comment,
                    childThreads: children,
                    endCursor: thread.endCursor,
                    totalInThread: thread.totalInThread,
                });
            }
            return result;
        }
        const threads = shapeThreads(childThreads);
        // Return result
        return {
            totalThreads: totalInThread,
            threads,
            endCursor,
        }
    }
})

const forMapper = {
    [CommentFor.Project]: 'projectId',
    [CommentFor.Routine]: 'routineId',
    [CommentFor.Standard]: 'standardId',
}

/**
 * Handles authorized creates, updates, and deletes
 */
export const commentMutater = (prisma: PrismaType): Mutater<Comment> => ({
    shapeCreate(userId: string, data: CommentCreateInput): Prisma.commentUpsertArgs['create'] {
        return {
            id: data.id,
            translations: TranslationModel.relationshipBuilder(userId, data, { create: commentTranslationCreate, update: commentTranslationUpdate }, true),
            userId,
            [forMapper[data.createdFor]]: data.forId,
            parentId: data.parentId ?? null,
        }
    },
    shapeUpdate(userId: string, data: CommentUpdateInput): Prisma.commentUpsertArgs['update'] {
        return {
            translations: TranslationModel.relationshipBuilder(userId, data, { create: commentTranslationCreate, update: commentTranslationUpdate }, false),
        }
    },
    cud(params: CUDInput<CommentCreateInput, CommentUpdateInput>): Promise<CUDResult<Comment>> {
        return cudHelper({
            ...params,
            objectType: 'Comment',
            prisma,
            yup: { yupCreate: commentsCreate, yupUpdate: commentsUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
            onCreated: (created) => {
                for (const c of created) {
                    Trigger(prisma).objectCreate('Comment', c.id as string, params.userId);
                }
            },
        })
    },
})

export const CommentModel = ({
    prismaObject: (prisma: PrismaType) => prisma.comment,
    format: commentFormatter(),
    mutate: commentMutater,
    permissions: commentPermissioner,
    query: commentQuerier,
    search: commentSearcher(),
    type: 'Comment' as GraphQLModelType,
})