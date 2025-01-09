import { StatsTeam } from "@local/shared";
import { ApiPartial } from "../types";

export const statsTeam: ApiPartial<StatsTeam> = {
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
