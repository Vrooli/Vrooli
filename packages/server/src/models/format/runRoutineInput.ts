import { RunRoutineInputModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "RunRoutineInput" as const;
export const RunRoutineInputFormat: Formatter<RunRoutineInputModelLogic> = {
    gqlRelMap: {
        __typename,
        input: "RoutineVersionInput",
        runRoutine: "RunRoutine",
    },
    prismaRelMap: {
        __typename,
        input: "RunRoutineInput",
        runRoutine: "RunRoutine",
    },
    countFields: {},
};
