import { StatsApi } from "@shared/consts";
import { GqlPartial } from "../types";

export const statsApi: GqlPartial<StatsApi> = {
    __typename: 'StatsApi',
    full: {
        id: true,
        created_at: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        calls: true,
        routineVersions: true,
    },
}