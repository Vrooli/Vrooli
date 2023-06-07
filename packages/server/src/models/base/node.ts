import { MaxObjects, nodeValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { CustomError } from "../../events";
import { bestTranslation, defaultPermissions, translationShapeHelper } from "../../utils";
import { NodeFormat } from "../format/node";
import { ModelLogic } from "../types";
import { RoutineModel } from "./routine";
import { NodeModelLogic } from "./types";

const __typename = "Node" as const;
const MAX_NODES_IN_ROUTINE = 100;
const suppFields = [] as const;
export const NodeModel: ModelLogic<NodeModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.node,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
    },
    format: NodeFormat,
    mutate: {
        shape: {
            pre: async ({ createList, deleteList, prisma, userData }) => {
                // Don't allow more than 100 nodes in a routine
                if (createList.length) {
                    const deltaAdding = createList.length - deleteList.length;
                    if (deltaAdding < 0) return;
                    const existingCount = await prisma.routine_version.findUnique({
                        where: { id: createList[0].routineVersionConnect },
                        include: { _count: { select: { nodes: true } } },
                    });
                    const totalCount = (existingCount?._count.nodes ?? 0) + deltaAdding;
                    if (totalCount > MAX_NODES_IN_ROUTINE) {
                        throw new CustomError("0052", "MaxNodesReached", userData.languages, { totalCount });
                    }
                }
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                columnIndex: noNull(data.columnIndex),
                nodeType: data.nodeType,
                rowIndex: noNull(data.rowIndex),
                ...(await shapeHelper({ relation: "end", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "NodeEnd", parentRelationshipName: "node", data, ...rest })),
                ...(await shapeHelper({ relation: "loop", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "NodeLoop", parentRelationshipName: "node", data, ...rest })),
                ...(await shapeHelper({ relation: "routineList", relTypes: ["Create"], isOneToOne: true, isRequired: false, objectType: "NodeRoutineList", parentRelationshipName: "node", data, ...rest })),
                ...(await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "RoutineVersion", parentRelationshipName: "nodes", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                id: data.id,
                columnIndex: noNull(data.columnIndex),
                nodeType: noNull(data.nodeType),
                rowIndex: noNull(data.rowIndex),
                ...(await shapeHelper({ relation: "end", relTypes: ["Create", "Update"], isOneToOne: true, isRequired: false, objectType: "NodeEnd", parentRelationshipName: "node", data, ...rest })),
                ...(await shapeHelper({ relation: "loop", relTypes: ["Create", "Update", "Delete"], isOneToOne: true, isRequired: false, objectType: "NodeLoop", parentRelationshipName: "node", data, ...rest })),
                ...(await shapeHelper({ relation: "routineList", relTypes: ["Create", "Update"], isOneToOne: true, isRequired: false, objectType: "NodeRoutineList", parentRelationshipName: "node", data, ...rest })),
                ...(await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "RoutineVersion", parentRelationshipName: "nodes", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
        },
        yup: nodeValidation,
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ routineVersion: "RoutineVersion" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => RoutineModel.validate!.owner(data.routineVersion as any, userId),
        isDeleted: (data, languages) => RoutineModel.validate!.isDeleted(data.routineVersion as any, languages),
        isPublic: (data, languages) => RoutineModel.validate!.isPublic(data.routineVersion as any, languages),
        visibility: {
            private: { routineVersion: RoutineModel.validate!.visibility.private },
            public: { routineVersion: RoutineModel.validate!.visibility.public },
            owner: (userId) => ({ routineVersion: RoutineModel.validate!.visibility.owner(userId) }),
        },
    },
});