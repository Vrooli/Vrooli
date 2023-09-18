import { ReactionSummaryModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ReactionSummaryFormat: Formatter<ReactionSummaryModelLogic> = {
    gqlRelMap: {
        __typename: "ReactionSummary",
    },
    prismaRelMap: {
        __typename: "ReactionSummary",
    },
    countFields: {},
};
