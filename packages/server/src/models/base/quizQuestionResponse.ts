import { MaxObjects, QuizQuestionResponseSortBy, quizQuestionResponseValidation } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { QuizQuestionResponseFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { QuizAttemptModelInfo, QuizAttemptModelLogic, QuizQuestionModelInfo, QuizQuestionModelLogic, QuizQuestionResponseModelInfo, QuizQuestionResponseModelLogic } from "./types";

const __typename = "QuizQuestionResponse" as const;
export const QuizQuestionResponseModel: QuizQuestionResponseModelLogic = ({
    __typename,
    dbTable: "quiz_question_response",
    display: () => ({
        label: {
            select: () => ({ id: true, quizQuestion: { select: ModelMap.get<QuizQuestionModelLogic>("QuizQuestion").display().label.select() } }),
            get: (select, languages) => i18next.t("common:QuizQuestionResponseLabel", {
                lng: languages.length > 0 ? languages[0] : "en",
                questionLabel: ModelMap.get<QuizQuestionModelLogic>("QuizQuestion").display().label.get(select.quizQuestion as QuizQuestionModelInfo["PrismaModel"], languages),
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
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<QuizQuestionResponseModelInfo["PrismaSelect"]>([["quizAttempt", "QuizAttempt"]], ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<QuizAttemptModelLogic>("QuizAttempt").validate().owner(data?.quizAttempt as QuizAttemptModelInfo["PrismaModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            quizAttempt: "QuizAttempt",
        }),
        visibility: {
            private: function getVisibilityPrivate() {
                return {};
            },
            public: function getVisibilityPublic() {
                return {};
            },
            owner: (userId) => ({
                quizAttempt: ModelMap.get<QuizAttemptModelLogic>("QuizAttempt").validate().visibility.owner(userId),
            }),
        },
    }),
});
