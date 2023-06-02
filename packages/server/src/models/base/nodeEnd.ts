import { nodeEndValidation } from "@local/shared";
import { noNull, selPad, shapeHelper } from "../../builders";
import { nodeEndNextShapeHelper } from "../../utils";
import { NodeEndFormat } from "../format/nodeEnd";
import { ModelLogic } from "../types";
import { NodeModel } from "./node";
import { NodeEndModelLogic } from "./types";

const __typename = "NodeEnd" as const;
const suppFields = [] as const;
export const NodeEndModel: ModelLogic<NodeEndModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.node_end,
    display: {
        label: {
            select: () => ({ id: true, node: selPad(NodeModel.display.label.select) }),
            get: (select, languages) => NodeModel.display.label.get(select.node as any, languages),
        },
    },
    format: NodeEndFormat,
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
