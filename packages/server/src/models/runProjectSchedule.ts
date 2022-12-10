import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer } from "./types";

const __typename = 'RunProjectSchedule' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.run_project_scheduleSelect,
    Prisma.run_project_scheduleGetPayload<SelectWrap<Prisma.run_project_scheduleSelect>>
> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const RunProjectScheduleModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_project_schedule,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})