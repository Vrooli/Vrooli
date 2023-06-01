import { MaxObjects, QuestionAnswerSortBy, questionAnswerValidation } from "@local/shared";
import { shapeHelper } from "../builders";
import { PrismaType } from "../types";
import { bestTranslation, defaultPermissions, translationShapeHelper } from "../utils";
import { ModelLogic, QuestionAnswerModelLogic } from "./types";

const __typename = "QuestionAnswer" as const;
const suppFields = [] as const;
export const QuestionAnswerModel: ModelLogic<QuestionAnswerModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.question_answer,
    display: {
        label: {
            select: () => ({ id: true, callLink: true, translations: { select: { language: true, text: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.text ?? "",
        },
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
