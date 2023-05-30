import { RunRoutineSearchInput, RunRoutineSortBy, ScheduleException, ScheduleExceptionCreateInput, ScheduleExceptionUpdateInput } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { ScheduleModel } from "./schedule";
import { ModelLogic } from "./types";

const __typename = "ScheduleException" as const;
const suppFields = [] as const;
export const ScheduleExceptionModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ScheduleExceptionCreateInput,
    GqlUpdate: ScheduleExceptionUpdateInput,
    GqlModel: ScheduleException,
    GqlPermission: object,
    GqlSearch: RunRoutineSearchInput,
    GqlSort: RunRoutineSortBy,
    PrismaCreate: Prisma.schedule_exceptionUpsertArgs["create"],
    PrismaUpdate: Prisma.schedule_exceptionUpsertArgs["update"],
    PrismaModel: Prisma.schedule_exceptionGetPayload<SelectWrap<Prisma.schedule_exceptionSelect>>,
    PrismaSelect: Prisma.schedule_exceptionSelect,
    PrismaWhere: Prisma.schedule_exceptionWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.schedule_exception,
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
