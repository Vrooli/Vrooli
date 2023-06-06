import { StatsStandardModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "StatsStandard" as const;
export const StatsStandardFormat: Formatter<StatsStandardModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        standard: "Standard",
    },
    countFields: {},
};
