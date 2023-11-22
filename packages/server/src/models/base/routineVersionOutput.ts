import { routineVersionOutputValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { RoutineVersionOutputFormat } from "../formats";
import { RoutineVersionModelInfo, RoutineVersionModelLogic, RoutineVersionOutputModelInfo, RoutineVersionOutputModelLogic } from "./types";

const __typename = "RoutineVersionOutput" as const;
export const RoutineVersionOutputModel: RoutineVersionOutputModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.routine_version_output,
    display: () => ({
        label: {
            select: () => ({
                id: true,
                name: true,
                routineVersion: { select: ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").display().label.select() },
            }),
            get: (select, languages) => select.name ?? ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").display().label.get(select.routineVersion as RoutineVersionModelInfo["PrismaModel"], languages),
        },
    }),
    format: RoutineVersionOutputFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                index: noNull(data.index),
                name: noNull(data.name),
                ...(await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "RoutineVersion", parentRelationshipName: "outputs", data, ...rest })),
                ...(await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "StandardVersion", parentRelationshipName: "routineVersionOutputs", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                index: noNull(data.index),
                name: noNull(data.name),
                ...(await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create", "Disconnect"], isOneToOne: true, objectType: "StandardVersion", parentRelationshipName: "routineVersionOutputs", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Create", "Delete"], data, ...rest })),
            }),
        },
        yup: routineVersionOutputValidation,
    },
    search: undefined,
    validate: () => ({
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<RoutineVersionOutputModelInfo["PrismaSelect"]>([["routineVersion", "RoutineVersion"]], ...rest),
        isTransferable: false,
        maxObjects: 100000,
        owner: (data, userId) => ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").validate().owner(data?.routineVersion as RoutineVersionModelInfo["PrismaModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({ id: true, routineVersion: "RoutineVersion" }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ routineVersion: ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").validate().visibility.owner(userId) }),
        },
    }),
});
