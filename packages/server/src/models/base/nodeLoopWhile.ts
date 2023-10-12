import { MaxObjects, nodeLoopWhileValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { NodeLoopWhileFormat } from "../formats";
import { NodeLoopModelInfo, NodeLoopModelLogic, NodeLoopWhileModelInfo, NodeLoopWhileModelLogic } from "./types";

const __typename = "NodeLoopWhile" as const;
export const NodeLoopWhileModel: NodeLoopWhileModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.node_loop_while,
    // Doesn't make sense to have a displayer for this model
    display: () => ({
        label: {
            select: () => ({ id: true }),
            get: () => "",
        },
    }),
    format: NodeLoopWhileFormat,
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
    search: undefined,
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, loop: "NodeLoop" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<NodeLoopModelLogic>("NodeLoop").validate().owner(data?.loop as NodeLoopModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<NodeLoopModelLogic>("NodeLoop").validate().isDeleted(data.loop as NodeLoopModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<NodeLoopWhileModelInfo["PrismaSelect"]>([["loop", "NodeLoop"]], ...rest),
        visibility: {
            private: { loop: ModelMap.get<NodeLoopModelLogic>("NodeLoop").validate().visibility.private },
            public: { loop: ModelMap.get<NodeLoopModelLogic>("NodeLoop").validate().visibility.public },
            owner: (userId) => ({ loop: ModelMap.get<NodeLoopModelLogic>("NodeLoop").validate().visibility.owner(userId) }),
        },
    }),
});
