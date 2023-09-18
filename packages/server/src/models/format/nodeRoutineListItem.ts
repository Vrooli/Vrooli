import { NodeRoutineListItemModelLogic } from "../base/types";
import { Formatter } from "../types";

export const NodeRoutineListItemFormat: Formatter<NodeRoutineListItemModelLogic> = {
    gqlRelMap: {
        __typename: "NodeRoutineListItem",
        routineVersion: "RoutineVersion",
    },
    prismaRelMap: {
        __typename: "NodeRoutineListItem",
        list: "NodeRoutineList",
        routineVersion: "RoutineVersion",
    },
    countFields: {},
};
