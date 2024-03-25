import { MaxObjects, QuestionAnswerSortBy, questionAnswerValidation } from "@local/shared";
import { shapeHelper } from "../../builders/shapeHelper";
import { bestTranslation, defaultPermissions } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { QuestionAnswerFormat } from "../formats";
import { QuestionAnswerModelLogic } from "./types";

const __typename = "QuestionAnswer" as const;
export const QuestionAnswerModel: QuestionAnswerModelLogic = ({
    __typename,
    delegate: (p) => p.question_answer,
    display: () => ({
        label: {
            select: () => ({ id: true, callLink: true, translations: { select: { language: true, text: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.text ?? "",
        },
    }),
    format: QuestionAnswerFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                createdBy: { connect: { id: rest.userData.id } },
                question: await shapeHelper({ relation: "question", relTypes: ["Connect"], isOneToOne: true, objectType: "Question", parentRelationshipName: "answers", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest }),
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
    validate: () => ({
        isDeleted: () => false,
        isPublic: () => true,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data?.createdBy,
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
    }),
});
