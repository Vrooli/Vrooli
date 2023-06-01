import { MaxObjects, NodeLoop, NodeLoopCreateInput, NodeLoopUpdateInput, nodeLoopValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions } from "../../utils";
import { NodeModel } from "./node";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "NodeLoop" as const;
export const NodeLoopFormat: Formatter<ModelNodeLoopLogic> = {
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
        visibility: {
            private: { node: NodeModel.validate!.visibility.private },
            public: { node: NodeModel.validate!.visibility.public },
            owner: (userId) => ({ node: NodeModel.validate!.visibility.owner(userId) }),
};
