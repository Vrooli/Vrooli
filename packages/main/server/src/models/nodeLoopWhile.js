import { MaxObjects } from "@local/consts";
import { nodeLoopWhileValidation } from "@local/validation";
import { noNull, shapeHelper } from "../builders";
import { defaultPermissions, translationShapeHelper } from "../utils";
import { NodeLoopModel } from "./nodeLoop";
const __typename = "NodeLoopWhile";
const suppFields = [];
export const NodeLoopWhileModel = ({
    __typename,
    delegate: (prisma) => prisma.node_loop_while,
    display: {
        select: () => ({ id: true }),
        label: () => "",
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            loop: "NodeLoop",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                condition: data.condition,
                ...(await shapeHelper({ relation: "loop", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "NodeLoop", parentRelationshipName: "whiles", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                condition: noNull(data.condition),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
        },
        yup: nodeLoopWhileValidation,
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ loop: "NodeLoop" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => NodeLoopModel.validate.owner(data.loop, userId),
        isDeleted: (data, languages) => NodeLoopModel.validate.isDeleted(data.loop, languages),
        isPublic: (data, languages) => NodeLoopModel.validate.isPublic(data.loop, languages),
        visibility: {
            private: { loop: NodeLoopModel.validate.visibility.private },
            public: { loop: NodeLoopModel.validate.visibility.public },
            owner: (userId) => ({ loop: NodeLoopModel.validate.visibility.owner(userId) }),
        },
    },
});
//# sourceMappingURL=nodeLoopWhile.js.map