import { StatsStandard } from "@shared/consts";
import { GqlPartial } from "../types";

export const statsStandardPartial: GqlPartial<StatsStandard> = {
    __typename: 'StatsStandard',
    full: {
        id: true,
        created_at: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        linksToInputs: true,
        linksToOutputs: true,
        timesUsedInCompletedRoutines: true,
    },
}