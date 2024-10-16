import { MaxObjects, scheduleExceptionValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { ScheduleExceptionFormat } from "../formats";
import { ScheduleExceptionModelInfo, ScheduleExceptionModelLogic, ScheduleModelInfo, ScheduleModelLogic } from "./types";

const __typename = "ScheduleException" as const;
export const ScheduleExceptionModel: ScheduleExceptionModelLogic = ({
    __typename,
    dbTable: "schedule_exception",
    display: () => ({
        label: {
            select: () => ({ id: true, schedule: { select: ModelMap.get<ScheduleModelLogic>("Schedule").display().label.select() } }),
            get: (select, languages) => ModelMap.get<ScheduleModelLogic>("Schedule").display().label.get(select.schedule as ScheduleModelInfo["PrismaModel"], languages),
        },
    }),
    format: ScheduleExceptionFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
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
        owner: (data, userId) => ModelMap.get<ScheduleModelLogic>("Schedule").validate().owner(data?.schedule as ScheduleModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<ScheduleModelLogic>("Schedule").validate().isDeleted(data.schedule as ScheduleModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<ScheduleExceptionModelInfo["PrismaSelect"]>([["schedule", "Schedule"]], ...rest),
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
