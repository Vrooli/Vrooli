import { RunRoutineSearchInput, RunRoutineSortBy, ScheduleRecurrence, ScheduleRecurrenceCreateInput, ScheduleRecurrenceUpdateInput } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { ScheduleModel } from "./schedule";
import { ModelLogic } from "./types";

const __typename = "ScheduleRecurrence" as const;
const suppFields = [] as const;
export const ScheduleRecurrenceModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ScheduleRecurrenceCreateInput,
    GqlUpdate: ScheduleRecurrenceUpdateInput,
    GqlModel: ScheduleRecurrence,
    GqlPermission: object,
    GqlSearch: RunRoutineSearchInput,
    GqlSort: RunRoutineSortBy,
    PrismaCreate: Prisma.schedule_recurrenceUpsertArgs["create"],
    PrismaUpdate: Prisma.schedule_recurrenceUpsertArgs["update"],
    PrismaModel: Prisma.schedule_recurrenceGetPayload<SelectWrap<Prisma.schedule_recurrenceSelect>>,
    PrismaSelect: Prisma.schedule_recurrenceSelect,
    PrismaWhere: Prisma.schedule_recurrenceWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.schedule_recurrence,
    display: {
        label: {
            select: () => ({ id: true, schedule: { select: ScheduleModel.display.label.select() } }),
            get: (select, languages) => ScheduleModel.display.label.get(select.schedule as any, languages),
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            schedule: "Schedule",
        },
        prismaRelMap: {
            __typename,
            schedule: "Schedule",
        },
        countFields: {},
    },
    mutate: {} as any,
    validate: {} as any,
});
