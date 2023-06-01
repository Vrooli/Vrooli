import { MaxObjects, NodeRoutineList, NodeRoutineListCreateInput, NodeRoutineListUpdateInput, nodeRoutineListValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, selPad, shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions } from "../../utils";
import { NodeModel } from "./node";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "NodeRoutineList" as const;
export const NodeRoutineListFormat: Formatter<ModelNodeRoutineListLogic> = {
        gqlRelMap: {
            __typename,
            items: "NodeRoutineListItem",
        },
        prismaRelMap: {
            __typename,
            node: "Node",
            items: "NodeRoutineListItem",
        },
        countFields: {},
    },
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
        visibility: {
            private: { node: NodeModel.validate!.visibility.private },
            public: { node: NodeModel.validate!.visibility.public },
            owner: (userId) => ({ node: NodeModel.validate!.visibility.owner(userId) }),
};
