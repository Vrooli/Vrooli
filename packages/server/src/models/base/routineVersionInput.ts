import { routineVersionInputValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { RoutineVersionInputFormat } from "../formats";
import { RoutineVersionInputModelInfo, RoutineVersionInputModelLogic, RoutineVersionModelInfo, RoutineVersionModelLogic } from "./types";

const __typename = "RoutineVersionInput" as const;
export const RoutineVersionInputModel: RoutineVersionInputModelLogic = ({
    __typename,
    dbTable: "routine_version_input",
    dbTranslationTable: "routine_version_input_translation",
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
    format: RoutineVersionInputFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                index: noNull(data.index),
                isRequired: noNull(data.isRequired),
                name: noNull(data.name),
                routineVersion: await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "RoutineVersion", parentRelationshipName: "inputs", data, ...rest }),
                standardVersion: await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "StandardVersion", parentRelationshipName: "routineVersionInputs", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                index: noNull(data.index),
                isRequired: noNull(data.isRequired),
                name: noNull(data.name),
                standardVersion: await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create", "Disconnect"], isOneToOne: true, objectType: "StandardVersion", parentRelationshipName: "routineVersionInputs", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create", "Create", "Delete"], data, ...rest }),
            }),
        },
        yup: routineVersionInputValidation,
    },
    search: undefined,
    validate: () => ({
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<RoutineVersionInputModelInfo["PrismaSelect"]>([["routineVersion", "RoutineVersion"]], ...rest),
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
