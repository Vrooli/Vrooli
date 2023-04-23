import { CommentSortBy, MaxObjects } from "@local/consts";
import { lowercaseFirstLetter } from "@local/utils";
import { commentValidation } from "@local/validation";
import { getUser } from "../auth";
import { addSupplementalFields, combineQueries, modelToGql, selectHelper, toPartialGqlInfo } from "../builders";
import { getSearchStringQuery } from "../getters";
import { bestLabel, defaultPermissions, onCommonPlain, oneIsPublic, SearchMap, translationShapeHelper } from "../utils";
import { SortMap } from "../utils/sortMap";
import { getSingleTypePermissions } from "../validators";
import { BookmarkModel } from "./bookmark";
import { ReactionModel } from "./reaction";
const __typename = "Comment";
const suppFields = ["you"];
export const CommentModel = ({
    __typename,
    delegate: (prisma) => prisma.comment,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, text: true } } }),
        label: (select, languages) => bestLabel(select.translations, "text", languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            owner: {
                ownedByUser: "User",
                ownedByOrganization: "Organization",
            },
            commentedOn: {
                apiVersion: "ApiVersion",
                issue: "Issue",
                noteVersion: "NoteVersion",
                post: "Post",
                projectVersion: "ProjectVersion",
                pullRequest: "PullRequest",
                question: "Question",
                questionAnswer: "QuestionAnswer",
                routineVersion: "RoutineVersion",
                smartContractVersion: "SmartContractVersion",
                standardVersion: "StandardVersion",
            },
            reports: "Report",
            bookmarkedBy: "User",
        },
        prismaRelMap: {
            __typename: "Comment",
            ownedByUser: "User",
            ownedByOrganization: "Organization",
            apiVersion: "ApiVersion",
            issue: "Issue",
            noteVersion: "NoteVersion",
            parent: "Comment",
            post: "Post",
            projectVersion: "ProjectVersion",
            pullRequest: "PullRequest",
            question: "Question",
            questionAnswer: "QuestionAnswer",
            routineVersion: "RoutineVersion",
            smartContractVersion: "SmartContractVersion",
            standardVersion: "StandardVersion",
            reports: "Report",
            bookmarkedBy: "User",
            reactions: "Reaction",
            parents: "Comment",
        },
        joinMap: { bookmarkedBy: "user" },
        countFields: {
            reportsCount: true,
            translationsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
                        isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        reaction: await ReactionModel.query.getReactions(prisma, userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                ownedByUser: { connect: { id: rest.userData.id } },
                [lowercaseFirstLetter(data.createdFor)]: { connect: { id: data.forConnect } },
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
        },
        trigger: {
            onCommon: async (params) => {
                await onCommonPlain({
                    ...params,
                    objectType: __typename,
                    ownerOrganizationField: "ownedByOrganization",
                    ownerUserField: "ownedByUser",
                });
            },
        },
        yup: commentValidation,
    },
    query: {
        async searchThreads(prisma, userData, input, info, nestLimit = 2) {
            const partialInfo = toPartialGqlInfo(info, CommentModel.format.gqlRelMap, userData?.languages ?? ["en"], true);
            const idQuery = (Array.isArray(input.ids)) ? ({ id: { in: input.ids } }) : undefined;
            const where = { ...idQuery };
            const orderByField = input.sortBy ?? CommentModel.search.defaultSort;
            const orderByIsValid = CommentModel.search.sortBy[orderByField] === undefined;
            const orderBy = orderByIsValid ? SortMap[input.sortBy ?? CommentModel.search.defaultSort] : undefined;
            const searchResults = await prisma.comment.findMany({
                where,
                orderBy,
                take: input.take ?? 10,
                ...selectHelper(partialInfo),
            });
            if (searchResults.length === 0)
                return [];
            const threads = [];
            for (const result of searchResults) {
                const totalInThread = await prisma.comment.count({
                    where: {
                        ...where,
                        parentId: result.id,
                    },
                });
                const nestedThreads = nestLimit > 0 ? await prisma.comment.findMany({
                    where: {
                        ...where,
                        parentId: result.id,
                    },
                    take: input.take ?? 10,
                    ...selectHelper(partialInfo),
                }) : [];
                const endCursor = nestedThreads.length > 0 ? nestedThreads[nestedThreads.length - 1].id : undefined;
                const childThreads = nestLimit > 0 ? await this.searchThreads(prisma, userData, {
                    ids: nestedThreads.map(n => n.id),
                    take: input.take ?? 10,
                    sortBy: input.sortBy,
                }, info, nestLimit - 1) : [];
                threads.push({
                    __typename: "CommentThread",
                    childThreads,
                    comment: result,
                    endCursor,
                    totalInThread,
                });
            }
            return threads;
        },
        async searchNested(prisma, req, input, info, nestLimit = 2) {
            const partialInfo = toPartialGqlInfo(info, CommentModel.format.gqlRelMap, req.languages, true);
            const searchQuery = input.searchString ? getSearchStringQuery({ objectType: "Comment", searchString: input.searchString }) : undefined;
            const customQueries = [];
            for (const field of Object.keys(CommentModel.search.searchFields)) {
                if (input[field] !== undefined) {
                    customQueries.push(SearchMap[field](input, getUser(req), __typename));
                }
            }
            const where = combineQueries([searchQuery, ...customQueries]);
            const orderByField = input.sortBy ?? CommentModel.search.defaultSort;
            const orderByIsValid = CommentModel.search.sortBy[orderByField] === undefined;
            const orderBy = orderByIsValid ? SortMap[input.sortBy ?? CommentModel.search.defaultSort] : undefined;
            const searchResults = await prisma.comment.findMany({
                where,
                orderBy,
                take: input.take ?? 10,
                skip: input.after ? 1 : undefined,
                cursor: input.after ? {
                    id: input.after,
                } : undefined,
                ...selectHelper(partialInfo),
            });
            if (searchResults.length === 0)
                return {
                    __typename: "CommentSearchResult",
                    totalThreads: 0,
                    threads: [],
                };
            const totalInThread = input.after ? undefined : await prisma.comment.count({
                where: { ...where },
            });
            const endCursor = searchResults[searchResults.length - 1].id;
            const childThreads = nestLimit > 0 ? await this.searchThreads(prisma, getUser(req), {
                ids: searchResults.map(r => r.id),
                take: input.take ?? 10,
                sortBy: input.sortBy ?? CommentModel.search.defaultSort,
            }, info, nestLimit) : [];
            const flattenThreads = (threads) => {
                const result = [];
                for (const thread of threads) {
                    result.push(thread.comment);
                    result.push(...flattenThreads(thread.childThreads));
                }
                return result;
            };
            let comments = flattenThreads(childThreads);
            comments = comments.map((c) => modelToGql(c, partialInfo));
            comments = await addSupplementalFields(prisma, getUser(req), comments, partialInfo);
            const shapeThreads = (threads) => {
                const result = [];
                for (const thread of threads) {
                    const comment = comments.find((c) => c.id === thread.comment.id);
                    const children = shapeThreads(thread.childThreads);
                    result.push({
                        __typename: "CommentThread",
                        comment,
                        childThreads: children,
                        endCursor: thread.endCursor,
                        totalInThread: thread.totalInThread,
                    });
                }
                return result;
            };
            const threads = shapeThreads(childThreads);
            return {
                __typename: "CommentSearchResult",
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
        searchStringQuery: () => ({ translations: "transText" }),
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            apiVersion: "Api",
            issue: "Issue",
            ownedByOrganization: "Organization",
            ownedByUser: "User",
            post: "Post",
            projectVersion: "Project",
            pullRequest: "PullRequest",
            question: "Question",
            questionAnswer: "QuestionAnswer",
            routineVersion: "Routine",
            smartContractVersion: "SmartContract",
            standardVersion: "Standard",
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
        isPublic: (data, languages) => oneIsPublic(data, [
            ["apiVersion", "ApiVersion"],
            ["issue", "Issue"],
            ["post", "Post"],
            ["projectVersion", "ProjectVersion"],
            ["pullRequest", "PullRequest"],
            ["question", "Question"],
            ["questionAnswer", "QuestionAnswer"],
            ["routineVersion", "RoutineVersion"],
            ["smartContractVersion", "SmartContractVersion"],
            ["standardVersion", "StandardVersion"],
        ], languages),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ ownedByUser: { id: userId } }),
        },
    },
});
//# sourceMappingURL=comment.js.map