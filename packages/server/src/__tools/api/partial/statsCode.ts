import { StatsCode } from "@local/shared";
import { ApiPartial } from "../types.js";

export const statsCode: ApiPartial<StatsCode> = {
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        calls: true,
        routineVersions: true,
    },
};
