import { StatsRoutineModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "StatsRoutine" as const;
export const StatsRoutineFormat: Formatter<StatsRoutineModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        routine: "Routine",
    },
    countFields: {},
};
