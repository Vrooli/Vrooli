import { nodeLinkValidation } from "@local/validation";
import { noNull, selPad, shapeHelper } from "../builders";
import { NodeModel } from "./node";
const __typename = "NodeLink";
const suppFields = [];
export const NodeLinkModel = ({
    __typename,
    delegate: (prisma) => prisma.node_link,
    display: {
        select: () => ({
            id: true,
            from: selPad(NodeModel.display.select),
            to: selPad(NodeModel.display.select),
        }),
        label: (select, languages) => {
            return `${NodeModel.display.label(select.from, languages)} -> ${NodeModel.display.label(select.to, languages)}`;
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            whens: "NodeLinkWhen",
        },
        prismaRelMap: {
            __typename,
            from: "Node",
            to: "Node",
            routineVersion: "RoutineVersion",
            whens: "NodeLinkWhen",
        },
        countFields: {},
    },
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
//# sourceMappingURL=nodeLink.js.map