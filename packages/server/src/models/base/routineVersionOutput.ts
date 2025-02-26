import { MaxObjects, routineVersionOutputValidation } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { RoutineVersionOutputFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { RoutineVersionModelInfo, RoutineVersionModelLogic, RoutineVersionOutputModelInfo, RoutineVersionOutputModelLogic } from "./types.js";

const __typename = "RoutineVersionOutput" as const;
export const RoutineVersionOutputModel: RoutineVersionOutputModelLogic = ({
    __typename,
    dbTable: "routine_version_output",
    dbTranslationTable: "routine_version_output_translation",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                name: true,
                routineVersion: { select: ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").display().label.select() },
            }),
            get: (select, languages) => select.name ?? ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").display().label.get(select.routineVersion as RoutineVersionModelInfo["DbModel"], languages),
        },
    }),
    format: RoutineVersionOutputFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                index: noNull(data.index),
                name: noNull(data.name),
                routineVersion: await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "RoutineVersion", parentRelationshipName: "outputs", data, ...rest }),
                standardVersion: await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "StandardVersion", parentRelationshipName: "routineVersionOutputs", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                index: noNull(data.index),
                name: noNull(data.name),
                standardVersion: await shapeHelper({ relation: "standardVersion", relTypes: ["Connect", "Create", "Disconnect"], isOneToOne: true, objectType: "StandardVersion", parentRelationshipName: "routineVersionOutputs", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create", "Create", "Delete"], data, ...rest }),
            }),
        },
        yup: routineVersionOutputValidation,
    },
    search: undefined,
    validate: () => ({
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<RoutineVersionOutputModelInfo["DbSelect"]>([["routineVersion", "RoutineVersion"]], ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").validate().owner(data?.routineVersion as RoutineVersionModelInfo["DbModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({ id: true, routineVersion: "RoutineVersion" }),
        visibility: {
            own: function getOwn(data) {
                return {
                    routineVersion: useVisibility("RoutineVersion", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    routineVersion: useVisibility("RoutineVersion", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    routineVersion: useVisibility("RoutineVersion", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    routineVersion: useVisibility("RoutineVersion", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    routineVersion: useVisibility("RoutineVersion", "Public", data),
                };
            },
        },
    }),
});
