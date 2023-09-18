import { RoutineVersionOutputModelLogic } from "../base/types";
import { Formatter } from "../types";

export const RoutineVersionOutputFormat: Formatter<RoutineVersionOutputModelLogic> = {
    gqlRelMap: {
        __typename: "RoutineVersionOutput",
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
    },
    prismaRelMap: {
        __typename: "RoutineVersionOutput",
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
    },
    countFields: {},
};
