import { CommentSortBy } from "@shared/consts";
import { commentsCreate, commentsUpdate } from "@shared/validation";
import { Comment, CommentCreateInput, CommentFor, CommentPermission, CommentSearchInput, CommentSearchResult, CommentThread, CommentUpdateInput, SessionUser } from "../schema/types";
import { PrismaType } from "../types";
import { addJoinTablesHelper, removeJoinTablesHelper, selectHelper, modelToGraphQL, toPartialGraphQLInfo, timeFrameToPrisma, addSupplementalFields, addCountFieldsHelper, removeCountFieldsHelper, combineQueries, permissionsSelectHelper, getUser, getSearchString } from "./builder";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { CustomError, Trigger } from "../events";
import { Formatter, Searcher, GraphQLInfo, PartialGraphQLInfo, GraphQLModelType, Mutater, Validator } from "./types";
import { Prisma } from "@prisma/client";
import { oneIsPublic, translationRelationshipBuilder } from "./utils";
import { Request } from "express";
import { getSingleTypePermissions } from "./validators";

const joinMapper = { starredBy: 'user' };
const countMapper = { reportsCount: 'reports' };
type SupplementalFields = 'isStarred' | 'isUpvoted' | 'permissionsComment';
const formatter = (): Formatter<Comment, SupplementalFields> => ({
    relationshipMap: {
        __typename: 'Comment',
        creator: {
            User: 'User',
            Organization: 'Organization',
        },
        commentedOn: {
            Project: 'Project',
            Routine: 'Routine',
            Standard: 'Standard',
        },
        reports: 'Report',
        starredBy: 'User',
    },
    addJoinTables: (partial) => addJoinTablesHelper(partial, joinMapper),
    removeJoinTables: (data) => removeJoinTablesHelper(data, joinMapper),
    addCountFields: (partial) => addCountFieldsHelper(partial, countMapper),
    removeCountFields: (data) => removeCountFieldsHelper(data, countMapper),
    supplemental: {
        graphqlFields: ['isStarred', 'isUpvoted', 'permissionsComment'],
        toGraphQL: ({ ids, prisma, userData }) => [
            ['isStarred', async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, 'Comment')],
            ['isUpvoted', async () => await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, 'Routine')],
            ['permissionsComment', async () => await getSingleTypePermissions('Comment', ids, prisma, userData)],
        ],
    },
})

const searcher = (): Searcher<
    CommentSearchInput,
    CommentSortBy,
    Prisma.commentOrderByWithRelationInput,
    Prisma.commentWhereInput
