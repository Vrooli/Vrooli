import { Prisma } from "@prisma/client";
import { Comment, CommentCreateInput, CommentSearchInput, CommentSearchResult, CommentSortBy, CommentThread, CommentUpdateInput, CommentYou, MaxObjects } from "@shared/consts";
import { lowercaseFirstLetter } from "@shared/utils";
import { commentValidation } from "@shared/validation";
import { Request } from "express";
import { getUser } from "../auth";
import { addSupplementalFields, combineQueries, modelToGql, selectHelper, toPartialGqlInfo } from "../builders";
import { GraphQLInfo, PartialGraphQLInfo, SelectWrap } from "../builders/types";
import { getSearchStringQuery } from "../getters";
import { PrismaType, SessionUserToken } from "../types";
import { bestLabel, defaultPermissions, onCommonPlain, oneIsPublic, SearchMap, translationShapeHelper } from "../utils";
import { SortMap } from "../utils/sortMap";
import { getSingleTypePermissions } from "../validators";
import { BookmarkModel } from "./bookmark";
import { ReactionModel } from "./reaction";
import { ModelLogic } from "./types";

const __typename = 'Comment' as const;
type Permissions = Pick<CommentYou, 'canDelete' | 'canUpdate' | 'canBookmark' | 'canReply' | 'canReport' | 'canReact'>;
const suppFields = ['you'] as const;
export const CommentModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: CommentCreateInput,
    GqlUpdate: CommentUpdateInput,
    GqlModel: Comment,
    GqlPermission: Permissions,
    GqlSearch: CommentSearchInput,
    GqlSort: CommentSortBy,
    PrismaCreate: Prisma.commentUpsertArgs['create'],
    PrismaUpdate: Prisma.commentUpsertArgs['update'],
    PrismaModel: Prisma.commentGetPayload<SelectWrap<Prisma.commentSelect>>,
    PrismaSelect: Prisma.commentSelect,
    PrismaWhere: Prisma.commentWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.comment,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, text: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'text', languages),
    },
    format: {
        gqlRelMap: {
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
            bookmarkedBy: 'User',
        },
        prismaRelMap: {
            __typename: 'Comment',
            ownedByUser: 'User',
            ownedByOrganization: 'Organization',
            apiVersion: 'ApiVersion',
            issue: 'Issue',
            noteVersion: 'NoteVersion',
            parent: 'Comment',
            post: 'Post',
            projectVersion: 'ProjectVersion',
            pullRequest: 'PullRequest',
            question: 'Question',
            questionAnswer: 'QuestionAnswer',
            routineVersion: 'RoutineVersion',
            smartContractVersion: 'SmartContractVersion',
            standardVersion: 'StandardVersion',
            reports: 'Report',
            bookmarkedBy: 'User',
            reactions: 'Reaction',
            parents: 'Comment',
        },
        joinMap: { bookmarkedBy: 'user' },
        countFields: {
            reportsCount: true,
            translationsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        reaction: await ReactionModel.query.getReactions(prisma, userData?.id, ids, __typename),
                    }
                }
            },
        },
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                ownedByUser: { connect: { id: rest.userData.id } },
                [lowercaseFirstLetter(data.createdFor)]: { connect: { id: data.forConnect } },
                ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, ...rest })),
            })
        },
        trigger: {
            onCommon: async (params) => {
                await onCommonPlain({
                    ...params,
                    objectType: __typename,
                    ownerOrganizationField: 'ownedByOrganization',
                    ownerUserField: 'ownedByUser',
                });
            },
        },
        yup: commentValidation,
    },
    query: {
        /**
         * Custom search query for querying comment threads
         */
        async searchThreads(
            prisma: PrismaType,
            userData: SessionUserToken | null,
            input: { ids: string[], take: number, sortBy: CommentSortBy },
            info: GraphQLInfo | PartialGraphQLInfo,
            nestLimit: number = 2,
        ): Promise<CommentThread[]> {
            // Partially convert info type
            let partialInfo = toPartialGqlInfo(info, CommentModel.format.gqlRelMap, userData?.languages ?? ['en'], true);
            const idQuery = (Array.isArray(input.ids)) ? ({ id: { in: input.ids } }) : undefined;
            // Combine queries
            const where = { ...idQuery };
            // Determine sort order
            // Make sure sort field is valid
            const orderByField = input.sortBy ?? CommentModel.search!.defaultSort;
            const orderByIsValid = CommentModel.search!.sortBy[orderByField] === undefined
            const orderBy = orderByIsValid ? SortMap[input.sortBy ?? CommentModel.search!.defaultSort] : undefined;
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
                    __typename: 'CommentThread' as const,
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
            let partialInfo = toPartialGqlInfo(info, CommentModel.format.gqlRelMap, req.languages, true);
            // Determine text search query
            const searchQuery = input.searchString ? getSearchStringQuery({ objectType: 'Comment', searchString: input.searchString }) : undefined;
            // Loop through search fields and add each to the search query, 
            // if the field is specified in the input
            const customQueries: { [x: string]: any }[] = [];
            for (const field of Object.keys(CommentModel.search!.searchFields)) {
                if (input[field as string] !== undefined) {
                    customQueries.push(SearchMap[field as string](input, getUser(req), __typename));
                }
            }
            // Combine queries
            const where = combineQueries([searchQuery, ...customQueries]);
            // Determine sort order
            // Make sure sort field is valid
            const orderByField = input.sortBy ?? CommentModel.search!.defaultSort;
            const orderByIsValid = CommentModel.search!.sortBy[orderByField] === undefined
            const orderBy = orderByIsValid ? SortMap[input.sortBy ?? CommentModel.search!.defaultSort] : undefined;
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
                __typename: 'CommentSearchResult' as const,
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
                sortBy: input.sortBy ?? CommentModel.search!.defaultSort,
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
            comments = comments.map((c: any) => modelToGql(c, partialInfo as PartialGraphQLInfo));
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
                        __typename: 'CommentThread' as const,
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
                __typename: 'CommentSearchResult' as const,
                totalThreads: totalInThread,
                threads,
                endCursor,
            }
        }
    },
    search: {
        defaultSort: CommentSortBy.ScoreDesc,
        searchFields: {
            apiVersionId: true,
            createdTimeFrame: true,
            issueId: true,
            minScore: true,
            minBookmarks: true,
            noteVersionId: true,
            ownedByOrganizationId: true,
            ownedByUserId: true,
            postId: true,
            projectVersionId: true,
            pullRequestId: true,
            questionAnswerId: true,
            questionId: true,
            routineVersionId: true,
            smartContractVersionId: true,
            standardVersionId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        sortBy: CommentSortBy,
        searchStringQuery: () => ({ translations: 'transText' }),
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            apiVersion: 'Api',
            issue: 'Issue',
            ownedByOrganization: 'Organization',
            ownedByUser: 'User',
            post: 'Post',
            projectVersion: 'Project',
            pullRequest: 'PullRequest',
            question: 'Question',
            questionAnswer: 'QuestionAnswer',
            routineVersion: 'Routine',
            smartContractVersion: 'SmartContract',
            standardVersion: 'Standard',
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isLoggedIn, isPublic }) => ({
            ...defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic }),
            canReply: () => isLoggedIn && (isAdmin || isPublic),
        }),
        owner: (data) => ({
            Organization: data.ownedByOrganization,
            User: data.ownedByUser,
        }),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.commentSelect>(data, [
            ['apiVersion', 'ApiVersion'],
            ['issue', 'Issue'],
            ['post', 'Post'],
            ['projectVersion', 'ProjectVersion'],
            ['pullRequest', 'PullRequest'],
            ['question', 'Question'],
            ['questionAnswer', 'QuestionAnswer'],
            ['routineVersion', 'RoutineVersion'],
            ['smartContractVersion', 'SmartContractVersion'],
            ['standardVersion', 'StandardVersion'],
        ], languages),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ ownedByUser: { id: userId } }),
        }
    },
})