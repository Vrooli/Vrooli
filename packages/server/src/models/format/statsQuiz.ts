import { StatsQuizModelLogic } from "../base/types";
import { Formatter } from "../types";

export const StatsQuizFormat: Formatter<StatsQuizModelLogic> = {
    gqlRelMap: {
        __typename: "StatsQuiz",
    },
    prismaRelMap: {
        __typename: "StatsQuiz",
        quiz: "Quiz",
    },
    countFields: {},
};
