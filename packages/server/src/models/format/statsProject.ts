import { StatsProjectModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "StatsProject" as const;
export const StatsProjectFormat: Formatter<StatsProjectModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        project: "Api",
    },
    countFields: {},
};
