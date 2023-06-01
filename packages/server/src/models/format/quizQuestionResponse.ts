import { Formatter } from "../types";

const __typename = "QuizQuestionResponse" as const;
export const QuizQuestionResponseFormat: Formatter<ModelQuizQuestionResponseLogic> = {
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
};
