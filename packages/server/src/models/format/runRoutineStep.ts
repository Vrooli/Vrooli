import { RunRoutineStepModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "RunRoutineStep" as const;
export const RunRoutineStepFormat: Formatter<RunRoutineStepModelLogic> = {
    gqlRelMap: {
        __typename,
        run: "RunRoutine",
        node: "Node",
        subroutine: "Routine",
    },
    prismaRelMap: {
        __typename,
        node: "Node",
        runRoutine: "RunRoutine",
        subroutine: "RoutineVersion",
    },
    countFields: {},
};
