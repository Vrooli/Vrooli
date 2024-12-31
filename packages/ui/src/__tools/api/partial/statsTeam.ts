import { StatsTeam } from "@local/shared";
import { GqlPartial } from "../types";

export const statsTeam: GqlPartial<StatsTeam> = {
    __typename: "StatsTeam",
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        apis: true,
        codes: true,
        members: true,
        notes: true,
        projects: true,
        routines: true,
        standards: true,
    },
};
