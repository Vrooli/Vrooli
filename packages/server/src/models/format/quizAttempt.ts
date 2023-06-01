import { Formatter } from "../types";

const __typename = "QuizAttempt" as const;
export const QuizAttemptFormat: Formatter<ModelQuizAttemptLogic> = {
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
};
