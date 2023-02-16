import { StatsOrganization } from "@shared/consts";
import { GqlPartial } from "../types";

export const statsOrganization: GqlPartial<StatsOrganization> = {
    __typename: 'StatsOrganization',
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        apis: true,
        members: true,
        notes: true,
        projects: true,
        routines: true,
        smartContracts: true,
        standards: true,
    },
}