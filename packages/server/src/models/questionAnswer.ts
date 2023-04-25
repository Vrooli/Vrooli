import { MaxObjects, QuestionAnswer, QuestionAnswerCreateInput, QuestionAnswerSearchInput, QuestionAnswerSortBy, QuestionAnswerUpdateInput, questionAnswerValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel, defaultPermissions, translationShapeHelper } from "../utils";
import { ModelLogic } from "./types";

const __typename = "QuestionAnswer" as const;
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
    PrismaCreate: Prisma.question_answerUpsertArgs["create"],
    PrismaUpdate: Prisma.question_answerUpsertArgs["update"],
    PrismaModel: Prisma.question_answerGetPayload<SelectWrap<Prisma.question_answerSelect>>,
    PrismaSelect: Prisma.question_answerSelect,
    PrismaWhere: Prisma.question_answerWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.question_answer,
    display: {
        select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations as any, "name", languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            bookmarkedBy: "User",
            createdBy: "User",
            comments: "Comment",
            question: "Question",
        },
        prismaRelMap: {
            __typename,
            bookmarkedBy: "User",
            createdBy: "User",
            comments: "Comment",
            question: "Question",
            reactions: "Reaction",
        },
        countFields: {
            commentsCount: true,
        },
        joinMap: { bookmarkedBy: "user" },
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                createdBy: { connect: { id: rest.userData.id } },
                ...(await shapeHelper({ relation: "question", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Question", parentRelationshipName: "answers", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
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
                "transTextWrapped",
            ],
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
            createdBy: "User",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                createdBy: { id: userId },
            }),
        },
    },
});
