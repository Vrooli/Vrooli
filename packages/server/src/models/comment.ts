import { CommentSortBy } from "@shared/consts";
import { commentsCreate, commentsUpdate } from "@shared/validation";
import { Comment, CommentCreateInput, CommentFor, CommentPermission, CommentSearchInput, CommentSearchResult, CommentThread, CommentUpdateInput, SessionUser } from "../endpoints/types";
import { PrismaType } from "../types";
import { StarModel } from "./star";
import { VoteModel } from "./vote";
import { Trigger } from "../events";
import { Formatter, Searcher, Mutater, Validator, Displayer } from "./types";
import { Prisma } from "@prisma/client";
import { Request } from "express";
import { getSingleTypePermissions } from "../validators";
import { addSupplementalFields, combineQueries, modelToGraphQL, permissionsSelectHelper, selectHelper, toPartialGraphQLInfo } from "../builders";
import { bestLabel, oneIsPublic, SearchMap, translationRelationshipBuilder } from "../utils";
import { GraphQLInfo, PartialGraphQLInfo, SelectWrap } from "../builders/types";
import { getSearchStringQuery } from "../getters";
import { getUser } from "../auth";
import { SortMap } from "../utils/sortMap";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: CommentCreateInput,
    GqlUpdate: CommentUpdateInput,
    GqlModel: Comment,
    GqlPermission: CommentPermission,
    GqlSearch: CommentSearchInput,
    GqlSort: CommentSortBy,
    PrismaCreate: Prisma.commentUpsertArgs['create'],
    PrismaUpdate: Prisma.commentUpsertArgs['update'],
    PrismaModel: Prisma.commentGetPayload<SelectWrap<Prisma.commentSelect>>,
    PrismaSelect: Prisma.commentSelect,
    PrismaWhere: Prisma.commentWhereInput,
}

const __typename = 'Comment' as const;

const suppFields = ['isStarred', 'isUpvoted', 'permissionsComment'] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        owner: {
            ownedByUser: 'User',
            ownedByOrganization: 'Organization',
        },
        commentedOn: {
            apiVersion: 'ApiVersion',
            issue: 'Issue',
            noteVersion: 'NoteVersion',
            post: 'Post',
            projectVersion: 'ProjectVersion',
            pullRequest: 'PullRequest',
            question: 'Question',
            questionAnswer: 'QuestionAnswer',
            routineVersion: 'RoutineVersion',
            smartContractVersion: 'SmartContractVersion',
            standardVersion: 'StandardVersion',
        },
        reports: 'Report',
        starredBy: 'User',
    },
    joinMap: { starredBy: 'user' },
    countFields: ['reportsCount', 'translationsCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, prisma, userData }) => [
            ['isStarred', async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename)],
            ['isUpvoted', async () => await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename)],
            ['permissionsComment', async () => await getSingleTypePermissions(__typename, ids, prisma, userData)],
        ],
    },
})

const searcher = (): Searcher<Model> => ({
    defaultSort: CommentSortBy.ScoreDesc,
    searchFields: [
        'apiVersionId',
        'createdTimeFrame',
        'issueId',
        'minScore',
        'minStars',
        'noteVersionId',
        'ownedByOrganizationId',
        'ownedByUserId',
        'postId',
        'projectVersionId',
        'pullRequestId',
        'questionAnswerId',
        'questionId',
        'routineVersionId',
        'smartContractVersionId',
        'standardVersionId',
        'translationLanguages',
        'updatedTimeFrame',
    ],
    sortBy: CommentSortBy,
    searchStringQuery: () => ({ translations: 'transText' }),
})

const validator = (): Validator<Model> => ({
    validateMap: {
        __typename: 'Comment',
        ownedByUser: 'User',
        ownedByOrganization: 'Organization',
    },
    isTransferable: false,
    maxObjects: {
        User: {
            private: 0,
            public: 10000,
        },
        Organization: 0,
    },
    permissionsSelect: (...params) => ({
        id: true,
        ...permissionsSelectHelper([
            ['apiVersion', 'Api'],
            ['issue', 'Issue'],
            ['ownedByOrganization', 'Organization'],
            ['post', 'Post'],
            ['projectVersion', 'Project'],
            ['pullRequest', 'PullRequest'],
            ['question', 'Question'],
            ['questionAnswer', 'QuestionAnswer'],
            ['routineVersion', 'Routine'],
            ['smartContractVersion', 'SmartContract'],
            ['standardVersion', 'Standard'],
            ['ownedByUser', 'User'],
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
        Organization: data.ownedByOrganization,
        User: data.ownedByUser,
    }),
    isDeleted: () => false,
    isPublic: (data, languages) => oneIsPublic<Prisma.commentSelect>(data, [
        ['apiVersion', 'Api'],
        ['issue', 'Issue'],
        ['post', 'Post'],
        ['projectVersion', 'Project'],
        ['pullRequest', 'PullRequest'],
        ['question', 'Question'],
        ['questionAnswer', 'QuestionAnswer'],
        ['routineVersion', 'Routine'],
        ['smartContractVersion', 'SmartContract'],
        ['standardVersion', 'Standard'],
    ], languages),
    visibility: {
        private: {},
        public: {},
        owner: (userId) => ({ ownedByUser: { id: userId } }),
    }
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
        let partialInfo = toPartialGraphQLInfo(info, formatter().relationshipMap, userData?.languages ?? ['en'], true);
        const idQuery = (Array.isArray(input.ids)) ? ({ id: { in: input.ids } }) : undefined;
        // Combine queries
        const where = { ...idQuery };
        // Determine sort order
        // Make sure sort field is valid
        const orderByField = input.sortBy ?? searcher().defaultSort;
        const orderByIsValid = searcher().sortBy[orderByField] === undefined
        const orderBy = orderByIsValid ? SortMap[input.sortBy ?? searcher().defaultSort] : undefined;
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
        let partialInfo = toPartialGraphQLInfo(info, formatter().relationshipMap, req.languages, true);
        // Determine text search query
        const searchQuery = input.searchString ? getSearchStringQuery({ objectType: 'Comment', searchString: input.searchString }) : undefined;
        // Loop through search fields and add each to the search query, 
        // if the field is specified in the input
        const customQueries: { [x: string]: any }[] = [];
        for (const field of searcher().searchFields) {
            if (input[field as string] !== undefined) {
                customQueries.push(SearchMap[field as string](input, getUser(req), __typename));
            }
        }
        // Combine queries
        const where = combineQueries([searchQuery, ...customQueries]);
        // Determine sort order
        // Make sure sort field is valid
        const orderByField = input.sortBy ?? searcher().defaultSort;
        const orderByIsValid = searcher().sortBy[orderByField] === undefined
        const orderBy = orderByIsValid ? SortMap[input.sortBy ?? searcher().defaultSort] : undefined;
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
    ApiVersion: 'apiVersionId',
    Issue: 'issueId',
    NoteVersion: 'noteVersionId',
    Post: 'postId',
    ProjectVersion: 'projectVersionId',
    PullRequest: 'pullRequestId',
    Question: 'questionId',
    QuestionAnswer: 'questionAnswerId',
    RoutineVersion: 'routineVersionId',
    SmartContractVersion: 'smartContractVersionId',
    StandardVersion: 'standardVersionId',
}

const mutater = (): Mutater<Model> => ({
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
                Trigger(prisma, userData.languages).createComment(userData.id, c.id as string);
            }
        },
    },
    yup: { create: commentsCreate, update: commentsUpdate },
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, translations: { select: { language: true, text: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'text', languages),
})

export const CommentModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.comment,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    query: querier(),
    search: searcher(),
    validate: validator(),
})