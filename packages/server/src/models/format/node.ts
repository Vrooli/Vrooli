import { NodeModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Node" as const;
export const NodeFormat: Formatter<NodeModelLogic> = {
    gqlRelMap: {
        __typename,
        end: "NodeEnd",
        loop: "NodeLoop",
        routineList: "NodeRoutineList",
        routineVersion: "RoutineVersion",
    },
    prismaRelMap: {
        __typename,
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
