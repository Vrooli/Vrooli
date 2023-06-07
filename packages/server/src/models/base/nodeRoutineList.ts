import { MaxObjects, nodeRoutineListValidation } from "@local/shared";
import { noNull, selPad, shapeHelper } from "../../builders";
import { defaultPermissions } from "../../utils";
import { NodeRoutineListFormat } from "../format/nodeRoutineList";
import { ModelLogic } from "../types";
import { NodeModel } from "./node";
import { NodeRoutineListModelLogic } from "./types";

const __typename = "NodeRoutineList" as const;
const suppFields = [] as const;
export const NodeRoutineListModel: ModelLogic<NodeRoutineListModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.node_routine_list,
    display: {
        label: {
            select: () => ({ id: true, node: selPad(NodeModel.display.label.select) }),
            get: (select, languages) => NodeModel.display.label.get(select.node as any, languages),
        },
    },
    format: NodeRoutineListFormat,
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
        owner: (data, userId) => NodeModel.validate!.owner(data.node as any, userId),
        isDeleted: (data, languages) => NodeModel.validate!.isDeleted(data.node as any, languages),
        isPublic: (data, languages) => NodeModel.validate!.isPublic(data.node as any, languages),
        visibility: {
            private: { node: NodeModel.validate!.visibility.private },
            public: { node: NodeModel.validate!.visibility.public },
            owner: (userId) => ({ node: NodeModel.validate!.visibility.owner(userId) }),
        },
    },
});