import { StatsApi } from "@local/shared";
import { ApiPartial } from "../types.js";

export const statsApi: ApiPartial<StatsApi> = {
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        calls: true,
        routineVersions: true,
    },
};
