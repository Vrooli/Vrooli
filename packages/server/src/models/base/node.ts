import { MaxObjects, nodeValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { CustomError } from "../../events/error";
import { bestTranslation, defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { NodeFormat } from "../formats";
import { NodeModelInfo, NodeModelLogic, RoutineVersionModelInfo, RoutineVersionModelLogic } from "./types";

const __typename = "Node" as const;
const MAX_NODES_IN_ROUTINE = 100;
export const NodeModel: NodeModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.node,
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
    }),
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
                ...(await shapeHelper({ relation: "end", relTypes: ["Create"], isOneToOne: true, objectType: "NodeEnd", parentRelationshipName: "node", data, ...rest })),
                ...(await shapeHelper({ relation: "loop", relTypes: ["Create"], isOneToOne: true, objectType: "NodeLoop", parentRelationshipName: "node", data, ...rest })),
                ...(await shapeHelper({ relation: "routineList", relTypes: ["Create"], isOneToOne: true, objectType: "NodeRoutineList", parentRelationshipName: "node", data, ...rest })),
                ...(await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "RoutineVersion", parentRelationshipName: "nodes", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                id: data.id,
                columnIndex: noNull(data.columnIndex),
                nodeType: noNull(data.nodeType),
                rowIndex: noNull(data.rowIndex),
                ...(await shapeHelper({ relation: "end", relTypes: ["Create", "Update"], isOneToOne: true, objectType: "NodeEnd", parentRelationshipName: "node", data, ...rest })),
                ...(await shapeHelper({ relation: "loop", relTypes: ["Create", "Update", "Delete"], isOneToOne: true, objectType: "NodeLoop", parentRelationshipName: "node", data, ...rest })),
                ...(await shapeHelper({ relation: "routineList", relTypes: ["Create", "Update"], isOneToOne: true, objectType: "NodeRoutineList", parentRelationshipName: "node", data, ...rest })),
                ...(await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "RoutineVersion", parentRelationshipName: "nodes", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest })),
            }),
        },
        yup: nodeValidation,
    },
    search: undefined,
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, routineVersion: "RoutineVersion" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").validate().owner(data?.routineVersion as RoutineVersionModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").validate().isDeleted(data.routineVersion as RoutineVersionModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<NodeModelInfo["PrismaSelect"]>([["routineVersion", "RoutineVersion"]], ...rest),
        visibility: {
            private: { routineVersion: ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").validate().visibility.private },
            public: { routineVersion: ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").validate().visibility.public },
            owner: (userId) => ({ routineVersion: ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").validate().visibility.owner(userId) }),
        },
    }),
});
