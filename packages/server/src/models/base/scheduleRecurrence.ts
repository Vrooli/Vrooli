import { scheduleRecurrenceValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { ScheduleRecurrenceFormat } from "../formats";
import { ModelLogic } from "../types";
import { ScheduleModel } from "./schedule";
import { ScheduleModelLogic, ScheduleRecurrenceModelLogic } from "./types";

const __typename = "ScheduleRecurrence" as const;
const suppFields = [] as const;
export const ScheduleRecurrenceModel: ModelLogic<ScheduleRecurrenceModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.schedule_recurrence,
    display: {
        label: {
            select: () => ({ id: true, schedule: { select: ScheduleModel.display.label.select() } }),
            get: (select, languages) => ScheduleModel.display.label.get(select.schedule as ScheduleModelLogic["PrismaModel"], languages),
        },
    },
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
    validate: {
        isTransferable: false,
        maxObjects: 100000,
        permissionsSelect: () => ({ schedule: "Schedule" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ScheduleModel.validate.owner(data?.schedule as ScheduleModelLogic["PrismaModel"], userId),
        isDeleted: (data, languages) => ScheduleModel.validate.isDeleted(data.schedule as ScheduleModelLogic["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<ScheduleRecurrenceModelLogic["PrismaSelect"]>([["schedule", "Schedule"]], ...rest),
        visibility: {
            private: { schedule: ScheduleModel.validate.visibility.private },
            public: { schedule: ScheduleModel.validate.visibility.public },
            owner: (userId) => ({ schedule: ScheduleModel.validate.visibility.owner(userId) }),
        },
    },
});
