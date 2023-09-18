import { StatsApiModelLogic } from "../base/types";
import { Formatter } from "../types";

export const StatsApiFormat: Formatter<StatsApiModelLogic> = {
    gqlRelMap: {
        __typename: "StatsApi",
    },
    prismaRelMap: {
        __typename: "StatsApi",
        api: "Api",
    },
    countFields: {},
};
