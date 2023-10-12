import { scheduleExceptionValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { ScheduleExceptionFormat } from "../formats";
import { ScheduleExceptionModelInfo, ScheduleExceptionModelLogic, ScheduleModelInfo, ScheduleModelLogic } from "./types";

const __typename = "ScheduleException" as const;
export const ScheduleExceptionModel: ScheduleExceptionModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.schedule_exception,
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
                    ...(await shapeHelper({ relation: "schedule", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Schedule", parentRelationshipName: "exceptions", data, ...rest })),
                };
            },
            update: async ({ data, ...rest }) => {
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
        maxObjects: 100000,
        permissionsSelect: () => ({ schedule: "Schedule" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<ScheduleModelLogic>("Schedule").validate().owner(data?.schedule as ScheduleModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<ScheduleModelLogic>("Schedule").validate().isDeleted(data.schedule as ScheduleModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<ScheduleExceptionModelInfo["PrismaSelect"]>([["schedule", "Schedule"]], ...rest),
        visibility: {
            private: { schedule: ModelMap.get<ScheduleModelLogic>("Schedule").validate().visibility.private },
            public: { schedule: ModelMap.get<ScheduleModelLogic>("Schedule").validate().visibility.public },
            owner: (userId) => ({ schedule: ModelMap.get<ScheduleModelLogic>("Schedule").validate().visibility.owner(userId) }),
        },
    }),
});
