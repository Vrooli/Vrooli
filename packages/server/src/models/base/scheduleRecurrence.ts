import { MaxObjects, scheduleRecurrenceValidation } from "@vrooli/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { ScheduleRecurrenceFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { type ScheduleModelInfo, type ScheduleModelLogic, type ScheduleRecurrenceModelInfo, type ScheduleRecurrenceModelLogic } from "./types.js";

const __typename = "ScheduleRecurrence" as const;
export const ScheduleRecurrenceModel: ScheduleRecurrenceModelLogic = ({
    __typename,
    dbTable: "schedule_recurrence",
    display: () => ({
        label: {
            select: () => ({ id: true, schedule: { select: ModelMap.get<ScheduleModelLogic>("Schedule").display().label.select() } }),
            get: (select, languages) => ModelMap.get<ScheduleModelLogic>("Schedule").display().label.get(select.schedule as ScheduleModelInfo["DbModel"], languages),
        },
    }),
    format: ScheduleRecurrenceFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: BigInt(data.id),
                    recurrenceType: data.recurrenceType,
                    interval: data.interval,
                    dayOfWeek: noNull(data.dayOfWeek),
                    dayOfMonth: noNull(data.dayOfMonth),
                    duration: noNull(data.duration),
                    month: noNull(data.month),
                    endDate: noNull(data.endDate),
                    schedule: await shapeHelper({ relation: "schedule", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "Schedule", parentRelationshipName: "recurrences", data, ...rest }),
                };
            },
            update: async ({ data }) => {
                return {
                    recurrenceType: noNull(data.recurrenceType),
                    interval: noNull(data.interval),
                    dayOfWeek: noNull(data.dayOfWeek),
                    dayOfMonth: noNull(data.dayOfMonth),
                    duration: noNull(data.duration),
                    month: noNull(data.month),
                    endDate: noNull(data.endDate),
                };
            },
        },
        yup: scheduleRecurrenceValidation,
    },
    search: {} as any,
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ schedule: "Schedule" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<ScheduleModelLogic>("Schedule").validate().owner(data?.schedule as ScheduleModelInfo["DbModel"], userId),
        isDeleted: (data) => ModelMap.get<ScheduleModelLogic>("Schedule").validate().isDeleted(data.schedule as ScheduleModelInfo["DbModel"]),
        isPublic: (...rest) => oneIsPublic<ScheduleRecurrenceModelInfo["DbSelect"]>([["schedule", "Schedule"]], ...rest),
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
