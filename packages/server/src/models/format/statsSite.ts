import { StatsSiteModelLogic } from "../base/types";
import { Formatter } from "../types";

export const StatsSiteFormat: Formatter<StatsSiteModelLogic> = {
    gqlRelMap: {
        __typename: "StatsSite",
    },
    prismaRelMap: {
        __typename: "StatsSite",
    },
    countFields: {},
};
