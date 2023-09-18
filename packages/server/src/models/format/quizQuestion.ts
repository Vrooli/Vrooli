import { QuizQuestionModelLogic } from "../base/types";
import { Formatter } from "../types";

export const QuizQuestionFormat: Formatter<QuizQuestionModelLogic> = {
    gqlRelMap: {
        __typename: "QuizQuestion",
        quiz: "Quiz",
        responses: "QuizQuestionResponse",
        standardVersion: "StandardVersion",
    },
    prismaRelMap: {
        __typename: "QuizQuestion",
        quiz: "Quiz",
        responses: "QuizQuestionResponse",
        standardVersion: "StandardVersion",
    },
    countFields: {
        responsesCount: true,
    },
};
