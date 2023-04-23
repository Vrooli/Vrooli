import { BookmarkSortBy, MaxObjects } from "@local/consts";
import { exists, uppercaseFirstLetter } from "@local/utils";
import { bookmarkValidation } from "@local/validation";
import { ApiModel, BookmarkListModel, IssueModel, PostModel, QuestionAnswerModel, QuestionModel, QuizModel, SmartContractModel, UserModel } from ".";
import { findFirstRel, onlyValidIds, selPad, shapeHelper } from "../builders";
import { Trigger } from "../events";
import { getLogic } from "../getters";
import { defaultPermissions } from "../utils";
import { CommentModel } from "./comment";
import { NoteModel } from "./note";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { StandardModel } from "./standard";
import { TagModel } from "./tag";
const forMapper = {
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
const __typename = "Bookmark";
const suppFields = [];
export const BookmarkModel = ({
    __typename,
    delegate: (prisma) => prisma.bookmark,
    display: {
        select: () => ({
            id: true,
            api: selPad(ApiModel.display.select),
            comment: selPad(CommentModel.display.select),
            issue: selPad(IssueModel.display.select),
            note: selPad(NoteModel.display.select),
            organization: selPad(OrganizationModel.display.select),
            post: selPad(PostModel.display.select),
            project: selPad(ProjectModel.display.select),
            question: selPad(QuestionModel.display.select),
            questionAnswer: selPad(QuestionAnswerModel.display.select),
            quiz: selPad(QuizModel.display.select),
            routine: selPad(RoutineModel.display.select),
            smartContract: selPad(SmartContractModel.display.select),
            standard: selPad(StandardModel.display.select),
            tag: selPad(TagModel.display.select),
            user: selPad(UserModel.display.select),
        }),
        label: (select, languages) => {
            if (select.api)
                return ApiModel.display.label(select.api, languages);
            if (select.comment)
                return CommentModel.display.label(select.comment, languages);
            if (select.issue)
                return IssueModel.display.label(select.issue, languages);
            if (select.note)
                return NoteModel.display.label(select.note, languages);
            if (select.organization)
                return OrganizationModel.display.label(select.organization, languages);
            if (select.post)
                return PostModel.display.label(select.post, languages);
            if (select.project)
                return ProjectModel.display.label(select.project, languages);
            if (select.question)
                return QuestionModel.display.label(select.question, languages);
            if (select.questionAnswer)
                return QuestionAnswerModel.display.label(select.questionAnswer, languages);
            if (select.quiz)
                return QuizModel.display.label(select.quiz, languages);
            if (select.routine)
                return RoutineModel.display.label(select.routine, languages);
            if (select.smartContract)
                return SmartContractModel.display.label(select.smartContract, languages);
            if (select.standard)
                return StandardModel.display.label(select.standard, languages);
            if (select.tag)
                return TagModel.display.label(select.tag, languages);
            if (select.user)
                return UserModel.display.label(select.user, languages);
            return "";
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            by: "User",
            list: "BookmarkList",
            to: {
                api: "Api",
                comment: "Comment",
                issue: "Issue",
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
            },
        },
        prismaRelMap: {
            __typename,
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
        },
        countFields: {},
        supplemental: {
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
            onCreated: async ({ created, prisma, userData }) => {
                for (const c of created) {
                    const [objectRel, objectId] = findFirstRel(c, ["apiId", "commentId", "issueId", "noteId", "organizationId", "postId", "projectId", "questionId", "questionAnswerId", "quizId", "routineId", "smartContractId", "standardId", "tagId", "userId"]);
                    if (!objectRel || !objectId)
                        return;
                    const objectType = uppercaseFirstLetter(objectRel.slice(0, -2));
                    const { delegate } = getLogic(["delegate"], objectType, userData.languages, "bookmark onCreated");
                    await delegate(prisma).update({ where: { id: objectId }, data: { bookmarks: { increment: 1 } } });
                    Trigger(prisma, userData.languages).objectBookmark(true, objectType, objectId, userData.id);
                }
            },
            beforeDeleted: async ({ deletingIds, prisma }) => {
                const deleting = await prisma.bookmark.findMany({
                    where: { id: { in: deletingIds } },
                    select: { apiId: true, commentId: true, issueId: true, noteId: true, organizationId: true, postId: true, projectId: true, questionId: true, questionAnswerId: true, quizId: true, routineId: true, smartContractId: true, standardId: true, tagId: true, userId: true },
                });
                const bookmarkedPairs = deleting.map(c => {
                    const [objectRel, objectId] = findFirstRel(c, ["apiId", "commentId", "issueId", "noteId", "organizationId", "postId", "projectId", "questionId", "questionAnswerId", "quizId", "routineId", "smartContractId", "standardId", "tagId", "userId"]);
                    if (!objectRel || !objectId)
                        return [null, null];
                    const objectType = uppercaseFirstLetter(objectRel.slice(0, -2));
                    return [objectType, objectId];
                }).filter(([objectType, objectId]) => objectType && objectId);
                const grouped = bookmarkedPairs.reduce((acc, [objectType, objectId]) => {
                    if (!acc[objectType])
                        acc[objectType] = [];
                    acc[objectType].push(objectId);
                    return acc;
                }, {});
                return grouped;
            },
            onDeleted: async ({ beforeDeletedData, prisma, userData }) => {
                for (const [objectType, objectIds] of Object.entries(beforeDeletedData)) {
                    const { delegate } = getLogic(["delegate"], objectType, userData.languages, "bookmark onDeleted");
                    await delegate(prisma).updateMany({ where: { id: { in: objectIds } }, data: { bookmarks: { decrement: 1 } } });
                    for (const objectId of objectIds) {
                        Trigger(prisma, userData.languages).objectBookmark(false, objectType, objectId, userData.id);
                    }
                }
            },
        },
        yup: bookmarkValidation,
    },
    query: {
        async getIsBookmarkeds(prisma, userId, ids, bookmarkFor) {
            const result = new Array(ids.length).fill(false);
            if (!userId)
                return result;
            const idsFiltered = onlyValidIds(ids);
            const fieldName = `${bookmarkFor.toLowerCase()}Id`;
            const isBookmarkredArray = await prisma.bookmark.findMany({ where: { list: { user: { id: userId } }, [fieldName]: { in: idsFiltered } } });
            for (let i = 0; i < ids.length; i++) {
                if (exists(ids[i]) && isBookmarkredArray.find((bookmark) => bookmark[fieldName] === ids[i])) {
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
                { list: BookmarkListModel.search.searchStringQuery() },
                { api: ApiModel.search.searchStringQuery() },
                { comment: CommentModel.search.searchStringQuery() },
                { issue: IssueModel.search.searchStringQuery() },
                { note: NoteModel.search.searchStringQuery() },
                { organization: OrganizationModel.search.searchStringQuery() },
                { post: PostModel.search.searchStringQuery() },
                { project: ProjectModel.search.searchStringQuery() },
                { question: QuestionModel.search.searchStringQuery() },
                { questionAnswer: QuestionAnswerModel.search.searchStringQuery() },
                { quiz: QuizModel.search.searchStringQuery() },
                { routine: RoutineModel.search.searchStringQuery() },
                { smartContract: SmartContractModel.search.searchStringQuery() },
                { standard: StandardModel.search.searchStringQuery() },
                { tag: TagModel.search.searchStringQuery() },
                { user: UserModel.search.searchStringQuery() },
            ],
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => BookmarkListModel.validate.owner(data.list, userId),
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
                list: BookmarkListModel.validate.visibility.owner(userId),
            }),
        },
    },
});
//# sourceMappingURL=bookmark.js.map