import { StatsProjectModelLogic } from "../base/types";
import { Formatter } from "../types";

export const StatsProjectFormat: Formatter<StatsProjectModelLogic> = {
    gqlRelMap: {
        __typename: "StatsProject",
    },
    prismaRelMap: {
        __typename: "StatsProject",
        project: "Api",
    },
    countFields: {},
};
