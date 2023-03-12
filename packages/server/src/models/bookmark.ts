import { BookmarkCreateInput, BookmarkFor, BookmarkSortBy, BookmarkUpdateInput, GqlModelType, MaxObjects } from "@shared/consts";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { StandardModel } from "./standard";
import { TagModel } from "./tag";
import { CommentModel } from "./comment";
import { Bookmark, BookmarkSearchInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { ApiModel, BookmarkListModel, IssueModel, PostModel, QuestionAnswerModel, QuestionModel, QuizModel, SmartContractModel, UserModel } from ".";
import { SelectWrap } from "../builders/types";
import { findFirstRel, onlyValidIds, selPad, shapeHelper, uppercaseFirstLetter } from "../builders";
import { NoteModel } from "./note";
import { exists } from "@shared/utils";
import { bookmarkValidation } from "@shared/validation";
import { Trigger } from "../events";
import { getLogic } from "../getters";
import { defaultPermissions } from "../utils";

const forMapper: { [key in BookmarkFor]: keyof Prisma.bookmarkUpsertArgs['create'] } = {
    Api: 'api',
    Comment: 'comment',
    Issue: 'issue',
    Note: 'note',
    Organization: 'organization',
    Post: 'post',
    Project: 'project',
    Question: 'question',
    QuestionAnswer: 'questionAnswer',
    Quiz: 'quiz',
    Routine: 'routine',
    SmartContract: 'smartContract',
    Standard: 'standard',
    Tag: 'tag',
    User: 'user',
}

const __typename = 'Bookmark' as const;
const suppFields = [] as const;
export const BookmarkModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: BookmarkCreateInput,
    GqlUpdate: BookmarkUpdateInput,
    GqlModel: Bookmark,
    GqlSearch: BookmarkSearchInput,
    GqlSort: BookmarkSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.bookmarkUpsertArgs['create'],
    PrismaUpdate: Prisma.bookmarkUpsertArgs['update'],
    PrismaModel: Prisma.bookmarkGetPayload<SelectWrap<Prisma.bookmarkSelect>>,
    PrismaSelect: Prisma.bookmarkSelect,
    PrismaWhere: Prisma.bookmarkWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.bookmark,
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
            if (select.api) return ApiModel.display.label(select.api as any, languages);
            if (select.comment) return CommentModel.display.label(select.comment as any, languages);
            if (select.issue) return IssueModel.display.label(select.issue as any, languages);
            if (select.note) return NoteModel.display.label(select.note as any, languages);
            if (select.organization) return OrganizationModel.display.label(select.organization as any, languages);
            if (select.post) return PostModel.display.label(select.post as any, languages);
            if (select.project) return ProjectModel.display.label(select.project as any, languages);
            if (select.question) return QuestionModel.display.label(select.question as any, languages);
            if (select.questionAnswer) return QuestionAnswerModel.display.label(select.questionAnswer as any, languages);
            if (select.quiz) return QuizModel.display.label(select.quiz as any, languages);
            if (select.routine) return RoutineModel.display.label(select.routine as any, languages);
            if (select.smartContract) return SmartContractModel.display.label(select.smartContract as any, languages);
            if (select.standard) return StandardModel.display.label(select.standard as any, languages);
            if (select.tag) return TagModel.display.label(select.tag as any, languages);
            if (select.user) return UserModel.display.label(select.user as any, languages);
            return '';
        }
    },
    format: {
        gqlRelMap: {
            __typename,
            by: 'User',
            to: {
                api: 'Api',
                comment: 'Comment',
                issue: 'Issue',
                list: 'BookmarkList',
                note: 'Note',
                organization: 'Organization',
                post: 'Post',
                project: 'Project',
                question: 'Question',
                questionAnswer: 'QuestionAnswer',
                quiz: 'Quiz',
                routine: 'Routine',
                smartContract: 'SmartContract',
                standard: 'Standard',
                tag: 'Tag',
                user: 'User',
            }
        },
        prismaRelMap: {
            __typename,
            api: 'Api',
            comment: 'Comment',
            issue: 'Issue',
            list: 'BookmarkList',
            note: 'Note',
            organization: 'Organization',
            post: 'Post',
            project: 'Project',
            question: 'Question',
            questionAnswer: 'QuestionAnswer',
            quiz: 'Quiz',
            routine: 'Routine',
            smartContract: 'SmartContract',
            standard: 'Standard',
            tag: 'Tag',
            user: 'User',
        },
        countFields: {},
        supplemental: {
            // Make sure to query for every bookmarked item, 
            // so we can ensure that the mutation trigger can increment the bookmark count
            dbFields: [
                'apiId',
                'commentId',
                'issueId',
                'noteId',
                'organizationId',
                'postId',
                'projectId',
                'questionId',
                'questionAnswerId',
                'quizId',
                'routineId',
                'smartContractId',
                'standardId',
                'tagId',
                'userId',
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
                ...(await shapeHelper({ relation: 'list', relTypes: ['Connect', 'Create'], isOneToOne: true, isRequired: true, objectType: 'BookmarkList', parentRelationshipName: 'bookmarks', data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                id: data.id,
                ...(await shapeHelper({ relation: 'list', relTypes: ['Connect', 'Update'], isOneToOne: true, isRequired: false, objectType: 'BookmarkList', parentRelationshipName: 'bookmarks', data, ...rest })),
            })
        },
        trigger: {
            onCreated: async ({ created, prisma, userData }) => {
                for (const c of created) {
                    // Find type and id of bookmarked object
                    const [objectRel, objectId] = findFirstRel(c, ['apiId', 'commentId', 'issueId', 'noteId', 'organizationId', 'postId', 'projectId', 'questionId', 'questionAnswerId', 'quizId', 'routineId', 'smartContractId', 'standardId', 'tagId', 'userId'])
                    if (!objectRel || !objectId) return;
                    // Object type is objectRel with "Id" removed and first letter capitalized
                    const objectType: BookmarkFor = uppercaseFirstLetter(objectRel.slice(0, -2)) as BookmarkFor;
                    // Update "bookmarks" count for bookmarked object
                    const { delegate } = getLogic(['delegate'], objectType, userData.languages, 'bookmark onCreated');
                    await delegate(prisma).update({ where: { id: objectId }, data: { bookmarks: { increment: 1 } } });
                    // Trigger bookmarkCreated event
                    Trigger(prisma, userData.languages).objectBookmark(true, objectType, objectId, userData.id)
                }
            },
            beforeDeleted: async ({ deletingIds, prisma }) => {
                // Grab bookmarked object id and type
                const deleting = await prisma.bookmark.findMany({
                    where: { id: { in: deletingIds } },
                    select: { apiId: true, commentId: true, issueId: true, noteId: true, organizationId: true, postId: true, projectId: true, questionId: true, questionAnswerId: true, quizId: true, routineId: true, smartContractId: true, standardId: true, tagId: true, userId: true }
                });
                // Find type and id of bookmarked object
                const bookmarkedPairs: [BookmarkFor, string][] = deleting.map(c => {
                    const [objectRel, objectId] = findFirstRel(c, ['apiId', 'commentId', 'issueId', 'noteId', 'organizationId', 'postId', 'projectId', 'questionId', 'questionAnswerId', 'quizId', 'routineId', 'smartContractId', 'standardId', 'tagId', 'userId'])
                    if (!objectRel || !objectId) return [null, null];
                    // Object type is objectRel with "Id" removed and first letter capitalized
                    const objectType: BookmarkFor = uppercaseFirstLetter(objectRel.slice(0, -2)) as BookmarkFor;
                    return [objectType, objectId]
                }).filter(([objectType, objectId]) => objectType && objectId) as [BookmarkFor, string][];
                // Group by object type
                const grouped: { [key in BookmarkFor]?: string[] } = bookmarkedPairs.reduce((acc, [objectType, objectId]) => {
                    if (!acc[objectType]) acc[objectType] = [];
                    acc[objectType].push(objectId);
                    return acc;
                }, {});
                return grouped
            },
            onDeleted: async ({ beforeDeletedData, prisma, userData }) => {
                // For each bookmarked object type, decrement the bookmark count
                for (const [objectType, objectIds] of Object.entries(beforeDeletedData)) {
                    const { delegate } = getLogic(['delegate'], objectType as GqlModelType, userData.languages, 'bookmark onDeleted');
                    await (delegate(prisma) as any).updateMany({ where: { id: { in: objectIds } }, data: { bookmarks: { decrement: 1 } } });
                    // For each bookmarked object, trigger bookmarkDeleted event
                    for (const objectId of (objectIds as string[])) {
                        Trigger(prisma, userData.languages).objectBookmark(false, objectType as BookmarkFor, objectId, userData.id)
                    }
                }
            }
        },
        yup: bookmarkValidation,
    },
    query: {
        async getIsBookmarkeds(
            prisma: PrismaType,
            userId: string | null | undefined,
            ids: string[],
            bookmarkFor: keyof typeof BookmarkFor
        ): Promise<boolean[]> {
            // Create result array that is the same length as ids
            const result = new Array(ids.length).fill(false);
            // If userId not passed, return result
            if (!userId) return result;
            // Filter out nulls and undefineds from ids
            const idsFiltered = onlyValidIds(ids);
            const fieldName = `${bookmarkFor.toLowerCase()}Id`;
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
            excludeLinkedToTag: true,
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
            ]
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => BookmarkListModel.validate!.owner(data.list as any),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            api: 'Api',
            comment: 'Comment',
            issue: 'Issue',
            list: 'BookmarkList',
            note: 'Note',
            organization: 'Organization',
            post: 'Post',
            project: 'Project',
            question: 'Question',
            questionAnswer: 'QuestionAnswer',
            quiz: 'Quiz',
            routine: 'Routine',
            smartContract: 'SmartContract',
            standard: 'Standard',
            tag: 'Tag',
            user: 'User',
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                list: BookmarkListModel.validate!.visibility.owner(userId),
            }),
        },
    },
})