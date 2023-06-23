import { scheduleRecurrenceValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { defaultPermissions } from "../../utils";
import { ScheduleRecurrenceFormat } from "../format/scheduleRecurrence";
import { ModelLogic } from "../types";
import { ScheduleModel } from "./schedule";
import { ScheduleRecurrenceModelLogic } from "./types";

const __typename = "ScheduleRecurrence" as const;
const suppFields = [] as const;
export const ScheduleRecurrenceModel: ModelLogic<ScheduleRecurrenceModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.schedule_recurrence,
    display: {
        label: {
            select: () => ({ id: true, schedule: { select: ScheduleModel.display.label.select() } }),
            get: (select, languages) => ScheduleModel.display.label.get(select.schedule as any, languages),
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
            update: async ({ data, ...rest }) => {
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
    validate: {
        isTransferable: false,
        maxObjects: 100000,
        permissionsSelect: () => ({ schedule: "Schedule" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ScheduleModel.validate!.owner(data.schedule as any, userId),
        isDeleted: (data, languages) => ScheduleModel.validate!.isDeleted(data.schedule as any, languages),
        isPublic: (data, languages) => ScheduleModel.validate!.isPublic(data.schedule as any, languages),
        visibility: {
            private: { schedule: ScheduleModel.validate!.visibility.private },
            public: { schedule: ScheduleModel.validate!.visibility.public },
            owner: (userId) => ({ schedule: ScheduleModel.validate!.visibility.owner(userId) }),
        },
    },
});
