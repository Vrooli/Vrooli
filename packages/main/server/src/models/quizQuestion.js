import { MaxObjects, QuizQuestionSortBy } from "@local/consts";
import { quizQuestionValidation } from "@local/validation";
import { noNull, shapeHelper } from "../builders";
import { bestLabel, defaultPermissions, translationShapeHelper } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { QuizModel } from "./quiz";
const __typename = "QuizQuestion";
const suppFields = ["you"];
export const QuizQuestionModel = ({
    __typename,
    delegate: (prisma) => prisma.quiz_question,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, questionText: true } } }),
        label: (select, languages) => bestLabel(select.translations, "questionText", languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            quiz: "Quiz",
            responses: "QuizQuestionResponse",
            standardVersion: "StandardVersion",
        },
        prismaRelMap: {
            __typename,
            quiz: "Quiz",
            responses: "QuizQuestionResponse",
            standardVersion: "StandardVersion",
        },
        countFields: {
            responsesCount: true,
        },
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
                order: noNull(data.order),
                points: noNull(data.points),
                ...(await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "StandardVersion", parentRelationshipName: "quizQuestions", data, ...rest })),
                ...(await shapeHelper({ relation: "quiz", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Quiz", parentRelationshipName: "quizQuestions", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                order: noNull(data.order),
                points: noNull(data.points),
                ...(await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create", "Update"], isOneToOne: true, isRequired: false, objectType: "StandardVersion", parentRelationshipName: "quizQuestions", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
        },
        yup: quizQuestionValidation,
    },
    search: {
        defaultSort: QuizQuestionSortBy.OrderAsc,
        sortBy: QuizQuestionSortBy,
        searchFields: {
            createdTimeFrame: true,
            translationLanguages: true,
            quizId: true,
            standardId: true,
            userId: true,
            responseId: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transQuestionTextWrapped",
            ],
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => QuizModel.validate.isPublic(data.quiz, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => QuizModel.validate.owner(data.quiz, userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            quiz: "Quiz",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                quiz: QuizModel.validate.visibility.owner(userId),
            }),
        },
    },
});
//# sourceMappingURL=quizQuestion.js.map