import { StatsStandardModelLogic } from "../base/types";
import { Formatter } from "../types";

export const StatsStandardFormat: Formatter<StatsStandardModelLogic> = {
    gqlRelMap: {
        __typename: "StatsStandard",
    },
    prismaRelMap: {
        __typename: "StatsStandard",
        standard: "Standard",
    },
    countFields: {},
};
