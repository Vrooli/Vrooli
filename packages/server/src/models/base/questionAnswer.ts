import { MaxObjects, QuestionAnswerSortBy, getTranslation, questionAnswerValidation } from "@local/shared";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { QuestionAnswerFormat } from "../formats.js";
import { QuestionAnswerModelLogic } from "./types.js";

const __typename = "QuestionAnswer" as const;
export const QuestionAnswerModel: QuestionAnswerModelLogic = ({
    __typename,
    dbTable: "question_answer",
    dbTranslationTable: "question_answer_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, text: true } } }),
            get: (select, languages) => getTranslation(select, languages).text ?? "",
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
            own: function getOwn(data) {
                return {
                    createdBy: { id: data.userId },
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        { createdBy: { id: data.userId } },
                        { question: useVisibility("Question", "Public", data) },
                    ],
                };
            },
            // Search method not useful for this object because answers are not explicitly set as private, so we'll return "Own"
            // TODO when drafts are added, this and related functions will have to change
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("QuestionAnswer", "Own", data);
            },
            ownPublic: function getOwnPublic(data) {
                return useVisibility("QuestionAnswer", "Own", data);
            },
            public: function getPublic(data) {
                return {
                    question: useVisibility("Question", "Public", data),
                };
            },
        },
    }),
});
