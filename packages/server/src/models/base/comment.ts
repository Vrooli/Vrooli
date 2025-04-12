import { Comment, CommentFor, CommentSearchInput, CommentSearchResult, CommentSortBy, CommentThread, MaxObjects, ModelType, SessionUser, VisibilityType, commentValidation, getTranslation, lowercaseFirstLetter } from "@local/shared";
import { Prisma } from "@prisma/client";
import { Request } from "express";
import { SessionService } from "../../auth/session.js";
import { combineQueries } from "../../builders/combineQueries.js";
import { InfoConverter, addSupplementalFields } from "../../builders/infoConverter.js";
import { PartialApiInfo } from "../../builders/types.js";
import { useVisibility, useVisibilityMapper, visibilityBuilderPrisma } from "../../builders/visibilityBuilder.js";
import { DbProvider } from "../../db/provider.js";
import { getSearchStringQuery } from "../../getters/getSearchStringQuery.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { SearchMap } from "../../utils/searchMap.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { SortMap } from "../../utils/sortMap.js";
import { afterMutationsPlain } from "../../utils/triggers/afterMutationsPlain.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { CommentFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { BookmarkModelLogic, CommentModelInfo, CommentModelLogic, ReactionModelLogic, TeamModelLogic } from "./types.js";

const DEFAULT_TAKE = 10;

