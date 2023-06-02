import { NodeRoutineListItemModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "NodeRoutineListItem" as const;
export const NodeRoutineListItemFormat: Formatter<NodeRoutineListItemModelLogic> = {
    gqlRelMap: {
        __typename,
        routineVersion: "RoutineVersion",
    },
    prismaRelMap: {
        __typename,
        list: "NodeRoutineList",
        routineVersion: "RoutineVersion",
    },
    countFields: {},
};
