import { NodeModelLogic } from "../base/types";
import { Formatter } from "../types";

export const NodeFormat: Formatter<NodeModelLogic> = {
    gqlRelMap: {
        __typename: "Node",
        end: "NodeEnd",
        loop: "NodeLoop",
        routineList: "NodeRoutineList",
        routineVersion: "RoutineVersion",
    },
    prismaRelMap: {
        __typename: "Node",
        end: "NodeEnd",
        loop: "NodeLoop",
        next: "NodeLink",
        previous: "NodeLink",
        routineList: "NodeRoutineList",
        routineVersion: "RoutineVersion",
        runSteps: "RunRoutineStep",
    },
    countFields: {},
};