const forMapper: { [key in CommentFor]: keyof Prisma.commentUpsertArgs["create"] } = {
    ApiVersion: "apiVersion",
    CodeVersion: "codeVersion",
    Issue: "issue",
    NoteVersion: "noteVersion",
    Post: "post",
    ProjectVersion: "projectVersion",
    PullRequest: "pullRequest",
    Question: "question",
    QuestionAnswer: "questionAnswer",
    RoutineVersion: "routineVersion",
    StandardVersion: "standardVersion",
};
const reversedForMapper: { [key in keyof Prisma.commentUpsertArgs["create"]]: CommentFor } = Object.fromEntries(
    Object.entries(forMapper).map(([key, value]) => [value, key]),
);

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
            userData: SessionUser | null,
            input: { ids: string[], take: number, sortBy: CommentSortBy },
            info: PartialApiInfo,
            nestLimit = 2,
        ): Promise<CommentThread[]> {
            // Partially convert info type
            const partialInfo = InfoConverter.get().fromApiToPartialApi(info, ModelMap.get<CommentModelLogic>("Comment").format.apiRelMap, true);
            const idQuery = (Array.isArray(input.ids)) ? ({ id: { in: input.ids } }) : undefined;
            // Combine queries
            const where = { ...idQuery };
            // Determine sort order
            // Make sure sort field is valid
            const orderByField = input.sortBy ?? ModelMap.get<CommentModelLogic>("Comment").search.defaultSort;
            const orderByIsValid = ModelMap.get<CommentModelLogic>("Comment").search.sortBy[orderByField] === undefined;
            const orderBy = orderByIsValid ? SortMap[input.sortBy ?? ModelMap.get<CommentModelLogic>("Comment").search.defaultSort] : undefined;
            // Find requested search array
            const searchResults = await DbProvider.get().comment.findMany({
                where,
                orderBy,
                take: input.take ?? DEFAULT_TAKE,
                ...InfoConverter.get().fromPartialApiToPrismaSelect(partialInfo),
            });
            // If there are no results
            if (searchResults.length === 0) return [];
            // Initialize result 
            const threads: CommentThread[] = [];
            // For each result
            for (const result of searchResults) {
                // Find total in thread
                const totalInThread = await DbProvider.get().comment.count({
                    where: {
                        ...where,
                        parentId: result.id,
                    },
                });
                // Query for nested threads
                const nestedThreads = nestLimit > 0 ? await DbProvider.get().comment.findMany({
                    where: {
                        ...where,
                        parentId: result.id,
                    },
                    take: input.take ?? DEFAULT_TAKE,
                    ...InfoConverter.get().fromPartialApiToPrismaSelect(partialInfo),
                }) : [];
                // Find end cursor of nested threads
                const endCursor = nestedThreads.length > 0 ? nestedThreads[nestedThreads.length - 1].id : undefined;
                // For nested threads, recursively call this function
                const childThreads = nestLimit > 0 ? await this.searchThreads(userData, {
                    ids: nestedThreads.map(n => n.id),
                    take: input.take ?? DEFAULT_TAKE,
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
            info: PartialApiInfo,
            nestLimit = 2,
        ): Promise<CommentSearchResult> {
            const userData = SessionService.getUser(req);
            // Partially convert info type
            const partialInfo = InfoConverter.get().fromApiToPartialApi(info, ModelMap.get<CommentModelLogic>("Comment").format.apiRelMap, true);
            // Determine text search query
            const searchQuery = input.searchString ? getSearchStringQuery({ objectType: "Comment", searchString: input.searchString }) : undefined;
            const searchData = {
                objectType: __typename,
                req,
                searchInput: input,
                userData,
                visibility: VisibilityType.Public,
            };
            // Loop through search fields and add each to the search query, 
            // if the field is specified in the input
            const customQueries: { [x: string]: any }[] = [];
            for (const field of Object.keys(ModelMap.get<CommentModelLogic>("Comment").search.searchFields)) {
                const fieldInput = input[field];
                const searchMapper = SearchMap[field];
                if (fieldInput !== undefined && searchMapper !== undefined) {
                    customQueries.push(searchMapper(fieldInput, searchData));
                }
            }
            // Create query for visibility
            const { query: visibilityQuery } = visibilityBuilderPrisma(searchData);
            // Combine queries
            const where = combineQueries([searchQuery, visibilityQuery, ...customQueries]);
            // Determine sort order
            // Make sure sort field is valid
            const orderByField = input.sortBy ?? ModelMap.get<CommentModelLogic>("Comment").search.defaultSort;
            const orderBy = orderByField in SortMap ? SortMap[orderByField] : undefined;
            // Find requested search array
            const searchResults = await DbProvider.get().comment.findMany({
                where,
                orderBy,
                take: input.take ?? DEFAULT_TAKE,
                skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
                cursor: input.after ? {
                    id: input.after,
                } : undefined,
                ...InfoConverter.get().fromPartialApiToPrismaSelect(partialInfo),
            });
            // If there are no results
            if (searchResults.length === 0) return {
                __typename: "CommentSearchResult" as const,
                totalThreads: 0,
                threads: [],
            };
            // Query total in thread, if cursor is not provided (since this means this data was already given to the user earlier)
            const totalInThread = input.after ? undefined : await DbProvider.get().comment.count({
                where: { ...where },
            });
            // Calculate end cursor
            const endCursor = searchResults[searchResults.length - 1].id;
            // If not as nestLimit, recurse with all result IDs
            const childThreads = nestLimit > 0 ? await this.searchThreads(userData, {
                ids: searchResults.map(r => r.id),
                take: input.take ?? DEFAULT_TAKE,
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
            comments = comments.map((c: any) => InfoConverter.get().fromDbToApi(c, partialInfo as PartialApiInfo));
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
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<CommentModelInfo["ApiPermission"]>(__typename, ids, userData)),
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
        isPublic: (...rest) => oneIsPublic<CommentModelInfo["DbSelect"]>([
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
            own: function getOwn(data) {
                return {
                    OR: [
                        { ownedByTeam: ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(data.userId) },
                        { ownedByUser: { id: data.userId } },
                    ],
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        useVisibility("Comment", "Own", data),
                        useVisibility("Comment", "Public", data),
                    ],
                };
            },
            // Search method not useful for this object because comments are not explicitly set as private, so we'll return "Own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("Comment", "Own", data);
            },
            // Search method not useful for this object because comments are not explicitly set as private, so we'll return "Own"
            ownPublic: function getPrivate(data) {
                return useVisibility("Comment", "Own", data);
            },
            public: function getVisibilityPublic(data) {
                const searchInput = data.searchInput as CommentSearchInput;
                // If the search input has a relation ID, return that relation only
                const forSearch = Object.keys(searchInput).find(searchKey =>
                    searchKey.endsWith("Id") &&
                    reversedForMapper[searchKey.substring(0, searchKey.length - "Id".length)],
                );
                if (forSearch) {
                    const relation = forSearch.substring(0, forSearch.length - "Id".length);
                    return { [relation]: useVisibility(reversedForMapper[relation] as ModelType, "Public", data) };
                }
                // Otherwise, use an OR on all relations
                return {
                    OR: [
                        ...useVisibilityMapper("Public", data, forMapper, false),
                    ],
                };
            },
        },
    }),
});
