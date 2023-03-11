import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MaxObjects, QuestionAnswer, QuestionAnswerCreateInput, QuestionAnswerSearchInput, QuestionAnswerSortBy, QuestionAnswerUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel, defaultPermissions, translationShapeHelper } from "../utils";
import { ModelLogic } from "./types";
import { questionAnswerValidation } from "@shared/validation";
import { shapeHelper } from "../builders";

const __typename = 'QuestionAnswer' as const;
const suppFields = [] as const;
export const QuestionAnswerModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuestionAnswerCreateInput,
    GqlUpdate: QuestionAnswerUpdateInput,
    GqlModel: QuestionAnswer,
    GqlSearch: QuestionAnswerSearchInput,
    GqlSort: QuestionAnswerSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.question_answerUpsertArgs['create'],
    PrismaUpdate: Prisma.question_answerUpsertArgs['update'],
    PrismaModel: Prisma.question_answerGetPayload<SelectWrap<Prisma.question_answerSelect>>,
    PrismaSelect: Prisma.question_answerSelect,
    PrismaWhere: Prisma.question_answerWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.question_answer,
    display: {
        select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations as any, 'name', languages)
    },
    format: {
        gqlRelMap: {
            __typename,
            bookmarkedBy: 'User',
            createdBy: 'User',
            comments: 'Comment',
            question: 'Question',
        },
        prismaRelMap: {
            __typename,
            bookmarkedBy: 'User',
            createdBy: 'User',
            comments: 'Comment',
            question: 'Question',
            votedBy: 'Vote',
        },
        countFields: {
            commentsCount: true,
        },
        joinMap: { bookmarkedBy: 'user' },
    },
    mutate: {
        shape: {
            create: async ({ data, prisma, userData }) => ({
                id: data.id,
                createdBy: { connect: { id: userData.id } },
                ...(await shapeHelper({ relation: 'question', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'Question', parentRelationshipName: 'answers', data, prisma, userData })),
                ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, prisma, userData })),
            }),
            update: async ({ data, prisma, userData }) => ({
                ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, prisma, userData })),
            })
        },
        yup: questionAnswerValidation,
    },
    search: {
        defaultSort: QuestionAnswerSortBy.DateUpdatedDesc,
        sortBy: QuestionAnswerSortBy,
        searchFields: {
            createdTimeFrame: true,
            excludeIds: true,
            translationLanguages: true,
            minScore: true,
            minBookmarks: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                'transTextWrapped',
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