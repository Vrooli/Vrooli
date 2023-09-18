import { RoutineVersionInputModelLogic } from "../base/types";
import { Formatter } from "../types";

export const RoutineVersionInputFormat: Formatter<RoutineVersionInputModelLogic> = {
    gqlRelMap: {
        __typename: "RoutineVersionInput",
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
    },
    prismaRelMap: {
        __typename: "RoutineVersionInput",
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
        runInputs: "RunRoutineInput",
    },
    countFields: {},
};
