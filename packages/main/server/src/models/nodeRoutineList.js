import { MaxObjects } from "@local/consts";
import { nodeRoutineListValidation } from "@local/validation";
import { noNull, selPad, shapeHelper } from "../builders";
import { defaultPermissions } from "../utils";
import { NodeModel } from "./node";
const __typename = "NodeRoutineList";
const suppFields = [];
export const NodeRoutineListModel = ({
    __typename,
    delegate: (prisma) => prisma.node_routine_list,
    display: {
        select: () => ({ id: true, node: selPad(NodeModel.display.select) }),
        label: (select, languages) => NodeModel.display.label(select.node, languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            items: "NodeRoutineListItem",
        },
        prismaRelMap: {
            __typename,
            node: "Node",
            items: "NodeRoutineListItem",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                isOrdered: noNull(data.isOrdered),
                isOptional: noNull(data.isOptional),
                ...(await shapeHelper({ relation: "node", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Node", parentRelationshipName: "node", data, ...rest })),
                ...(await shapeHelper({ relation: "items", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "NodeRoutineListItem", parentRelationshipName: "list", data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                isOrdered: noNull(data.isOrdered),
                isOptional: noNull(data.isOptional),
                ...(await shapeHelper({ relation: "items", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "NodeRoutineListItem", parentRelationshipName: "list", data, ...rest })),
            }),
        },
        yup: nodeRoutineListValidation,
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ node: "Node" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => NodeModel.validate.owner(data.node, userId),
        isDeleted: (data, languages) => NodeModel.validate.isDeleted(data.node, languages),
        isPublic: (data, languages) => NodeModel.validate.isPublic(data.node, languages),
        visibility: {
            private: { node: NodeModel.validate.visibility.private },
            public: { node: NodeModel.validate.visibility.public },
            owner: (userId) => ({ node: NodeModel.validate.visibility.owner(userId) }),
        },
    },
});
//# sourceMappingURL=nodeRoutineList.js.map