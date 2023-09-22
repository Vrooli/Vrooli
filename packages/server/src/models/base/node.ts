import { MaxObjects, nodeValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { CustomError } from "../../events";
import { bestTranslation, defaultPermissions } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { NodeFormat } from "../formats";
import { ModelLogic } from "../types";
import { RoutineVersionModel } from "./routineVersion";
import { NodeModelLogic, RoutineVersionModelLogic } from "./types";

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
            pre: async ({ Create, Delete, prisma, userData }) => {
                // Don't allow more than 100 nodes in a routine
                if (Create.length) {
                    const deltaAdding = Create.length - Delete.length;
                    if (deltaAdding < 0) return;
                    const existingCount = await prisma.routine_version.findUnique({
                        where: { id: Create[0].input.routineVersionConnect },
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
    search: undefined,
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, routineVersion: "RoutineVersion" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => RoutineVersionModel.validate.owner(data?.routineVersion as RoutineVersionModelLogic["PrismaModel"], userId),
        isDeleted: (data, languages) => RoutineVersionModel.validate.isDeleted(data.routineVersion as RoutineVersionModelLogic["PrismaModel"], languages),
        isPublic: (data, languages) => RoutineVersionModel.validate.isPublic(data.routineVersion as RoutineVersionModelLogic["PrismaModel"], languages),
        visibility: {
            private: { routineVersion: RoutineVersionModel.validate.visibility.private },
            public: { routineVersion: RoutineVersionModel.validate.visibility.public },
            owner: (userId) => ({ routineVersion: RoutineVersionModel.validate.visibility.owner(userId) }),
        },
    },
});
