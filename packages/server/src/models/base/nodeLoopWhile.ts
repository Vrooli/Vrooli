import { MaxObjects, nodeLoopWhileValidation } from "@local/shared";
import { noNull, shapeHelper } from "../builders";
import { PrismaType } from "../types";
import { defaultPermissions, translationShapeHelper } from "../utils";
import { NodeLoopModel } from "./nodeLoop";
import { ModelLogic, NodeLoopWhileModelLogic } from "./types";

const __typename = "NodeLoopWhile" as const;
const suppFields = [] as const;
export const NodeLoopWhileModel: ModelLogic<NodeLoopWhileModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_loop_while,
    // Doesn't make sense to have a displayer for this model
    display: {
        label: {
            select: () => ({ id: true }),
            get: () => "",
        },
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
        owner: (data, userId) => NodeLoopModel.validate!.owner(data.loop as any, userId),
        isDeleted: (data, languages) => NodeLoopModel.validate!.isDeleted(data.loop as any, languages),
        isPublic: (data, languages) => NodeLoopModel.validate!.isPublic(data.loop as any, languages),
        visibility: {
            private: { loop: NodeLoopModel.validate!.visibility.private },
            public: { loop: NodeLoopModel.validate!.visibility.public },
            owner: (userId) => ({ loop: NodeLoopModel.validate!.visibility.owner(userId) }),
        },
    },
});
