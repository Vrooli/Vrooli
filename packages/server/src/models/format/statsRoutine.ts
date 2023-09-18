import { StatsRoutineModelLogic } from "../base/types";
import { Formatter } from "../types";

export const StatsRoutineFormat: Formatter<StatsRoutineModelLogic> = {
    gqlRelMap: {
        __typename: "StatsRoutine",
    },
    prismaRelMap: {
        __typename: "StatsRoutine",
        routine: "Routine",
    },
    countFields: {},
};
