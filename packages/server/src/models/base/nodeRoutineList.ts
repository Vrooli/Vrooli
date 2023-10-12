import { MaxObjects, nodeRoutineListValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { NodeRoutineListFormat } from "../formats";
import { NodeModelInfo, NodeModelLogic, NodeRoutineListModelInfo, NodeRoutineListModelLogic } from "./types";

const __typename = "NodeRoutineList" as const;
export const NodeRoutineListModel: NodeRoutineListModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.node_routine_list,
    display: () => ({
        label: {
            select: () => ({ id: true, node: { select: ModelMap.get<NodeModelLogic>("Node").display().label.select() } }),
            get: (select, languages) => ModelMap.get<NodeModelLogic>("Node").display().label.get(select.node as NodeModelInfo["PrismaModel"], languages),
        },
    }),
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
    search: undefined,
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, node: "Node" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<NodeModelLogic>("Node").validate().owner(data?.node as NodeModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<NodeModelLogic>("Node").validate().isDeleted(data.node as NodeModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<NodeRoutineListModelInfo["PrismaSelect"]>([["node", "Node"]], ...rest),
        visibility: {
            private: { node: ModelMap.get<NodeModelLogic>("Node").validate().visibility.private },
            public: { node: ModelMap.get<NodeModelLogic>("Node").validate().visibility.public },
            owner: (userId) => ({ node: ModelMap.get<NodeModelLogic>("Node").validate().visibility.owner(userId) }),
        },
    }),
});
