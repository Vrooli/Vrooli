import { RoutineVersionOutputModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "RoutineVersionOutput" as const;
export const RoutineVersionOutputFormat: Formatter<RoutineVersionOutputModelLogic> = {
    gqlRelMap: {
        __typename,
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
    },
    prismaRelMap: {
        __typename,
        routineVersion: "RoutineVersion",
        standardVersion: "StandardVersion",
    },
    countFields: {},
};
