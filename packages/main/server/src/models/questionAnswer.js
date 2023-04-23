import { MaxObjects, QuestionAnswerSortBy } from "@local/consts";
import { questionAnswerValidation } from "@local/validation";
import { shapeHelper } from "../builders";
import { bestLabel, defaultPermissions, translationShapeHelper } from "../utils";
const __typename = "QuestionAnswer";
const suppFields = [];
export const QuestionAnswerModel = ({
    __typename,
    delegate: (prisma) => prisma.question_answer,
    display: {
        select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, "name", languages),
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
//# sourceMappingURL=questionAnswer.js.map