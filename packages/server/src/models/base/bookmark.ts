import { BookmarkFor, BookmarkSortBy, bookmarkValidation, exists, GqlModelType, lowercaseFirstLetter, MaxObjects, uppercaseFirstLetter } from "@local/shared";
import { Prisma } from "@prisma/client";
import { ModelMap } from ".";
import { findFirstRel } from "../../builders/findFirstRel";
import { onlyValidIds } from "../../builders/onlyValidIds";
import { shapeHelper } from "../../builders/shapeHelper";
import { Trigger } from "../../events/trigger";
import { PrismaType } from "../../types";
import { defaultPermissions } from "../../utils";
import { BookmarkFormat } from "../formats";
import { ApiModelInfo, ApiModelLogic, BookmarkListModelInfo, BookmarkListModelLogic, BookmarkModelLogic, CommentModelInfo, CommentModelLogic, IssueModelInfo, IssueModelLogic, NoteModelInfo, NoteModelLogic, OrganizationModelInfo, OrganizationModelLogic, PostModelInfo, PostModelLogic, ProjectModelInfo, ProjectModelLogic, QuestionAnswerModelInfo, QuestionAnswerModelLogic, QuestionModelInfo, QuestionModelLogic, QuizModelInfo, QuizModelLogic, RoutineModelInfo, RoutineModelLogic, SmartContractModelInfo, SmartContractModelLogic, StandardModelInfo, StandardModelLogic, TagModelInfo, TagModelLogic, UserModelInfo, UserModelLogic } from "./types";

const forMapper: { [key in BookmarkFor]: keyof Prisma.bookmarkUpsertArgs["create"] } = {
    Api: "api",
    Comment: "comment",
    Issue: "issue",
    Note: "note",
    Organization: "organization",
    Post: "post",
    Project: "project",
    Question: "question",
    QuestionAnswer: "questionAnswer",
    Quiz: "quiz",
    Routine: "routine",
    SmartContract: "smartContract",
    Standard: "standard",
    Tag: "tag",
    User: "user",
};

