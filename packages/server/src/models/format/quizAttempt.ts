import { QuizAttemptModelLogic } from "../base/types";
import { Formatter } from "../types";

export const QuizAttemptFormat: Formatter<QuizAttemptModelLogic> = {
    gqlRelMap: {
        __typename: "QuizAttempt",
        quiz: "Quiz",
        responses: "QuizQuestionResponse",
        user: "User",
    },
    prismaRelMap: {
        __typename: "QuizAttempt",
        quiz: "Quiz",
        responses: "QuizQuestionResponse",
        user: "User",
    },
    countFields: {
        responsesCount: true,
    },
};
