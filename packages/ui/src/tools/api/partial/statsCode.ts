import { StatsCode } from "@local/shared";
import { GqlPartial } from "../types";

export const statsCode: GqlPartial<StatsCode> = {
    __typename: "StatsCode",
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        calls: true,
        routineVersions: true,
    },
};
