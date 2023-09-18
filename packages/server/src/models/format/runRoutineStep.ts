import { RunRoutineStepModelLogic } from "../base/types";
import { Formatter } from "../types";

export const RunRoutineStepFormat: Formatter<RunRoutineStepModelLogic> = {
    gqlRelMap: {
        __typename: "RunRoutineStep",
        run: "RunRoutine",
        node: "Node",
        subroutine: "Routine",
    },
    prismaRelMap: {
        __typename: "RunRoutineStep",
        node: "Node",
        runRoutine: "RunRoutine",
        subroutine: "RoutineVersion",
    },
    countFields: {},
};