> => ({
    defaultSort: CommentSortBy.VotesDesc,
    sortMap: {
        DateCreatedAsc: { created_at: 'asc' },
        DateCreatedDesc: { created_at: 'desc' },
        DateUpdatedAsc: { updated_at: 'asc' },
        DateUpdatedDesc: { updated_at: 'desc' },
        StarsAsc: { stars: 'asc' },
        StarsDesc: { stars: 'desc' },
        VotesAsc: { votes: 'asc' },
        VotesDesc: { votes: 'desc' },
    },
    searchStringQuery: ({ insensitive, languages }) => ({
        translations: { some: { language: languages ? { in: languages } : undefined, text: insensitive } }
    }),
    customQueries(input) {
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

const validator = (): Validator<
    CommentCreateInput,
    CommentUpdateInput,
    Comment,
    Prisma.commentGetPayload<{ select: { [K in keyof Required<Prisma.commentSelect>]: true } }>,
    CommentPermission,
    Prisma.commentSelect,
    Prisma.commentWhereInput
> => ({
    validateMap: {
        __typename: 'Comment',
        user: 'User',
        organization: 'Organization',
    },
    permissionsSelect: (...params) => ({
        id: true,
        ...permissionsSelectHelper([
            // ['apiVersion', 'Api'],
            // ['issue', 'Issue'],
            ['organization', 'Organization'],
            // ['post', 'Post'],
            ['projectVersion', 'Project'],
            // ['pullRequest', 'PullRequest'],
            // ['question', 'Question'],
            // ['questionAnswer', 'QuestionAnswer'],
            ['routineVersion', 'Routine'],
            // ['smartContractVersion', 'SmartContract'],
            ['standardVersion', 'Standard'],
            ['user', 'User'],
        ], ...params)
    }),
    permissionResolvers: ({ isAdmin, isPublic }) => ([
        ['canDelete', async () => isAdmin],
        ['canEdit', async () => isAdmin],
        ['canReply', async () => isAdmin || isPublic],
        ['canReport', async () => !isAdmin && isPublic],
        ['canStar', async () => isAdmin || isPublic],
        ['canView', async () => isAdmin || isPublic],
        ['canVote', async () => isAdmin || isPublic],
    ]),
    owner: (data) => ({
        Organization: data.organization,
        User: data.user,
    }),
    isDeleted: () => false,
    isPublic: (data, languages) => oneIsPublic<Prisma.commentSelect>(data, [
        // ['apiVersion', 'Api'],
        // ['issue', 'Issue'],
        // ['post', 'Post'],
        ['projectVersion', 'Project'],
        // ['pullRequest', 'PullRequest'],
        // ['question', 'Question'],
        // ['questionAnswer', 'QuestionAnswer'],
        ['routineVersion', 'Routine'],
        // ['smartContractVersion', 'SmartContract'],
        ['standardVersion', 'Standard'],
    ], languages),
    ownerOrMemberWhere: (userId) => ({ user: { id: userId } }),
})

const querier = () => ({
    /**
     * Custom search query for querying comment threads
     */
    async searchThreads(
        prisma: PrismaType,
        userData: SessionUser | null,
        input: { ids: string[], take: number, sortBy: CommentSortBy },
        info: GraphQLInfo | PartialGraphQLInfo,
        nestLimit: number = 2,
    ): Promise<CommentThread[]> {
        // Partially convert info type
        let partialInfo = toPartialGraphQLInfo(info, formatter().relationshipMap);
        if (!partialInfo)
            throw new CustomError('0023', 'InternalError', userData?.languages ?? ['en']);
        // Create query for specified ids
        const idQuery = (Array.isArray(input.ids)) ? ({ id: { in: input.ids } }) : undefined;
        // Combine queries
        const where = { ...idQuery };
        // Determine sort order
        const orderBy = searcher().sortMap[input.sortBy ?? searcher().defaultSort];
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
            const childThreads = nestLimit > 0 ? await this.searchThreads(prisma, userData, {
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
        prisma: PrismaType,
        req: Request,
        input: CommentSearchInput,
        info: GraphQLInfo | PartialGraphQLInfo,
        nestLimit: number = 2,
    ): Promise<CommentSearchResult> {
        // Partially convert info type
        let partialInfo = toPartialGraphQLInfo(info, formatter().relationshipMap);
        if (!partialInfo)
            throw new CustomError('0322', 'InternalError', req.languages);
        // Determine text search query
        const searchQuery = input.searchString ? getSearchString({ objectType: 'Comment', searchString: input.searchString }) : undefined;
        // Determine createdTimeFrame query
        const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
        // Determine updatedTimeFrame query
        const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
        // Create type-specific queries
        let typeQuery = (searcher() as any).customQueries(input);
        // Combine queries
        const where = { ...searchQuery, ...createdQuery, ...updatedQuery, ...typeQuery };
        // Determine sort order
        const orderBy = searcher().sortMap[input.sortBy ?? searcher().defaultSort];
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
        const childThreads = nestLimit > 0 ? await this.searchThreads(prisma, getUser(req), {
            ids: searchResults.map(r => r.id),
            take: input.take ?? 10,
            sortBy: input.sortBy ?? searcher().defaultSort,
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
        comments = await addSupplementalFields(prisma, getUser(req), comments, partialInfo);
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

const forMapper: { [key in CommentFor]: string } = {
    Project: 'projectId',
    Routine: 'routineId',
    Standard: 'standardId',
}

/**
 * Handles authorized creates, updates, and deletes
 */
const mutater = (): Mutater<
    Comment,
    { graphql: CommentCreateInput, db: Prisma.commentUpsertArgs['create'] },
    { graphql: CommentUpdateInput, db: Prisma.commentUpsertArgs['update'] },
    false,
    false
> => ({
    shape: {
        create: async ({ data, prisma, userData }) => {
            return {
                id: data.id,
                translations: await translationRelationshipBuilder(prisma, userData, data, true),
                userId: userData.id,
                [forMapper[data.createdFor]]: data.forId,
                parentId: data.parentId ?? null,
            }
        },
        update: async ({ data, prisma, userData }) => {
            return {
                translations: await translationRelationshipBuilder(prisma, userData, data, false),
            }
        }
    },
    trigger: {
        onCreated: ({ created, prisma, userData }) => {
            for (const c of created) {
                Trigger(prisma, userData.languages).objectCreate('Comment', c.id as string, userData.id);
            }
        },
    },
    yup: { create: commentsCreate, update: commentsUpdate },
})

export const CommentModel = ({
    delegate: (prisma: PrismaType) => prisma.comment,
    format: formatter(),
    mutate: mutater(),
    query: querier(),
    search: searcher(),
    type: 'Comment' as GraphQLModelType,
    validate: validator(),
})