import { RoutineVersionInputModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "RoutineVersionInput" as const;
export const RoutineVersionInputFormat: Formatter<RoutineVersionInputModelLogic> = {
    gqlRelMap: {
        __typename,
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
    },
    prismaRelMap: {
        __typename,
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
        runInputs: "RunRoutineInput",
    },
    countFields: {},
};
