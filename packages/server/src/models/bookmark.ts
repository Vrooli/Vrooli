import { BookmarkCreateInput, BookmarkFor, BookmarkSortBy, BookmarkUpdateInput } from "@shared/consts";
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
import { ApiModel, IssueModel, PostModel, QuestionAnswerModel, QuestionModel, QuizModel, SmartContractModel, UserModel } from ".";
import { SelectWrap } from "../builders/types";
import { onlyValidIds, selPad } from "../builders";
import { NoteModel } from "./note";
import { exists } from "@shared/utils";

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
            by: 'User',
            api: 'Api',
            comment: 'Comment',
            issue: 'Issue',
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
    },
    mutate: {} as any,
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
            const isBookmarkredArray = await prisma.bookmark.findMany({ where: { byId: userId, [fieldName]: { in: idsFiltered } } });
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
                'labelWrapped',
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
    validate: {} as any,
    // TODO replace with mutate and validate. Trigger should update "bookmarks" count of object being bookmarkred
    // bookmark: async (prisma: PrismaType, userData: SessionUser, input: BookmarkInput): Promise<boolean> => {
    //     prisma.bookmark.findMany({
    //         where: {
    //             tagId: null
    //         }
    //     })
    //     // Get prisma delegate for type of object being bookmarkred
    //     const { delegate } = getLogic(['delegate'], input.bookmarkFor, userData.languages, 'bookmark');
    //     // Check if object being bookmarkred exists
    //     const bookmarkringFor: null | { id: string, bookmarks: number } = await delegate(prisma).findUnique({ where: { id: input.forConnect }, select: { id: true, bookmarks: true } }) as any;
    //     if (!bookmarkringFor)
    //         throw new CustomError('0110', 'ErrorUnknown', userData.languages, { bookmarkFor: input.bookmarkFor, forId: input.forConnect });
    //     // Check if bookmark already exists on object by this user TODO fix for tags
    //     const bookmark = await prisma.bookmark.findFirst({
    //         where: {
    //             byId: userData.id,
    //             [`${forMapper[input.bookmarkFor]}Id`]: input.forConnect
    //         }
    //     })
    //     // If bookmark already existed and we want to bookmark, 
    //     // or if bookmark did not exist and we don't want to bookmark, skip
    //     if ((bookmark && input.isBookmark) || (!bookmark && !input.isBookmark)) return true;
    //     // If bookmark did not exist and we want to bookmark, create
    //     if (!bookmark && input.isBookmark) {
    //         // Create
    //         await prisma.bookmark.create({
    //             data: {
    //                 byId: userData.id,
    //                 [`${forMapper[input.bookmarkFor]}Id`]: input.forConnect
    //             }
    //         })
    //         // Increment bookmark count on object
    //         await delegate(prisma).update({
    //             where: { id: input.forConnect },
    //             data: { bookmarks: bookmarkringFor.bookmarks + 1 }
    //         })
    //         // Handle trigger
    //         await Trigger(prisma, userData.languages).objectBookmark(true, input.bookmarkFor, input.forConnect, userData.id);
    //     }
    //     // If bookmark did exist and we don't want to bookmark, delete
    //     else if (bookmark && !input.isBookmark) {
    //         // Delete bookmark
    //         await prisma.bookmark.delete({ where: { id: bookmark.id } })
    //         // Decrement bookmark count on object
    //         await delegate(prisma).update({
    //             where: { id: input.forConnect },
    //             data: { bookmarks: bookmarkringFor.bookmarks - 1 }
    //         })
    //         // Handle trigger
    //         await Trigger(prisma, userData.languages).objectBookmark(false, input.bookmarkFor, input.forConnect, userData.id);
    //     }
    //     return true;
    // }
})