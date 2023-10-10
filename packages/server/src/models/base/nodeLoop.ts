import { MaxObjects, nodeLoopValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { NodeLoopFormat } from "../formats";
import { NodeLoopModelInfo, NodeLoopModelLogic, NodeModelInfo, NodeModelLogic } from "./types";

const __typename = "NodeLoop" as const;
export const NodeLoopModel: NodeLoopModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.node_loop,
    // Doesn't make sense to have a displayer for this model
    display: {
        label: {
            select: () => ({ id: true }),
            get: () => "",
        },
    },
    format: NodeLoopFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                loops: noNull(data.loops),
                maxLoops: noNull(data.maxLoops),
                operation: noNull(data.operation),
                ...(await shapeHelper({ relation: "node", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Node", parentRelationshipName: "loop", data, ...rest })),
                ...(await shapeHelper({ relation: "whiles", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "NodeLoopWhile", parentRelationshipName: "loop", data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                loops: noNull(data.loops),
                maxLoops: noNull(data.maxLoops),
                operation: noNull(data.operation),
                ...(await shapeHelper({ relation: "node", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "Node", parentRelationshipName: "loop", data, ...rest })),
                ...(await shapeHelper({ relation: "whiles", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "NodeLoopWhile", parentRelationshipName: "loop", data, ...rest })),
            }),
        },
        yup: nodeLoopValidation,
    },
    search: undefined,
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, node: "Node" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<NodeModelLogic>("Node").validate.owner(data?.node as NodeModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<NodeModelLogic>("Node").validate.isDeleted(data.node as NodeModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<NodeLoopModelInfo["PrismaSelect"]>([["node", "Node"]], ...rest),
        visibility: {
            private: { node: ModelMap.get<NodeModelLogic>("Node").validate.visibility.private },
            public: { node: ModelMap.get<NodeModelLogic>("Node").validate.visibility.public },
            owner: (userId) => ({ node: ModelMap.get<NodeModelLogic>("Node").validate.visibility.owner(userId) }),
        },
    },
});
