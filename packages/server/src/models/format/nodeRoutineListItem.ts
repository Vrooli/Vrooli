import { MaxObjects, NodeRoutineListItem, NodeRoutineListItemCreateInput, NodeRoutineListItemUpdateInput, nodeRoutineListItemValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, selPad, shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { bestTranslation, defaultPermissions, translationShapeHelper } from "../../utils";
import { NodeRoutineListModel } from "./nodeRoutineList";
import { RoutineModel } from "./routine";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "NodeRoutineListItem" as const;
export const NodeRoutineListItemFormat: Formatter<ModelNodeRoutineListItemLogic> = {
        gqlRelMap: {
            __typename,
            routineVersion: "RoutineVersion",
        },
        prismaRelMap: {
            __typename,
            list: "NodeRoutineList",
            routineVersion: "RoutineVersion",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                index: data.index,
                isOptional: noNull(data.isOptional),
                ...(await shapeHelper({ relation: "list", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "NodeRoutineList", parentRelationshipName: "list", data, ...rest })),
                ...(await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "RoutineVersion", parentRelationshipName: "nodeLists", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                index: noNull(data.index),
                isOptional: noNull(data.isOptional),
                ...(await shapeHelper({ relation: "routineVersion", relTypes: ["Update"], isOneToOne: true, isRequired: false, objectType: "RoutineVersion", parentRelationshipName: "nodeLists", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
        visibility: {
            private: { list: NodeRoutineListModel.validate!.visibility.private },
            public: { list: NodeRoutineListModel.validate!.visibility.public },
            owner: (userId) => ({ list: NodeRoutineListModel.validate!.visibility.owner(userId) }),
};
