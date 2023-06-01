import { Formatter } from "../types";

const __typename = "QuizQuestion" as const;
export const QuizQuestionFormat: Formatter<ModelQuizQuestionLogic> = {
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
};
