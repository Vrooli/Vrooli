import { StatsUserModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "StatsUser" as const;
export const StatsUserFormat: Formatter<StatsUserModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        user: "User",
    },
    countFields: {},
};
