import { RunRoutineModelLogic } from "../base/types";
import { Formatter } from "../types";

export const RunRoutineFormat: Formatter<RunRoutineModelLogic> = {
    gqlRelMap: {
        __typename: "RunRoutine",
        inputs: "RunRoutineInput",
        organization: "Organization",
        routineVersion: "RoutineVersion",
        runProject: "RunProject",
        schedule: "Schedule",
        steps: "RunRoutineStep",
        user: "User",
    },
    prismaRelMap: {
        __typename: "RunRoutine",
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
