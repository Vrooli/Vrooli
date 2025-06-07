import { MaxObjects, scheduleExceptionValidation } from "@vrooli/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { ScheduleExceptionFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { type ScheduleExceptionModelInfo, type ScheduleExceptionModelLogic, type ScheduleModelInfo, type ScheduleModelLogic } from "./types.js";

const __typename = "ScheduleException" as const;
export const ScheduleExceptionModel: ScheduleExceptionModelLogic = ({
    __typename,
    dbTable: "schedule_exception",
    display: () => ({
        label: {
            select: () => ({ id: true, schedule: { select: ModelMap.get<ScheduleModelLogic>("Schedule").display().label.select() } }),
            get: (select, languages) => ModelMap.get<ScheduleModelLogic>("Schedule").display().label.get(select.schedule as ScheduleModelInfo["DbModel"], languages),
        },
    }),
    format: ScheduleExceptionFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: BigInt(data.id),
                    originalStartTime: data.originalStartTime,
                    newStartTime: noNull(data.newStartTime),
                    newEndTime: noNull(data.newEndTime),
                    schedule: await shapeHelper({ relation: "schedule", relTypes: ["Connect"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "exceptions", data, ...rest }),
                };
            },
            update: async ({ data }) => {
                return {
                    originalStartTime: noNull(data.originalStartTime),
                    newStartTime: noNull(data.newStartTime),
                    newEndTime: noNull(data.newEndTime),
                };
            },
        },
        yup: scheduleExceptionValidation,
    },
    search: {} as any,
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ schedule: "Schedule" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<ScheduleModelLogic>("Schedule").validate().owner(data?.schedule as ScheduleModelInfo["DbModel"], userId),
        isDeleted: (data) => ModelMap.get<ScheduleModelLogic>("Schedule").validate().isDeleted(data.schedule as ScheduleModelInfo["DbModel"]),
        isPublic: (...rest) => oneIsPublic<ScheduleExceptionModelInfo["DbSelect"]>([["schedule", "Schedule"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    schedule: useVisibility("Schedule", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    schedule: useVisibility("Schedule", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    schedule: useVisibility("Schedule", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    schedule: useVisibility("Schedule", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    schedule: useVisibility("Schedule", "Public", data),
                };
            },
        },
    }),
});
