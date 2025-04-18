import { StatsStandard } from "@local/shared";
import { ApiPartial } from "../types.js";

export const statsStandard: ApiPartial<StatsStandard> = {
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        linksToInputs: true,
        linksToOutputs: true,
    },
};
