import { StatsApi } from "@local/shared";
import { GqlPartial } from "../types";

export const statsApi: GqlPartial<StatsApi> = {
    __typename: "StatsApi",
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        calls: true,
        routineVersions: true,
    },
};
