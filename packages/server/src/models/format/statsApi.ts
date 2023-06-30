import { StatsApiModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "StatsApi" as const;
export const StatsApiFormat: Formatter<StatsApiModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        api: "Api",
    },
    countFields: {},
};
