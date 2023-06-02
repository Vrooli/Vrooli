import { QuizQuestionModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "QuizQuestion" as const;
export const QuizQuestionFormat: Formatter<QuizQuestionModelLogic> = {
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
