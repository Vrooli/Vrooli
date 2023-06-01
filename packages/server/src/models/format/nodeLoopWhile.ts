import { MaxObjects, NodeLoopWhile, NodeLoopWhileCreateInput, NodeLoopWhileUpdateInput, nodeLoopWhileValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions, translationShapeHelper } from "../../utils";
import { NodeLoopModel } from "./nodeLoop";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "NodeLoopWhile" as const;
export const NodeLoopWhileFormat: Formatter<ModelNodeLoopWhileLogic> = {
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
        visibility: {
            private: { loop: NodeLoopModel.validate!.visibility.private },
            public: { loop: NodeLoopModel.validate!.visibility.public },
            owner: (userId) => ({ loop: NodeLoopModel.validate!.visibility.owner(userId) }),
};
