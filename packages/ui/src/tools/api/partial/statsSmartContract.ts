import { StatsSmartContract } from "@local/shared";
import { GqlPartial } from "../types";

export const statsSmartContract: GqlPartial<StatsSmartContract> = {
    __typename: 'StatsSmartContract',
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        calls: true,
        routineVersions: true,
    },
}