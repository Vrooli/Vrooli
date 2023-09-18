import { QuizQuestionResponseModelLogic } from "../base/types";
import { Formatter } from "../types";

export const QuizQuestionResponseFormat: Formatter<QuizQuestionResponseModelLogic> = {
    gqlRelMap: {
        __typename: "QuizQuestionResponse",
        quizAttempt: "QuizAttempt",
        quizQuestion: "QuizQuestion",
    },
    prismaRelMap: {
        __typename: "QuizQuestionResponse",
        quizAttempt: "QuizAttempt",
        quizQuestion: "QuizQuestion",
    },
    countFields: {},
};
