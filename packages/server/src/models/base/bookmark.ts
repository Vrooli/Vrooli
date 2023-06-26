import { BookmarkFor, BookmarkSortBy, bookmarkValidation, exists, GqlModelType, lowercaseFirstLetter, MaxObjects, uppercaseFirstLetter } from "@local/shared";
import { Prisma } from "@prisma/client";
import { ApiModel, BookmarkListModel, IssueModel, PostModel, QuestionAnswerModel, QuestionModel, QuizModel, SmartContractModel, UserModel } from ".";
import { findFirstRel, onlyValidIds, selPad, shapeHelper } from "../../builders";
import { Trigger } from "../../events";
import { getLogic } from "../../getters";
import { PrismaType } from "../../types";
import { defaultPermissions } from "../../utils";
import { BookmarkFormat } from "../format/bookmark";
import { ModelLogic } from "../types";
import { CommentModel } from "./comment";
import { NoteModel } from "./note";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { StandardModel } from "./standard";
import { TagModel } from "./tag";
import { BookmarkModelLogic } from "./types";

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
const suppFields = [] as const;
export const BookmarkModel: ModelLogic<BookmarkModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.bookmark,
    display: {
        label: {
            select: () => ({
                id: true,
                api: selPad(ApiModel.display.label.select),
                comment: selPad(CommentModel.display.label.select),
                issue: selPad(IssueModel.display.label.select),
                note: selPad(NoteModel.display.label.select),
                organization: selPad(OrganizationModel.display.label.select),
                post: selPad(PostModel.display.label.select),
                project: selPad(ProjectModel.display.label.select),
                question: selPad(QuestionModel.display.label.select),
                questionAnswer: selPad(QuestionAnswerModel.display.label.select),
                quiz: selPad(QuizModel.display.label.select),
                routine: selPad(RoutineModel.display.label.select),
                smartContract: selPad(SmartContractModel.display.label.select),
                standard: selPad(StandardModel.display.label.select),
                tag: selPad(TagModel.display.label.select),
                user: selPad(UserModel.display.label.select),
            }),
            get: (select, languages) => {
                if (select.api) return ApiModel.display.label.get(select.api as any, languages);
                if (select.comment) return CommentModel.display.label.get(select.comment as any, languages);
                if (select.issue) return IssueModel.display.label.get(select.issue as any, languages);
                if (select.note) return NoteModel.display.label.get(select.note as any, languages);
                if (select.organization) return OrganizationModel.display.label.get(select.organization as any, languages);
                if (select.post) return PostModel.display.label.get(select.post as any, languages);
                if (select.project) return ProjectModel.display.label.get(select.project as any, languages);
                if (select.question) return QuestionModel.display.label.get(select.question as any, languages);
                if (select.questionAnswer) return QuestionAnswerModel.display.label.get(select.questionAnswer as any, languages);
                if (select.quiz) return QuizModel.display.label.get(select.quiz as any, languages);
                if (select.routine) return RoutineModel.display.label.get(select.routine as any, languages);
                if (select.smartContract) return SmartContractModel.display.label.get(select.smartContract as any, languages);
                if (select.standard) return StandardModel.display.label.get(select.standard as any, languages);
                if (select.tag) return TagModel.display.label.get(select.tag as any, languages);
                if (select.user) return UserModel.display.label.get(select.user as any, languages);
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
            onCreated: async ({ created, prisma, userData }) => {
                for (const c of created) {
                    // Find type and id of bookmarked object
                    const [objectRel, objectId] = findFirstRel(c, ["apiId", "commentId", "issueId", "noteId", "organizationId", "postId", "projectId", "questionId", "questionAnswerId", "quizId", "routineId", "smartContractId", "standardId", "tagId", "userId"]);
                    if (!objectRel || !objectId) return;
                    // Object type is objectRel with "Id" removed and first letter capitalized
                    const objectType: BookmarkFor = uppercaseFirstLetter(objectRel.slice(0, -2)) as BookmarkFor;
                    // Update "bookmarks" count for bookmarked object
                    const { delegate } = getLogic(["delegate"], objectType, userData.languages, "bookmark onCreated");
                    await delegate(prisma).update({ where: { id: objectId }, data: { bookmarks: { increment: 1 } } });
                    // Trigger bookmarkCreated event
                    Trigger(prisma, userData.languages).objectBookmark(true, objectType, objectId, userData.id);
                }
            },
            beforeDeleted: async ({ deletingIds, prisma }) => {
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
                return grouped;
            },
            onDeleted: async ({ beforeDeletedData, prisma, userData }) => {
                // For each bookmarked object type, decrement the bookmark count
                for (const [objectType, objectIds] of Object.entries(beforeDeletedData)) {
                    const { delegate } = getLogic(["delegate"], objectType as GqlModelType, userData.languages, "bookmark onDeleted");
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
                { list: BookmarkListModel.search!.searchStringQuery() },
                { api: ApiModel.search!.searchStringQuery() },
                { comment: CommentModel.search!.searchStringQuery() },
                { issue: IssueModel.search!.searchStringQuery() },
                { note: NoteModel.search!.searchStringQuery() },
                { organization: OrganizationModel.search!.searchStringQuery() },
                { post: PostModel.search!.searchStringQuery() },
                { project: ProjectModel.search!.searchStringQuery() },
                { question: QuestionModel.search!.searchStringQuery() },
                { questionAnswer: QuestionAnswerModel.search!.searchStringQuery() },
                { quiz: QuizModel.search!.searchStringQuery() },
                { routine: RoutineModel.search!.searchStringQuery() },
                { smartContract: SmartContractModel.search!.searchStringQuery() },
                { standard: StandardModel.search!.searchStringQuery() },
                { tag: TagModel.search!.searchStringQuery() },
                { user: UserModel.search!.searchStringQuery() },
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
        owner: (data, userId) => BookmarkListModel.validate.owner(data.list as any, userId),
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
