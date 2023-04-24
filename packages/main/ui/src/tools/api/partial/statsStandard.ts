import { StatsStandard } from "@local/consts";
import { GqlPartial } from "../types";

export const statsStandard: GqlPartial<StatsStandard> = {
    __typename: "StatsStandard",
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        linksToInputs: true,
        linksToOutputs: true,
    },
};