import { nodeEndValidation } from "@local/validation";
import { noNull, selPad, shapeHelper } from "../builders";
import { nodeEndNextShapeHelper } from "../utils";
import { NodeModel } from "./node";
const __typename = "NodeEnd";
const suppFields = [];
export const NodeEndModel = ({
    __typename,
    delegate: (prisma) => prisma.node_end,
    display: {
        select: () => ({ id: true, node: selPad(NodeModel.display.select) }),
        label: (select, languages) => NodeModel.display.label(select.node, languages),
    },
    format: {
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
        },
        yup: nodeEndValidation,
    },
});
//# sourceMappingURL=nodeEnd.js.map