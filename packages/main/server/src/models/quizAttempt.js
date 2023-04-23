import { QuizAttemptSortBy } from "@local/consts";
import { quizAttemptValidation } from "@local/validation";
import { noNull, shapeHelper } from "../builders";
import { getSingleTypePermissions } from "../validators";
import { QuizModel } from "./quiz";
const __typename = "QuizAttempt";
const suppFields = ["you"];
export const QuizAttemptModel = ({
    __typename,
    delegate: (prisma) => prisma.quiz_attempt,
    display: {
        select: () => ({
            id: true,
            created_at: true,
            quiz: { select: QuizModel.display.select() },
        }),
        label: (select, languages) => {
            const quizName = QuizModel.display.label(select.quiz, languages);
            const date = new Date(select.created_at).toLocaleDateString();
            return `${quizName} - ${date}`;
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            quiz: "Quiz",
            responses: "QuizQuestionResponse",
            user: "User",
        },
        prismaRelMap: {
            __typename,
            quiz: "Quiz",
            responses: "QuizQuestionResponse",
            user: "User",
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
                contextSwitches: noNull(data.contextSwitches),
                timeTaken: noNull(data.timeTaken),
                language: data.language,
                user: { connect: { id: rest.userData.id } },
                ...(await shapeHelper({ relation: "quiz", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Quiz", parentRelationshipName: "attempts", data, ...rest })),
                ...(await shapeHelper({ relation: "responses", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "QuizQuestionResponse", parentRelationshipName: "quizAttempt", data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                contextSwitches: noNull(data.contextSwitches),
                timeTaken: noNull(data.timeTaken),
                ...(await shapeHelper({ relation: "responses", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "QuizQuestionResponse", parentRelationshipName: "quizAttempt", data, ...rest })),
            }),
        },
        yup: quizAttemptValidation,
    },
    search: {
        defaultSort: QuizAttemptSortBy.TimeTakenDesc,
        sortBy: QuizAttemptSortBy,
        searchFields: {
            createdTimeFrame: true,
            status: true,
            languageIn: true,
            maxPointsEarned: true,
            minPointsEarned: true,
            userId: true,
            quizId: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({}),
    },
    validate: {},
});
//# sourceMappingURL=quizAttempt.js.map