const __typename = "Bookmark" as const;
export const BookmarkModel: BookmarkModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.bookmark,
    display: {
        label: {
            select: () => ({
                id: true,
                api: { select: ModelMap.get<ApiModelLogic>("Api").display.label.select() },
                comment: { select: ModelMap.get<CommentModelLogic>("Comment").display.label.select() },
                issue: { select: ModelMap.get<IssueModelLogic>("Issue").display.label.select() },
                note: { select: ModelMap.get<NoteModelLogic>("Note").display.label.select() },
                organization: { select: ModelMap.get<OrganizationModelLogic>("Organization").display.label.select() },
                post: { select: ModelMap.get<PostModelLogic>("Post").display.label.select() },
                project: { select: ModelMap.get<ProjectModelLogic>("Project").display.label.select() },
                question: { select: ModelMap.get<QuestionModelLogic>("Question").display.label.select() },
                questionAnswer: { select: ModelMap.get<QuestionAnswerModelLogic>("QuestionAnswer").display.label.select() },
                quiz: { select: ModelMap.get<QuizModelLogic>("Quiz").display.label.select() },
                routine: { select: ModelMap.get<RoutineModelLogic>("Routine").display.label.select() },
                smartContract: { select: ModelMap.get<SmartContractModelLogic>("SmartContract").display.label.select() },
                standard: { select: ModelMap.get<StandardModelLogic>("Standard").display.label.select() },
                tag: { select: ModelMap.get<TagModelLogic>("Tag").display.label.select() },
                user: { select: ModelMap.get<UserModelLogic>("User").display.label.select() },
            }),
            get: (select, languages) => {
                if (select.api) return ModelMap.get<ApiModelLogic>("Api").display.label.get(select.api as ApiModelInfo["PrismaModel"], languages);
                if (select.comment) return ModelMap.get<CommentModelLogic>("Comment").display.label.get(select.comment as CommentModelInfo["PrismaModel"], languages);
                if (select.issue) return ModelMap.get<IssueModelLogic>("Issue").display.label.get(select.issue as IssueModelInfo["PrismaModel"], languages);
                if (select.note) return ModelMap.get<NoteModelLogic>("Note").display.label.get(select.note as NoteModelInfo["PrismaModel"], languages);
                if (select.organization) return ModelMap.get<OrganizationModelLogic>("Organization").display.label.get(select.organization as OrganizationModelInfo["PrismaModel"], languages);
                if (select.post) return ModelMap.get<PostModelLogic>("Post").display.label.get(select.post as PostModelInfo["PrismaModel"], languages);
                if (select.project) return ModelMap.get<ProjectModelLogic>("Project").display.label.get(select.project as ProjectModelInfo["PrismaModel"], languages);
                if (select.question) return ModelMap.get<QuestionModelLogic>("Question").display.label.get(select.question as QuestionModelInfo["PrismaModel"], languages);
                if (select.questionAnswer) return ModelMap.get<QuestionAnswerModelLogic>("QuestionAnswer").display.label.get(select.questionAnswer as QuestionAnswerModelInfo["PrismaModel"], languages);
                if (select.quiz) return ModelMap.get<QuizModelLogic>("Quiz").display.label.get(select.quiz as QuizModelInfo["PrismaModel"], languages);
                if (select.routine) return ModelMap.get<RoutineModelLogic>("Routine").display.label.get(select.routine as RoutineModelInfo["PrismaModel"], languages);
                if (select.smartContract) return ModelMap.get<SmartContractModelLogic>("SmartContract").display.label.get(select.smartContract as SmartContractModelInfo["PrismaModel"], languages);
                if (select.standard) return ModelMap.get<StandardModelLogic>("Standard").display.label.get(select.standard as StandardModelInfo["PrismaModel"], languages);
                if (select.tag) return ModelMap.get<TagModelLogic>("Tag").display.label.get(select.tag as TagModelInfo["PrismaModel"], languages);
                if (select.user) return ModelMap.get<UserModelLogic>("User").display.label.get(select.user as UserModelInfo["PrismaModel"], languages);
                return "";
            },
        },
    },
    format: BookmarkFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                [forMapper[data.bookmarkFor]]: { connect: { id: data.forConnect } },
                ...(await shapeHelper({ relation: "list", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "BookmarkList", parentRelationshipName: "bookmarks", data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                id: data.id,
                ...(await shapeHelper({ relation: "list", relTypes: ["Connect", "Update"], isOneToOne: true, isRequired: false, objectType: "BookmarkList", parentRelationshipName: "bookmarks", data, ...rest })),
            }),
        },
        trigger: {
            beforeDeleted: async ({ beforeDeletedData, deletingIds, prisma }) => {
                // Grab bookmarked object id and type
                const deleting = await prisma.bookmark.findMany({
                    where: { id: { in: deletingIds } },
                    select: { apiId: true, commentId: true, issueId: true, noteId: true, organizationId: true, postId: true, projectId: true, questionId: true, questionAnswerId: true, quizId: true, routineId: true, smartContractId: true, standardId: true, tagId: true, userId: true },
                });
                // Find type and id of bookmarked object
                const bookmarkedPairs: [BookmarkFor, string][] = deleting.map(c => {
                    const [objectRel, objectId] = findFirstRel(c, ["apiId", "commentId", "issueId", "noteId", "organizationId", "postId", "projectId", "questionId", "questionAnswerId", "quizId", "routineId", "smartContractId", "standardId", "tagId", "userId"]);
                    if (!objectRel || !objectId) return [null, null];
                    // Object type is objectRel with "Id" removed and first letter capitalized
                    const objectType: BookmarkFor = uppercaseFirstLetter(objectRel.slice(0, -2)) as BookmarkFor;
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
            afterMutations: async ({ beforeDeletedData, createInputs, prisma, userData }) => {
                for (const c of createInputs) {
                    // Find type and id of bookmarked object
                    const [objectRel, objectId] = findFirstRel(c, ["apiId", "commentId", "issueId", "noteId", "organizationId", "postId", "projectId", "questionId", "questionAnswerId", "quizId", "routineId", "smartContractId", "standardId", "tagId", "userId"]);
                    if (!objectRel || !objectId) return;
                    // Object type is objectRel with "Id" removed and first letter capitalized
                    const objectType: BookmarkFor = uppercaseFirstLetter(objectRel.slice(0, -2)) as BookmarkFor;
                    // Update "bookmarks" count for bookmarked object
                    const delegate = ModelMap.get(objectType, true, "bookmark onCreated").delegate;
                    await delegate(prisma).update({ where: { id: objectId }, data: { bookmarks: { increment: 1 } } });
                    // Trigger bookmarkCreated event
                    Trigger(prisma, userData.languages).objectBookmark(true, objectType, objectId, userData.id);
                }
                // For each bookmarked object type, decrement the bookmark count
                for (const [objectType, objectIds] of Object.entries((beforeDeletedData[__typename] ?? {}) as { [key in BookmarkFor]?: string[] })) {
                    const delegate = ModelMap.get(objectType as GqlModelType, true, "bookmark onDeleted").delegate;
                    await (delegate(prisma) as any).updateMany({ where: { id: { in: objectIds } }, data: { bookmarks: { decrement: 1 } } });
                    // For each bookmarked object, trigger bookmarkDeleted event
                    for (const objectId of (objectIds as string[])) {
                        Trigger(prisma, userData.languages).objectBookmark(false, objectType as BookmarkFor, objectId, userData.id);
                    }
                }
            },
        },
        yup: bookmarkValidation,
    },
    query: {
        async getIsBookmarkeds(
            prisma: PrismaType,
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
            const isBookmarkredArray = await prisma.bookmark.findMany({ where: { list: { user: { id: userId } }, [fieldName]: { in: idsFiltered } } });
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
            commentId: true,
            excludeLinkedToTag: true,
            issueId: true,
            listId: true,
            noteId: true,
            organizationId: true,
            postId: true,
            projectId: true,
            questionId: true,
            questionAnswerId: true,
            quizId: true,
            routineId: true,
            smartContractId: true,
            standardId: true,
            tagId: true,
            userId: true,
        },
        searchStringQuery: () => ({
            OR: [
                { list: ModelMap.get<BookmarkListModelLogic>("BookmarkList").search.searchStringQuery() },
                { api: ModelMap.get<ApiModelLogic>("Api").search.searchStringQuery() },
                { comment: ModelMap.get<CommentModelLogic>("Comment").search.searchStringQuery() },
                { issue: ModelMap.get<IssueModelLogic>("Issue").search.searchStringQuery() },
                { note: ModelMap.get<NoteModelLogic>("Note").search.searchStringQuery() },
                { organization: ModelMap.get<OrganizationModelLogic>("Organization").search.searchStringQuery() },
                { post: ModelMap.get<PostModelLogic>("Post").search.searchStringQuery() },
                { project: ModelMap.get<ProjectModelLogic>("Project").search.searchStringQuery() },
                { question: ModelMap.get<QuestionModelLogic>("Question").search.searchStringQuery() },
                { questionAnswer: ModelMap.get<QuestionAnswerModelLogic>("QuestionAnswer").search.searchStringQuery() },
                { quiz: ModelMap.get<QuizModelLogic>("Quiz").search.searchStringQuery() },
                { routine: ModelMap.get<RoutineModelLogic>("Routine").search.searchStringQuery() },
                { smartContract: ModelMap.get<SmartContractModelLogic>("SmartContract").search.searchStringQuery() },
                { standard: ModelMap.get<StandardModelLogic>("Standard").search.searchStringQuery() },
                { tag: ModelMap.get<TagModelLogic>("Tag").search.searchStringQuery() },
                { user: ModelMap.get<UserModelLogic>("User").search.searchStringQuery() },
            ],
        }),
        supplemental: {
            // Make sure to query for every bookmarked item, 
            // so we can ensure that the mutation trigger can increment the bookmark count
            dbFields: [
                "apiId",
                "commentId",
                "issueId",
                "noteId",
                "organizationId",
                "postId",
                "projectId",
                "questionId",
                "questionAnswerId",
                "quizId",
                "routineId",
                "smartContractId",
                "standardId",
                "tagId",
                "userId",
            ],
            graphqlFields: [],
            toGraphQL: async () => ({}),
        },
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<BookmarkListModelLogic>("BookmarkList").validate.owner(data?.list as BookmarkListModelInfo["PrismaModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            api: "Api",
            comment: "Comment",
            issue: "Issue",
            list: "BookmarkList",
            note: "Note",
            organization: "Organization",
            post: "Post",
            project: "Project",
            question: "Question",
            questionAnswer: "QuestionAnswer",
            quiz: "Quiz",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
            tag: "Tag",
            user: "User",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                list: ModelMap.get<BookmarkListModelLogic>("BookmarkList").validate.visibility.owner(userId),
            }),
        },
    },
});
