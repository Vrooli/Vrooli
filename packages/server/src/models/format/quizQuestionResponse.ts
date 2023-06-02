import { QuizQuestionResponseModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "QuizQuestionResponse" as const;
export const QuizQuestionResponseFormat: Formatter<QuizQuestionResponseModelLogic> = {
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
