import { RunRoutineInputModelLogic } from "../base/types";
import { Formatter } from "../types";

export const RunRoutineInputFormat: Formatter<RunRoutineInputModelLogic> = {
    gqlRelMap: {
        __typename: "RunRoutineInput",
        input: "RoutineVersionInput",
        runRoutine: "RunRoutine",
    },
    prismaRelMap: {
        __typename: "RunRoutineInput",
        input: "RunRoutineInput",
        runRoutine: "RunRoutine",
    },
    countFields: {},
};
