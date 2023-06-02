import { RunRoutineModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "RunRoutine" as const;
export const RunRoutineFormat: Formatter<RunRoutineModelLogic> = {
    gqlRelMap: {
        __typename,
        inputs: "RunRoutineInput",
        organization: "Organization",
        routineVersion: "RoutineVersion",
        runProject: "RunProject",
        schedule: "Schedule",
        steps: "RunRoutineStep",
        user: "User",
    },
    prismaRelMap: {
        __typename,
        inputs: "RunRoutineInput",
        organization: "Organization",
        routineVersion: "RoutineVersion",
        runProject: "RunProject",
        schedule: "Schedule",
        steps: "RunRoutineStep",
        user: "User",
    },
    countFields: {
        inputsCount: true,
        stepsCount: true,
    },
};
