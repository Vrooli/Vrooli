import { MaxObjects, QuizQuestionResponseSortBy } from "@local/consts";
import { quizQuestionResponseValidation } from "@local/validation";
import i18next from "i18next";
import { noNull, selPad, shapeHelper } from "../builders";
import { defaultPermissions } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { QuizAttemptModel } from "./quizAttempt";
import { QuizQuestionModel } from "./quizQuestion";
const __typename = "QuizQuestionResponse";
const suppFields = ["you"];
export const QuizQuestionResponseModel = ({
    __typename,
    delegate: (prisma) => prisma.quiz_question_response,
    display: {
        select: () => ({ id: true, quizQuestion: selPad(QuizQuestionModel.display.select) }),
        label: (select, languages) => i18next.t("common:QuizQuestionResponseLabel", {
            lng: languages.length > 0 ? languages[0] : "en",
            questionLabel: QuizQuestionModel.display.label(select.quizQuestion, languages),
        }),
    },
    format: {
        gqlRelMap: {
            __typename,
            quizAttempt: "QuizAttempt",
            quizQuestion: "QuizQuestion",
        },
        prismaRelMap: {
            __typename,
            quizAttempt: "QuizAttempt",
            quizQuestion: "QuizQuestion",
        },
        countFields: {},
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                response: data.response,
                ...(await shapeHelper({ relation: "quizAttempt", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "QuizAttempt", parentRelationshipName: "responses", data, ...rest })),
                ...(await shapeHelper({ relation: "quizQuestion", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "QuizQuestion", parentRelationshipName: "responses", data, ...rest })),
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
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transResponseWrapped",
            ],
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => QuizAttemptModel.validate.isPublic(data.quizAttempt, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => QuizAttemptModel.validate.owner(data.quizAttempt, userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            quizAttempt: "QuizAttempt",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                quizAttempt: QuizAttemptModel.validate.visibility.owner(userId),
            }),
        },
    },
});
//# sourceMappingURL=quizQuestionResponse.js.map