import { NodeEnd, NodeEndCreateInput, NodeEndUpdateInput, nodeEndValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, selPad, shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { nodeEndNextShapeHelper } from "../../utils";
import { NodeModel } from "./node";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "NodeEnd" as const;
export const NodeEndFormat: Formatter<ModelNodeEndLogic> = {
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
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    wasSuccessful: noNull(data.wasSuccessful),
                    ...(await shapeHelper({ relation: "node", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Node", parentRelationshipName: "end", data, ...rest })),
                    ...(await nodeEndNextShapeHelper({ relTypes: ["Connect"], data, ...rest })),
                };
            },
            update: async ({ data, ...rest }) => {
                return {
                    wasSuccessful: noNull(data.wasSuccessful),
                    ...(await nodeEndNextShapeHelper({ relTypes: ["Connect", "Disconnect"], data, ...rest })),
                };
            },
};
