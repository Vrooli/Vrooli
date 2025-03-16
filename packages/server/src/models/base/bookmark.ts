import { BookmarkFor, BookmarkSortBy, MaxObjects, ModelType, bookmarkValidation, exists, lowercaseFirstLetter, uppercaseFirstLetter } from "@local/shared";
import { Prisma } from "@prisma/client";
import { findFirstRel } from "../../builders/findFirstRel.js";
import { onlyValidIds } from "../../builders/onlyValid.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { type PrismaDelegate } from "../../builders/types.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { DbProvider } from "../../db/provider.js";
import { Trigger } from "../../events/trigger.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { BookmarkFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { BookmarkListModelInfo, BookmarkListModelLogic, BookmarkModelLogic } from "./types.js";

const forMapper: { [key in BookmarkFor]: keyof Prisma.bookmarkUpsertArgs["create"] } = {
    Api: "api",
    Code: "code",
    Comment: "comment",
    Issue: "issue",
    Note: "note",
    Post: "post",
    Project: "project",
    Question: "question",
    QuestionAnswer: "questionAnswer",
    Quiz: "quiz",
    Routine: "routine",
    Standard: "standard",
    Tag: "tag",
    Team: "team",
    User: "user",
};

const __typename = "Bookmark" as const;
export const BookmarkModel: BookmarkModelLogic = ({
    __typename,
    dbTable: "bookmark",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                ...Object.fromEntries(Object.entries(forMapper).map(([key, value]) =>
                    [value, { select: ModelMap.get(key as ModelType).display().label.select() }])),
            }),
            get: (select, languages) => {
                for (const [key, value] of Object.entries(forMapper)) {
                    if (select[value]) return ModelMap.get(key as ModelType).display().label.get(select[value], languages);
                }
                return "";
            },
        },
    }),
    format: BookmarkFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                [forMapper[data.bookmarkFor]]: { connect: { id: data.forConnect } },
                list: await shapeHelper({ relation: "list", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "BookmarkList", parentRelationshipName: "bookmarks", data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                id: data.id,
                list: await shapeHelper({ relation: "list", relTypes: ["Connect", "Update"], isOneToOne: true, objectType: "BookmarkList", parentRelationshipName: "bookmarks", data, ...rest }),
            }),
        },
        trigger: {
            beforeDeleted: async ({ beforeDeletedData, deletingIds }) => {
                // Grab bookmarked object id and type
                const deleting = await DbProvider.get().bookmark.findMany({
                    where: { id: { in: deletingIds } },
                    select: {
                        apiId: true,
                        codeId: true,
                        commentId: true,
                        issueId: true,
                        noteId: true,
                        postId: true,
                        projectId: true,
                        questionId: true,
                        questionAnswerId: true,
                        quizId: true,
                        routineId: true,
                        standardId: true,
                        tagId: true,
                        teamId: true,
                        userId: true,
                    },
                });
                // Find type and id of bookmarked object
                const bookmarkedPairs: [BookmarkFor, string][] = deleting.map(c => {
                    const [objectRel, objectId] = findFirstRel(c, [
                        "apiId",
                        "codeId",
                        "commentId",
                        "issueId",
                        "noteId",
                        "postId",
                        "projectId",
                        "questionId",
                        "questionAnswerId",
                        "quizId",
                        "routineId",
                        "standardId",
                        "tagId",
                        "teamId",
                        "userId",
                    ]);
                    if (!objectRel || !objectId) return [null, null];
                    // Object type is objectRel with "Id" removed and first letter capitalized
                    const objectType: BookmarkFor = uppercaseFirstLetter(objectRel.slice(0, -"Id".length)) as BookmarkFor;
                    return [objectType, objectId];
                }).filter(([objectType, objectId]) => objectType && objectId) as [BookmarkFor, string][];
                // Group by object type
                const grouped: { [key in BookmarkFor]?: string[] } = bookmarkedPairs.reduce((acc, [objectType, objectId]) => {
                    if (!acc[objectType]) acc[objectType] = [];
                    acc[objectType].push(objectId);
                    return acc;
                }, {});
                // Add result to beforeDeletedData
                if (beforeDeletedData[__typename]) beforeDeletedData[__typename] = { ...beforeDeletedData[__typename], ...grouped };
                else beforeDeletedData[__typename] = grouped;
            },
            afterMutations: async ({ beforeDeletedData, createInputs, userData }) => {
                for (const c of createInputs) {
                    // Find type and id of bookmarked object
                    const [objectRel, objectId] = findFirstRel(c, [
                        "apiId",
                        "codeId",
                        "commentId",
                        "issueId",
                        "noteId",
                        "postId",
                        "projectId",
                        "questionId",
                        "questionAnswerId",
                        "quizId",
                        "routineId",
                        "standardId",
                        "tagId",
                        "teamId",
                        "userId",
                    ]);
                    if (!objectRel || !objectId) return;
                    // Object type is objectRel with "Id" removed and first letter capitalized
                    const objectType: BookmarkFor = uppercaseFirstLetter(objectRel.slice(0, -"Id".length)) as BookmarkFor;
                    // Update "bookmarks" count for bookmarked object
                    const delegate = (DbProvider.get()[ModelMap.get(objectType, true, "bookmark onCreated").dbTable] as PrismaDelegate);
                    await delegate.update({ where: { id: objectId }, data: { bookmarks: { increment: 1 } } });
                    // Trigger bookmarkCreated event
                    Trigger(userData.languages).objectBookmark(true, objectType, objectId, userData.id);
                }
                // For each bookmarked object type, decrement the bookmark count
                for (const [objectType, objectIds] of Object.entries((beforeDeletedData[__typename] ?? {}) as { [key in BookmarkFor]?: string[] })) {
                    const delegate = (DbProvider.get()[ModelMap.get(objectType as ModelType, true, "bookmark onDeleted").dbTable] as PrismaDelegate);
                    await (delegate as any).updateMany({ where: { id: { in: objectIds } }, data: { bookmarks: { decrement: 1 } } });
                    // For each bookmarked object, trigger bookmarkDeleted event
                    for (const objectId of (objectIds as string[])) {
                        Trigger(userData.languages).objectBookmark(false, objectType as BookmarkFor, objectId, userData.id);
                    }
                }
            },
        },
        yup: bookmarkValidation,
    },
    query: {
        async getIsBookmarkeds(
            userId: string | null | undefined,
            ids: string[],
            bookmarkFor: keyof typeof BookmarkFor,
        ): Promise<boolean[]> {
            // Create result array that is the same length as ids
            const result = new Array(ids.length).fill(false);
            // If userId not passed, return result
            if (!userId) return result;
            // Filter out nulls and undefineds from ids
            const idsFiltered = onlyValidIds(ids);
            const fieldName = `${lowercaseFirstLetter(bookmarkFor)}Id`;
            const isBookmarkredArray = await DbProvider.get().bookmark.findMany({ where: { list: { user: { id: userId } }, [fieldName]: { in: idsFiltered } } });
            // Replace the nulls in the result array with true or false
            for (let i = 0; i < ids.length; i++) {
                // check if this id is in isBookmarkredArray
                if (exists(ids[i]) && isBookmarkredArray.find((bookmark: any) => bookmark[fieldName] === ids[i])) {
                    result[i] = true;
                }
            }
            return result;
        },
    },
    search: {
        defaultSort: BookmarkSortBy.DateUpdatedDesc,
        sortBy: BookmarkSortBy,
        searchFields: {
            apiId: true,
            codeId: true,
            commentId: true,
            excludeLinkedToTag: true,
            issueId: true,
            listLabel: true,
            limitTo: true,
            listId: true,
            noteId: true,
            postId: true,
            projectId: true,
            questionId: true,
            questionAnswerId: true,
            quizId: true,
            routineId: true,
            standardId: true,
            tagId: true,
            teamId: true,
            userId: true,
        },
        searchStringQuery: () => ({
            OR: [
                { list: ModelMap.get<BookmarkListModelLogic>("BookmarkList").search.searchStringQuery() },
                ...Object.entries(forMapper).map(([key, value]) => ({ [value]: ModelMap.getLogic(["search"], key as ModelType).search.searchStringQuery() })),
            ],
        }),
        supplemental: {
            // Make sure to query for every bookmarked item, 
            // so we can ensure that the mutation trigger can increment the bookmark count
            dbFields: [
                "apiId",
                "codeId",
                "commentId",
                "issueId",
                "noteId",
                "postId",
                "projectId",
                "questionId",
                "questionAnswerId",
                "quizId",
                "routineId",
                "standardId",
                "tagId",
                "teamId",
                "userId",
            ],
            suppFields: [],
            getSuppFields: async () => ({}),
        },
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<BookmarkListModelLogic>("BookmarkList").validate().owner(data?.list as BookmarkListModelInfo["DbModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            list: "BookmarkList",
            ...Object.fromEntries(Object.entries(forMapper).map(([key, value]) => [value, key as ModelType])),
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    list: useVisibility("BookmarkList", "Own", data),
                };
            },
            // Always private, so it's the same as "own"
            ownOrPublic: function getOwnOrPublic(data) {
                return useVisibility("Bookmark", "Own", data);
            },
            // Always private, so it's the same as "own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("Bookmark", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
