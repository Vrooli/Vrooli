import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer } from "./types";

const __typename = 'RunRoutineSchedule' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.run_routine_scheduleSelect,
    Prisma.run_routine_scheduleGetPayload<SelectWrap<Prisma.run_routine_scheduleSelect>>
> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const RunRoutineScheduleModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_routine_schedule,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})