import { StatsUserModelLogic } from "../base/types";
import { Formatter } from "../types";

export const StatsUserFormat: Formatter<StatsUserModelLogic> = {
    gqlRelMap: {
        __typename: "StatsUser",
    },
    prismaRelMap: {
        __typename: "StatsUser",
        user: "User",
    },
    countFields: {},
};
