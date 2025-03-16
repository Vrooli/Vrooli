import { MaxObjects, getTranslation, quizQuestionValidation } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { QuizQuestionFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { QuizModelInfo, QuizModelLogic, QuizQuestionModelInfo, QuizQuestionModelLogic } from "./types.js";

const __typename = "QuizQuestion" as const;
export const QuizQuestionModel: QuizQuestionModelLogic = ({
    __typename,
    dbTable: "quiz_question",
    dbTranslationTable: "quiz_question_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, questionText: true } } }),
            get: (select, languages) => getTranslation(select, languages).questionText ?? "",
        },
    }),
    format: QuizQuestionFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                order: noNull(data.order),
                points: noNull(data.points),
                standardVersion: await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "StandardVersion", parentRelationshipName: "quizQuestions", data, ...rest }),
                quiz: await shapeHelper({ relation: "quiz", relTypes: ["Connect"], isOneToOne: true, objectType: "Quiz", parentRelationshipName: "quizQuestions", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                order: noNull(data.order),
                points: noNull(data.points),
                standardVersion: await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create", "Update"], isOneToOne: true, objectType: "StandardVersion", parentRelationshipName: "quizQuestions", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest }),
            }),
        },
        yup: quizQuestionValidation,
    },
    search: undefined,
    validate: () => ({
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<QuizQuestionModelInfo["DbSelect"]>([["quiz", "Quiz"]], ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<QuizModelLogic>("Quiz").validate().owner(data?.quiz as QuizModelInfo["DbModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            quiz: "Quiz",
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    quiz: useVisibility("Quiz", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    quiz: useVisibility("Quiz", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    quiz: useVisibility("Quiz", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    quiz: useVisibility("Quiz", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    quiz: useVisibility("Quiz", "Public", data),
                };
            },
        },
    }),
});
