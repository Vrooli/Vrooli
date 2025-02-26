import { DEFAULT_LANGUAGE, MaxObjects, QuizQuestionResponseSortBy, quizQuestionResponseValidation } from "@local/shared";
import i18next from "i18next";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { QuizQuestionResponseFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { QuizAttemptModelInfo, QuizAttemptModelLogic, QuizQuestionModelInfo, QuizQuestionModelLogic, QuizQuestionResponseModelInfo, QuizQuestionResponseModelLogic } from "./types.js";

const __typename = "QuizQuestionResponse" as const;
export const QuizQuestionResponseModel: QuizQuestionResponseModelLogic = ({
    __typename,
    dbTable: "quiz_question_response",
    display: () => ({
        label: {
            select: () => ({ id: true, quizQuestion: { select: ModelMap.get<QuizQuestionModelLogic>("QuizQuestion").display().label.select() } }),
            get: (select, languages) => i18next.t("common:QuizQuestionResponseLabel", {
                lng: languages && languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE,
                questionLabel: ModelMap.get<QuizQuestionModelLogic>("QuizQuestion").display().label.get(select.quizQuestion as QuizQuestionModelInfo["DbModel"], languages),
            }),
        },
    }),
    format: QuizQuestionResponseFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                response: data.response,
                quizAttempt: await shapeHelper({ relation: "quizAttempt", relTypes: ["Connect"], isOneToOne: true, objectType: "QuizAttempt", parentRelationshipName: "responses", data, ...rest }),
                quizQuestion: await shapeHelper({ relation: "quizQuestion", relTypes: ["Connect"], isOneToOne: true, objectType: "QuizQuestion", parentRelationshipName: "responses", data, ...rest }),
            }),
            update: async ({ data }) => ({
                response: noNull(data.response),
            }),
        },
        yup: quizQuestionResponseValidation,
    },
    search: {
        defaultSort: QuizQuestionResponseSortBy.QuestionOrderAsc,
        sortBy: QuizQuestionResponseSortBy,
        searchFields: {
            createdTimeFrame: true,
            quizAttemptId: true,
            quizQuestionId: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transResponseWrapped",
            ],
        }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<QuizQuestionResponseModelInfo["ApiPermission"]>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<QuizQuestionResponseModelInfo["DbSelect"]>([["quizAttempt", "QuizAttempt"]], ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<QuizAttemptModelLogic>("QuizAttempt").validate().owner(data?.quizAttempt as QuizAttemptModelInfo["DbModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            quizAttempt: "QuizAttempt",
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    quizAttempt: useVisibility("QuizAttempt", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    quizAttempt: useVisibility("QuizAttempt", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    quizAttempt: useVisibility("QuizAttempt", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    quizAttempt: useVisibility("QuizAttempt", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    quizAttempt: useVisibility("QuizAttempt", "Public", data),
                };
            },
        },
    }),
});
