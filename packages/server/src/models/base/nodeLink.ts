import { nodeLinkValidation } from "@local/shared";
import { noNull, selPad, shapeHelper } from "../../builders";
import { NodeLinkFormat } from "../format/nodeLink";
import { ModelLogic } from "../types";
import { NodeModel } from "./node";
import { NodeLinkModelLogic } from "./types";

const __typename = "NodeLink" as const;
const suppFields = [] as const;
export const NodeLinkModel: ModelLogic<NodeLinkModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.node_link,
    display: {
        label: {
            select: () => ({
                id: true,
                from: selPad(NodeModel.display.label.select),
                to: selPad(NodeModel.display.label.select),
            }),
            // Label combines from and to labels
            get: (select, languages) => {
                return `${NodeModel.display.label.get(select.from as any, languages)} -> ${NodeModel.display.label.get(select.to as any, languages)}`;
            },
        },
    },
    format: NodeLinkFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                operation: noNull(data.operation),
                ...(await shapeHelper({ relation: "from", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Node", parentRelationshipName: "next", data, ...rest })),
                ...(await shapeHelper({ relation: "to", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Node", parentRelationshipName: "previous", data, ...rest })),
                ...(await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "RoutineVersion", parentRelationshipName: "nodeLinks", data, ...rest })),
                ...(await shapeHelper({ relation: "whens", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "NodeLinkWhen", parentRelationshipName: "link", data, ...rest })),

            }),
            update: async ({ data, ...rest }) => ({
                operation: noNull(data.operation),
                ...(await shapeHelper({ relation: "from", relTypes: ["Connect", "Disconnect"], isOneToOne: true, isRequired: false, objectType: "Node", parentRelationshipName: "next", data, ...rest })),
                ...(await shapeHelper({ relation: "to", relTypes: ["Connect", "Disconnect"], isOneToOne: true, isRequired: false, objectType: "Node", parentRelationshipName: "previous", data, ...rest })),
                ...(await shapeHelper({ relation: "whens", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "NodeLinkWhen", parentRelationshipName: "link", data, ...rest })),
            }),
        },
        yup: nodeLinkValidation,
    },
});
