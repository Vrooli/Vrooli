import { NodeEndModelLogic } from "../base/types";
import { Formatter } from "../types";

export const NodeEndFormat: Formatter<NodeEndModelLogic> = {
    gqlRelMap: {
        __typename: "NodeEnd",
        suggestedNextRoutineVersions: "RoutineVersion",
    },
    prismaRelMap: {
        __typename: "NodeEnd",
        suggestedNextRoutineVersions: "RoutineVersion",
        node: "Node",
    },
    joinMap: { suggestedNextRoutineVersions: "toRoutineVersion" },
    countFields: {},
};
