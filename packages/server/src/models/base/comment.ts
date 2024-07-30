import { Comment, CommentSearchInput, CommentSearchResult, CommentSortBy, CommentThread, commentValidation, getTranslation, lowercaseFirstLetter, MaxObjects, VisibilityType } from "@local/shared";
import { Request } from "express";
import { ModelMap } from ".";
import { getUser } from "../../auth/request";
import { addSupplementalFields } from "../../builders/addSupplementalFields";
import { combineQueries } from "../../builders/combineQueries";
import { modelToGql } from "../../builders/modelToGql";
import { selectHelper } from "../../builders/selectHelper";
import { toPartialGqlInfo } from "../../builders/toPartialGqlInfo";
import { GraphQLInfo, PartialGraphQLInfo } from "../../builders/types";
import { visibilityBuilderPrisma } from "../../builders/visibilityBuilder";
import { prismaInstance } from "../../db/instance";
import { getSearchStringQuery } from "../../getters";
import { SessionUserToken } from "../../types";
import { defaultPermissions, oneIsPublic, SearchMap } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { SortMap } from "../../utils/sortMap";
import { afterMutationsPlain } from "../../utils/triggers";
import { getSingleTypePermissions } from "../../validators";
import { CommentFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { BookmarkModelLogic, CommentModelInfo, CommentModelLogic, ReactionModelLogic } from "./types";

const __typename = "Comment" as const;
export const CommentModel: CommentModelLogic = ({
    __typename,
    dbTable: "comment",
    dbTranslationTable: "comment_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, text: true } } }),
            get: (select, languages) => getTranslation(select, languages).text ?? "",
        },
    }),
    format: CommentFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                ownedByUser: { connect: { id: rest.userData.id } },
                [lowercaseFirstLetter(data.createdFor)]: { connect: { id: data.forConnect } },
                translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest }),
            }),
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsPlain({
                    ...params,
                    objectType: __typename,
                    ownerTeamField: "ownedByTeam",
                    ownerUserField: "ownedByUser",
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
            userData: SessionUserToken | null,
            input: { ids: string[], take: number, sortBy: CommentSortBy },
            info: GraphQLInfo | PartialGraphQLInfo,
            nestLimit = 2,
        ): Promise<CommentThread[]> {
            // Partially convert info type
            const partialInfo = toPartialGqlInfo(info, ModelMap.get<CommentModelLogic>("Comment").format.gqlRelMap, userData?.languages ?? ["en"], true);
            const idQuery = (Array.isArray(input.ids)) ? ({ id: { in: input.ids } }) : undefined;
            // Combine queries
            const where = { ...idQuery };
            // Determine sort order
            // Make sure sort field is valid
            const orderByField = input.sortBy ?? ModelMap.get<CommentModelLogic>("Comment").search.defaultSort;
            const orderByIsValid = ModelMap.get<CommentModelLogic>("Comment").search.sortBy[orderByField] === undefined;
            const orderBy = orderByIsValid ? SortMap[input.sortBy ?? ModelMap.get<CommentModelLogic>("Comment").search.defaultSort] : undefined;
            // Find requested search array
            const searchResults = await prismaInstance.comment.findMany({
                where,
                orderBy,
                take: input.take ?? 10,
                ...selectHelper(partialInfo),
            });
            // If there are no results
            if (searchResults.length === 0) return [];
            // Initialize result 
            const threads: CommentThread[] = [];
            // For each result
            for (const result of searchResults) {
                // Find total in thread
                const totalInThread = await prismaInstance.comment.count({
                    where: {
                        ...where,
                        parentId: result.id,
                    },
                });
                // Query for nested threads
                const nestedThreads = nestLimit > 0 ? await prismaInstance.comment.findMany({
                    where: {
                        ...where,
                        parentId: result.id,
                    },
                    take: input.take ?? 10,
                    ...selectHelper(partialInfo),
                }) : [];
                // Find end cursor of nested threads
                const endCursor = nestedThreads.length > 0 ? nestedThreads[nestedThreads.length - 1].id : undefined;
                // For nested threads, recursively call this function
                const childThreads = nestLimit > 0 ? await this.searchThreads(userData, {
                    ids: nestedThreads.map(n => n.id),
                    take: input.take ?? 10,
                    sortBy: input.sortBy,
                }, info, nestLimit - 1) : [];
                // Add thread to result
                threads.push({
                    __typename: "CommentThread" as const,
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
            req: Request,
            input: CommentSearchInput,
            info: GraphQLInfo | PartialGraphQLInfo,
            nestLimit = 2,
        ): Promise<CommentSearchResult> {
            const userData = getUser(req.session);
            // Partially convert info type
            const partialInfo = toPartialGqlInfo(info, ModelMap.get<CommentModelLogic>("Comment").format.gqlRelMap, req.session.languages, true);
            // Determine text search query
            const searchQuery = input.searchString ? getSearchStringQuery({ objectType: "Comment", searchString: input.searchString }) : undefined;
            // Loop through search fields and add each to the search query, 
            // if the field is specified in the input
            const customQueries: { [x: string]: any }[] = [];
            for (const field of Object.keys(ModelMap.get<CommentModelLogic>("Comment").search.searchFields)) {
                const fieldInput = input[field];
                const searchMapper = SearchMap[field];
                if (fieldInput !== undefined && searchMapper !== undefined) {
                    const searchData = { objectType: __typename, userData, visibility: VisibilityType.Public };
                    customQueries.push(searchMapper(fieldInput, searchData));
                }
            }
            // Create query for visibility
            const visibilityQuery = visibilityBuilderPrisma({ objectType: "Comment", userData, visibility: VisibilityType.Public });
            // Combine queries
            const where = combineQueries([searchQuery, visibilityQuery, ...customQueries]);
            // Determine sort order
            // Make sure sort field is valid
            const orderByField = input.sortBy ?? ModelMap.get<CommentModelLogic>("Comment").search.defaultSort;
            const orderBy = orderByField in SortMap ? SortMap[orderByField] : undefined;
            // Find requested search array
            const searchResults = await prismaInstance.comment.findMany({
                where,
                orderBy,
                take: input.take ?? 10,
                skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
                cursor: input.after ? {
                    id: input.after,
                } : undefined,
                ...selectHelper(partialInfo),
            });
            // If there are no results
            if (searchResults.length === 0) return {
                __typename: "CommentSearchResult" as const,
                totalThreads: 0,
                threads: [],
            };
            // Query total in thread, if cursor is not provided (since this means this data was already given to the user earlier)
            const totalInThread = input.after ? undefined : await prismaInstance.comment.count({
                where: { ...where },
            });
            // Calculate end cursor
            const endCursor = searchResults[searchResults.length - 1].id;
            // If not as nestLimit, recurse with all result IDs
            const childThreads = nestLimit > 0 ? await this.searchThreads(userData, {
                ids: searchResults.map(r => r.id),
                take: input.take ?? 10,
                sortBy: input.sortBy ?? ModelMap.get<CommentModelLogic>("Comment").search.defaultSort,
            }, info, nestLimit) : [];
            // Find every comment in "childThreads", and put into 1D array. This uses a helper function to handle recursion
            function flattenThreads(threads: CommentThread[]) {
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
            comments = await addSupplementalFields(userData, comments, partialInfo);
            // Put comments back into "threads" object, using another helper function. 
            // Comments can be matched by their ID
            function shapeThreads(threads: CommentThread[]) {
                const result: CommentThread[] = [];
                for (const thread of threads) {
                    // Find current-level comment
                    const comment = comments.find((c: any) => c.id === thread.comment.id);
                    // Recurse
                    const children = shapeThreads(thread.childThreads);
                    // Add thread to result
                    result.push({
                        __typename: "CommentThread" as const,
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
                __typename: "CommentSearchResult" as const,
                totalThreads: totalInThread,
                threads,
                endCursor,
            };
        },
    },
    search: {
        defaultSort: CommentSortBy.ScoreDesc,
        searchFields: {
            apiVersionId: true,
            createdTimeFrame: true,
            codeVersionId: true,
            issueId: true,
            minScore: true,
            minBookmarks: true,
            noteVersionId: true,
            ownedByTeamId: true,
            ownedByUserId: true,
            postId: true,
            projectVersionId: true,
            pullRequestId: true,
            questionAnswerId: true,
            questionId: true,
            routineVersionId: true,
            standardVersionId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        sortBy: CommentSortBy,
        searchStringQuery: () => ({ translations: "transText" }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<CommentModelInfo["GqlPermission"]>(__typename, ids, userData)),
                        isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(userData?.id, ids, __typename),
                        reaction: await ModelMap.get<ReactionModelLogic>("Reaction").query.getReactions(userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            apiVersion: "ApiVersion",
            codeVersion: "CodeVersion",
            issue: "Issue",
            ownedByTeam: "Team",
            ownedByUser: "User",
            post: "Post",
            projectVersion: "ProjectVersion",
            pullRequest: "PullRequest",
            question: "Question",
            questionAnswer: "QuestionAnswer",
            routineVersion: "RoutineVersion",
            standardVersion: "StandardVersion",
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isLoggedIn, isPublic }) => ({
            ...defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic }),
            canReply: () => isLoggedIn && (isAdmin || isPublic),
        }),
        owner: (data) => ({
            Team: data?.ownedByTeam,
            User: data?.ownedByUser,
        }),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<CommentModelInfo["PrismaSelect"]>([
            ["apiVersion", "ApiVersion"],
            ["codeVersion", "CodeVersion"],
            ["issue", "Issue"],
            ["post", "Post"],
            ["projectVersion", "ProjectVersion"],
            ["pullRequest", "PullRequest"],
            ["question", "Question"],
            ["questionAnswer", "QuestionAnswer"],
            ["routineVersion", "RoutineVersion"],
            ["standardVersion", "StandardVersion"],
        ], ...rest),
        visibility: {
            private: function getVisibilityPrivate() {
                return {};
            },
            public: function getVisibilityPublic() {
                return {};
            },
            owner: (userId) => ({ ownedByUser: { id: userId } }),
        },
    }),
});
