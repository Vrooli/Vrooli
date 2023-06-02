import { NodeEndModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "NodeEnd" as const;
export const NodeEndFormat: Formatter<NodeEndModelLogic> = {
    gqlRelMap: {
        __typename,
        suggestedNextRoutineVersions: "RoutineVersion",
    },
    prismaRelMap: {
        __typename,
        suggestedNextRoutineVersions: "RoutineVersion",
        node: "Node",
    },
    joinMap: { suggestedNextRoutineVersions: "toRoutineVersion" },
    countFields: {},
};
