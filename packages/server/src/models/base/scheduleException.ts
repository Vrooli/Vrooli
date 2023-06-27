import { scheduleExceptionValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { defaultPermissions } from "../../utils";
import { ScheduleExceptionFormat } from "../format/scheduleException";
import { ModelLogic } from "../types";
import { ScheduleModel } from "./schedule";
import { ScheduleExceptionModelLogic } from "./types";

const __typename = "ScheduleException" as const;
const suppFields = [] as const;
export const ScheduleExceptionModel: ModelLogic<ScheduleExceptionModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.schedule_exception,
    display: {
        label: {
            select: () => ({ id: true, schedule: { select: ScheduleModel.display.label.select() } }),
            get: (select, languages) => ScheduleModel.display.label.get(select.schedule as any, languages),
        },
    },
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
    validate: {
        isTransferable: false,
        maxObjects: 100000,
        permissionsSelect: () => ({ schedule: "Schedule" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ScheduleModel.validate.owner(data.schedule as any, userId),
        isDeleted: (data, languages) => ScheduleModel.validate.isDeleted(data.schedule as any, languages),
        isPublic: (data, languages) => ScheduleModel.validate.isPublic(data.schedule as any, languages),
        visibility: {
            private: { schedule: ScheduleModel.validate.visibility.private },
            public: { schedule: ScheduleModel.validate.visibility.public },
            owner: (userId) => ({ schedule: ScheduleModel.validate.visibility.owner(userId) }),
        },
    },
});
