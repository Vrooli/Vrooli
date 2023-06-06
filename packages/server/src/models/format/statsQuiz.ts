import { StatsQuizModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "StatsQuiz" as const;
export const StatsQuizFormat: Formatter<StatsQuizModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        quiz: "Quiz",
    },
    countFields: {},
};
