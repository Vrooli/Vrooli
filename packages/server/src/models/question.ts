import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MaxObjects, Question, QuestionCreateInput, QuestionForType, QuestionSearchInput, QuestionSortBy, QuestionUpdateInput, QuestionYou } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel, defaultPermissions, onCommonPlain, tagShapeHelper, translationShapeHelper } from "../utils";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";
import { BookmarkModel } from "./bookmark";
import { VoteModel } from "./vote";
import { questionValidation } from "@shared/validation";
import { noNull } from "../builders";

const forMapper: { [key in QuestionForType]: keyof Prisma.questionUpsertArgs['create'] } = {
    Api: 'api',
    Note: 'note',
    Organization: 'organization',
    Project: 'project',
    Routine: 'routine',
    SmartContract: 'smartContract',
    Standard: 'standard',
}

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
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                referencing: noNull(data.referencing),
                createdBy: { connect: { id: rest.userData.id } },
                [forMapper[data.forType]]: { connect: { id: data.forConnect } },
                ...(await tagShapeHelper({ relTypes: ['Connect', 'Create'], parentType: 'Question', relation: 'tags', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                ...(data.acceptedAnswerConnect ? {
                    answers: {
                        update: {
                            where: { id: data.acceptedAnswerConnect },
                            data: { isAccepted: true },
                        },
                    },
                } : {}),
                ...(await tagShapeHelper({ relTypes: ['Connect', 'Create', 'Disconnect'], parentType: 'Question', relation: 'tags', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, ...rest })),
            })
        },
        trigger: {
            onCommon: async (params) => {
                await onCommonPlain({
                    ...params,
                    objectType: __typename,
                    ownerUserField: 'createdBy',
                });
            },
        },
        yup: questionValidation,
    },
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
            private: { isPrivate: true, },
            public: { isPrivate: false },
            owner: (userId) => ({
                createdBy: { id: userId },
            }),
        },
    },
})