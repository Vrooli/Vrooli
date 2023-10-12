import { scheduleRecurrenceValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { ScheduleRecurrenceFormat } from "../formats";
import { ScheduleModelInfo, ScheduleModelLogic, ScheduleRecurrenceModelInfo, ScheduleRecurrenceModelLogic } from "./types";

const __typename = "ScheduleRecurrence" as const;
export const ScheduleRecurrenceModel: ScheduleRecurrenceModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.schedule_recurrence,
    display: () => ({
        label: {
            select: () => ({ id: true, schedule: { select: ModelMap.get<ScheduleModelLogic>("Schedule").display().label.select() } }),
            get: (select, languages) => ModelMap.get<ScheduleModelLogic>("Schedule").display().label.get(select.schedule as ScheduleModelInfo["PrismaModel"], languages),
        },
    }),
    format: ScheduleRecurrenceFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    recurrenceType: data.recurrenceType,
                    interval: data.interval,
                    dayOfWeek: noNull(data.dayOfWeek),
                    dayOfMonth: noNull(data.dayOfMonth),
                    month: noNull(data.month),
                    endDate: noNull(data.endDate),
                    ...(await shapeHelper({ relation: "schedule", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "Schedule", parentRelationshipName: "recurrences", data, ...rest })),
                };
            },
            update: async ({ data }) => {
                return {
                    recurrenceType: noNull(data.recurrenceType),
                    interval: noNull(data.interval),
                    dayOfWeek: noNull(data.dayOfWeek),
                    dayOfMonth: noNull(data.dayOfMonth),
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
        maxObjects: 100000,
        permissionsSelect: () => ({ schedule: "Schedule" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<ScheduleModelLogic>("Schedule").validate().owner(data?.schedule as ScheduleModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<ScheduleModelLogic>("Schedule").validate().isDeleted(data.schedule as ScheduleModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<ScheduleRecurrenceModelInfo["PrismaSelect"]>([["schedule", "Schedule"]], ...rest),
        visibility: {
            private: { schedule: ModelMap.get<ScheduleModelLogic>("Schedule").validate().visibility.private },
            public: { schedule: ModelMap.get<ScheduleModelLogic>("Schedule").validate().visibility.public },
            owner: (userId) => ({ schedule: ModelMap.get<ScheduleModelLogic>("Schedule").validate().visibility.owner(userId) }),
        },
    }),
});
