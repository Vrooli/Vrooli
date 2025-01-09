import { StatsApi } from "@local/shared";
import { ApiPartial } from "../types";

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
