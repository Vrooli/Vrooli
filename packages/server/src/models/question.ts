import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MaxObjects, Question, QuestionCreateInput, QuestionSearchInput, QuestionSortBy, QuestionUpdateInput, QuestionYou } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel, defaultPermissions } from "../utils";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";
import { BookmarkModel } from "./bookmark";
import { VoteModel } from "./vote";

const __typename = 'Question' as const;
type Permissions = Pick<QuestionYou, 'canDelete' | 'canUpdate' | 'canBookmark' | 'canRead' | 'canVote'>;
const suppFields = ['you'] as const;
export const QuestionModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuestionCreateInput,
    GqlUpdate: QuestionUpdateInput,
    GqlModel: Question,
    GqlSearch: QuestionSearchInput,
    GqlSort: QuestionSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.questionUpsertArgs['create'],
    PrismaUpdate: Prisma.questionUpsertArgs['update'],
    PrismaModel: Prisma.questionGetPayload<SelectWrap<Prisma.questionSelect>>,
    PrismaSelect: Prisma.questionSelect,
    PrismaWhere: Prisma.questionWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.question,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            createdBy: 'User',
            answers: 'QuestionAnswer',
            comments: 'Comment',
            forObject: {
                api: 'Api',
                note: 'Note',
                organization: 'Organization',
                project: 'Project',
                routine: 'Routine',
                smartContract: 'SmartContract',
                standard: 'Standard',
            },
            reports: 'Report',
            bookmarkedBy: 'User',
            tags: 'Tag',
        },
        prismaRelMap: {
            __typename,
            createdBy: 'User',
            api: 'Api',
            note: 'Note',
            organization: 'Organization',
            project: 'Project',
            routine: 'Routine',
            smartContract: 'SmartContract',
            standard: 'Standard',
            comments: 'Comment',
            answers: 'QuestionAnswer',
            reports: 'Report',
            tags: 'Tag',
            bookmarkedBy: 'User',
            votedBy: 'User',
            viewedBy: 'User',
        },
        joinMap: { bookmarkedBy: 'user', tags: 'tag' },
        countFields: {
            answersCount: true,
            commentsCount: true,
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
                        isUpvoted: await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename),
                    },
                }
            },
        },
    },
    mutate: {} as any,
    search: {
        defaultSort: QuestionSortBy.ScoreDesc,
        sortBy: QuestionSortBy,
        searchFields: {
            createdTimeFrame: true,
            excludeIds: true,
            hasAcceptedAnswer: true,
            createdById: true,
            apiId: true,
            noteId: true,
            organizationId: true,
            projectId: true,
            routineId: true,
            smartContractId: true,
            standardId: true,
            translationLanguages: true,
            maxScore: true,
            maxBookmarks: true,
            minScore: true,
            minBookmarks: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                'descriptionWrapped',
                'nameWrapped',
            ]
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => true,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data.createdBy,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            createdBy: 'User',
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                createdBy: { id: userId },
            }),
        },
    },
})