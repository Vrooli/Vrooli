import { MaxObjects } from "@local/consts";
import { nodeLoopValidation } from "@local/validation";
import { noNull, shapeHelper } from "../builders";
import { defaultPermissions } from "../utils";
import { NodeModel } from "./node";
const __typename = "NodeLoop";
const suppFields = [];
export const NodeLoopModel = ({
    __typename,
    delegate: (prisma) => prisma.node_loop,
    display: {
        select: () => ({ id: true }),
        label: () => "",
    },
    format: {
        gqlRelMap: {
            __typename,
            whiles: "NodeLoopWhile",
        },
        prismaRelMap: {
            __typename,
            node: "Node",
            whiles: "NodeLoopWhile",
        },
        countFields: {},
    },
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
//# sourceMappingURL=nodeLoop.js.map