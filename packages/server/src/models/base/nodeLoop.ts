import { MaxObjects, nodeLoopValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { defaultPermissions } from "../../utils";
import { NodeLoopFormat } from "../format/nodeLoop";
import { ModelLogic } from "../types";
import { NodeModel } from "./node";
import { NodeLoopModelLogic } from "./types";

const __typename = "NodeLoop" as const;
const suppFields = [] as const;
export const NodeLoopModel: ModelLogic<NodeLoopModelLogic, typeof suppFields> = ({
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
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, node: "Node" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => NodeModel.validate.owner(data.node as any, userId),
        isDeleted: (data, languages) => NodeModel.validate.isDeleted(data.node as any, languages),
        isPublic: (data, languages) => NodeModel.validate.isPublic(data.node as any, languages),
        visibility: {
            private: { node: NodeModel.validate.visibility.private },
            public: { node: NodeModel.validate.visibility.public },
            owner: (userId) => ({ node: NodeModel.validate.visibility.owner(userId) }),
        },
    },
});
