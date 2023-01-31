import { StatsProject } from "@shared/consts";
import { GqlPartial } from "../types";

export const statsProjectPartial: GqlPartial<StatsProject> = {
    __typename: 'StatsProject',
    full: {
        id: true,
        created_at: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        directories: true,
        notes: true,
        routines: true,
        smartContracts: true,
        standards: true,
    },
}